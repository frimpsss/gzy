package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frimpsss/gzy/internal/config"
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
