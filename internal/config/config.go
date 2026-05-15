package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const CurrentVersion = 1

var aliasPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

type Config struct {
	Version  int       `json:"version"`
	Accounts []Account `json:"accounts"`
}

type Account struct {
	Alias       string `json:"alias"`
	Command     string `json:"command"`
	GitHubUser  string `json:"github_user"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
	GitHubKeyID int64  `json:"github_key_id,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func ValidateAlias(alias string) error {
	if len(alias) > 40 {
		return fmt.Errorf("alias %q is longer than 40 characters", alias)
	}
	if alias == "git" || strings.HasPrefix(alias, "git-") {
		return fmt.Errorf("alias %q is reserved", alias)
	}
	if !aliasPattern.MatchString(alias) {
		return fmt.Errorf("alias %q must start with a letter or number and contain only letters, numbers, underscores, or hyphens", alias)
	}
	return nil
}

func (cfg *Config) AddAccount(account Account) error {
	if err := ValidateAlias(account.Alias); err != nil {
		return err
	}
	if _, ok := cfg.Find(account.Alias); ok {
		return fmt.Errorf("account alias %q already exists", account.Alias)
	}
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	if account.Command == "" {
		account.Command = "git-" + account.Alias
	}
	cfg.Accounts = append(cfg.Accounts, account)
	return nil
}

func (cfg Config) Find(alias string) (Account, bool) {
	for _, account := range cfg.Accounts {
		if account.Alias == alias {
			return account, true
		}
	}
	return Account{}, false
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{Version: CurrentVersion}, nil
	}
	if err != nil {
		return Config{}, err
	}
	return Import(data)
}

func Save(path string, cfg Config) error {
	data, err := Export(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

func Export(cfg Config) ([]byte, error) {
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func Import(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.Version < 0 {
		return fmt.Errorf("config version %d is invalid", cfg.Version)
	}
	if cfg.Version > CurrentVersion {
		return fmt.Errorf("config version %d is newer than supported version %d", cfg.Version, CurrentVersion)
	}
	seen := make(map[string]struct{}, len(cfg.Accounts))
	for _, account := range cfg.Accounts {
		if err := ValidateAlias(account.Alias); err != nil {
			return err
		}
		if _, ok := seen[account.Alias]; ok {
			return fmt.Errorf("account alias %q already exists", account.Alias)
		}
		seen[account.Alias] = struct{}{}
	}
	return nil
}
