#!/usr/bin/env sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <tangled-core-commit>" >&2
  exit 2
fi

commit="$1"
case "$commit" in
*[!0123456789abcdef]*)
  echo "commit must be a 40-character lowercase hexadecimal SHA" >&2
  exit 2
  ;;
esac
if [ "${#commit}" -ne 40 ]; then
  echo "commit must be a 40-character lowercase hexadecimal SHA" >&2
  exit 2
fi
source_url="https://tangled.org/tangled.org/core.git"
temporary_directory=$(mktemp -d)
trap 'rm -rf "$temporary_directory"' EXIT

git init --quiet "$temporary_directory"
git -C "$temporary_directory" remote add origin "$source_url"
git -C "$temporary_directory" fetch --quiet --depth=1 origin "$commit"
git -C "$temporary_directory" checkout --quiet --detach FETCH_HEAD

resolved_commit=$(git -C "$temporary_directory" rev-parse HEAD)
if [ "$resolved_commit" != "$commit" ]; then
  echo "resolved $resolved_commit, expected $commit" >&2
  exit 1
fi

while IFS= read -r schema_path; do
  [ -n "$schema_path" ] || continue
  destination="schemas/$schema_path"
  mkdir -p "$(dirname "$destination")"
  git -C "$temporary_directory" show "HEAD:lexicons/$schema_path" >"$destination"
done <schemas.txt

printf '{\n  "source": "%s",\n  "commit": "%s",\n  "schemaPaths": "schemas.txt"\n}\n' "$source_url" "$resolved_commit" >lock.json
go generate .
