#!/usr/bin/env sh
set -eu
indigo_lexicons=$(go list -m -f '{{.Dir}}' github.com/bluesky-social/indigo)/lexicons
lexgen() {
  go run github.com/bluesky-social/indigo/cmd/lexgen --build-file lexgen.json \
    --external-lexicons "$indigo_lexicons/com/atproto/repo/strongRef.json" \
    --external-lexicons "$indigo_lexicons/com/atproto/label/defs.json" schemas
}
lexgen
# Bootstrapping allows union codecs to compile before their map codecs exist.
sed -i.bak '/lexutil.RegisterType/s/^/\/\//' ./*.go
sed -i.bak '/func init() {/a\
    _ = lexutil.ErrUnrecognizedType
' ./*.go
go run ./cborgen/bootstrap.go
go run ./cborgen
lexgen
rm -f ./*.bak
