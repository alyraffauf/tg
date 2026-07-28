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

## Quick start

```bash
# Log in (OAuth, or pass an app password for headless use)
tg auth login alice.example.com

# Clone a repository
tg repo clone microcosm.blue/microcosm-rs

# Work with issues and pull requests
tg issue list
tg issue create --body "Details" "Bug report"
tg pr create --title "Add feature" --base main
tg pr merge <rkey>
```

`tg` auto-detects the repository from Git remotes when run inside a cloned Tangled repo, checking `origin` first. Hosted Tangled remotes support SSH, Git, HTTP, and HTTPS URLs. Custom Knot remotes support SSH URLs and are accepted only when the host matches the repository's canonical record. You can also pass a fully qualified `handle/repo` argument.

## Documentation

- Command reference — `tg <command> --help`, or the man pages installed by the Nix package (`man tg`, `man tg-issue-list`, ...)
- [Authentication](docs/authentication.md) — OAuth and app-password login, multiple accounts, keyring token storage
- [Configuration](docs/configuration.md) — config file, environment variables, and flags

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and an overview of the codebase.

## License

See [LICENSE](LICENSE.md).
