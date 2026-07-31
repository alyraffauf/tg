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

    dirty="$(git status --porcelain -- ':!'"${versionfile}")"
    if [ -n "$dirty" ]; then
        echo "error: working tree has changes outside ${versionfile}:" >&2
        echo "$dirty" >&2
        exit 1
    fi

    nix build

    sed -i -E "s/^(\s*version\s*=\s*)\"[^\"]+\"/\1\"${version}\"/" "${versionfile}"
    if ! grep -qE "version\s*=\s*\"${version}\"" "${versionfile}"; then
        echo "error: failed to update version in ${versionfile}" >&2
        exit 1
    fi

    git add "${versionfile}"
    git commit -m "release: ${tag}"
    git tag -a "${tag}" -m "${tag}"
    git push upstream master "${tag}"
    git push origin master "${tag}"
