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
