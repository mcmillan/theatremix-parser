package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// kitchenSink is the committed real show file (see testdata/real/README.md).
const kitchenSink = "../../testdata/real/kitchen_sink.tmix"

var update = flag.Bool("update", false, "rewrite golden files")

func realPath() string { return filepath.FromSlash(kitchenSink) }

func runCLI(t *testing.T, stdin []byte, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, bytes.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

// brokenCopy copies the real fixture and runs stmts on it. The sqlite driver
// is registered by the tmix package import in main.go.
func brokenCopy(t *testing.T, stmts ...string) string {
	t.Helper()
	data, err := os.ReadFile(realPath())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "broken.tmix")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: path}).String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	return path
}

func TestGolden(t *testing.T) {
	code, out, errs := runCLI(t, nil, "-validate", realPath())
	if code != 0 || errs != "" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	golden := filepath.Join("testdata", "kitchen_sink.golden.json")
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
}

func TestStdinMatchesPath(t *testing.T) {
	data, err := os.ReadFile(realPath())
	if err != nil {
		t.Fatal(err)
	}
	_, fromPath, _ := runCLI(t, nil, realPath())
	codeDash, fromDash, _ := runCLI(t, data, "-")
	codeNoArg, fromNoArg, _ := runCLI(t, data)
	if codeDash != 0 || codeNoArg != 0 || fromPath != fromDash || fromPath != fromNoArg {
		t.Error("stdin output must be byte-identical to path output")
	}
}

func TestCompact(t *testing.T) {
	code, out, _ := runCLI(t, nil, "-compact", realPath())
	if code != 0 || strings.Count(out, "\n") != 1 || !json.Valid([]byte(out)) {
		t.Errorf("compact output invalid: code %d, %d lines", code, strings.Count(out, "\n"))
	}
}

func TestValidateFlag(t *testing.T) {
	if code, out, errs := runCLI(t, nil, "-validate", realPath()); code != 0 || errs != "" || out == "" {
		t.Errorf("clean file: code %d stderr %q", code, errs)
	}
	// Channel 11 is already on DCA 1 in cue 1; putting it on DCA 2 too breaks I2.
	broken := brokenCopy(t, `UPDATE cues SET dca02Channels = '11' WHERE number = 1 AND point = 0`)
	code, out, errs := runCLI(t, nil, "-validate", broken)
	if code != exitViolated || !strings.Contains(errs, "violation: I2") || !json.Valid([]byte(out)) {
		t.Errorf("broken file: code %d stderr %q", code, errs)
	}
}

func TestErrors(t *testing.T) {
	dir := t.TempDir()
	notes := filepath.Join(dir, "notes.txt")
	os.WriteFile(notes, []byte("not a database"), 0o644)
	cases := []struct {
		name  string
		stdin []byte
		args  []string
	}{
		{"missing path", nil, []string{filepath.Join(dir, "nope.tmix")}},
		{"text file", nil, []string{notes}},
		{"sqlite without tmix tables", nil, []string{brokenCopy(t, `DROP TABLE cues`)}},
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

// TestRealFixtures runs the CLI over every show file in testdata/real,
// including any added later.
func TestRealFixtures(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("..", "..", "testdata", "real", "*.tmix"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("no real show files found: %v", err)
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
