package tokens

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "p", "ghu_secret-token"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := Load(dir, "p")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != "ghu_secret-token" {
		t.Fatalf("Load() = %q, want %q", got, "ghu_secret-token")
	}
}

func TestSaveCreatesDirAndPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "tokens")
	if err := Save(dir, "p", "tok"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "p"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %v, want 0600", info.Mode().Perm())
	}
}

func TestLoadMissingFileReturnsNotExist(t *testing.T) {
	_, err := Load(t.TempDir(), "absent")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load() error = %v, want os.ErrNotExist", err)
	}
}

func TestLoadTrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "p"), []byte("  ghu_tok  \n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := Load(dir, "p")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != "ghu_tok" {
		t.Fatalf("Load() = %q, want %q", got, "ghu_tok")
	}
}
