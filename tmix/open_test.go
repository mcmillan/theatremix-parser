package tmix

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mcmillan/theatremix-parser/internal/fixture"
)

func TestOpenBytesMatchesOpenFile(t *testing.T) {
	// A directory name with a space and '#' exercises the file: URI encoding.
	dir := filepath.Join(t.TempDir(), "with space#hash")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := fixture.Create(t, filepath.Join(dir, "a show.tmix"), "C")
	fromFile, err := OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fromBytes, err := OpenBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(fromFile)
	b, _ := json.Marshal(fromBytes)
	if !bytes.Equal(a, b) {
		t.Error("OpenFile and OpenBytes produced different documents")
	}
}

func TestOpenErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := OpenBytes(nil); err == nil {
		t.Error("empty input should fail")
	}
	if _, err := OpenBytes([]byte("this is not sqlite at all, but long enough")); !errors.Is(err, ErrNotSQLite) {
		t.Errorf("garbage bytes: got %v, want ErrNotSQLite", err)
	}
	txt := filepath.Join(dir, "notes.txt")
	os.WriteFile(txt, []byte("hello"), 0o644)
	if _, err := OpenFile(txt); !errors.Is(err, ErrNotSQLite) {
		t.Errorf("text file: got %v, want ErrNotSQLite", err)
	}
	if _, err := OpenFile(filepath.Join(dir, "missing.tmix")); err == nil {
		t.Error("missing file should fail")
	}
	bare := fixture.CreateBare(t, filepath.Join(dir, "bare.sqlite"),
		`CREATE TABLE config (param TEXT, value TEXT)`, `CREATE TABLE profiles (id INTEGER)`)
	if _, err := OpenFile(bare); !errors.Is(err, ErrNotTmix) {
		t.Errorf("sqlite without cues: got %v, want ErrNotTmix", err)
	}
}

func TestOpenDBIsReadOnly(t *testing.T) {
	path := fixture.Create(t, filepath.Join(t.TempDir(), "ro.tmix"), "C")
	db, err := openDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO config (param, value) VALUES ('x', 'y')`); err == nil {
		t.Error("write through read-only connection should fail")
	}
}
