# gzy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `gzy`, a cross-platform Go CLI that creates account-specific Git wrapper commands like `git-p`, including guided SSH setup, GitHub browser authentication, export/import, doctor checks, and curl/wget/PowerShell installers.

**Architecture:** The `cmd/gzy` binary delegates to focused internal packages for config, paths, wrappers, Git command execution, SSH key management, GitHub OAuth, and setup orchestration. Generated `git-*` wrappers stay tiny and call `gzy run <alias> -- <git args...>`, so all real behavior remains testable in Go. GitHub device authentication uploads public SSH keys during initial setup, while manual paste remains a fallback when OAuth is unavailable.

**Tech Stack:** Go 1.22+, standard library only for the CLI; shell and PowerShell scripts for installers; Go unit tests using `testing`, `httptest`, and temporary directories.

---

## File Structure

- Create `go.mod`: Go module definition for `github.com/frimpsss/gzy`.
- Create `cmd/gzy/main.go`: thin binary entrypoint that calls `internal/cli`.
- Create `internal/cli/cli.go`: command routing, help text, process exit codes, and dependency wiring.
- Create `internal/cli/cli_test.go`: CLI routing tests using in-memory IO and fakes.
- Create `internal/app/app.go`: production adapter that joins config, setup, wrappers, doctor, and git runner packages.
- Create `internal/app/app_test.go`: production adapter tests with temporary config and wrapper directories.
- Create `internal/config/config.go`: account model, JSON load/save/export/import, and alias validation.
- Create `internal/config/config_test.go`: config persistence and validation tests.
- Create `internal/paths/paths.go`: platform-aware config, SSH, and bin directory selection.
- Create `internal/paths/paths_test.go`: path selection tests for darwin, linux, and windows using injected environment values.
- Create `internal/wrapper/wrapper.go`: shell and Windows wrapper rendering, install, and removal.
- Create `internal/wrapper/wrapper_test.go`: wrapper content and install tests.
- Create `internal/gitrun/gitrun.go`: account-specific Git command construction and execution.
- Create `internal/gitrun/gitrun_test.go`: tests for `git -c user.name -c user.email` args and `GIT_SSH_COMMAND`.
- Create `internal/sshkeys/sshkeys.go`: SSH key discovery, fingerprinting, creation, public-key reading, and clipboard/browser helpers.
- Create `internal/sshkeys/sshkeys_test.go`: key discovery and fake subprocess tests.
- Create `internal/githubauth/githubauth.go`: GitHub OAuth device flow and SSH public-key upload client.
- Create `internal/githubauth/githubauth_test.go`: fake GitHub HTTP server tests for success, pending, denied, timeout, and duplicate-key behavior.
- Create `internal/setup/setup.go`: guided `init`, `add`, and `auth` orchestration.
- Create `internal/setup/setup_test.go`: setup-flow tests with fake prompts, fake key manager, fake GitHub client, and fake wrapper installer.
- Create `internal/doctor/doctor.go`: checks for Git, SSH, configured key files, wrapper files, and GitHub SSH readiness.
- Create `internal/doctor/doctor_test.go`: doctor result tests with fake command lookup and fake filesystem.
- Create `internal/release/release.go`: release target naming helpers shared by tests and docs.
- Create `internal/release/release_test.go`: OS/architecture artifact naming tests.
- Create `install.sh`: macOS/Linux installer using curl or wget.
- Create `install.ps1`: Windows PowerShell installer.
- Create `scripts/build-release.sh`: local release build helper for all supported targets.
- Create `README.md`: installation, first-run, daily usage, HTTPS caveat, and development notes.

## Task 1: Bootstrap Go Module And CLI Entrypoint

**Files:**
- Create: `go.mod`
- Create: `cmd/gzy/main.go`
- Create: `internal/cli/cli.go`
- Create: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing CLI tests**

Create `internal/cli/cli_test.go`:

```go
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
		t.Fatalf("Run() code = %d, want 0", code)
	}
	text := out.String()
	for _, want := range []string{"gzy", "init", "add", "run", "doctor"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help missing %q in:\n%s", want, text)
		}
	}
}

func TestRunVersionPrintsVersion(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"version"}, &out, &out, Deps{Version: "dev-test"})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if got := strings.TrimSpace(out.String()); got != "gzy dev-test" {
		t.Fatalf("version output = %q", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"wat"}, &stdout, &stderr, Deps{Version: "dev-test"})
	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command: wat") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/cli
```

Expected: FAIL because `go.mod` and `internal/cli` do not exist yet.

- [ ] **Step 3: Add module and minimal CLI**

Create `go.mod`:

```go
module github.com/frimpsss/gzy

go 1.22
```

Create `cmd/gzy/main.go`:

```go
package main

import (
	"os"

	"github.com/frimpsss/gzy/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, cli.Deps{Version: version}))
}
```

Create `internal/cli/cli.go`:

```go
package cli

import (
	"fmt"
	"io"
)

type Deps struct {
	Version string
}

func Run(args []string, stdout io.Writer, stderr io.Writer, deps Deps) int {
	if deps.Version == "" {
		deps.Version = "dev"
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "gzy %s\n", deps.Version)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `gzy - Git account aliases without the hassle

Usage:
  gzy init
  gzy add
  gzy list
  gzy remove <alias>
  gzy install
  gzy export
  gzy import <file>
  gzy auth <alias>
  gzy doctor
  gzy run <alias> -- <git args...>
  gzy version`)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/cli
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add go.mod cmd/gzy/main.go internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: bootstrap gzy cli"
```

## Task 2: Config Model, Alias Validation, And JSON Persistence

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing config tests**

Create `internal/config/config_test.go`:

```go
package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAlias(t *testing.T) {
	valid := []string{"p", "work", "client-1", "client_2"}
	for _, alias := range valid {
		if err := ValidateAlias(alias); err != nil {
			t.Fatalf("ValidateAlias(%q) returned %v", alias, err)
		}
	}

	invalid := []string{"", "bad alias", "../x", "git-p", ".hidden", "x/y"}
	for _, alias := range invalid {
		if err := ValidateAlias(alias); err == nil {
			t.Fatalf("ValidateAlias(%q) returned nil", alias)
		}
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := Config{
		Version: 1,
		Accounts: []Account{{
			Alias:       "p",
			Command:     "git-p",
			GitHubUser:  "frimpsss",
			Name:        "Akwasi Frimpong",
			Email:       "84427404+frimpsss@users.noreply.github.com",
			PrivateKey:  "~/.ssh/gzy_p",
			PublicKey:   "~/.ssh/gzy_p.pub",
			GitHubKeyID: 123,
			CreatedAt:   "2026-05-15T00:00:00Z",
		}},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Accounts) != 1 || loaded.Accounts[0].Alias != "p" {
		t.Fatalf("loaded config = %#v", loaded)
	}
}

func TestLoadMissingConfigReturnsEmptyVersionedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Version != 1 || len(cfg.Accounts) != 0 {
		t.Fatalf("Load() = %#v", cfg)
	}
}

func TestAddAccountRejectsDuplicateAlias(t *testing.T) {
	cfg := Config{Version: 1, Accounts: []Account{{Alias: "p"}}}
	err := cfg.AddAccount(Account{Alias: "p"})
	if err == nil {
		t.Fatal("AddAccount duplicate returned nil")
	}
}

func TestExportDoesNotIncludePrivateKeyContents(t *testing.T) {
	cfg := Config{Version: 1, Accounts: []Account{{Alias: "p", PrivateKey: "/tmp/key"}}}
	data, err := Export(cfg)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if !strings.Contains(string(data), `"privateKey": "/tmp/key"`) {
		t.Fatalf("export missing private key path: %s", data)
	}
	if strings.Contains(string(data), "BEGIN OPENSSH PRIVATE KEY") {
		t.Fatalf("export leaked private key contents")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/config
```

Expected: FAIL because `internal/config` does not exist yet.

- [ ] **Step 3: Implement config package**

Create `internal/config/config.go`:

```go
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const CurrentVersion = 1

var aliasPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

type Config struct {
	Version  int       `json:"version"`
	Accounts []Account `json:"accounts"`
}

type Account struct {
	Alias       string `json:"alias"`
	Command     string `json:"command"`
	GitHubUser  string `json:"githubUser"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PrivateKey  string `json:"privateKey"`
	PublicKey   string `json:"publicKey"`
	GitHubKeyID int64  `json:"githubKeyId,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

func ValidateAlias(alias string) error {
	if !aliasPattern.MatchString(alias) {
		return fmt.Errorf("alias %q must start with a letter or number and contain only letters, numbers, dashes, or underscores", alias)
	}
	if alias == "git" || len(alias) > 40 {
		return fmt.Errorf("alias %q is not allowed", alias)
	}
	return nil
}

func (c *Config) AddAccount(account Account) error {
	if err := ValidateAlias(account.Alias); err != nil {
		return err
	}
	if account.Command == "" {
		account.Command = "git-" + account.Alias
	}
	for _, existing := range c.Accounts {
		if existing.Alias == account.Alias {
			return fmt.Errorf("alias %q already exists", account.Alias)
		}
	}
	if c.Version == 0 {
		c.Version = CurrentVersion
	}
	c.Accounts = append(c.Accounts, account)
	return nil
}

func (c Config) Find(alias string) (Account, bool) {
	for _, account := range c.Accounts {
		if account.Alias == alias {
			return account, true
		}
	}
	return Account{}, false
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{Version: CurrentVersion}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func Export(cfg Config) ([]byte, error) {
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func Import(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	for _, account := range cfg.Accounts {
		if err := ValidateAlias(account.Alias); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/config
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add config persistence"
```

## Task 3: Platform Paths And User Bin Selection

**Files:**
- Create: `internal/paths/paths.go`
- Create: `internal/paths/paths_test.go`

- [ ] **Step 1: Write failing path tests**

Create `internal/paths/paths_test.go`:

```go
package paths

import (
	"path/filepath"
	"testing"
)

func TestConfigPathByOS(t *testing.T) {
	cases := []struct {
		goos string
		env  map[string]string
		want string
	}{
		{"darwin", map[string]string{"HOME": "/Users/alex"}, "/Users/alex/Library/Application Support/gzy/config.json"},
		{"linux", map[string]string{"HOME": "/home/alex"}, "/home/alex/.config/gzy/config.json"},
		{"windows", map[string]string{"APPDATA": `C:\Users\alex\AppData\Roaming`, "USERPROFILE": `C:\Users\alex`}, `C:\Users\alex\AppData\Roaming\gzy\config.json`},
	}
	for _, tc := range cases {
		p := New(tc.goos, tc.env)
		if got := p.ConfigFile(); got != filepath.Clean(tc.want) {
			t.Fatalf("%s ConfigFile() = %q, want %q", tc.goos, got, filepath.Clean(tc.want))
		}
	}
}

func TestDefaultKeyPaths(t *testing.T) {
	p := New("linux", map[string]string{"HOME": "/home/alex"})
	priv, pub := p.DefaultKeyPair("work")
	if priv != "/home/alex/.ssh/gzy_work" {
		t.Fatalf("private key = %q", priv)
	}
	if pub != "/home/alex/.ssh/gzy_work.pub" {
		t.Fatalf("public key = %q", pub)
	}
}

func TestPreferredBinDir(t *testing.T) {
	p := New("linux", map[string]string{"HOME": "/home/alex", "PATH": "/usr/bin:/home/alex/.local/bin"})
	if got := p.BinDir(); got != "/home/alex/.local/bin" {
		t.Fatalf("BinDir() = %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/paths
```

Expected: FAIL because `internal/paths` does not exist yet.

- [ ] **Step 3: Implement path helper**

Create `internal/paths/paths.go`:

```go
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Paths struct {
	goos string
	env  map[string]string
}

func New(goos string, env map[string]string) Paths {
	if goos == "" {
		goos = runtime.GOOS
	}
	return Paths{goos: goos, env: env}
}

func FromOS() Paths {
	env := map[string]string{}
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return New(runtime.GOOS, env)
}

func (p Paths) ConfigFile() string {
	switch p.goos {
	case "darwin":
		return filepath.Clean(filepath.Join(p.home(), "Library", "Application Support", "gzy", "config.json"))
	case "windows":
		base := p.env["APPDATA"]
		if base == "" {
			base = filepath.Join(p.home(), "AppData", "Roaming")
		}
		return filepath.Clean(filepath.Join(base, "gzy", "config.json"))
	default:
		base := p.env["XDG_CONFIG_HOME"]
		if base == "" {
			base = filepath.Join(p.home(), ".config")
		}
		return filepath.Clean(filepath.Join(base, "gzy", "config.json"))
	}
}

func (p Paths) SSHDir() string {
	return filepath.Clean(filepath.Join(p.home(), ".ssh"))
}

func (p Paths) DefaultKeyPair(alias string) (string, string) {
	privateKey := filepath.Join(p.SSHDir(), "gzy_"+alias)
	return filepath.Clean(privateKey), filepath.Clean(privateKey + ".pub")
}

func (p Paths) BinDir() string {
	if p.goos == "windows" {
		return filepath.Clean(filepath.Join(p.home(), "bin"))
	}
	local := filepath.Join(p.home(), ".local", "bin")
	if pathHasDir(p.env["PATH"], local) {
		return filepath.Clean(local)
	}
	return filepath.Clean(filepath.Join(p.home(), "bin"))
}

func (p Paths) home() string {
	if p.goos == "windows" {
		if value := p.env["USERPROFILE"]; value != "" {
			return value
		}
	}
	return p.env["HOME"]
}

func pathHasDir(pathValue string, dir string) bool {
	for _, part := range filepath.SplitList(pathValue) {
		if filepath.Clean(part) == filepath.Clean(dir) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/paths
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/paths/paths.go internal/paths/paths_test.go
git commit -m "feat: add platform paths"
```

## Task 4: Wrapper Rendering And Installation

**Files:**
- Create: `internal/wrapper/wrapper.go`
- Create: `internal/wrapper/wrapper_test.go`

- [ ] **Step 1: Write failing wrapper tests**

Create `internal/wrapper/wrapper_test.go`:

```go
package wrapper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderUnixWrapper(t *testing.T) {
	got := RenderUnix("p", "/usr/local/bin/gzy")
	want := "#!/bin/sh\nexec \"/usr/local/bin/gzy\" run \"p\" -- \"$@\"\n"
	if got != want {
		t.Fatalf("RenderUnix() = %q, want %q", got, want)
	}
}

func TestRenderWindowsCMDWrapper(t *testing.T) {
	got := RenderWindowsCMD("work", `C:\Users\alex\bin\gzy.exe`)
	if !strings.Contains(got, `"C:\Users\alex\bin\gzy.exe" run "work" -- %*`) {
		t.Fatalf("RenderWindowsCMD() = %q", got)
	}
}

func TestInstallUnixWrapper(t *testing.T) {
	dir := t.TempDir()
	path, err := Install("linux", dir, "p", "/tmp/gzy")
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if filepath.Base(path) != "git-p" {
		t.Fatalf("wrapper path = %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("wrapper is not executable: %v", info.Mode())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/wrapper
```

Expected: FAIL because `internal/wrapper` does not exist yet.

- [ ] **Step 3: Implement wrapper package**

Create `internal/wrapper/wrapper.go`:

```go
package wrapper

import (
	"fmt"
	"os"
	"path/filepath"
)

func RenderUnix(alias string, gzyPath string) string {
	return fmt.Sprintf("#!/bin/sh\nexec %q run %q -- \"$@\"\n", gzyPath, alias)
}

func RenderWindowsCMD(alias string, gzyPath string) string {
	return fmt.Sprintf("@echo off\r\n%q run %q -- %%*\r\n", gzyPath, alias)
}

func Install(goos string, binDir string, alias string, gzyPath string) (string, error) {
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}
	if goos == "windows" {
		path := filepath.Join(binDir, "git-"+alias+".cmd")
		return path, os.WriteFile(path, []byte(RenderWindowsCMD(alias, gzyPath)), 0o644)
	}
	path := filepath.Join(binDir, "git-"+alias)
	return path, os.WriteFile(path, []byte(RenderUnix(alias, gzyPath)), 0o755)
}

func Remove(goos string, binDir string, alias string) error {
	name := "git-" + alias
	if goos == "windows" {
		name += ".cmd"
	}
	err := os.Remove(filepath.Join(binDir, name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/wrapper
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/wrapper/wrapper.go internal/wrapper/wrapper_test.go
git commit -m "feat: add git wrapper generation"
```

## Task 5: Account-Specific Git Runner

**Files:**
- Create: `internal/gitrun/gitrun.go`
- Create: `internal/gitrun/gitrun_test.go`

- [ ] **Step 1: Write failing Git runner tests**

Create `internal/gitrun/gitrun_test.go`:

```go
package gitrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frimpsss/gzy/internal/config"
)

func TestBuildCommandAddsIdentityAndGitArgs(t *testing.T) {
	account := config.Account{Name: "Akwasi Frimpong", Email: "me@example.com", PrivateKey: "/tmp/key"}
	cmd := BuildCommand(account, []string{"status"})
	wantArgs := []string{"-c", "user.name=Akwasi Frimpong", "-c", "user.email=me@example.com", "status"}
	if strings.Join(cmd.Args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("Args = %#v, want %#v", cmd.Args, wantArgs)
	}
	if !strings.Contains(cmd.Env["GIT_SSH_COMMAND"], "ssh -i /tmp/key -o IdentitiesOnly=yes") {
		t.Fatalf("GIT_SSH_COMMAND = %q", cmd.Env["GIT_SSH_COMMAND"])
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/gitrun
```

Expected: FAIL because `internal/gitrun` does not exist yet.

- [ ] **Step 3: Implement Git runner**

Create `internal/gitrun/gitrun.go`:

```go
package gitrun

import (
	"os"
	"os/exec"

	"github.com/frimpsss/gzy/internal/config"
)

type Command struct {
	Args []string
	Env  map[string]string
}

type Runner struct {
	GitPath string
}

func BuildCommand(account config.Account, gitArgs []string) Command {
	args := []string{"-c", "user.name=" + account.Name, "-c", "user.email=" + account.Email}
	args = append(args, gitArgs...)
	env := map[string]string{
		"GIT_SSH_COMMAND": "ssh -i " + account.PrivateKey + " -o IdentitiesOnly=yes",
	}
	return Command{Args: args, Env: env}
}

func (r Runner) Run(account config.Account, gitArgs []string) (int, error) {
	gitPath := r.GitPath
	if gitPath == "" {
		gitPath = "git"
	}
	built := BuildCommand(account, gitArgs)
	cmd := exec.Command(gitPath, built.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	for key, value := range built.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	return 1, err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/gitrun
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/gitrun/gitrun.go internal/gitrun/gitrun_test.go
git commit -m "feat: add account git runner"
```

## Task 6: SSH Key Discovery, Creation, And Public Key Reading

**Files:**
- Create: `internal/sshkeys/sshkeys.go`
- Create: `internal/sshkeys/sshkeys_test.go`

- [ ] **Step 1: Write failing SSH key tests**

Create `internal/sshkeys/sshkeys_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/sshkeys
```

Expected: FAIL because `internal/sshkeys` does not exist yet.

- [ ] **Step 3: Implement SSH key package**

Create `internal/sshkeys/sshkeys.go`:

```go
package sshkeys

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Key struct {
	Name        string
	PrivatePath string
	PublicPath  string
	Fingerprint string
}

type Manager struct {
	SSHKeygenPath string
}

func Discover(sshDir string) ([]Key, error) {
	patterns := []string{"id_ed25519", "id_rsa", "github_*", "gzy_*"}
	seen := map[string]bool{}
	var keys []Key
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(sshDir, pattern))
		if err != nil {
			return nil, err
		}
		for _, privatePath := range matches {
			if strings.HasSuffix(privatePath, ".pub") || seen[privatePath] {
				continue
			}
			publicPath := privatePath + ".pub"
			if _, err := os.Stat(publicPath); err != nil {
				continue
			}
			seen[privatePath] = true
			keys = append(keys, Key{
				Name:        filepath.Base(privatePath),
				PrivatePath: privatePath,
				PublicPath:  publicPath,
			})
		}
	}
	return keys, nil
}

func ReadPublicKey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (m Manager) Create(privatePath string, email string) error {
	if err := os.MkdirAll(filepath.Dir(privatePath), 0o700); err != nil {
		return err
	}
	keygen := m.SSHKeygenPath
	if keygen == "" {
		keygen = "ssh-keygen"
	}
	cmd := exec.Command(keygen, "-t", "ed25519", "-C", email, "-f", privatePath, "-N", "")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/sshkeys
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/sshkeys/sshkeys.go internal/sshkeys/sshkeys_test.go
git commit -m "feat: add ssh key management"
```

## Task 7: GitHub Device Flow And SSH Key Upload

**Files:**
- Create: `internal/githubauth/githubauth.go`
- Create: `internal/githubauth/githubauth_test.go`

- [ ] **Step 1: Write failing GitHub auth tests**

Create `internal/githubauth/githubauth_test.go`:

```go
package githubauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeviceFlowSuccessAndKeyUpload(t *testing.T) {
	var uploadedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/device/code":
			_ = json.NewEncoder(w).Encode(DeviceCode{
				DeviceCode:      "device",
				UserCode:        "ABCD-1234",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       900,
				Interval:        1,
			})
		case "/login/oauth/access_token":
			_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "token", TokenType: "bearer", Scope: "write:public_key"})
		case "/user/keys":
			if got := r.Header.Get("Authorization"); got != "Bearer token" {
				t.Fatalf("Authorization = %q", got)
			}
			var body CreateKeyRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			uploadedKey = body.Key
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(KeyResponse{ID: 42, Key: body.Key, Title: body.Title})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := Client{
		ClientID: "client",
		BaseURL:  server.URL,
		HTTP:     server.Client(),
		Sleep:    func(time.Duration) {},
	}
	code, err := client.StartDeviceFlow()
	if err != nil {
		t.Fatalf("StartDeviceFlow() error = %v", err)
	}
	token, err := client.WaitForToken(code, time.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("WaitForToken() error = %v", err)
	}
	key, err := client.UploadKey(token.AccessToken, "gzy-p-host", "ssh-ed25519 AAAATEST")
	if err != nil {
		t.Fatalf("UploadKey() error = %v", err)
	}
	if key.ID != 42 || uploadedKey != "ssh-ed25519 AAAATEST" {
		t.Fatalf("key=%#v uploaded=%q", key, uploadedKey)
	}
}

func TestWaitForTokenDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/device/code" {
			_ = json.NewEncoder(w).Encode(DeviceCode{DeviceCode: "device", UserCode: "CODE", VerificationURI: "https://github.com/login/device", ExpiresIn: 900, Interval: 1})
			return
		}
		_ = json.NewEncoder(w).Encode(TokenResponse{Error: "access_denied"})
	}))
	defer server.Close()
	client := Client{ClientID: "client", BaseURL: server.URL, HTTP: server.Client(), Sleep: func(time.Duration) {}}
	code, err := client.StartDeviceFlow()
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.WaitForToken(code, time.Now().Add(time.Second))
	if err == nil {
		t.Fatal("WaitForToken denied returned nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/githubauth
```

Expected: FAIL because `internal/githubauth` does not exist yet.

- [ ] **Step 3: Implement GitHub auth client**

Create `internal/githubauth/githubauth.go`:

```go
package githubauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	ClientID string
	BaseURL  string
	HTTP     *http.Client
	Sleep    func(time.Duration)
}

type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

type CreateKeyRequest struct {
	Title string `json:"title"`
	Key   string `json:"key"`
}

type KeyResponse struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Title string `json:"title"`
}

func (c Client) StartDeviceFlow() (DeviceCode, error) {
	if c.ClientID == "" {
		return DeviceCode{}, errors.New("GitHub OAuth client id is not configured")
	}
	form := "client_id=" + c.ClientID + "&scope=write:public_key"
	req, err := http.NewRequest(http.MethodPost, c.endpoint("/login/device/code"), strings.NewReader(form))
	if err != nil {
		return DeviceCode{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var code DeviceCode
	if err := c.doJSON(req, &code); err != nil {
		return DeviceCode{}, err
	}
	return code, nil
}

func (c Client) WaitForToken(code DeviceCode, deadline time.Time) (TokenResponse, error) {
	interval := time.Duration(code.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	for time.Now().Before(deadline) {
		form := "client_id=" + c.ClientID + "&device_code=" + code.DeviceCode + "&grant_type=urn:ietf:params:oauth:grant-type:device_code"
		req, err := http.NewRequest(http.MethodPost, c.endpoint("/login/oauth/access_token"), strings.NewReader(form))
		if err != nil {
			return TokenResponse{}, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var token TokenResponse
		if err := c.doJSON(req, &token); err != nil {
			return TokenResponse{}, err
		}
		switch token.Error {
		case "":
			if token.AccessToken == "" {
				return TokenResponse{}, errors.New("GitHub returned an empty access token")
			}
			return token, nil
		case "authorization_pending":
			c.sleep(interval)
		case "slow_down":
			interval += 5 * time.Second
			c.sleep(interval)
		case "expired_token", "access_denied":
			return TokenResponse{}, fmt.Errorf("GitHub authorization failed: %s", token.Error)
		default:
			return TokenResponse{}, fmt.Errorf("GitHub authorization failed: %s", token.Error)
		}
	}
	return TokenResponse{}, errors.New("GitHub authorization timed out")
}

func (c Client) UploadKey(token string, title string, publicKey string) (KeyResponse, error) {
	body, err := json.Marshal(CreateKeyRequest{Title: title, Key: publicKey})
	if err != nil {
		return KeyResponse{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.endpoint("/user/keys"), bytes.NewReader(body))
	if err != nil {
		return KeyResponse{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	var response KeyResponse
	if err := c.doJSON(req, &response); err != nil {
		return KeyResponse{}, err
	}
	return response, nil
}

func (c Client) endpoint(path string) string {
	base := c.BaseURL
	if base == "" {
		base = "https://github.com"
	}
	if path == "/user/keys" && c.BaseURL == "" {
		return "https://api.github.com/user/keys"
	}
	return strings.TrimRight(base, "/") + path
}

func (c Client) doJSON(req *http.Request, into any) error {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("GitHub request failed with status %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func (c Client) sleep(duration time.Duration) {
	if c.Sleep != nil {
		c.Sleep(duration)
		return
	}
	time.Sleep(duration)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/githubauth
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/githubauth/githubauth.go internal/githubauth/githubauth_test.go
git commit -m "feat: add github device authentication"
```

## Task 8: Guided Setup Orchestration For init, add, And auth

**Files:**
- Create: `internal/setup/setup.go`
- Create: `internal/setup/setup_test.go`

- [ ] **Step 1: Write failing setup tests**

Create `internal/setup/setup_test.go`:

```go
package setup

import (
	"bytes"
	"testing"

	"github.com/frimpsss/gzy/internal/config"
)

func TestAddAccountCreatesConfigAndWrapper(t *testing.T) {
	prompts := NewStaticPrompter(map[string]string{
		"alias":       "p",
		"githubUser":  "frimpsss",
		"name":        "Akwasi Frimpong",
		"email":       "me@example.com",
		"keyChoice":   "create",
		"authChoice":  "manual",
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/setup
```

Expected: FAIL because `internal/setup` does not exist yet.

- [ ] **Step 3: Implement setup interfaces and service**

Create `internal/setup/setup.go` with these public interfaces and methods:

```go
package setup

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/frimpsss/gzy/internal/config"
)

type Prompter interface {
	Ask(key string, label string) (string, error)
}

type Store interface {
	Load() (config.Config, error)
	Save(config.Config) error
}

type KeyManager interface {
	DefaultKeyPair(alias string) (string, string)
	Create(privatePath string, email string) error
	ReadPublic(publicPath string) (string, error)
}

type WrapperInstaller interface {
	Install(alias string) error
}

type GitHubAuthenticator interface {
	UploadWithDeviceFlow(title string, publicKey string) (int64, error)
}

type Service struct {
	Prompts  Prompter
	Store    Store
	Keys     KeyManager
	Wrappers WrapperInstaller
	GitHub   GitHubAuthenticator
	Stdout   io.Writer
	Now      func() time.Time
}

func (s Service) Init() error {
	return s.Add()
}

func (s Service) Add() error {
	alias, err := s.Prompts.Ask("alias", "Alias command suffix")
	if err != nil {
		return err
	}
	if err := config.ValidateAlias(alias); err != nil {
		return err
	}
	githubUser, err := s.Prompts.Ask("githubUser", "GitHub username")
	if err != nil {
		return err
	}
	name, err := s.Prompts.Ask("name", "Git commit name")
	if err != nil {
		return err
	}
	email, err := s.Prompts.Ask("email", "Git commit email")
	if err != nil {
		return err
	}
	keyChoice, err := s.Prompts.Ask("keyChoice", "SSH key choice")
	if err != nil {
		return err
	}
	authChoice, err := s.Prompts.Ask("authChoice", "GitHub authentication method")
	if err != nil {
		return err
	}

	privateKey, publicKey := s.Keys.DefaultKeyPair(alias)
	if keyChoice == "create" {
		if err := s.Keys.Create(privateKey, email); err != nil {
			return err
		}
	}
	account := config.Account{
		Alias:      alias,
		Command:    "git-" + alias,
		GitHubUser: githubUser,
		Name:       name,
		Email:      email,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		CreatedAt:  s.now().UTC().Format(time.RFC3339),
	}

	cfg, err := s.Store.Load()
	if err != nil {
		return err
	}
	if err := cfg.AddAccount(account); err != nil {
		return err
	}
	if authChoice == "browser" && s.GitHub != nil {
		keyID, err := s.uploadKey(account)
		if err != nil {
			return err
		}
		cfg.Accounts[len(cfg.Accounts)-1].GitHubKeyID = keyID
	}
	if err := s.Store.Save(cfg); err != nil {
		return err
	}
	return s.Wrappers.Install(alias)
}

func (s Service) Auth(alias string) error {
	cfg, err := s.Store.Load()
	if err != nil {
		return err
	}
	for i, account := range cfg.Accounts {
		if account.Alias == alias {
			keyID, err := s.uploadKey(account)
			if err != nil {
				return err
			}
			cfg.Accounts[i].GitHubKeyID = keyID
			return s.Store.Save(cfg)
		}
	}
	return fmt.Errorf("alias %q is not configured", alias)
}

func (s Service) uploadKey(account config.Account) (int64, error) {
	if s.GitHub == nil {
		return 0, errors.New("GitHub browser authentication is not configured")
	}
	publicKey, err := s.Keys.ReadPublic(account.PublicKey)
	if err != nil {
		return 0, err
	}
	return s.GitHub.UploadWithDeviceFlow("gzy-"+account.Alias, publicKey)
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
```

Add test fakes in `internal/setup/setup_test.go` below the tests:

```go
type StaticPrompter struct{ Answers map[string]string }

func NewStaticPrompter(answers map[string]string) StaticPrompter {
	return StaticPrompter{Answers: answers}
}

func (p StaticPrompter) Ask(key string, label string) (string, error) {
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

func (k *FakeKeys) DefaultKeyPair(alias string) (string, string) { return k.PrivatePath, k.PublicPath }
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
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/setup
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/setup/setup.go internal/setup/setup_test.go
git commit -m "feat: add guided setup orchestration"
```

## Task 9: Route CLI Commands Through An Application Interface

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`
- Create: `internal/cli/deps.go`

- [ ] **Step 1: Add failing CLI command tests**

Append to `internal/cli/cli_test.go`:

```go
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
```

Add fakes to the test file:

```go
type fakeApp struct {
	Accounts   []config.Account
	ExportData []byte
	RunCode    int
	RunAlias   string
	RunArgs    []string
}

func (a *fakeApp) Init() error { return nil }
func (a *fakeApp) Add() error { return nil }
func (a *fakeApp) List() ([]config.Account, error) { return a.Accounts, nil }
func (a *fakeApp) Remove(alias string) error { return nil }
func (a *fakeApp) Install() error { return nil }
func (a *fakeApp) Export() ([]byte, error) { return a.ExportData, nil }
func (a *fakeApp) Import(path string) error { return nil }
func (a *fakeApp) Auth(alias string) error { return nil }
func (a *fakeApp) Doctor() ([]string, error) { return []string{"OK   git"}, nil }
func (a *fakeApp) RunGit(alias string, args []string) (int, error) {
	a.RunAlias = alias
	a.RunArgs = append([]string{}, args...)
	return a.RunCode, nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/cli
```

Expected: FAIL because `Deps` does not have the application collaborator and commands.

- [ ] **Step 3: Update CLI routing**

Create `internal/cli/deps.go`:

```go
package cli

import "github.com/frimpsss/gzy/internal/config"

type App interface {
	Init() error
	Add() error
	List() ([]config.Account, error)
	Remove(alias string) error
	Install() error
	Export() ([]byte, error)
	Import(path string) error
	Auth(alias string) error
	Doctor() ([]string, error)
	RunGit(alias string, args []string) (int, error)
}
```

Modify `internal/cli/cli.go`:

```go
package cli

import (
	"fmt"
	"io"
	"strings"
)

type Deps struct {
	Version string
	App     App
}

func Run(args []string, stdout io.Writer, stderr io.Writer, deps Deps) int {
	if deps.Version == "" {
		deps.Version = "dev"
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "gzy %s\n", deps.Version)
		return 0
	case "init":
		return runNoArg(stderr, deps, "init")
	case "add":
		return runNoArg(stderr, deps, "add")
	case "list":
		return runList(stdout, stderr, deps)
	case "remove":
		return runOneArg(args[1:], stderr, deps, "remove")
	case "install":
		return runNoArg(stderr, deps, "install")
	case "export":
		return runExport(stdout, stderr, deps)
	case "import":
		return runOneArg(args[1:], stderr, deps, "import")
	case "auth":
		return runOneArg(args[1:], stderr, deps, "auth")
	case "doctor":
		return runDoctor(stdout, stderr, deps)
	case "run":
		return runGitAlias(args[1:], stderr, deps)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func runList(stdout io.Writer, stderr io.Writer, deps Deps) int {
	if deps.App == nil {
		fmt.Fprintln(stderr, "application is not configured")
		return 1
	}
	accounts, err := deps.App.List()
	if err != nil {
		fmt.Fprintf(stderr, "could not list accounts: %v\n", err)
		return 1
	}
	for _, account := range accounts {
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", account.Command, account.GitHubUser, account.Email)
	}
	return 0
}

func runGitAlias(args []string, stderr io.Writer, deps Deps) int {
	if deps.App == nil {
		fmt.Fprintln(stderr, "application is not configured")
		return 1
	}
	if len(args) < 3 || args[1] != "--" {
		fmt.Fprintln(stderr, "usage: gzy run <alias> -- <git args...>")
		return 2
	}
	code, err := deps.App.RunGit(args[0], args[2:])
	if err != nil {
		fmt.Fprintf(stderr, "git failed: %v\n", err)
		return 1
	}
	return code
}

func runNoArg(stderr io.Writer, deps Deps, command string) int {
	if deps.App == nil {
		fmt.Fprintln(stderr, "application is not configured")
		return 1
	}
	var err error
	switch command {
	case "init":
		err = deps.App.Init()
	case "add":
		err = deps.App.Add()
	case "install":
		err = deps.App.Install()
	}
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	return 0
}

func runOneArg(args []string, stderr io.Writer, deps Deps, command string) int {
	if deps.App == nil {
		fmt.Fprintln(stderr, "application is not configured")
		return 1
	}
	if len(args) != 1 {
		fmt.Fprintf(stderr, "usage: gzy %s %s\n", command, usageArg(command))
		return 2
	}
	var err error
	switch command {
	case "remove":
		err = deps.App.Remove(args[0])
	case "import":
		err = deps.App.Import(args[0])
	case "auth":
		err = deps.App.Auth(args[0])
	}
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	return 0
}

func usageArg(command string) string {
	switch command {
	case "import":
		return "<file>"
	default:
		return "<alias>"
	}
}

func runExport(stdout io.Writer, stderr io.Writer, deps Deps) int {
	if deps.App == nil {
		fmt.Fprintln(stderr, "application is not configured")
		return 1
	}
	data, err := deps.App.Export()
	if err != nil {
		fmt.Fprintf(stderr, "could not export config: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, strings.TrimSpace(string(data)))
	return 0
}

func runDoctor(stdout io.Writer, stderr io.Writer, deps Deps) int {
	if deps.App == nil {
		fmt.Fprintln(stderr, "application is not configured")
		return 1
	}
	lines, err := deps.App.Doctor()
	if err != nil {
		fmt.Fprintf(stderr, "doctor failed: %v\n", err)
		return 1
	}
	for _, line := range lines {
		fmt.Fprintln(stdout, line)
	}
	return 0
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, strings.TrimSpace(`gzy - Git account aliases without the hassle

Usage:
  gzy init
  gzy add
  gzy list
  gzy remove <alias>
  gzy install
  gzy export
  gzy import <file>
  gzy auth <alias>
  gzy doctor
  gzy run <alias> -- <git args...>
  gzy version`)+"\n")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/cli
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/cli/cli.go internal/cli/cli_test.go internal/cli/deps.go
git commit -m "feat: route cli commands"
```

## Task 10: Doctor Checks

**Files:**
- Create: `internal/doctor/doctor.go`
- Create: `internal/doctor/doctor_test.go`

- [ ] **Step 1: Write failing doctor tests**

Create `internal/doctor/doctor_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/doctor
```

Expected: FAIL because `internal/doctor` does not exist yet.

- [ ] **Step 3: Implement doctor package**

Create `internal/doctor/doctor.go`:

```go
package doctor

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/frimpsss/gzy/internal/config"
)

type Check struct {
	Name    string
	OK      bool
	Message string
}

type Result struct {
	Checks []Check
}

func (r Result) HasProblem(name string) bool {
	for _, check := range r.Checks {
		if check.Name == name && !check.OK {
			return true
		}
	}
	return false
}

type Checker struct {
	LookPath func(string) (string, error)
	Exists   func(string) bool
}

func (c Checker) Check(cfg config.Config) Result {
	lookPath := c.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	exists := c.Exists
	if exists == nil {
		exists = func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		}
	}

	var checks []Check
	for _, binary := range []string{"git", "ssh", "ssh-keygen"} {
		_, err := lookPath(binary)
		checks = append(checks, Check{Name: binary, OK: err == nil, Message: messageForBinary(binary, err)})
	}
	for _, account := range cfg.Accounts {
		privateOK := exists(account.PrivateKey)
		publicOK := exists(account.PublicKey)
		checks = append(checks, Check{
			Name:    account.Alias,
			OK:      privateOK && publicOK,
			Message: fmt.Sprintf("%s key files private=%t public=%t", account.Command, privateOK, publicOK),
		})
	}
	return Result{Checks: checks}
}

func messageForBinary(binary string, err error) string {
	if err == nil {
		return binary + " found"
	}
	return binary + " is missing; install it and rerun gzy doctor"
}

func errNotFound(name string) error {
	return fmt.Errorf("%s not found", name)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```sh
go test ./internal/doctor
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/doctor/doctor.go internal/doctor/doctor_test.go
git commit -m "feat: add doctor checks"
```

## Task 11: Installer Scripts And Release Build Helper

**Files:**
- Create: `internal/release/release.go`
- Create: `internal/release/release_test.go`
- Create: `scripts/build-release.sh`
- Create: `install.sh`
- Create: `install.ps1`

- [ ] **Step 1: Write failing release tests**

Create `internal/release/release_test.go`:

```go
package release

import "testing"

func TestArtifactName(t *testing.T) {
	cases := []struct {
		goos string
		arch string
		want string
	}{
		{"darwin", "arm64", "gzy_Darwin_arm64.tar.gz"},
		{"linux", "amd64", "gzy_Linux_x86_64.tar.gz"},
		{"windows", "amd64", "gzy_Windows_x86_64.zip"},
	}
	for _, tc := range cases {
		if got := ArtifactName(tc.goos, tc.arch); got != tc.want {
			t.Fatalf("ArtifactName(%q,%q)=%q want %q", tc.goos, tc.arch, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```sh
go test ./internal/release
```

Expected: FAIL because `internal/release` does not exist yet.

- [ ] **Step 3: Implement release helper**

Create `internal/release/release.go`:

```go
package release

func ArtifactName(goos string, arch string) string {
	osName := map[string]string{"darwin": "Darwin", "linux": "Linux", "windows": "Windows"}[goos]
	archName := map[string]string{"amd64": "x86_64", "arm64": "arm64"}[arch]
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return "gzy_" + osName + "_" + archName + ext
}
```

- [ ] **Step 4: Add installer scripts**

Create `scripts/build-release.sh`:

```sh
#!/bin/sh
set -eu

version="${VERSION:-dev}"
mkdir -p dist

build_one() {
  goos="$1"
  goarch="$2"
  name="$3"
  out="dist/gzy"
  if [ "$goos" = "windows" ]; then
    out="dist/gzy.exe"
  fi
  GOOS="$goos" GOARCH="$goarch" go build -ldflags "-X main.version=$version" -o "$out" ./cmd/gzy
  if [ "$goos" = "windows" ]; then
    (cd dist && zip -q "$name" gzy.exe && rm gzy.exe)
  else
    (cd dist && tar -czf "$name" gzy && rm gzy)
  fi
}

build_one darwin amd64 gzy_Darwin_x86_64.tar.gz
build_one darwin arm64 gzy_Darwin_arm64.tar.gz
build_one linux amd64 gzy_Linux_x86_64.tar.gz
build_one linux arm64 gzy_Linux_arm64.tar.gz
build_one windows amd64 gzy_Windows_x86_64.zip
build_one windows arm64 gzy_Windows_arm64.zip
```

Create `install.sh`:

```sh
#!/bin/sh
set -eu

repo="${GZY_REPO:-frimpsss/gzy}"
version="${GZY_VERSION:-latest}"
bin_dir="${GZY_BIN_DIR:-$HOME/.local/bin}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Darwin) os_name="Darwin" ;;
  Linux) os_name="Linux" ;;
  *) echo "gzy: unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) arch_name="x86_64" ;;
  arm64|aarch64) arch_name="arm64" ;;
  *) echo "gzy: unsupported architecture: $arch" >&2; exit 1 ;;
esac

artifact="gzy_${os_name}_${arch_name}.tar.gz"
if [ "$version" = "latest" ]; then
  url="https://github.com/${repo}/releases/latest/download/${artifact}"
else
  url="https://github.com/${repo}/releases/download/${version}/${artifact}"
fi

mkdir -p "$bin_dir"
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp_dir/$artifact"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp_dir/$artifact" "$url"
else
  echo "gzy: install requires curl or wget" >&2
  exit 1
fi

tar -xzf "$tmp_dir/$artifact" -C "$tmp_dir"
install "$tmp_dir/gzy" "$bin_dir/gzy"
echo "gzy installed to $bin_dir/gzy"
case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *) echo "Add this to your shell profile: export PATH=\"$bin_dir:\$PATH\"" ;;
esac
```

Create `install.ps1`:

```powershell
$ErrorActionPreference = "Stop"

$Repo = if ($env:GZY_REPO) { $env:GZY_REPO } else { "frimpsss/gzy" }
$Version = if ($env:GZY_VERSION) { $env:GZY_VERSION } else { "latest" }
$BinDir = if ($env:GZY_BIN_DIR) { $env:GZY_BIN_DIR } else { Join-Path $HOME "bin" }
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture -eq "Arm64") { "arm64" } else { "x86_64" }
$Artifact = "gzy_Windows_$Arch.zip"

if ($Version -eq "latest") {
  $Url = "https://github.com/$Repo/releases/latest/download/$Artifact"
} else {
  $Url = "https://github.com/$Repo/releases/download/$Version/$Artifact"
}

$Temp = New-Item -ItemType Directory -Path ([System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), [System.Guid]::NewGuid().ToString()))
try {
  New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
  $ZipPath = Join-Path $Temp.FullName $Artifact
  Invoke-WebRequest -Uri $Url -OutFile $ZipPath
  Expand-Archive -Path $ZipPath -DestinationPath $Temp.FullName -Force
  Copy-Item (Join-Path $Temp.FullName "gzy.exe") (Join-Path $BinDir "gzy.exe") -Force
  Write-Host "gzy installed to $BinDir\gzy.exe"
  if (($env:PATH -split ";") -notcontains $BinDir) {
    Write-Host "Add this directory to PATH: $BinDir"
  }
} finally {
  Remove-Item -Recurse -Force $Temp.FullName
}
```

- [ ] **Step 5: Run release tests and script syntax checks**

Run:

```sh
go test ./internal/release
sh -n install.sh
sh -n scripts/build-release.sh
```

Expected: PASS and no shell syntax output.

- [ ] **Step 6: Commit**

Run:

```sh
git add internal/release/release.go internal/release/release_test.go scripts/build-release.sh install.sh install.ps1
git commit -m "feat: add release installers"
```

## Task 12: README And End-To-End Verification

**Files:**
- Create: `README.md`
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`
- Modify: `cmd/gzy/main.go`

- [ ] **Step 1: Write README**

Create `README.md`:

```markdown
# gzy

`gzy` lets you use multiple GitHub accounts on one computer through simple Git aliases like `git-p` and `git-w`.

## Install

macOS and Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/frimpsss/gzy/main/install.sh | sh
```

```sh
wget -qO- https://raw.githubusercontent.com/frimpsss/gzy/main/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/frimpsss/gzy/main/install.ps1 | iex
```

## First Setup

```sh
gzy init
```

`gzy` asks for an alias, GitHub username, commit name, commit email, and SSH key choice. Browser authentication is the default path. It uploads the public SSH key to GitHub after you approve the app in your browser.

## Daily Use

```sh
git-p clone git@github.com:frimpsss/example.git
git-p status
git-p push
git-w commit -m "message"
```

## HTTPS Remotes

`gzy` sets the commit name and email for both SSH and HTTPS remotes. GitHub login for HTTPS remotes still depends on your system Git credential manager. SSH remotes provide the smoothest multi-account flow.

## Transfer To Another Machine

```sh
gzy export > gzy-config.json
gzy import gzy-config.json
gzy install
gzy doctor
```

Private SSH keys are not exported.

## Development

```sh
go test ./...
go build ./cmd/gzy
```
```

- [ ] **Step 2: Write failing app adapter tests**

Create `internal/app/app_test.go`:

```go
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
```

- [ ] **Step 3: Run test to verify it fails**

Run:

```sh
go test ./internal/app
```

Expected: FAIL because `internal/app` does not exist yet.

- [ ] **Step 4: Create production app adapter**

Create `internal/app/app.go`:

```go
package app

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/frimpsss/gzy/internal/config"
	"github.com/frimpsss/gzy/internal/doctor"
	"github.com/frimpsss/gzy/internal/gitrun"
	"github.com/frimpsss/gzy/internal/githubauth"
	"github.com/frimpsss/gzy/internal/paths"
	"github.com/frimpsss/gzy/internal/setup"
	"github.com/frimpsss/gzy/internal/sshkeys"
	"github.com/frimpsss/gzy/internal/wrapper"
)

type Config struct {
	ConfigPath     string
	BinDir         string
	GZYPath        string
	GOOS           string
	GitHubClientID string
	Stdin          io.Reader
	Stdout         io.Writer
}

type App struct {
	cfg Config
}

func New(cfg Config) *App {
	if cfg.GOOS == "" {
		cfg.GOOS = runtime.GOOS
	}
	if cfg.Stdin == nil {
		cfg.Stdin = os.Stdin
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	return &App{cfg: cfg}
}

func (a *App) Init() error { return a.Add() }

func (a *App) Add() error {
	return a.setupService().Add()
}

func (a *App) List() ([]config.Account, error) {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return nil, err
	}
	return cfg.Accounts, nil
}

func (a *App) Remove(alias string) error {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return err
	}
	kept := cfg.Accounts[:0]
	found := false
	for _, account := range cfg.Accounts {
		if account.Alias == alias {
			found = true
			continue
		}
		kept = append(kept, account)
	}
	if !found {
		return fmt.Errorf("alias %q is not configured", alias)
	}
	cfg.Accounts = kept
	if err := config.Save(a.cfg.ConfigPath, cfg); err != nil {
		return err
	}
	return wrapper.Remove(a.cfg.GOOS, a.cfg.BinDir, alias)
}

func (a *App) Install() error {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return err
	}
	for _, account := range cfg.Accounts {
		if _, err := wrapper.Install(a.cfg.GOOS, a.cfg.BinDir, account.Alias, a.cfg.GZYPath); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) Export() ([]byte, error) {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return nil, err
	}
	return config.Export(cfg)
}

func (a *App) Import(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cfg, err := config.Import(data)
	if err != nil {
		return err
	}
	return config.Save(a.cfg.ConfigPath, cfg)
}

func (a *App) Auth(alias string) error {
	return a.setupService().Auth(alias)
}

func (a *App) Doctor() ([]string, error) {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return nil, err
	}
	result := doctor.Checker{}.Check(cfg)
	lines := make([]string, 0, len(result.Checks))
	for _, check := range result.Checks {
		status := "FAIL"
		if check.OK {
			status = "OK"
		}
		lines = append(lines, fmt.Sprintf("%s   %s", status, check.Name))
	}
	return lines, nil
}

func (a *App) RunGit(alias string, args []string) (int, error) {
	cfg, err := config.Load(a.cfg.ConfigPath)
	if err != nil {
		return 1, err
	}
	account, ok := cfg.Find(alias)
	if !ok {
		return 1, fmt.Errorf("alias %q is not configured", alias)
	}
	return gitrun.Runner{}.Run(account, args)
}

func (a *App) setupService() setup.Service {
	return setup.Service{
		Prompts:  terminalPrompter{in: a.cfg.Stdin, out: a.cfg.Stdout},
		Store:    fileStore{path: a.cfg.ConfigPath},
		Keys:     keyAdapter{paths: paths.FromOS(), manager: sshkeys.Manager{}},
		Wrappers: wrapperAdapter{goos: a.cfg.GOOS, binDir: a.cfg.BinDir, gzyPath: a.cfg.GZYPath},
		GitHub:   githubAdapter{clientID: a.cfg.GitHubClientID, out: a.cfg.Stdout},
		Stdout:   a.cfg.Stdout,
	}
}

type terminalPrompter struct {
	in  io.Reader
	out io.Writer
}

func (p terminalPrompter) Ask(key string, label string) (string, error) {
	fmt.Fprintf(p.out, "%s: ", label)
	reader := bufio.NewReader(p.in)
	text, err := reader.ReadString('\n')
	if err != nil && len(text) == 0 {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

type fileStore struct{ path string }

func (s fileStore) Load() (config.Config, error) { return config.Load(s.path) }
func (s fileStore) Save(cfg config.Config) error { return config.Save(s.path, cfg) }

type keyAdapter struct {
	paths   paths.Paths
	manager sshkeys.Manager
}

func (k keyAdapter) DefaultKeyPair(alias string) (string, string) { return k.paths.DefaultKeyPair(alias) }
func (k keyAdapter) Create(privatePath string, email string) error { return k.manager.Create(privatePath, email) }
func (k keyAdapter) ReadPublic(publicPath string) (string, error)  { return sshkeys.ReadPublicKey(publicPath) }

type wrapperAdapter struct {
	goos    string
	binDir  string
	gzyPath string
}

func (w wrapperAdapter) Install(alias string) error {
	_, err := wrapper.Install(w.goos, w.binDir, alias, w.gzyPath)
	return err
}

type githubAdapter struct {
	clientID string
	out      io.Writer
}

func (g githubAdapter) UploadWithDeviceFlow(title string, publicKey string) (int64, error) {
	client := githubauth.Client{ClientID: g.clientID, HTTP: http.DefaultClient}
	code, err := client.StartDeviceFlow()
	if err != nil {
		return 0, err
	}
	fmt.Fprintf(g.out, "Open %s and enter code %s\n", code.VerificationURI, code.UserCode)
	token, err := client.WaitForToken(code, time.Now().Add(time.Duration(code.ExpiresIn)*time.Second))
	if err != nil {
		return 0, err
	}
	key, err := client.UploadKey(token.AccessToken, title, publicKey)
	if err != nil {
		return 0, err
	}
	return key.ID, nil
}
```

- [ ] **Step 5: Wire main to production app**

Modify `cmd/gzy/main.go`:

```go
package main

import (
	"os"

	"github.com/frimpsss/gzy/internal/app"
	"github.com/frimpsss/gzy/internal/cli"
	"github.com/frimpsss/gzy/internal/paths"
)

var version = "dev"

func main() {
	p := paths.FromOS()
	configPath := os.Getenv("GZY_CONFIG")
	if configPath == "" {
		configPath = p.ConfigFile()
	}
	binDir := os.Getenv("GZY_BIN_DIR")
	if binDir == "" {
		binDir = p.BinDir()
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "gzy"
	}
	deps := cli.Deps{
		Version: version,
		App: app.New(app.Config{
			ConfigPath:     configPath,
			BinDir:         binDir,
			GZYPath:        exe,
			GitHubClientID: os.Getenv("GZY_GITHUB_CLIENT_ID"),
		}),
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, deps))
}
```

- [ ] **Step 6: Verify command behavior**

The command behavior must match these outputs after a config file exists:

```text
gzy list
git-p    frimpsss    84427404+frimpsss@users.noreply.github.com
```

```text
gzy doctor
OK   git
OK   ssh
OK   ssh-keygen
OK   git-p
```

- [ ] **Step 7: Run full test suite**

Run:

```sh
go test ./...
```

Expected: PASS.

- [ ] **Step 8: Build local binary**

Run:

```sh
go build -o ./dist/gzy ./cmd/gzy
./dist/gzy version
```

Expected:

```text
gzy dev
```

- [ ] **Step 9: Test wrapper delegation with a temporary config**

Run:

```sh
tmp="$(mktemp -d)"
printf '%s\n' '{"version":1,"accounts":[{"alias":"p","command":"git-p","githubUser":"frimpsss","name":"Akwasi Frimpong","email":"84427404+frimpsss@users.noreply.github.com","privateKey":"'"$HOME"'/.ssh/id_ed25519","publicKey":"'"$HOME"'/.ssh/id_ed25519.pub","createdAt":"2026-05-15T00:00:00Z"}]}' > "$tmp/config.json"
GZY_CONFIG="$tmp/config.json" GZY_BIN_DIR="$tmp/bin" ./dist/gzy install
"$tmp/bin/git-p" status
```

Expected: the wrapper calls `gzy run p -- status`. If run outside a Git repo, Git may print `fatal: not a git repository`; that is acceptable because it proves delegation reached real Git.

- [ ] **Step 10: Commit**

Run:

```sh
git add README.md cmd/gzy/main.go internal/app/app.go internal/app/app_test.go
git commit -m "docs: add gzy usage guide"
```

## Self-Review Checklist

- Spec coverage: The tasks cover Go CLI creation, account config, alias wrappers, SSH key management, OAuth device flow, key upload, Git execution, export/import foundations, doctor checks, installers, release build scripts, README usage, and tests.
- Scope check: GitHub key deletion, long-lived token storage, private key export, and advanced HTTPS credential manipulation remain out of scope as specified.
- Type consistency: `config.Account`, `config.Config`, `setup.Service`, `githubauth.Client`, and `gitrun.Runner` are named consistently across tasks.
- Verification command before completion: run `go test ./...`, `go build -o ./dist/gzy ./cmd/gzy`, `sh -n install.sh`, and `sh -n scripts/build-release.sh`.
