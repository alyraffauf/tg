# Show available recipes
default:
    @just --list

# Cut a new release: bump version, run the gate, commit, tag, push
release version:
    #!/usr/bin/env bash
    set -euo pipefail
    version="{{version}}"
    tag="v${version}"
    versionfile="internal/cli/root.go"
    mcpversionfile="mcp/pyproject.toml"
    mcplockfile="mcp/uv.lock"

    if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "error: '$version' is not valid semver (expected X.Y.Z)" >&2
        exit 1
    fi

    branch="$(git symbolic-ref --short HEAD)"
    if [ "$branch" != "master" ]; then
        echo "error: must be on master (currently on $branch)" >&2
        exit 1
    fi

    if git rev-parse --verify --quiet "refs/tags/${tag}" >/dev/null; then
        echo "error: tag ${tag} already exists" >&2
        exit 1
    fi

    nix fmt

    dirty="$(git status --porcelain -- ':!'"${versionfile}" ':!'"${mcpversionfile}" ':!'"${mcplockfile}")"
    if [ -n "$dirty" ]; then
        echo "error: working tree has changes outside version files:" >&2
        echo "$dirty" >&2
        exit 1
    fi

    sed -i -E "s/^(\s*version\s*=\s*)\"[^\"]+\"/\1\"${version}\"/" "${versionfile}"
    if ! grep -qE "version\s*=\s*\"${version}\"" "${versionfile}"; then
        echo "error: failed to update version in ${versionfile}" >&2
        exit 1
    fi

    sed -i -E "s/^(version = )\"[^\"]+\"/\1\"${version}\"/" "${mcpversionfile}"
    if ! grep -qFx "version = \"${version}\"" "${mcpversionfile}"; then
        echo "error: failed to update version in ${mcpversionfile}" >&2
        exit 1
    fi

    (cd mcp && uv lock)
    if ! grep -qFx "version = \"${version}\"" "${mcplockfile}"; then
        echo "error: failed to update version in ${mcplockfile}" >&2
        exit 1
    fi

    nix build
    nix build .#tgmcp
    (cd mcp && uv run --locked pytest)

    git add "${versionfile}" "${mcpversionfile}" "${mcplockfile}"
    git commit -m "release: ${tag}"
    git tag -a "${tag}" -m "${tag}"
    git push upstream master "${tag}"
    git push origin master "${tag}"
