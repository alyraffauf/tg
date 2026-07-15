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
tg pr diff <rkey>
tg pr comment <rkey> --body "Looks good"
tg pr close <rkey>
tg pr merge <rkey>
```

### Other commands

`tg repo edit`, `tg repo set-default-branch`, `tg repo delete --yes`, `tg repo fork`,
`tg ssh-key delete`, `tg browse`, `tg completion`, `tg auth token`, and `tg api` are available.

Changing scopes requires existing users to run `tg auth login` again.

## Platform limits

Tangled currently has no API for releases, gists, workflows/runs, repository archive/sync/rename,
PR checks, projects, codespaces, a status dashboard, or global text search. `tg pr checkout` is
deferred until PRs expose an immutable base commit/ref, allowing a safe isolated-worktree
implementation. `auth switch` is deferred because the local auth store currently supports one
account.

## Architecture

- `cmd/tg/` — CLI entry point
- `internal/cli/` — Cobra command tree (`repo`, `issue`, `pr`)
- `internal/gitutil/` — Git operations (clone, fetch, patch apply)
- `tangled/` — Typed client for the Bobbin API (`api.tangled.org`)
- `atproto/` — Identity resolution (handle ↔ DID, PDS discovery)

## Dependencies

- Go 1.26+
- `git` and `ssh` (for clone and PR checkout)
- [`github.com/bluesky-social/indigo`](https://github.com/bluesky-social/indigo) — atproto SDK
- [`github.com/spf13/cobra`](https://github.com/spf13/cobra) — CLI framework

## License

See [LICENSE](LICENSE.md).
