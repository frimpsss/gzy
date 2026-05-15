package sshkeys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverFindsPrivateKeysWithPublicPairs(t *testing.T) {
	dir := t.TempDir()
	privateKey := filepath.Join(dir, "id_ed25519")
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST me@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	keys, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(keys) != 1 || keys[0].PrivatePath != privateKey || keys[0].PublicPath != publicKey {
		t.Fatalf("Discover() = %#v", keys)
	}
}

func TestReadPublicKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key.pub")
	if err := os.WriteFile(path, []byte("ssh-ed25519 AAAATEST me@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPublicKey(path)
	if err != nil {
		t.Fatalf("ReadPublicKey() error = %v", err)
	}
	if got != "ssh-ed25519 AAAATEST me@example.com" {
		t.Fatalf("ReadPublicKey() = %q", got)
	}
}

func TestCreateRunsSSHKeygen(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	fakeKeygen := filepath.Join(dir, "ssh-keygen")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + log + "\ntouch \"$6\" \"$6.pub\"\n"
	if err := os.WriteFile(fakeKeygen, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	manager := Manager{SSHKeygenPath: fakeKeygen}
	privateKey := filepath.Join(dir, "gzy_p")
	if err := manager.Create(privateKey, "me@example.com"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"-t", "ed25519", "-C", "me@example.com", "-f"} {
		if !strings.Contains(text, want) {
			t.Fatalf("ssh-keygen log missing %q: %q", want, text)
		}
	}
}
