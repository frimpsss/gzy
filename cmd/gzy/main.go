package main

import (
	"os"

	"github.com/frimpsss/gzy/internal/app"
	"github.com/frimpsss/gzy/internal/cli"
	"github.com/frimpsss/gzy/internal/paths"
)

var (
	version               = "dev"
	defaultGitHubClientID = ""
)

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
	tokenDir := os.Getenv("GZY_TOKEN_DIR")
	if tokenDir == "" {
		tokenDir = p.TokenDir()
	}
	clientID := os.Getenv("GZY_GITHUB_CLIENT_ID")
	if clientID == "" {
		clientID = defaultGitHubClientID
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
			TokenDir:       tokenDir,
			GitHubClientID: clientID,
		}),
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, deps))
}
