package setup

import (
	"bytes"
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

func (g *FakeGitHub) UploadWithDeviceFlow(title string, publicKey string) (int64, error) {
	return g.KeyID, nil
}
