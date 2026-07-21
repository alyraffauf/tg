# AGENTS.md

`tg` is a Go 1.26+ CLI for Tangled (git forge on atproto), built with Cobra + Viper. Entry point: `cmd/tg/main.go`.

## Verify changes (in order)

1. `nix fmt` — treefmt wrapper (gofmt, prettier, alejandra, shfmt/shellcheck, statix, deadnix). Without Nix, run those formatters directly.
2. `nix build` — the real gate: compiles, runs `go test ./...`, and generates man pages + shell completions. **`go build` passing is not enough.**
3. Quick iteration only: `go build ./cmd/tg`, `go test ./...`.

## Nix gotchas

- Changing `go.mod`/`go.sum` invalidates `vendorHash` in `nix/tg.nix`. `nix build` fails with a hash mismatch; copy the `got:` hash into `vendorHash` and rebuild. Don't commit dependency changes until the build passes with the updated hash.
- Man pages are generated at build time by the hidden `tg man` command (invoked from `nix/tg.nix`). There is no separate docs step: command help text (`Short`, `Long`, flag descriptions) is the entire user-facing reference, so keep it accurate when adding/changing commands.

## CI

Tangled pipelines in `.tangled/workflows/` (nixery engine), not GitHub Actions: `nix build .#tg` and `go test ./...`. Default branch is `master`.

## Architecture

- `internal/cli/` — Cobra command tree, one file per command, all wired in `root.go`'s `init()`.
- `tangled/` — read-only client for the Bobbin appview XRPC API (default `https://bobbin.klbr.net`; override via `--appview`/`TG_APPVIEW`).
- Writes go two ways: PDS record mutations with the user's session (`atproto/`, `internal/cli/record_mutations.go`), and knot server RPCs (`knot/`) authed with a PDS-minted service-auth JWT.
- `atproto/` — handle↔DID resolution, PDS discovery, OAuth + app-password sessions stored in the OS keyring.
- `internal/gitutil/` — git subprocesses (clone, fetch, patch apply); its tests need `git` in PATH.

## Conventions

- New command: new file in `internal/cli/`, register with `AddCommand` in `root.go`.
- Commands returning data must support `--json` via the generic `output(data, humanFunc)` helper in `internal/cli/output.go` (human renderer + JSON struct with tags).
- Repo-targeting commands accept `handle/repo` or auto-detect from the `origin` remote (see `internal/cli/target.go`).
- Config is Viper: flag > `TG_` env var > `$XDG_CONFIG_HOME/tg/config.toml` > default.
- Tests are plain unit tests with table-driven style; keyring tests use an in-memory fake, no real keyring needed.
- Commit messages: lowercase `<area>: <change>`, area = package/command path (e.g. `cli:`, `atproto/auth:`, `nix/tg:`, `gitutil:`).
