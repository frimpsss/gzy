# gzy

`gzy` lets you use multiple GitHub accounts on one computer through simple Git aliases like `git-p` (personal) and `git-w` (work). Each alias uses its own SSH key, commit name, and email — no more `git config` juggling.

This README walks you through building `gzy` **from source** and using it on your own machine. No prior Go experience required.

---

## 1. Prerequisites

You need three things installed:

| Tool   | Check with         | Get it from                                     |
| ------ | ------------------ | ----------------------------------------------- |
| Go 1.22+ | `go version`     | https://go.dev/dl/                              |
| Git    | `git --version`    | https://git-scm.com/downloads                   |
| ssh-keygen | `ssh-keygen -V` | Already on macOS / Linux. Windows: install Git Bash or OpenSSH. |

If `go version` prints something like `go version go1.22.x` or higher, you're good.

---

## 2. Get the code

```sh
git clone https://github.com/frimpsss/gzy.git
cd gzy
```

---

## 3. Create a GitHub OAuth App (one-time, ~2 minutes)

`gzy init` opens your browser to sign in to GitHub and upload an SSH key for you. To do that, it needs a free **OAuth App Client ID**. This is a public ID — not a secret.

1. Go to https://github.com/settings/developers → **New OAuth App**.
2. Fill in:
   - **Application name:** `gzy-local` (anything is fine)
   - **Homepage URL:** `http://localhost`
   - **Authorization callback URL:** `http://localhost`
3. Click **Register application**.
4. On the next page, click **Enable Device Flow** and **Update application**.
5. Copy the **Client ID** (looks like `Iv1.abc123...` or `Ov23li...`).

Now wire it into your local build:

```sh
cp .env.example .env.local
```

Open `.env.local` and replace the placeholder:

```
GZY_GITHUB_CLIENT_ID=Iv1.your-client-id-here
```

> `.env.local` is gitignored, so this stays on your machine.

---

## 4. Build the binary

One command builds `gzy` and drops it in `~/.local/bin`:

```sh
./scripts/install-local.sh
```

You should see:

```
embedding GZY_GITHUB_CLIENT_ID from .env.local
built /Users/you/.local/bin/gzy
installed: /Users/you/.local/bin/gzy
```

Verify it works:

```sh
gzy version
```

> **"command not found: gzy"?** Your shell can't find `~/.local/bin`. Add it to your `PATH`:
>
> ```sh
> echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
> source ~/.zshrc
> ```
>
> (Use `~/.bashrc` instead if you're on bash.)

**Alternative: build without installing**

```sh
./scripts/build.sh           # produces ./dist/gzy
./dist/gzy version
```

---

## 5. Set up your first account

```sh
gzy init
```

`gzy` will ask you for:

- **Alias** — short tag like `p` (personal) or `w` (work). This becomes the command `git-p` / `git-w`.
- **GitHub username** — your `github.com/<username>`.
- **Commit name** — what shows up as the author on commits.
- **Commit email** — use the `@users.noreply.github.com` email from https://github.com/settings/emails if you want to keep your real email private.
- **SSH key** — let `gzy` generate a fresh one (recommended) or point at an existing key.

When prompted, `gzy` opens your browser to authorize the OAuth app. After you approve it, `gzy`:

1. Uploads your new SSH public key to that GitHub account.
2. Writes a `git-<alias>` wrapper into `~/.local/bin` (or `~/bin` on macOS by default).
3. Saves your config to the platform config directory.

Repeat `gzy init` for each extra account (e.g. once with alias `p`, again with alias `w`).

---

## 6. Use it

Use the `git-<alias>` wrappers exactly like `git`:

```sh
git-p clone git@github.com:you/personal-repo.git
cd personal-repo
git-p status
git-p commit -m "first commit"
git-p push

# In a different repo:
git-w clone git@github.com:your-company/work-repo.git
cd work-repo
git-w pull
```

Each wrapper transparently uses the right SSH key, name, and email for that account. Your normal `git` command still works the same way it did before.

---

## 7. Useful commands

```sh
gzy list                # show configured accounts
gzy doctor              # check that SSH keys, wrappers, and PATH look right
gzy install             # regenerate wrappers (e.g. after editing config)
gzy reset               # wipe gzy config, wrappers, and generated keys
gzy export > my.json    # export config (without private keys)
gzy import my.json      # import config on a new machine
gzy version
```

---

## Where things live

| What                        | Path                                             |
| --------------------------- | ------------------------------------------------ |
| Built binary                | `~/.local/bin/gzy`                               |
| Generated wrappers          | `~/.local/bin/git-<alias>` (Linux) or `~/bin/git-<alias>` (macOS) |
| Config file                 | Run `gzy doctor` to see the platform path        |
| SSH keys (when gzy generates them) | `~/.ssh/gzy_<alias>` and `~/.ssh/gzy_<alias>.pub` |

---

## Troubleshooting

**`git-p: command not found`** — The wrappers directory isn't on your `PATH`. Run `gzy doctor`; it will print the exact directory. Add it to `PATH` as in step 4.

**Browser auth fails / `GZY_GITHUB_CLIENT_ID is not set`** — You skipped step 3, or built before creating `.env.local`. Re-run `./scripts/install-local.sh`. You can also set it at runtime: `GZY_GITHUB_CLIENT_ID=Iv1.xxx gzy init`.

**Want to skip the browser?** — Set `GZY_NO_BROWSER=1`. `gzy` will print the device-flow code for you to enter manually.

**Pushing to the wrong account?** — You're probably using plain `git push`. Use `git-<alias> push` so the right SSH key is selected.

**Want to start over?** — `gzy reset` wipes config, wrappers, and gzy-generated keys.

---

## Trying it without touching your real setup

Want to experiment with `gzy` against a throwaway config? Point it at a temp directory:

```sh
tmp="$(mktemp -d)"
export GZY_CONFIG="$tmp/config.json"
export GZY_BIN_DIR="$tmp/bin"

./dist/gzy init     # writes only into $tmp, leaves your real config alone
./dist/gzy list
```

When you're done, `unset GZY_CONFIG GZY_BIN_DIR` and `gzy` is back to its real config.

---

## Environment variables (advanced)

| Variable                | Purpose                                                                                          |
| ----------------------- | ------------------------------------------------------------------------------------------------ |
| `GZY_CONFIG`            | Override the config file path.                                                                   |
| `GZY_BIN_DIR`           | Override where `git-<alias>` wrappers are written.                                               |
| `GZY_GITHUB_CLIENT_ID`  | OAuth Client ID for the browser flow. Overrides whatever was baked in from `.env.local`.         |
| `GZY_NO_BROWSER`        | Set to any value to skip auto-opening the browser during device-flow auth.                       |

---

## Developer reference

Run the test suite:

```sh
go test ./...
```

Run a single package:

```sh
go test ./internal/wrapper -v
```

Run `gzy` straight from source without building:

```sh
go run ./cmd/gzy version
go run ./cmd/gzy list
```

Cross-compile:

```sh
GOOS=linux  GOARCH=amd64 go build -o ./dist/gzy-linux-amd64  ./cmd/gzy
GOOS=darwin GOARCH=arm64 go build -o ./dist/gzy-darwin-arm64 ./cmd/gzy
```
