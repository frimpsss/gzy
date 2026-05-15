package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAliasAcceptsValidAliases(t *testing.T) {
	for _, alias := range []string{"p", "work", "client-1", "client_2"} {
		t.Run(alias, func(t *testing.T) {
			if err := ValidateAlias(alias); err != nil {
				t.Fatalf("ValidateAlias(%q) returned error: %v", alias, err)
			}
		})
	}
}

func TestValidateAliasRejectsInvalidAliases(t *testing.T) {
	for _, alias := range []string{"", "bad alias", "../x", "git-p", ".hidden", "x/y"} {
		t.Run(alias, func(t *testing.T) {
			if err := ValidateAlias(alias); err == nil {
				t.Fatalf("ValidateAlias(%q) returned nil error", alias)
			}
		})
	}
}

func TestSaveLoadRoundTripsConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := Config{
		Version: 1,
		Accounts: []Account{
			{
				Alias:       "p",
				Command:     "git-p",
				GitHubUser:  "frimpsss",
				Name:        "Akwasi Frimpong",
				Email:       "84427404+frimpsss@users.noreply.github.com",
				PrivateKey:  "~/.ssh/gzy_p",
				PublicKey:   "~/.ssh/gzy_p.pub",
				GitHubKeyID: 123,
				CreatedAt:   "2026-05-15T00:00:00Z",
			},
		},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.Version != want.Version {
		t.Fatalf("Version = %d, want %d", got.Version, want.Version)
	}
	if len(got.Accounts) != 1 {
		t.Fatalf("len(Accounts) = %d, want 1", len(got.Accounts))
	}
	if got.Accounts[0] != want.Accounts[0] {
		t.Fatalf("Account = %#v, want %#v", got.Accounts[0], want.Accounts[0])
	}
}

func TestLoadMissingConfigReturnsEmptyCurrentConfig(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing", "config.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d", got.Version, CurrentVersion)
	}
	if len(got.Accounts) != 0 {
		t.Fatalf("len(Accounts) = %d, want 0", len(got.Accounts))
	}
}

func TestAddAccountRejectsDuplicateAlias(t *testing.T) {
	var cfg Config
	if err := cfg.AddAccount(Account{Alias: "p"}); err != nil {
		t.Fatalf("AddAccount returned error: %v", err)
	}

	if err := cfg.AddAccount(Account{Alias: "p"}); err == nil {
		t.Fatal("AddAccount returned nil error for duplicate alias")
	}
}

func TestExportIncludesPrivateKeyPathButNotContents(t *testing.T) {
	privateKeyPath := filepath.Join(t.TempDir(), "gzy_p")
	privateKeyContents := "BEGIN OPENSSH PRIVATE KEY"
	if err := os.WriteFile(privateKeyPath, []byte(privateKeyContents), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	data, err := Export(Config{
		Version: CurrentVersion,
		Accounts: []Account{
			{
				Alias:      "p",
				Command:    "git-p",
				PrivateKey: privateKeyPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, privateKeyPath) {
		t.Fatalf("Export output missing private key path %q: %s", privateKeyPath, output)
	}
	if strings.Contains(output, privateKeyContents) {
		t.Fatalf("Export output included private key contents: %s", output)
	}
}
