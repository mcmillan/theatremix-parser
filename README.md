# theatremix-parser

[![CI](https://github.com/mcmillan/theatremix-parser/actions/workflows/ci.yml/badge.svg)](https://github.com/mcmillan/theatremix-parser/actions/workflows/ci.yml)

Reads a [TheatreMix](https://theatremix.com/) `.tmix` show file and writes a JSON
representation of it. The show file format was reverse-engineered; the
specification lives in [`docs/`](docs/):

- [`docs/TMIX_FORMAT_SPEC.md`](docs/TMIX_FORMAT_SPEC.md) — full specification with evidence
- [`docs/TMIX_FORMAT_SPEC.agent.md`](docs/TMIX_FORMAT_SPEC.agent.md) — condensed reference

## Provenance — this project was LLM-generated

Everything in this repository — the format specification, the Go library and
CLI, the tests and this README — was written by an LLM ([Claude](https://claude.com),
via Claude Code) working under human direction; the commit trailers name the
model. The human's contributions were the goal, the design decisions
(module layout, output shape, driver choice), review of the results, and the
real show file used as the test fixture.

Bear this in mind when relying on it:

- **The format specification is unofficial.** It was derived from a corpus of
  real show files, strings embedded in the TheatreMix application, and the
  app's bundled help — not from documentation or source provided by the
  vendor. Each statement carries a confidence marker (`[C]` confirmed, `[I]`
  inferred, `[?]` unknown); treat `[I]` and `[?]` items with appropriate
  caution and prefer keeping their raw values.
- **The parser is read-only** and does nothing to your show files, but its
  interpretation of some fields (console bus ids, button maps, colour order)
  rests on inference. Please report discrepancies.
- TheatreMix is a product of Mixing Technology Pty Ltd. This project is not
  affiliated with, endorsed by, or supported by them.

## Install

Prebuilt binaries for macOS, Linux and Windows (amd64 and arm64) are attached
to each [release](https://github.com/mcmillan/theatremix-parser/releases),
with a `SHA256SUMS` file. The macOS binaries are not notarised, so Gatekeeper
may require `xattr -d com.apple.quarantine theatremix-parser` on first run.

Or build from source:

```sh
go install github.com/mcmillan/theatremix-parser/cmd/theatremix-parser@latest
```

Pure Go (no cgo) — cross-compiles anywhere Go does.

## Usage

```
theatremix-parser [flags] [FILE]

  FILE        a .tmix show file; omit or use "-" to read it from stdin
  -compact    single-line JSON (default: indented)
  -validate   check the show against the format invariants; report violations
              on stderr and exit 2 if any are found
  -version    print the version and exit
```

```sh
theatremix-parser show.tmix | jq '.cues[] | {id, name, dcas}'
cat show.tmix | theatremix-parser -compact > show.json
```

Exit codes: `0` success · `1` error (nothing written to stdout) · `2` invariant
violations found with `-validate` (JSON is still written).

The file is opened read-only, so it is safe to run against a show that is
currently open in TheatreMix.

## Output

One JSON object with these top-level keys:

| key | content |
|---|---|
| `format` | schema variant, `minVersion`, `profileSchemaVersion`, tables and optional columns present |
| `show` | designer, venue, target console, last-connected console |
| `config` | typed settings (channels, DCAs, FX, feature switches, …) plus `raw` with every config row verbatim |
| `channels` | controlled channels with their names (`isAuxIn` for negative numbers) |
| `positions`, `profiles`, `ensembles`, `actors`, `actorProfiles`, `actorGroups`, `caches` | the remaining tables, decoded |
| `cues` | every stored cue, sorted; DCA assignments, per-channel positions / profiles / FX / level offsets (raw tenths and dB), snippets, scenes, playback cue, derived `id` (e.g. `5.20`), `colourName` and `mutedChannels` |

Values the specification marks as not yet understood (console bus ids, button
maps, `backupChannels`) are passed through unchanged.

## Library

```go
import "github.com/mcmillan/theatremix-parser/tmix"

show, err := tmix.OpenFile("show.tmix")   // or tmix.OpenBytes(data)
violations := tmix.Validate(show)
```

## Development

```sh
make            # build bin/theatremix-parser
make check      # gofmt + go vet + go test — what CI runs
make golden     # regenerate the CLI golden JSON after a reviewed output change
make dist-all   # package release archives for every OS/arch into dist/
make help       # list every target
```

Tests run against the real show file in [`testdata/real/`](testdata/real/);
see its README for what it covers and how to add more files.

CI (`.github/workflows/ci.yml`) runs the suite on Ubuntu, macOS and Windows,
cross-compiles all six release targets on every push, and on a `v*` tag
publishes them to a GitHub Release:

```sh
git tag v0.1.0 && git push origin v0.1.0
```

## License

[MIT](LICENSE).
