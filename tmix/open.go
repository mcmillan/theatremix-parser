package tmix

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// sqliteMagic is the 16-byte header every SQLite 3 database starts with.
const sqliteMagic = "SQLite format 3\x00"

// ErrNotSQLite reports input that is not a SQLite 3 database.
var ErrNotSQLite = errors.New("not a SQLite database")

// OpenFile reads and decodes the .tmix file at path. The database is opened
// read-only and immutable, so a file currently open in TheatreMix is safe to
// read.
func OpenFile(path string) (*Show, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	header := make([]byte, len(sqliteMagic))
	_, err = io.ReadFull(f, header)
	f.Close()
	if err != nil || string(header) != sqliteMagic {
		return nil, fmt.Errorf("%s: %w", path, ErrNotSQLite)
	}
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return Read(db)
}

// OpenBytes decodes a .tmix file held in memory (e.g. read from stdin). The
// bytes are staged in a temporary file for the SQLite driver and removed
// afterwards.
func OpenBytes(data []byte) (*Show, error) {
	if len(data) == 0 {
		return nil, errors.New("empty input")
	}
	if !bytes.HasPrefix(data, []byte(sqliteMagic)) {
		return nil, ErrNotSQLite
	}
	tmp, err := os.CreateTemp("", "tmix-*.sqlite")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	db, err := openDB(tmp.Name())
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return Read(db)
}

// openDB opens path read-only via a percent-encoded file: URI so that spaces
// and reserved characters in the path are safe.
func openDB(path string) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dsn := fileURI(abs) + "?mode=ro&immutable=1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return db, nil
}

// fileURI renders an absolute path as a file: URI. Windows drive paths become
// file:///C:/dir/file, the form SQLite expects; a bare C:\dir would otherwise be
// parsed as a URI authority.
func fileURI(abs string) string {
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}
