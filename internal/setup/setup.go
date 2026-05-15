package setup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/frimpsss/gzy/internal/config"
)

type Choice struct {
	Value string
	Label string
}

type Prompter interface {
	AskRequired(key string, label string) (string, error)
	AskWithDefault(key string, label string, def string) (string, error)
	AskChoice(key string, label string, choices []Choice) (string, error)
}

type Store interface {
	Load() (config.Config, error)
	Save(config.Config) error
}

type KeyManager interface {
	DefaultKeyPair(alias string) (string, string)
	Create(privatePath string, email string) error
	ReadPublic(publicPath string) (string, error)
}

type WrapperInstaller interface {
	Install(alias string) error
}

type GitHubAuthenticator interface {
	UploadWithDeviceFlow(title string, publicKey string) (int64, error)
}

type Service struct {
	Prompts     Prompter
	Store       Store
	Keys        KeyManager
	Wrappers    WrapperInstaller
	GitHub      GitHubAuthenticator
	Stdout      io.Writer
	Now         func() time.Time
	GitIdentity func() (name string, email string)
}

func (s Service) Init() error {
	return s.Add()
}

func (s Service) Add() error {
	alias, err := s.Prompts.AskRequired("alias", "Alias suffix (e.g. 'p' creates the command 'git-p')")
	if err != nil {
		return err
	}
	if err := config.ValidateAlias(alias); err != nil {
		return err
	}
	githubUser, err := s.Prompts.AskRequired("githubUser", "GitHub username for this account")
	if err != nil {
		return err
	}

	defaultName, defaultEmail := s.gitIdentity()
	name, err := s.Prompts.AskWithDefault("name", "Git commit name (the Author shown in 'git log')", defaultName)
	if err != nil {
		return err
	}
	email, err := s.Prompts.AskWithDefault("email", "Git commit email (the Author email shown in 'git log')", defaultEmail)
	if err != nil {
		return err
	}

	keyChoice, err := s.Prompts.AskChoice("keyChoice", "SSH key for this account", []Choice{
		{Value: "create", Label: "Create a new ed25519 SSH key (recommended)"},
		{Value: "existing", Label: "Use an existing SSH key already on disk"},
	})
	if err != nil {
		return err
	}

	privateKey, publicKey := s.Keys.DefaultKeyPair(alias)
	switch keyChoice {
	case "create":
		if err := s.Keys.Create(privateKey, email); err != nil {
			return err
		}
	case "existing":
		defaultExisting := defaultExistingKeyPath()
		privateKey, err = s.Prompts.AskWithDefault("privateKeyPath", "Path to existing private key", defaultExisting)
		if err != nil {
			return err
		}
		publicKey = privateKey + ".pub"
	}

	authChoice, err := s.Prompts.AskChoice("authChoice", "Register this SSH key with GitHub", []Choice{
		{Value: "browser", Label: "Browser: open GitHub and upload the key automatically"},
		{Value: "manual", Label: "Manual: print the public key for you to paste at github.com/settings/keys"},
	})
	if err != nil {
		return err
	}

	account := config.Account{
		Alias:      alias,
		Command:    "git-" + alias,
		GitHubUser: githubUser,
		Name:       name,
		Email:      email,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		CreatedAt:  s.now().UTC().Format(time.RFC3339),
	}

	cfg, err := s.Store.Load()
	if err != nil {
		return err
	}
	if err := cfg.AddAccount(account); err != nil {
		return err
	}
	if authChoice == "browser" && s.GitHub != nil {
		keyID, err := s.uploadKey(account)
		if err != nil {
			return err
		}
		cfg.Accounts[len(cfg.Accounts)-1].GitHubKeyID = keyID
	} else if authChoice == "manual" && s.Stdout != nil {
		pub, err := s.Keys.ReadPublic(account.PublicKey)
		if err == nil {
			fmt.Fprintf(s.Stdout, "\nAdd this public key at https://github.com/settings/keys :\n\n%s\n\n", pub)
		}
	}
	if err := s.Store.Save(cfg); err != nil {
		return err
	}
	return s.Wrappers.Install(alias)
}

func (s Service) Auth(alias string) error {
	cfg, err := s.Store.Load()
	if err != nil {
		return err
	}
	for i, account := range cfg.Accounts {
		if account.Alias == alias {
			keyID, err := s.uploadKey(account)
			if err != nil {
				return err
			}
			cfg.Accounts[i].GitHubKeyID = keyID
			return s.Store.Save(cfg)
		}
	}
	return fmt.Errorf("alias %q is not configured", alias)
}

func (s Service) uploadKey(account config.Account) (int64, error) {
	if s.GitHub == nil {
		return 0, errors.New("GitHub browser authentication is not configured")
	}
	publicKey, err := s.Keys.ReadPublic(account.PublicKey)
	if err != nil {
		return 0, err
	}
	return s.GitHub.UploadWithDeviceFlow("gzy-"+account.Alias, publicKey)
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s Service) gitIdentity() (string, string) {
	if s.GitIdentity != nil {
		return s.GitIdentity()
	}
	return "", ""
}

func defaultExistingKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "id_ed25519")
}
