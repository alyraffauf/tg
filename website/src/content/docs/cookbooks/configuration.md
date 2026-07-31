---
title: Configuration
description: Config file, environment variables, and flags.
---

`tg` resolves configuration values from the following sources, in increasing
precedence (later sources override earlier ones):

1. **Defaults** — `appview` is `https://bobbin.klbr.net`; `knot` is unset to permit automatic verified Knot discovery; `ssh-port` is `22`; `protocol` is `ssh`
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
protocol = "ssh"
```

`knot` and `ssh-port` are defaults for `tg repo create`. They select where a
repository is provisioned and the SSH port used when constructing its initial
clone or push remote. Existing repositories continue to use their configured
Git remotes.

`protocol` selects the URL used by `tg repo clone` and `tg repo create --clone`.
Set it to `ssh` (the default) or `https`. HTTPS clone URLs use the repository's
recorded Knot. `ssh-port` applies to `tg repo create` and its `--clone` or
`--push` setup; standalone SSH clones use port 22.

Override the config file location with `--config /path/to/config.toml`.

## Environment variables

| Variable      | Config key | Purpose                               |
| ------------- | ---------- | ------------------------------------- |
| `TG_APPVIEW`  | `appview`  | Appview host URL                      |
| `TG_ACCOUNT`  | `account`  | Account handle or DID                 |
| `TG_KNOT`     | `knot`     | Knot host for repo creation           |
| `TG_SSH_PORT` | `ssh-port` | SSH port used during repo creation    |
| `TG_PROTOCOL` | `protocol` | Clone URL protocol (`ssh` or `https`) |

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

## Shell completions

Put the relevant `tg completion` command in your shell config to source the
completions on startup _or_ pipe the generated completions to a file somewhere
your shell will source on startup.

### Bash

This goes in your config and increases shell startup latency because of command
invocation overhead, but doesn't require updating later

```bash
source <(tg completion bash)
```

Run these to generate completions that'll later need updating. They depend on
`bash-completion`.

```bash
mkdir -p "${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions"
# run to generate, rerun to update
tg completion bash > "${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions/tg.bash"
```

### Zsh

This goes in your config and increases shell startup latency because of command
invocation overhead, but doesn't require updating later. Put it somewhere
_after_ loading your Zsh framework or `compinit`.

```zsh
source <(tg completion zsh)
```

Run these to generate completions that'll later need updating

```zsh
mkdir -p "${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions"
# run to generate, rerun to update
tg completion zsh > "${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions/_tg"
```

Sourcing the generated completions automatically requires this somewhere in your
config _before_ loading your Zsh framework or `compinit`.

```zsh
fpath=("${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions" $fpath)
```

### Fish

This goes in your config and increases shell startup latency because of command
invocation overhead, but doesn't require updating later

```fish
tg completion fish | source
```

Run these to generate completions that'll later need updating

```fish
mkdir -p "$__fish_config_dir/completions"
# run to generate, rerun to update
tg completion fish > "$__fish_config_dir/completions/tg.fish"
```

### PowerShell

This goes in your PowerShell profile and increases shell startup latency because
of command invocation overhead, but doesn't require updating later

```powershell
tg completion powershell | Out-String | Invoke-Expression
```

Run these to generate completions that'll later need updating

```powershell
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $PROFILE) | Out-Null
# run to generate, rerun to update
tg completion powershell | Set-Content -Encoding utf8 -Path (Join-Path (Split-Path -Parent $PROFILE) "tg-completion.ps1")
```

Then add this to your PowerShell profile:

```powershell
. (Join-Path (Split-Path -Parent $PROFILE) "tg-completion.ps1")
```
