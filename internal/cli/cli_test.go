package cli

import (
	"bytes"
	"strings"
	"testing"
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

	if got, want := out.String(), "gzy dev-test"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
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
