package gitrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frimpsss/gzy/internal/config"
)

func TestBuildCommandAddsIdentityAndGitArgs(t *testing.T) {
	account := config.Account{Alias: "p", Name: "Akwasi Frimpong", Email: "me@example.com", PrivateKey: "/tmp/key"}
	cmd := BuildCommand(account, []string{"status"}, "")
	wantArgs := []string{
		"-c", "user.name=Akwasi Frimpong",
		"-c", "user.email=me@example.com",
		"status",
	}
	if strings.Join(cmd.Args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("Args = %#v, want %#v", cmd.Args, wantArgs)
	}
	if !strings.Contains(cmd.Env["GIT_SSH_COMMAND"], "ssh -i /tmp/key -o IdentitiesOnly=yes") {
		t.Fatalf("GIT_SSH_COMMAND = %q", cmd.Env["GIT_SSH_COMMAND"])
	}
}

func TestBuildCommandWithGZYPathAddsCredentialHelper(t *testing.T) {
	account := config.Account{Alias: "p", Name: "Akwasi", Email: "me@example.com", PrivateKey: "/tmp/key"}
	cmd := BuildCommand(account, []string{"pull"}, "/usr/local/bin/gzy")
	joined := strings.Join(cmd.Args, "|")
	if !strings.Contains(joined, "credential.https://github.com.helper=") {
		t.Fatalf("expected credential helper reset; args = %#v", cmd.Args)
	}
	if !strings.Contains(joined, "credential.https://github.com.helper=!/usr/local/bin/gzy credential p") {
		t.Fatalf("expected credential helper to invoke gzy credential; args = %#v", cmd.Args)
	}
}

func TestRunUsesFakeGitBinary(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	fakeGit := filepath.Join(dir, "git")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + log + "\nprintf '%s\\n' \"$GIT_SSH_COMMAND\" >> " + log + "\n"
	if err := os.WriteFile(fakeGit, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runner := Runner{GitPath: fakeGit}
	account := config.Account{Name: "Name", Email: "email@example.com", PrivateKey: "/tmp/key"}
	code, err := runner.Run(account, []string{"status"})
	if err != nil || code != 0 {
		t.Fatalf("Run() code=%d err=%v", code, err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "user.email=email@example.com") || !strings.Contains(text, "ssh -i /tmp/key") {
		t.Fatalf("fake git log = %q", text)
	}
}
