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

- `internal/app/` — frontend-independent application layer. `app.Service` bundles the resolver, appview, and auth dependencies and exposes every operation (target resolution, issue/PR/repo/string/SSH-key CRUD, auth flows) as methods returning typed domain structs. All application logic lives here.
- `internal/cli/` — thin Cobra frontend over `internal/app`: one file per command, all built in `NewRoot`. Each `RunE` parses flags/args into a service call and renders the returned struct.
- `tangled/` — read-only client for the Bobbin appview XRPC API (default `https://bobbin.klbr.net`; override via `--appview`/`TG_APPVIEW`).
- Writes go two ways: PDS record mutations with the user's session (`atproto/`), and knot server RPCs (`knot/`) authed with a PDS-minted service-auth JWT. Both are orchestrated by `internal/app`.
- `atproto/` — handle↔DID resolution, PDS discovery, OAuth + app-password sessions stored in the OS keyring.
- `internal/gitutil/` — git subprocesses (clone, fetch, patch apply); its tests need `git` in PATH.

## Conventions

- New command: add the service method to `internal/app/`, then a thin file in `internal/cli/` that parses flags and calls it, registered with `AddCommand` in `NewRoot`.
- Commands returning data must support `--json` via `output(cmd, data, human)` in `internal/cli/output.go`. The JSON structs are the canonical domain types in `internal/app/types.go`.
- Repo-targeting commands accept `handle/repo` or auto-detect from the `origin` remote (see `internal/app/target.go` and the `resolveTarget`/`resolveTargetFlag` shims in `internal/cli/target.go`).
- Config is Viper: flag > `TG_` env var > `$XDG_CONFIG_HOME/tg/config.toml` > default.
- Tests are plain unit tests with table-driven style; keyring tests use an in-memory fake, no real keyring needed. Tests for application logic live in `internal/app/`.
- Commit messages: lowercase `<area>: <change>`, area = package/command path (e.g. `app:`, `cli:`, `atproto/auth:`, `nix/tg:`, `gitutil:`).
