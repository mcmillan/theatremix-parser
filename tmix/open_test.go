package tmix

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenBytesMatchesOpenFile(t *testing.T) {
	// A directory name with a space and '#' exercises the file: URI encoding.
	dir := filepath.Join(t.TempDir(), "with space#hash")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := copyReal(t, filepath.Join(dir, "a show.tmix"))
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
	if _, err := OpenFile(mutatedCopy(t, `DROP TABLE cues`)); !errors.Is(err, ErrNotTmix) {
		t.Errorf("sqlite without cues: got %v, want ErrNotTmix", err)
	}
}

func TestOpenDBIsReadOnly(t *testing.T) {
	db, err := openDB(copyReal(t, filepath.Join(t.TempDir(), "ro.tmix")))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO config (param, value) VALUES ('x', 'y')`); err == nil {
		t.Error("write through read-only connection should fail")
	}
}

func TestFileURI(t *testing.T) {
	cases := map[string]string{
		"/tmp/a show#1.tmix":       "file:///tmp/a%20show%231.tmix",
		"/plain/path.tmix":         "file:///plain/path.tmix",
		"/q?uery/and%percent.tmix": "file:///q%3Fuery/and%25percent.tmix",
	}
	if runtime.GOOS == "windows" {
		// filepath.ToSlash only rewrites the native separator, so the drive
		// path form can only be exercised on Windows.
		cases = map[string]string{
			`C:\Users\me\show.tmix`: "file:///C:/Users/me/show.tmix",
			`D:\with space\s.tmix`:  "file:///D:/with%20space/s.tmix",
		}
	}
	for in, want := range cases {
		if got := fileURI(in); got != want {
			t.Errorf("fileURI(%q) = %q, want %q", in, got, want)
		}
	}
}
