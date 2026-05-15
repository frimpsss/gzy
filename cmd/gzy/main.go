package main

import (
	"os"

	"github.com/frimpsss/gzy/internal/cli"
)

var version = "dev"

func main() {
	code := cli.Run(os.Args[1:], os.Stdout, os.Stderr, cli.Deps{Version: version})
	os.Exit(code)
}
