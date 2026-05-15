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

## Test Locally

Build the binary and exercise it against a temporary config without touching your real one:

```sh
# Build into ./dist
go build -o ./dist/gzy ./cmd/gzy
./dist/gzy version

# Create a throwaway config
tmp="$(mktemp -d)"
cat > "$tmp/config.json" <<EOF
{
  "version": 1,
  "accounts": [{
    "alias": "p",
    "command": "git-p",
    "github_user": "frimpsss",
    "name": "Akwasi Frimpong",
    "email": "you@example.com",
    "private_key": "$HOME/.ssh/id_ed25519",
    "public_key": "$HOME/.ssh/id_ed25519.pub",
    "created_at": "2026-05-15T00:00:00Z"
  }]
}
EOF

# Point gzy at the temp config + bin dir
export GZY_CONFIG="$tmp/config.json"
export GZY_BIN_DIR="$tmp/bin"

./dist/gzy list
./dist/gzy doctor
./dist/gzy install        # writes git-p into $GZY_BIN_DIR
"$tmp/bin/git-p" status   # delegates through gzy run p -- status
```

Environment variables for local runs:

- `GZY_CONFIG` — path to the config JSON (defaults to the platform config location).
- `GZY_BIN_DIR` — directory where `git-<alias>` wrappers are written (defaults to the platform bin location).
- `GZY_GITHUB_CLIENT_ID` — OAuth client ID used by `gzy init` / `gzy auth` browser flow.

Run a single package's tests with verbose output:

```sh
go test ./internal/wrapper -v
go test ./internal/cli -run TestListPrintsAccounts -v
```

Cross-compile a release-style binary:

```sh
GOOS=linux GOARCH=amd64 go build -o ./dist/gzy-linux-amd64 ./cmd/gzy
```
