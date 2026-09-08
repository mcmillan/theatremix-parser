# Real show fixtures

## `kitchen_sink.tmix` (committed)

A show file saved by TheatreMix itself with every feature touched and no
personal data (designer `designer`, venue `venue`, no console ever connected).
It is the sole fixture: every end-to-end test uses it directly, copied under an
awkward path, or copied and altered with SQL to reproduce the older schema
variants (columns and tables dropped) and invalid states the app never writes.
It is pinned by:

- `tmix/kitchen_sink_test.go` — field-level assertions;
- `cmd/theatremix-parser/testdata/kitchen_sink.golden.json` — the exact CLI
  output (`go test ./cmd/... -update` regenerates it after a reviewed change).

If you regenerate it in TheatreMix, keep the existing content and add to it;
the assertions above describe what the tests rely on.

## Adding more

Drop further `.tmix` files in this directory. The test suites in `tmix/` and
`cmd/theatremix-parser/` discover every file here automatically
(`go test ./...`) and, for each one:

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
