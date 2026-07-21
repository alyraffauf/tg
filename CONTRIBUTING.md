# Contributing

## Before committing

- [ ] `nix fmt` — everything is formatted
- [ ] `nix build` passes — compiles, runs `go test ./...`, builds completions and man pages
- [ ] If `go.mod`/`go.sum` changed: `vendorHash` in `nix/tg.nix` is updated and the build passes with it (see [Updating `vendorHash`](#updating-vendorhash))
- [ ] Help text (`Short`, `Long`, flag descriptions) is accurate for any new or changed commands
- [ ] README and docs updated if user-facing behavior changed

## Development

```bash
# Build
go build ./cmd/tg

# Test
go test ./...
```

## Nix

Nix is a first-class citizen: the flake builds the package, runs the tests,
and generates the shell completions and man pages. `go build` passing is not
enough — verify changes with `nix build` before committing.

```bash
# Full package build (also runs go test ./...)
nix build

# Dev shell with Go and the treefmt formatters
nix develop

# Format Go, Nix, Markdown, and shell files
nix fmt
```

Without Nix, run the formatters `nix fmt` wraps directly: `gofmt -w .` for Go
(ships with the Go toolchain), `prettier --write .` for Markdown, and
`alejandra .` for Nix files.

### Updating `vendorHash`

The Nix package pins a `vendorHash` of the Go module dependencies in
`nix/tg.nix`. Any change to `go.mod` or `go.sum` — adding, removing, or
bumping a dependency — invalidates it, and `nix build` fails with a hash
mismatch:

```
error: hash mismatch in fixed-output derivation:
         specified: sha256-AAAA...
            got:    sha256-BBBB...
```

Copy the `got:` hash into `vendorHash` in `nix/tg.nix` and re-run `nix build`.
Do not commit dependency changes until the Nix build passes with the updated
hash.

## Documentation

Command help text (`Short`, `Long`, and flag descriptions) is the user-facing
reference: it powers `tg <command> --help` and the man pages, which the Nix
derivation generates at build time via the hidden `tg man` command. Keep it
accurate when adding or changing commands; no other documentation step is
needed.

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
