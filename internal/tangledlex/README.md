# Tangled lexicons

This package contains the generated Go representations of the Tangled record
and Bobbin query lexicons that `tg` consumes. It is deliberately internal:
`tangled/` remains the handwritten Bobbin client and turns these generated
types into the CLI's stable application types.

`lock.json` pins the upstream Tangled Core commit used for the checked-in
schemas. `schemas.txt` is the explicit allowlist of upstream lexicons that tg
depends on. Keeping both means an upstream schema addition cannot silently
become part of tg's wire contract.

## Updating

Choose a reviewed, full 40-character Tangled Core commit and run:

```sh
cd internal/tangledlex
./update-schemas.sh <tangled-core-commit>
```

The script fetches only that commit, replaces exactly the paths listed in
`schemas.txt`, updates `lock.json`, and regenerates the Go and CBOR code.
Review the resulting schema and generated-code diff, then run `nix fmt` and
`nix build path:.#tg` from the repository root.

To add support for a new lexicon, add its path (relative to Tangled Core's
`lexicons/`) to `schemas.txt`. If it defines a record type, also add that
type to the CBOR generator's list in `cborgen/main.go`; then run the update
script.
