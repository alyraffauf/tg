# Configuration

`tg` resolves configuration values from the following sources, in increasing
precedence (later sources override earlier ones):

1. **Defaults** — `appview` is `https://bobbin.klbr.net`; `knot` is unset to permit automatic verified Knot discovery; `ssh-port` is `22`
2. **Config file** — `$XDG_CONFIG_HOME/tg/config.toml` (or `~/.config/tg/config.toml`)
3. **Environment variables** — prefixed `TG_` (e.g. `TG_APPVIEW`)
4. **Command-line flags** — e.g. `--appview`

The config file is optional; a missing file is not an error.

## Config file

```toml
# ~/.config/tg/config.toml
appview = "https://bobbin.klbr.net"
knot = "knot.example.com"
ssh-port = 2222
```

`knot` and `ssh-port` are defaults for `tg repo create`. They select where a
repository is provisioned and the SSH port used when constructing its initial
clone or push remote. Existing repositories continue to use their configured
Git remotes.

Override the config file location with `--config /path/to/config.toml`.

## Environment variables

| Variable      | Config key | Purpose                            |
| ------------- | ---------- | ---------------------------------- |
| `TG_APPVIEW`  | `appview`  | Appview host URL                   |
| `TG_ACCOUNT`  | `account`  | Account handle or DID              |
| `TG_KNOT`     | `knot`     | Knot host for repo creation        |
| `TG_SSH_PORT` | `ssh-port` | SSH port used during repo creation |

Keys containing `.` or `-` map to `TG_`-prefixed underscore-separated names
(e.g. `foo.bar` → `TG_FOO_BAR`).

## Flags

| Flag        | Purpose                                                   |
| ----------- | --------------------------------------------------------- |
| `--config`  | Path to config file                                       |
| `--appview` | Appview host URL (overrides config file and `TG_APPVIEW`) |
| `--account` | Account handle or DID for this command                    |

## Automatic verified Knot discovery

When you don't specify a Knot with `--knot`, repo creation reads up to 10 Knot
registrations from your PDS and verifies each registration. Select a Knot
explicitly if you have more than 10 registrations. Exactly one verified Knot is
selected automatically. If no registrations exist, it falls back to
`knot1.tangled.sh`. One registration that can't be verified errors. If `tg`
successfully verifies multiple Knots, it errors and lists the candidates so you
can specify one with `--knot` or set it in the config file.

Set `knot` in your config, `TG_KNOT` in your environment, or pass `--knot` to
select a host explicitly and bypass automatic discovery. The flag overrides
the environment, which overrides the config file.
