# tg

`tg` is a command-line client for [Tangled](https://tangled.org), the git forge built on atproto. It is an analogue to the GitHub CLI (`gh`), using Bobbin for reads and authenticated PDS/knot writes.

## Installation

### Homebrew (stable, recommended, macOS+Linux)

```bash
brew tap alyraffauf/tap
brew trust alyraffauf/tap
brew install alyraffauf/tap/tg
```

### With Nix (latest dev release)

```bash
nix profile add github:alyraffauf/tg
```

### From source

```bash
go install github.com/alyraffauf/tg/cmd/tg@latest
```

## Quick start

```bash
# Log in (OAuth, or pass an app password for headless use)
tg auth login alice.example.com

# Clone a repository
tg repo clone microcosm.blue/microcosm-rs

# Clone by repository DID
tg repo clone did:plc:example

# Work with issues and pull requests
tg issue list
tg issue create --body "Details" "Bug report"
tg pr create --title "Add feature" --base main
tg pr merge <rkey>
```

`tg` auto-detects the repository from Git remotes when run inside a cloned Tangled repo, checking `origin` first. It recognizes remotes that identify a repository by `handle/repo` or by repository DID. Hosted Tangled remotes support SSH, Git, HTTP, and HTTPS URLs. Custom Knot remotes support SSH and HTTPS URLs. `tg` verifies a custom Knot against the repository record before it accepts the remote.

When a repository record contains a repository DID, `tg` uses the DID in new Git remotes. Automatically selected Knots use the `tangled.org` proxy. An explicitly configured Knot uses a direct remote and the configured SSH port. These remotes remain valid after handle changes, repository renames, and transfers. If `tg` cannot resolve the repository record for an SSH clone, it creates a `handle/repo` remote and prints the resolution error as a warning.

## Documentation

- Command reference — `tg <command> --help`, the man pages installed by the Nix package (`man tg`, `man tg-issue-list`, ...), or the [website](https://tgcli.wisp.place/reference/commands/tg/)
- [Authentication](https://tgcli.wisp.place/cookbooks/authentication/) — OAuth and app-password login, multiple accounts, keyring token storage
- [Configuration](https://tgcli.wisp.place/cookbooks/configuration/) — config file, environment variables, and flags

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and an overview of the codebase.

## License

See [LICENSE](LICENSE.md).
