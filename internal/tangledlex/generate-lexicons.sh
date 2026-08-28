#!/usr/bin/env sh
set -eu

indigo_lexicons=$(go list -m -f '{{.Dir}}' github.com/bluesky-social/indigo)/lexicons
strong_ref_schema="$indigo_lexicons/com/atproto/repo/strongRef.json"

go run github.com/bluesky-social/indigo/cmd/lexgen --build-file lexgen.json --external-lexicons "$strong_ref_schema" schemas

# cbor-gen needs to compile the generated package before it can emit its CBOR
# methods. Temporarily suppress the registry calls, which require those methods.
sed -i.bak '/lexutil.RegisterType/s/^/\/\//' ./*.go
sed -i.bak '/func init() {/a\
    _ = lexutil.ErrUnrecognizedType
' ./*.go
go run ./cborgen

# Restore the generated sources, including type registration, after CBOR methods exist.
go run github.com/bluesky-social/indigo/cmd/lexgen --build-file lexgen.json --external-lexicons "$strong_ref_schema" schemas
rm -f ./*.bak
