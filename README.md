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
