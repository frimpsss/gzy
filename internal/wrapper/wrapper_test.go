package wrapper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderUnix(t *testing.T) {
	got := RenderUnix("p", "/usr/local/bin/gzy")
	want := "#!/bin/sh\nexec \"/usr/local/bin/gzy\" run \"p\" -- \"$@\"\n"
	if got != want {
		t.Fatalf("RenderUnix() = %q, want %q", got, want)
	}
}

func TestRenderUnixQuotesPathsContainingSpaces(t *testing.T) {
	got := RenderUnix("p", "/Users/alex/My Tools/gzy")
	want := "#!/bin/sh\nexec \"/Users/alex/My Tools/gzy\" run \"p\" -- \"$@\"\n"
	if got != want {
		t.Fatalf("RenderUnix() = %q, want %q", got, want)
	}
}

func TestRenderWindowsCMD(t *testing.T) {
	got := RenderWindowsCMD("work", `C:\Users\alex\bin\gzy.exe`)
	want := `"C:\Users\alex\bin\gzy.exe" run "work" -- %*`
	if !strings.Contains(got, want) {
		t.Fatalf("RenderWindowsCMD() = %q, want it to contain %q", got, want)
	}
}

func TestInstallLinuxCreatesExecutableUnixWrapper(t *testing.T) {
	binDir := t.TempDir()

	got, err := Install("linux", binDir, "p", "/tmp/gzy")
	if err != nil {
		t.Fatalf("Install() returned error: %v", err)
	}

	wantPath := filepath.Join(binDir, "git-p")
	if got != wantPath {
		t.Fatalf("Install() path = %q, want %q", got, wantPath)
	}

	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile() returned error: %v", err)
	}
	wantContents := "#!/bin/sh\nexec \"/tmp/gzy\" run \"p\" -- \"$@\"\n"
	if string(data) != wantContents {
		t.Fatalf("wrapper contents = %q, want %q", string(data), wantContents)
	}

	info, err := os.Stat(wantPath)
	if err != nil {
		t.Fatalf("Stat() returned error: %v", err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0755 {
		t.Fatalf("wrapper mode = %v, want %v", gotMode, os.FileMode(0755))
	}
}

func TestInstallWindowsCreatesCMDWrapper(t *testing.T) {
	binDir := t.TempDir()

	got, err := Install("windows", binDir, "p", `C:\bin\gzy.exe`)
	if err != nil {
		t.Fatalf("Install() returned error: %v", err)
	}

	wantPath := filepath.Join(binDir, "git-p.cmd")
	if got != wantPath {
		t.Fatalf("Install() path = %q, want %q", got, wantPath)
	}

	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile() returned error: %v", err)
	}
	wantContents := "@echo off\r\n\"C:\\bin\\gzy.exe\" run \"p\" -- %*\r\n"
	if string(data) != wantContents {
		t.Fatalf("wrapper contents = %q, want %q", string(data), wantContents)
	}

	info, err := os.Stat(wantPath)
	if err != nil {
		t.Fatalf("Stat() returned error: %v", err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0644 {
		t.Fatalf("wrapper mode = %v, want %v", gotMode, os.FileMode(0644))
	}
}

func TestRemoveIgnoresMissingWrapperAndRemovesExistingWrapper(t *testing.T) {
	binDir := t.TempDir()

	if err := Remove("linux", binDir, "p"); err != nil {
		t.Fatalf("Remove() missing wrapper returned error: %v", err)
	}

	path, err := Install("linux", binDir, "p", "/tmp/gzy")
	if err != nil {
		t.Fatalf("Install() returned error: %v", err)
	}

	if err := Remove("linux", binDir, "p"); err != nil {
		t.Fatalf("Remove() existing wrapper returned error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("removed wrapper stat error = %v, want os.IsNotExist", err)
	}
}
