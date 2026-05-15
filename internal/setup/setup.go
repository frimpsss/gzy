package setup

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/frimpsss/gzy/internal/config"
)

type Prompter interface {
	Ask(key string, label string) (string, error)
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
	Prompts  Prompter
	Store    Store
	Keys     KeyManager
	Wrappers WrapperInstaller
	GitHub   GitHubAuthenticator
	Stdout   io.Writer
	Now      func() time.Time
}

func (s Service) Init() error {
	return s.Add()
}

func (s Service) Add() error {
	alias, err := s.Prompts.Ask("alias", "Alias command suffix")
	if err != nil {
		return err
	}
	if err := config.ValidateAlias(alias); err != nil {
		return err
	}
	githubUser, err := s.Prompts.Ask("githubUser", "GitHub username")
	if err != nil {
		return err
	}
	name, err := s.Prompts.Ask("name", "Git commit name")
	if err != nil {
		return err
	}
	email, err := s.Prompts.Ask("email", "Git commit email")
	if err != nil {
		return err
	}
	keyChoice, err := s.Prompts.Ask("keyChoice", "SSH key choice")
	if err != nil {
		return err
	}
	authChoice, err := s.Prompts.Ask("authChoice", "GitHub authentication method")
	if err != nil {
		return err
	}

	privateKey, publicKey := s.Keys.DefaultKeyPair(alias)
	if keyChoice == "create" {
		if err := s.Keys.Create(privateKey, email); err != nil {
			return err
		}
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
