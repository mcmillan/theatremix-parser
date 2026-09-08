# Real show fixtures

Drop one or more real TheatreMix `.tmix` show files in this directory. The test
suites in `tmix/` and `cmd/theatremix-parser/` discover them automatically
(`go test ./...`) and, for each file:

- parse it via `OpenFile` and via `OpenBytes` (the stdin path) and require
  byte-identical JSON from both;
- require every controlled channel to have a default profile and the cue list
  to be non-empty;
- run the format invariants (`tmix.Validate`) and fail on any violation — a
  violation on an app-written file means the spec has a gap worth recording;
- run the CLI with `-validate` and require exit 0 and valid JSON.

When the directory contains no `.tmix` files these tests are skipped.

`*.tmix` is git-ignored here because show files contain personal data
(designer, venue, console IP/MAC). Use `git add -f` to commit one on purpose.
