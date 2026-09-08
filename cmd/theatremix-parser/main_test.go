package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mcmillan/theatremix-parser/internal/fixture"
)

var update = flag.Bool("update", false, "rewrite golden files")

func runCLI(t *testing.T, stdin []byte, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, bytes.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestGolden(t *testing.T) {
	for _, variant := range []string{"A", "B", "C"} {
		t.Run(variant, func(t *testing.T) {
			path := fixture.Create(t, filepath.Join(t.TempDir(), "show.tmix"), variant)
			code, out, errs := runCLI(t, nil, path)
			if code != 0 || errs != "" {
				t.Fatalf("exit %d, stderr %q", code, errs)
			}
			golden := filepath.Join("testdata", "variant"+variant+".golden.json")
			if *update {
				if err := os.WriteFile(golden, []byte(out), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run with -update to create)", err)
			}
			if out != string(want) {
				t.Errorf("output differs from %s; run go test ./cmd/... -update after reviewing", golden)
			}
		})
	}
}

func TestStdinMatchesPath(t *testing.T) {
	path := fixture.Create(t, filepath.Join(t.TempDir(), "show.tmix"), "C")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, fromPath, _ := runCLI(t, nil, path)
	codeDash, fromDash, _ := runCLI(t, data, "-")
	codeNoArg, fromNoArg, _ := runCLI(t, data)
	if codeDash != 0 || codeNoArg != 0 || fromPath != fromDash || fromPath != fromNoArg {
		t.Error("stdin output must be byte-identical to path output")
	}
}

func TestCompact(t *testing.T) {
	path := fixture.Create(t, filepath.Join(t.TempDir(), "show.tmix"), "C")
	code, out, _ := runCLI(t, nil, "-compact", path)
	if code != 0 || strings.Count(out, "\n") != 1 || !json.Valid([]byte(out)) {
		t.Errorf("compact output invalid: code %d, %d lines", code, strings.Count(out, "\n"))
	}
}

func TestValidateFlag(t *testing.T) {
	dir := t.TempDir()
	clean := fixture.Create(t, filepath.Join(dir, "clean.tmix"), "C")
	if code, out, errs := runCLI(t, nil, "-validate", clean); code != 0 || errs != "" || out == "" {
		t.Errorf("clean file: code %d stderr %q", code, errs)
	}
	broken := fixture.Create(t, filepath.Join(dir, "broken.tmix"), "C")
	fixture.Mutate(t, broken, `UPDATE cues SET dca02Channels = '1' WHERE number = 1 AND point = 0`)
	code, out, errs := runCLI(t, nil, "-validate", broken)
	if code != exitViolated || !strings.Contains(errs, "violation: I2") || !json.Valid([]byte(out)) {
		t.Errorf("broken file: code %d stderr %q", code, errs)
	}
}

func TestErrors(t *testing.T) {
	dir := t.TempDir()
	notes := filepath.Join(dir, "notes.txt")
	os.WriteFile(notes, []byte("not a database"), 0o644)
	bare := fixture.CreateBare(t, filepath.Join(dir, "bare.sqlite"), `CREATE TABLE config (param TEXT, value TEXT)`)
	cases := []struct {
		name  string
		stdin []byte
		args  []string
	}{
		{"missing path", nil, []string{filepath.Join(dir, "nope.tmix")}},
		{"text file", nil, []string{notes}},
		{"sqlite without tmix tables", nil, []string{bare}},
		{"empty stdin", nil, nil},
		{"garbage stdin", []byte("garbage garbage garbage garbage"), []string{"-"}},
		{"too many args", nil, []string{notes, notes}},
		{"unknown flag", nil, []string{"-bogus"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, errs := runCLI(t, c.stdin, c.args...)
			if code != exitError || out != "" || errs == "" {
				t.Errorf("code %d stdout %q stderr %q", code, out, errs)
			}
		})
	}
}

func TestVersionAndHelp(t *testing.T) {
	if code, out, _ := runCLI(t, nil, "-version"); code != 0 || !strings.HasPrefix(out, "theatremix-parser ") {
		t.Errorf("-version: code %d out %q", code, out)
	}
	if code, _, errs := runCLI(t, nil, "-h"); code != 0 || !strings.Contains(errs, "usage:") {
		t.Errorf("-h: code %d stderr %q", code, errs)
	}
}

// TestRealFixtures runs the CLI over any real show files in testdata/real at
// the repository root; it is skipped when there are none.
func TestRealFixtures(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("..", "..", "testdata", "real", "*.tmix"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Skip("no real show files in testdata/real")
	}
	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			code, out, errs := runCLI(t, nil, "-validate", path)
			if code != 0 || errs != "" || !json.Valid([]byte(out)) {
				t.Errorf("code %d stderr %q valid=%v", code, errs, json.Valid([]byte(out)))
			}
		})
	}
}
