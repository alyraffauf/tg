#!/usr/bin/env sh
set -eu

go run github.com/bluesky-social/indigo/cmd/lexgen --build-file lexgen.json schemas

# cbor-gen needs to compile the generated package before it can emit its CBOR
# methods. Temporarily suppress the registry calls, which require those methods.
sed -i.bak '/lexutil.RegisterType/s/^/\/\//' ./*.go
sed -i.bak '/func init() {/a\
    _ = lexutil.ErrUnrecognizedType
' ./*.go
go run ./cborgen

# Restore the generated sources, including type registration, after CBOR methods exist.
go run github.com/bluesky-social/indigo/cmd/lexgen --build-file lexgen.json schemas
rm -f ./*.bak
