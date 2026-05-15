# gzy Design

## Goal

Build `gzy`, a tiny cross-platform CLI that lets users run Git through account-specific aliases such as `git-p` or `git-w`. The target user is a beginner who should not need to understand SSH key paths, Git credential internals, or OS-specific setup.

All project code and documentation live under:

```text
/Users/bigfrimps/Documents/workspace/experiments/gzy
```

## User Experience

A user installs the `gzy` binary, then runs:

```sh
gzy init
```

The setup asks for beginner-friendly inputs:

- Alias command suffix, such as `p` for `git-p`
- GitHub username
- Git commit name
- Git commit email
- SSH key choice: create a new key or use a detected existing key

For daily use, the generated commands behave like normal Git:

```sh
git-p clone git@github.com:user/repo.git
git-p status
git-w push
git-p commit -m "message"
```

## Installation

`gzy` should be installable without requiring users to clone the repository or install Go.

The project ships prebuilt release binaries for macOS, Linux, and Windows. It also ships installer scripts at the repository root:

- `install.sh` for macOS and Linux
- `install.ps1` for Windows PowerShell

The public README should support these install commands once the GitHub repository is published at `frimpsss/gzy`:

```sh
curl -fsSL https://raw.githubusercontent.com/frimpsss/gzy/main/install.sh | sh
```

```sh
wget -qO- https://raw.githubusercontent.com/frimpsss/gzy/main/install.sh | sh
```

For Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/frimpsss/gzy/main/install.ps1 | iex
```

Windows users who prefer `curl.exe` can download and run the PowerShell installer:

```powershell
curl.exe -fsSL https://raw.githubusercontent.com/frimpsss/gzy/main/install.ps1 -o install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

The installers detect OS and CPU architecture, download the matching release binary, install it into a user-writable bin directory, and print exact `PATH` instructions when the chosen directory is not already available in the shell.

## CLI Commands

`gzy init` starts a guided first-run setup and creates the first account alias.

`gzy add` adds another GitHub account and generates its alias wrapper.

`gzy list` shows configured accounts, aliases, commit identities, and SSH key status without printing private key contents.

`gzy remove <alias>` removes one configured account and its generated wrappers.

`gzy install` recreates wrapper commands from the saved config.

`gzy export` prints portable JSON config for moving setup to another machine. Private SSH keys are never exported by default.

`gzy import <file>` imports portable config and recreates missing wrappers. If referenced SSH keys are missing, `gzy` guides the user through creating replacement keys.

`gzy doctor` checks Git, SSH, wrapper install location, configured key files, and basic GitHub SSH authentication readiness.

`gzy run <alias> -- <git args...>` is the internal command used by generated wrappers.

## Architecture

The main application is a Go CLI named `gzy`.

Generated wrappers stay intentionally small. For example, `git-p status` delegates to:

```sh
gzy run p -- status
```

The Go binary owns all account lookup, OS-specific path handling, SSH key discovery, Git config injection, and process execution. This keeps wrapper files disposable and easy to regenerate.

## Account Model

Each account stores:

- Alias suffix, such as `p`
- Generated command name, such as `git-p`
- GitHub username
- Commit author name
- Commit author email
- SSH private key path
- Public key path
- Created or imported timestamp

The alias must be unique and safe for command names. Initial validation should allow letters, numbers, underscores, and dashes, while rejecting empty aliases and values that would produce invalid wrapper names.

## Config Locations

`gzy` stores config in the platform-standard user config location:

```text
macOS:   ~/Library/Application Support/gzy/config.json
Linux:   ~/.config/gzy/config.json
Windows: %APPDATA%\gzy\config.json
```

The config format is JSON so it is easy to inspect, export, import, and transfer.

## SSH Key Handling

Users should not need to find SSH key paths manually.

During setup, `gzy` scans common SSH locations:

```text
~/.ssh/id_ed25519
~/.ssh/id_rsa
~/.ssh/github_*
~/.ssh/*.pub
```

Detected keys are displayed with friendly names and fingerprints. If the user chooses to create a key, `gzy` creates an Ed25519 key at a predictable path:

```text
macOS/Linux: ~/.ssh/gzy_<alias>
Windows:     %USERPROFILE%\.ssh\gzy_<alias>
```

After creating or choosing a key, `gzy` copies the public key to the clipboard when possible and opens or prints:

```text
https://github.com/settings/keys
```

The user is instructed to paste the public key into GitHub. If clipboard support is unavailable, `gzy` prints the public key and the file path.

## Git Execution

For every wrapped Git command, `gzy run` executes the system `git` binary with account-specific configuration:

```sh
git -c user.name="<name>" -c user.email="<email>" <args...>
```

For SSH operations, it also injects:

```sh
GIT_SSH_COMMAND="ssh -i <account-key> -o IdentitiesOnly=yes"
```

This lets `git-p push` and `git-w push` use different GitHub SSH identities without requiring manual SSH config edits.

HTTPS remotes still rely on the OS credential manager for GitHub authentication. `gzy` will set the correct commit identity for HTTPS repos, but it cannot safely impersonate a different GitHub HTTPS account unless the user has separate credentials configured. `gzy doctor` should explain this clearly and recommend SSH remotes for the smoothest multi-account flow.

## Wrapper Installation

`gzy install` creates command wrappers in a user-local bin directory.

Preferred locations:

```text
macOS/Linux: ~/.local/bin, falling back to ~/bin
Windows:     %USERPROFILE%\bin or another user-writable directory documented by gzy
```

On macOS and Linux, wrappers are executable shell scripts. On Windows, `gzy` generates `.cmd` wrappers first, with optional PowerShell wrappers later if needed.

If the chosen install directory is not on `PATH`, `gzy` prints exact shell-specific instructions to add it.

## Portability

`gzy export` outputs config without private keys. This makes it safe to share between a user's own machines or store in a dotfiles repo.

On another machine:

```sh
gzy import gzy-config.json
gzy install
gzy doctor
```

If private keys do not exist on that machine, `gzy` guides the user through creating new keys and adding the new public keys to GitHub.

## Error Handling

Errors should be written for beginners. They should say what failed, why it matters, and the next command or action to take.

Examples:

- Git is missing: tell the user to install Git and rerun `gzy doctor`.
- SSH key is missing: offer to create a replacement key.
- Alias wrapper is missing: tell the user to run `gzy install`.
- Public key may not be added to GitHub: tell the user to add it at `https://github.com/settings/keys`.
- HTTPS remote is being used: explain that commit identity is handled, but account login depends on Git credentials.

## Testing

Core logic should be unit-tested without running real GitHub authentication:

- Config load, save, import, and export
- Alias validation
- Platform path selection
- Wrapper file generation
- Git command construction
- SSH key discovery parsing
- Installer OS and architecture selection

Integration tests can use temporary directories and fake `git` or `ssh` binaries on `PATH` to verify environment variables and subprocess arguments without touching the user's real Git configuration.

## Initial Scope

The first implementation includes the Go CLI, config management, guided account setup, SSH key creation or selection, wrapper generation, export/import, doctor checks, release build scripts, and curl/wget/PowerShell installers.

The first implementation does not need GitHub API integration, automatic browser login, private key export, or advanced credential-manager manipulation.
