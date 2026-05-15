package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frimpsss/gzy/internal/config"
	"github.com/frimpsss/gzy/internal/setup"
)

func TestListExportImportAndInstall(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	binDir := filepath.Join(dir, "bin")
	input := config.Config{Version: 1, Accounts: []config.Account{{
		Alias: "p", Command: "git-p", GitHubUser: "frimpsss", Name: "Akwasi", Email: "me@example.com", PrivateKey: "/tmp/key", PublicKey: "/tmp/key.pub",
	}}}
	if err := config.Save(cfgPath, input); err != nil {
		t.Fatal(err)
	}
	app := New(Config{ConfigPath: cfgPath, BinDir: binDir, GZYPath: "/usr/local/bin/gzy", GOOS: "linux"})
	accounts, err := app.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(accounts) != 1 || accounts[0].Alias != "p" {
		t.Fatalf("accounts = %#v", accounts)
	}
	data, err := app.Export()
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if !strings.Contains(string(data), `"alias": "p"`) {
		t.Fatalf("export = %s", data)
	}
	if err := app.Install(); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "git-p")); err != nil {
		t.Fatalf("wrapper missing: %v", err)
	}
}

func TestRemoveDeletesAccountAndWrapper(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	binDir := filepath.Join(dir, "bin")
	if err := config.Save(cfgPath, config.Config{Version: 1, Accounts: []config.Account{{Alias: "p", Command: "git-p"}}}); err != nil {
		t.Fatal(err)
	}
	app := New(Config{ConfigPath: cfgPath, BinDir: binDir, GZYPath: "/usr/local/bin/gzy", GOOS: "linux"})
	if err := app.Install(); err != nil {
		t.Fatal(err)
	}
	if err := app.Remove("p"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Accounts) != 0 {
		t.Fatalf("accounts after remove = %#v", loaded.Accounts)
	}
}

func TestTerminalPrompterAskWithDefaultUsesDefaultOnEmptyInput(t *testing.T) {
	p := newTerminalPrompter(strings.NewReader("\n"), io.Discard)
	got, err := p.AskWithDefault("k", "Name", "Akwasi")
	if err != nil {
		t.Fatalf("AskWithDefault error = %v", err)
	}
	if got != "Akwasi" {
		t.Fatalf("got %q, want %q", got, "Akwasi")
	}
}

func TestTerminalPrompterAskWithDefaultPrefersTypedValue(t *testing.T) {
	p := newTerminalPrompter(strings.NewReader("Bob\n"), io.Discard)
	got, err := p.AskWithDefault("k", "Name", "Akwasi")
	if err != nil {
		t.Fatalf("AskWithDefault error = %v", err)
	}
	if got != "Bob" {
		t.Fatalf("got %q, want Bob", got)
	}
}

func TestTerminalPrompterAskRequiredRetriesOnEmptyInput(t *testing.T) {
	p := newTerminalPrompter(strings.NewReader("\n   \nfoo\n"), io.Discard)
	got, err := p.AskRequired("k", "Name")
	if err != nil {
		t.Fatalf("AskRequired error = %v", err)
	}
	if got != "foo" {
		t.Fatalf("got %q, want foo", got)
	}
}

func TestTerminalPrompterAskChoiceRetriesOnInvalidInputAndPrintsMenu(t *testing.T) {
	var out bytes.Buffer
	p := newTerminalPrompter(strings.NewReader("99\nabc\n2\n"), &out)
	got, err := p.AskChoice("k", "Pick one", []setup.Choice{
		{Value: "a", Label: "Option A"},
		{Value: "b", Label: "Option B"},
	})
	if err != nil {
		t.Fatalf("AskChoice error = %v", err)
	}
	if got != "b" {
		t.Fatalf("got %q, want b", got)
	}
	output := out.String()
	if !strings.Contains(output, "1) Option A") || !strings.Contains(output, "2) Option B") {
		t.Fatalf("menu missing from output:\n%s", output)
	}
}
