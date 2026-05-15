package doctor

import (
	"testing"

	"github.com/frimpsss/gzy/internal/config"
)

func TestCheckReportsMissingGit(t *testing.T) {
	checker := Checker{LookPath: func(name string) (string, error) { return "", errNotFound(name) }}
	result := checker.Check(config.Config{Version: 1})
	if !result.HasProblem("git") {
		t.Fatalf("result = %#v", result)
	}
}

func TestCheckReportsMissingKey(t *testing.T) {
	checker := Checker{
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		Exists:   func(path string) bool { return false },
	}
	cfg := config.Config{Version: 1, Accounts: []config.Account{{Alias: "p", PrivateKey: "/tmp/missing", PublicKey: "/tmp/missing.pub"}}}
	result := checker.Check(cfg)
	if !result.HasProblem("p") {
		t.Fatalf("result = %#v", result)
	}
}
