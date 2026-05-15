package app

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
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
		Prompts:     newTerminalPrompter(a.cfg.Stdin, a.cfg.Stdout),
		Store:       fileStore{path: a.cfg.ConfigPath},
		Keys:        keyAdapter{paths: paths.FromOS(), manager: sshkeys.Manager{}},
		Wrappers:    wrapperAdapter{goos: a.cfg.GOOS, binDir: a.cfg.BinDir, gzyPath: a.cfg.GZYPath},
		GitHub:      githubAdapter{clientID: a.cfg.GitHubClientID, out: a.cfg.Stdout},
		Stdout:      a.cfg.Stdout,
		GitIdentity: defaultGitIdentity,
	}
}

type terminalPrompter struct {
	in  *bufio.Reader
	out io.Writer
}

func newTerminalPrompter(in io.Reader, out io.Writer) terminalPrompter {
	return terminalPrompter{in: bufio.NewReader(in), out: out}
}

func (p terminalPrompter) AskRequired(key string, label string) (string, error) {
	for {
		fmt.Fprintf(p.out, "%s: ", label)
		text, err := p.readLine()
		if err != nil && text == "" {
			return "", err
		}
		if text != "" {
			return text, nil
		}
		fmt.Fprintln(p.out, "  please enter a value")
	}
}

func (p terminalPrompter) AskWithDefault(key string, label string, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	text, err := p.readLine()
	if err != nil && text == "" {
		return "", err
	}
	if text == "" {
		if def == "" {
			return p.AskRequired(key, label)
		}
		return def, nil
	}
	return text, nil
}

func (p terminalPrompter) AskChoice(key string, label string, choices []setup.Choice) (string, error) {
	for {
		fmt.Fprintln(p.out, label+":")
		for i, c := range choices {
			fmt.Fprintf(p.out, "  %d) %s\n", i+1, c.Label)
		}
		fmt.Fprintf(p.out, "Choose 1-%d: ", len(choices))
		text, err := p.readLine()
		if err != nil && text == "" {
			return "", err
		}
		n, convErr := strconv.Atoi(text)
		if convErr == nil && n >= 1 && n <= len(choices) {
			return choices[n-1].Value, nil
		}
		fmt.Fprintf(p.out, "  please enter a number between 1 and %d\n", len(choices))
	}
}

func (p terminalPrompter) readLine() (string, error) {
	line, err := p.in.ReadString('\n')
	return strings.TrimSpace(line), err
}

func defaultGitIdentity() (string, string) {
	name, _ := exec.Command("git", "config", "--global", "user.name").Output()
	email, _ := exec.Command("git", "config", "--global", "user.email").Output()
	return strings.TrimSpace(string(name)), strings.TrimSpace(string(email))
}

type fileStore struct{ path string }

func (s fileStore) Load() (config.Config, error) { return config.Load(s.path) }
func (s fileStore) Save(cfg config.Config) error { return config.Save(s.path, cfg) }

type keyAdapter struct {
	paths   paths.Paths
	manager sshkeys.Manager
}

func (k keyAdapter) DefaultKeyPair(alias string) (string, string) {
	return k.paths.DefaultKeyPair(alias)
}
func (k keyAdapter) Create(privatePath string, email string) error {
	return k.manager.Create(privatePath, email)
}
func (k keyAdapter) ReadPublic(publicPath string) (string, error) {
	return sshkeys.ReadPublicKey(publicPath)
}

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
	fmt.Fprintf(g.out, "\nVisit %s and enter code: %s\n", code.VerificationURI, code.UserCode)
	if err := openBrowser(code.VerificationURI); err == nil {
		fmt.Fprintln(g.out, "(opened your browser; set GZY_NO_BROWSER=1 to skip)")
	}
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

func openBrowser(url string) error {
	if os.Getenv("GZY_NO_BROWSER") != "" {
		return fmt.Errorf("browser open disabled by GZY_NO_BROWSER")
	}
	bin, args := browserCommand(runtime.GOOS, url)
	if bin == "" {
		return fmt.Errorf("no browser opener for %s", runtime.GOOS)
	}
	return exec.Command(bin, args...).Start()
}

func browserCommand(goos string, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "cmd", []string{"/c", "start", "", url}
	case "linux":
		return "xdg-open", []string{url}
	default:
		return "", nil
	}
}
