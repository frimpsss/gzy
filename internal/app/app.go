package app

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/frimpsss/gzy/internal/config"
	"github.com/frimpsss/gzy/internal/doctor"
	"github.com/frimpsss/gzy/internal/githubauth"
	"github.com/frimpsss/gzy/internal/gitrun"
	"github.com/frimpsss/gzy/internal/paths"
	"github.com/frimpsss/gzy/internal/setup"
	"github.com/frimpsss/gzy/internal/sshkeys"
	"github.com/frimpsss/gzy/internal/wrapper"
)

type Config struct {
	ConfigPath     string
	BinDir         string
	GZYPath        string
	GOOS           string
	GitHubClientID string
	Stdin          io.Reader
	Stdout         io.Writer
}

type App struct {
	cfg Config
}

func New(cfg Config) *App {
	if cfg.GOOS == "" {
		cfg.GOOS = runtime.GOOS
	}
	if cfg.Stdin == nil {
		cfg.Stdin = os.Stdin
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	return &App{cfg: cfg}
}

func (a *App) Init() error { return a.Add() }

func (a *App) Add() error {
	return a.setupService().Add()
}

func (a *App) List() ([]config.Account, error) {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return nil, err
	}
	return cfg.Accounts, nil
}

func (a *App) Remove(alias string) error {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return err
	}
	kept := cfg.Accounts[:0]
	found := false
	for _, account := range cfg.Accounts {
		if account.Alias == alias {
			found = true
			continue
		}
		kept = append(kept, account)
	}
	if !found {
		return fmt.Errorf("alias %q is not configured", alias)
	}
	cfg.Accounts = kept
	if err := config.Save(a.cfg.ConfigPath, cfg); err != nil {
		return err
	}
	return wrapper.Remove(a.cfg.GOOS, a.cfg.BinDir, alias)
}

func (a *App) Install() error {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return err
	}
	for _, account := range cfg.Accounts {
		if _, err := wrapper.Install(a.cfg.GOOS, a.cfg.BinDir, account.Alias, a.cfg.GZYPath); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) Export() ([]byte, error) {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return nil, err
	}
	return config.Export(cfg)
}

func (a *App) Import(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cfg, err := config.Import(data)
	if err != nil {
		return err
	}
	return config.Save(a.cfg.ConfigPath, cfg)
}

func (a *App) Auth(alias string) error {
	return a.setupService().Auth(alias)
}

func (a *App) Doctor() ([]string, error) {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return nil, err
	}
	result := doctor.Checker{}.Check(cfg)
	lines := make([]string, 0, len(result.Checks))
	for _, check := range result.Checks {
		status := "FAIL"
		if check.OK {
			status = "OK"
		}
		lines = append(lines, fmt.Sprintf("%s   %s", status, check.Name))
	}
	return lines, nil
}

func (a *App) RunGit(alias string, args []string) (int, error) {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return 1, err
	}
	account, ok := cfg.Find(alias)
	if !ok {
		return 1, fmt.Errorf("alias %q is not configured", alias)
	}
	return gitrun.Runner{}.Run(account, args)
}

func (a *App) setupService() setup.Service {
	return setup.Service{
		Prompts:  terminalPrompter{in: a.cfg.Stdin, out: a.cfg.Stdout},
		Store:    fileStore{path: a.cfg.ConfigPath},
		Keys:     keyAdapter{paths: paths.FromOS(), manager: sshkeys.Manager{}},
		Wrappers: wrapperAdapter{goos: a.cfg.GOOS, binDir: a.cfg.BinDir, gzyPath: a.cfg.GZYPath},
		GitHub:   githubAdapter{clientID: a.cfg.GitHubClientID, out: a.cfg.Stdout},
		Stdout:   a.cfg.Stdout,
	}
}

type terminalPrompter struct {
	in  io.Reader
	out io.Writer
}

func (p terminalPrompter) Ask(key string, label string) (string, error) {
	fmt.Fprintf(p.out, "%s: ", label)
	reader := bufio.NewReader(p.in)
	text, err := reader.ReadString('\n')
	if err != nil && len(text) == 0 {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

type fileStore struct{ path string }

func (s fileStore) Load() (config.Config, error) { return config.Load(s.path) }
func (s fileStore) Save(cfg config.Config) error { return config.Save(s.path, cfg) }

type keyAdapter struct {
	paths   paths.Paths
	manager sshkeys.Manager
}

func (k keyAdapter) DefaultKeyPair(alias string) (string, string)  { return k.paths.DefaultKeyPair(alias) }
func (k keyAdapter) Create(privatePath string, email string) error { return k.manager.Create(privatePath, email) }
func (k keyAdapter) ReadPublic(publicPath string) (string, error)  { return sshkeys.ReadPublicKey(publicPath) }

type wrapperAdapter struct {
	goos    string
	binDir  string
	gzyPath string
}

func (w wrapperAdapter) Install(alias string) error {
	_, err := wrapper.Install(w.goos, w.binDir, alias, w.gzyPath)
	return err
}

type githubAdapter struct {
	clientID string
	out      io.Writer
}

func (g githubAdapter) UploadWithDeviceFlow(title string, publicKey string) (int64, error) {
	client := githubauth.Client{ClientID: g.clientID, HTTP: http.DefaultClient}
	code, err := client.StartDeviceFlow()
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(g.out, "Open %s and enter code %s\n", code.VerificationURI, code.UserCode)
	token, err := client.WaitForToken(code, time.Now().Add(time.Duration(code.ExpiresIn)*time.Second))
	if err != nil {
		return 0, err
	}
	key, err := client.UploadKey(token.AccessToken, title, publicKey)
	if err != nil {
		return 0, err
	}
	return key.ID, nil
}
