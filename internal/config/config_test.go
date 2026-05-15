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
	for _, alias := range []string{"", "bad alias", "../x", "git", "git-p", ".hidden", "x/y", strings.Repeat("a", 41)} {
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

func TestAddAccountFillsCommandAndVersion(t *testing.T) {
	var cfg Config

	if err := cfg.AddAccount(Account{Alias: "p"}); err != nil {
		t.Fatalf("AddAccount returned error: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}
	if len(cfg.Accounts) != 1 {
		t.Fatalf("len(Accounts) = %d, want 1", len(cfg.Accounts))
	}
	if got, want := cfg.Accounts[0].Command, "git-p"; got != want {
		t.Fatalf("Command = %q, want %q", got, want)
	}
}

func TestFindReturnsAccountWhenAliasExists(t *testing.T) {
	cfg := Config{
		Accounts: []Account{
			{Alias: "p", Command: "git-p"},
			{Alias: "work", Command: "git-work"},
		},
	}

	got, ok := cfg.Find("work")
	if !ok {
		t.Fatal("Find returned ok=false, want true")
	}
	if got.Alias != "work" {
		t.Fatalf("Alias = %q, want %q", got.Alias, "work")
	}
}

func TestFindReturnsFalseWhenAliasDoesNotExist(t *testing.T) {
	cfg := Config{
		Accounts: []Account{{Alias: "p", Command: "git-p"}},
	}

	got, ok := cfg.Find("missing")
	if ok {
		t.Fatalf("Find returned ok=true with account %#v, want false", got)
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
	if !strings.Contains(output, "\n  \"version\"") {
		t.Fatalf("Export output is not indented: %s", output)
	}
	if !strings.Contains(output, "\n      \"alias\"") {
		t.Fatalf("Export account output is not indented: %s", output)
	}
	if strings.Contains(output, privateKeyContents) {
		t.Fatalf("Export output included private key contents: %s", output)
	}
}

func TestImportParsesJSONAndDefaultsVersion(t *testing.T) {
	got, err := Import([]byte(`{
  "accounts": [
    {
      "alias": "p",
      "command": "git-p",
      "github_user": "frimpsss"
    }
  ]
}`))
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}

	if got.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d", got.Version, CurrentVersion)
	}
	if len(got.Accounts) != 1 {
		t.Fatalf("len(Accounts) = %d, want 1", len(got.Accounts))
	}
	if got.Accounts[0].Alias != "p" {
		t.Fatalf("Alias = %q, want %q", got.Accounts[0].Alias, "p")
	}
	if got.Accounts[0].GitHubUser != "frimpsss" {
		t.Fatalf("GitHubUser = %q, want %q", got.Accounts[0].GitHubUser, "frimpsss")
	}
}

func TestImportRejectsInvalidAlias(t *testing.T) {
	_, err := Import([]byte(`{
  "version": 1,
  "accounts": [
    {
      "alias": "../x",
      "command": "git-../x"
    }
  ]
}`))
	if err == nil {
		t.Fatal("Import returned nil error for invalid alias")
	}
}
