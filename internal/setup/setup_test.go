package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/frimpsss/gzy/internal/config"
)

func TestAddAccountCreatesConfigAndWrapper(t *testing.T) {
	prompts := NewStaticPrompter(map[string]string{
		"alias":      "p",
		"githubUser": "frimpsss",
		"name":       "Akwasi Frimpong",
		"email":      "me@example.com",
		"keyChoice":  "create",
		"authChoice": "manual",
	})
	store := &MemoryStore{Config: config.Config{Version: 1}}
	keys := &FakeKeys{PrivatePath: "/home/alex/.ssh/gzy_p", PublicPath: "/home/alex/.ssh/gzy_p.pub", PublicKey: "ssh-ed25519 AAAATEST"}
	wrappers := &FakeWrappers{}
	var out bytes.Buffer

	service := Service{Prompts: prompts, Store: store, Keys: keys, Wrappers: wrappers, Stdout: &out}
	if err := service.Add(); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if len(store.Config.Accounts) != 1 || store.Config.Accounts[0].Alias != "p" {
		t.Fatalf("accounts = %#v", store.Config.Accounts)
	}
	if !wrappers.Installed["p"] {
		t.Fatalf("wrapper was not installed")
	}
	if !keys.Created {
		t.Fatalf("key was not created")
	}
}

func TestAddReusesExistingKeyWhenChosen(t *testing.T) {
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "gzy_p")
	publicPath := filepath.Join(dir, "gzy_p.pub")
	if err := os.WriteFile(privatePath, []byte("existing-private"), 0o600); err != nil {
		t.Fatalf("seed private: %v", err)
	}
	if err := os.WriteFile(publicPath, []byte("existing-public"), 0o644); err != nil {
		t.Fatalf("seed public: %v", err)
	}

	prompts := NewStaticPrompter(map[string]string{
		"alias":             "p",
		"githubUser":        "frimpsss",
		"name":              "Akwasi",
		"email":             "me@example.com",
		"keyChoice":         "create",
		"existingKeyChoice": "reuse",
		"authChoice":        "manual",
	})
	store := &MemoryStore{Config: config.Config{Version: 1}}
	keys := &FakeKeys{PrivatePath: privatePath, PublicPath: publicPath, PublicKey: "ssh-ed25519 AAAATEST"}
	wrappers := &FakeWrappers{}
	var out bytes.Buffer

	service := Service{Prompts: prompts, Store: store, Keys: keys, Wrappers: wrappers, Stdout: &out}
	if err := service.Add(); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if keys.Created {
		t.Fatalf("Create should not be called when reusing existing key")
	}
	if _, err := os.Stat(privatePath); err != nil {
		t.Fatalf("private key should still exist: %v", err)
	}
	if _, err := os.Stat(publicPath); err != nil {
		t.Fatalf("public key should still exist: %v", err)
	}
}

func TestAddOverwritesExistingKeyWhenChosen(t *testing.T) {
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "gzy_p")
	publicPath := filepath.Join(dir, "gzy_p.pub")
	if err := os.WriteFile(privatePath, []byte("old-private"), 0o600); err != nil {
		t.Fatalf("seed private: %v", err)
	}
	if err := os.WriteFile(publicPath, []byte("old-public"), 0o644); err != nil {
		t.Fatalf("seed public: %v", err)
	}

	prompts := NewStaticPrompter(map[string]string{
		"alias":             "p",
		"githubUser":        "frimpsss",
		"name":              "Akwasi",
		"email":             "me@example.com",
		"keyChoice":         "create",
		"existingKeyChoice": "overwrite",
		"authChoice":        "manual",
	})
	store := &MemoryStore{Config: config.Config{Version: 1}}
	keys := &FakeKeys{PrivatePath: privatePath, PublicPath: publicPath, PublicKey: "ssh-ed25519 AAAATEST"}
	wrappers := &FakeWrappers{}
	var out bytes.Buffer

	service := Service{Prompts: prompts, Store: store, Keys: keys, Wrappers: wrappers, Stdout: &out}
	if err := service.Add(); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if !keys.Created {
		t.Fatalf("Create should be called when overwriting")
	}
	if _, err := os.Stat(privatePath); !os.IsNotExist(err) {
		t.Fatalf("private key should be removed before re-create, stat err = %v", err)
	}
	if _, err := os.Stat(publicPath); !os.IsNotExist(err) {
		t.Fatalf("public key should be removed before re-create, stat err = %v", err)
	}
}

func TestAuthStoresGitHubKeyID(t *testing.T) {
	store := &MemoryStore{Config: config.Config{Version: 1, Accounts: []config.Account{{
		Alias: "p", PublicKey: "/home/alex/.ssh/gzy_p.pub",
	}}}}
	keys := &FakeKeys{PublicKey: "ssh-ed25519 AAAATEST"}
	github := &FakeGitHub{KeyID: 99}
	service := Service{Store: store, Keys: keys, GitHub: github}
	if err := service.Auth("p"); err != nil {
		t.Fatalf("Auth() error = %v", err)
	}
	if store.Config.Accounts[0].GitHubKeyID != 99 {
		t.Fatalf("GitHubKeyID = %d", store.Config.Accounts[0].GitHubKeyID)
	}
}

type StaticPrompter struct{ Answers map[string]string }

func NewStaticPrompter(answers map[string]string) StaticPrompter {
	return StaticPrompter{Answers: answers}
}

func (p StaticPrompter) AskRequired(key string, label string) (string, error) {
	return p.Answers[key], nil
}

func (p StaticPrompter) AskWithDefault(key string, label string, def string) (string, error) {
	if v, ok := p.Answers[key]; ok && v != "" {
		return v, nil
	}
	return def, nil
}

func (p StaticPrompter) AskChoice(key string, label string, choices []Choice) (string, error) {
	return p.Answers[key], nil
}

type MemoryStore struct{ Config config.Config }

func (s *MemoryStore) Load() (config.Config, error) { return s.Config, nil }
func (s *MemoryStore) Save(cfg config.Config) error { s.Config = cfg; return nil }

type FakeKeys struct {
	PrivatePath string
	PublicPath  string
	PublicKey   string
	Created     bool
}

func (k *FakeKeys) DefaultKeyPair(alias string) (string, string)  { return k.PrivatePath, k.PublicPath }
func (k *FakeKeys) Create(privatePath string, email string) error { k.Created = true; return nil }
func (k *FakeKeys) ReadPublic(publicPath string) (string, error)  { return k.PublicKey, nil }

type FakeWrappers struct{ Installed map[string]bool }

func (w *FakeWrappers) Install(alias string) error {
	if w.Installed == nil {
		w.Installed = map[string]bool{}
	}
	w.Installed[alias] = true
	return nil
}

type FakeGitHub struct{ KeyID int64 }

func (g *FakeGitHub) UploadWithDeviceFlow(alias string, publicKey string) (int64, error) {
	return g.KeyID, nil
}
