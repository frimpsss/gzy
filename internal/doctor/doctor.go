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
