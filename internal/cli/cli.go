package cli

import (
	"fmt"
	"io"
)

type Deps struct {
	Version string
}

func Run(args []string, stdout io.Writer, stderr io.Writer, deps Deps) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "gzy %s", deps.Version)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 2
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "gzy")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  add")
	fmt.Fprintln(w, "  run")
	fmt.Fprintln(w, "  doctor")
}
