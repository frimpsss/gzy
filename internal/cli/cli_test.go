package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/frimpsss/gzy/internal/config"
)

func TestRunNoArgsPrintsHelp(t *testing.T) {
	var out bytes.Buffer

	code := Run([]string{}, &out, &out, Deps{})

	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}

	output := out.String()
	for _, want := range []string{"gzy", "init", "add", "run", "doctor"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output %q missing %q", output, want)
		}
	}
}

func TestRunVersionPrintsVersion(t *testing.T) {
	var out bytes.Buffer

	code := Run([]string{"version"}, &out, &out, Deps{Version: "dev-test"})

	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}

	if got := strings.TrimSpace(out.String()); got != "gzy dev-test" {
		t.Fatalf("output = %q, want %q", got, "gzy dev-test")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"wat"}, &stdout, &stderr, Deps{Version: "dev-test"})

	if code != 2 {
		t.Fatalf("Run returned code %d, want 2", code)
	}

	if got := stderr.String(); !strings.Contains(got, "unknown command: wat") {
		t.Fatalf("stderr %q missing unknown command message", got)
	}
}

func TestRunDelegatesGitArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := &fakeApp{RunCode: 0}
	deps := Deps{Version: "dev-test", App: app}
	code := Run([]string{"run", "p", "--", "status"}, &stdout, &stderr, deps)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%q", code, stderr.String())
	}
	if app.RunAlias != "p" || strings.Join(app.RunArgs, " ") != "status" {
		t.Fatalf("run alias=%q args=%#v", app.RunAlias, app.RunArgs)
	}
}

func TestListPrintsAccounts(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := Deps{App: &fakeApp{Accounts: []config.Account{{Alias: "p", Command: "git-p", GitHubUser: "frimpsss", Email: "me@example.com"}}}}
	code := Run([]string{"list"}, &stdout, &stderr, deps)
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "git-p") || !strings.Contains(stdout.String(), "me@example.com") {
		t.Fatalf("list output = %q", stdout.String())
	}
}

func TestRemoveRequiresAlias(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"remove"}, &stdout, &stderr, Deps{App: &fakeApp{}})
	if code != 2 {
		t.Fatalf("Run() code=%d stderr=%q", code, stderr.String())
	}
}

func TestExportWritesJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"export"}, &stdout, &stderr, Deps{App: &fakeApp{ExportData: []byte(`{"version":1}`)}})
	if code != 0 {
		t.Fatalf("Run() code=%d stderr=%q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != `{"version":1}` {
		t.Fatalf("export output = %q", stdout.String())
	}
}

type fakeApp struct {
	Accounts   []config.Account
	ExportData []byte
	RunCode    int
	RunAlias   string
	RunArgs    []string
}

func (a *fakeApp) Init() error                       { return nil }
func (a *fakeApp) Add() error                        { return nil }
func (a *fakeApp) List() ([]config.Account, error)   { return a.Accounts, nil }
func (a *fakeApp) Remove(alias string) error         { return nil }
func (a *fakeApp) Install() error                    { return nil }
func (a *fakeApp) Export() ([]byte, error)           { return a.ExportData, nil }
func (a *fakeApp) Import(path string) error          { return nil }
func (a *fakeApp) Auth(alias string) error           { return nil }
func (a *fakeApp) Doctor() ([]string, error)         { return []string{"OK   git"}, nil }
func (a *fakeApp) RunGit(alias string, args []string) (int, error) {
	a.RunAlias = alias
	a.RunArgs = append([]string{}, args...)
	return a.RunCode, nil
}
