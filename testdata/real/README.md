# Real show fixtures

## `kitchen_sink.tmix` (committed)

A show file saved by TheatreMix itself with every feature touched and no
personal data (designer `designer`, venue `venue`, no console ever connected).
It is the reference fixture for behaviour the synthetic fixtures cannot
reproduce faithfully and is pinned by:

- `tmix/kitchen_sink_test.go` — field-level assertions;
- `cmd/theatremix-parser/testdata/kitchen_sink.golden.json` — the exact CLI
  output (`go test ./cmd/... -update` regenerates it after a reviewed change).

## Adding more

Drop further `.tmix` files in this directory. The test suites in `tmix/` and
`cmd/theatremix-parser/` discover them automatically (`go test ./...`) and, for
each file:

- parse it via `OpenFile` and via `OpenBytes` (the stdin path) and require
  byte-identical JSON from both;
- require every controlled channel to have a default profile and the cue list
  to be non-empty;
- run the format invariants (`tmix.Validate`) and fail on any violation — a
  violation on an app-written file means the spec has a gap worth recording;
- run the CLI with `-validate` and require exit 0 and valid JSON.

Other `*.tmix` files here are git-ignored because show files usually contain
personal data (designer, venue, console IP/MAC). Use `git add -f` or add an
exception to `.gitignore` to commit one on purpose.
