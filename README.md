# tg

`tg` is a command-line client for [Tangled](https://tangled.org), the git forge built on atproto. It is an analogue to the GitHub CLI (`gh`), using Bobbin for reads and authenticated PDS/knot writes.

## Installation

### With Nix

```bash
nix profile add github:alyraffauf/tg
```

### From source

```bash
go install github.com/alyraffauf/tg/cmd/tg@latest
```

## Usage

### Authentication

Log in interactively with OAuth:

```bash
tg auth login alice.example.com
```

For headless use, pass an atproto app password as the second argument:

```bash
tg auth login alice.example.com xxxx-xxxx-xxxx-xxxx
```

To avoid exposing the app password in shell history, pass it on standard input:

```bash
printf '%s\n' "$ATPROTO_APP_PASSWORD" | tg auth login alice.example.com --password-stdin
```

Authentication is persisted locally. The current account is recorded in
`~/.config/tg/auth.json` (or `$XDG_CONFIG_HOME/tg/auth.json`); OAuth session
credentials are stored under `~/.config/tg/oauth/`, and app-password sessions
are stored in `~/.config/tg/password-session.json`. These files are created
with user-only permissions. Use `tg auth logout` to remove the active login.

`tg` auto-detects the repository from the `origin` remote when run inside a cloned Tangled repo. For now, only ssh origins are supported. You can also pass a fully-qualified `handle/repo` argument.

### Repositories

```bash
# List repositories for a user
tg repo list microcosm.blue

# Create a repository (requires `tg auth login`)
tg repo create my-tool --description "A small tool"

# Create and clone it into the current directory
tg repo create my-tool --clone

# Create and push an existing local repo (at the given path) to the new remote
tg repo create my-tool --push=.

# Clone a repository
tg repo clone microcosm.blue/microcosm-rs

# Clone into a custom directory
tg repo clone microcosm.blue/microcosm-rs my-fork
```

### Issues

```bash
# List issues for the current repository
tg issue list

# List issues for an explicit repository
tg issue list microcosm.blue/microcosm-rs

# Create, comment on, and update an issue
tg issue create --repo microcosm.blue/microcosm-rs --body "Details" "Bug report"
tg issue comment <rkey> --body "I can reproduce this"
tg issue close <rkey>
tg issue reopen <rkey>
tg issue edit <rkey> --title "Updated title"
```

### Pull Requests

```bash
# List pull requests
tg pr list

# Create, comment on, inspect, and update pull requests
tg pr create --title "Add feature" --base main
# Reconstruct the latest round on the current remote target branch
tg pr checkout <rkey>
tg pr diff <rkey>
tg pr comment <rkey> --body "Looks good"
tg pr close <rkey>
tg pr merge <rkey>
```

### Other commands

`tg repo edit`, `tg repo set-default-branch`, `tg repo delete --yes`, `tg repo fork`,
`tg ssh-key delete`, `tg browse`, `tg completion`, `tg auth token`, and `tg api` are available.

### Authentication & token storage

`tg` stores a single OAuth session in the system keyring: macOS Keychain or
the Secret Service on Linux (GNOME Keyring / KWallet). The keyring unlocks
with your login session, so no separate passphrase is needed.

Logging in again replaces the current session. The keyring is accessed on
first use (not at startup), so authentication only fails once you run a
command that needs a session. On Linux this requires a Secret Service
provider to be running; on a headless system without a D-Bus session bus
(e.g. a server or container), install and start `gnome-keyring-daemon` or
`kwalletd`, or set `DBUS_SESSION_BUS_ADDRESS`.

## Configuration

`tg` resolves configuration values from the following sources, in increasing
precedence (later sources override earlier ones):

1. **Defaults** — `appview` is `https://bobbin.klbr.net`
2. **Config file** — `$XDG_CONFIG_HOME/tg/config.toml` (or `~/.config/tg/config.toml`)
3. **Environment variables** — prefixed `TG_` (e.g. `TG_APPVIEW`)
4. **Command-line flags** — e.g. `--appview`

The config file is optional; a missing file is not an error.

### Config file

```toml
# ~/.config/tg/config.toml
appview = "https://bobbin.klbr.net"
```

Override the config file location with `--config /path/to/config.toml`.

### Environment variables

| Variable      | Config key | Purpose           |
|---------------|------------|-------------------|
| `TG_APPVIEW`  | `appview`  | Appview host URL  |

Keys containing `.` or `-` map to `TG_`-prefixed underscore-separated names
(e.g. `foo.bar` → `TG_FOO_BAR`).

### Flags

| Flag        | Purpose                                                          |
|-------------|------------------------------------------------------------------|
| `--config`  | Path to config file                                              |
| `--appview` | Appview host URL (overrides config file and `TG_APPVIEW`)       |

## Architecture

- `cmd/tg/` — CLI entry point
- `internal/cli/` — Cobra command tree (`repo`, `issue`, `pr`); Viper-backed configuration
- `internal/gitutil/` — Git operations (clone, fetch, patch apply)
- `tangled/` — Typed client for the Bobbin API (`api.tangled.org`)
- `atproto/` — Identity resolution (handle ↔ DID, PDS discovery); OAuth session storage (system keyring)

## Dependencies

- Go 1.26+
- `git` and `ssh` (for clone and PR checkout)
- [`github.com/bluesky-social/indigo`](https://github.com/bluesky-social/indigo) — atproto SDK
- [`github.com/spf13/cobra`](https://github.com/spf13/cobra) — CLI framework
- [`github.com/spf13/viper`](https://github.com/spf13/viper) — configuration
- [`github.com/zalando/go-keyring`](https://github.com/zalando/go-keyring) — platform keyring access

## License

See [LICENSE](LICENSE.md).
