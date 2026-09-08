package tmix

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// realFixtures returns the real show files placed in testdata/real at the
// repository root (see testdata/real/README.md), skipping when there are none.
func realFixtures(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join("..", "testdata", "real", "*.tmix"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Skip("no real show files in testdata/real")
	}
	return matches
}

func TestRealFixtures(t *testing.T) {
	for _, path := range realFixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			show, err := OpenFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(show.Cues) == 0 || len(show.Channels) == 0 {
				t.Errorf("%d cues, %d channels", len(show.Cues), len(show.Channels))
			}
			for _, ch := range show.Channels {
				if ch.DefaultProfileID == 0 {
					t.Errorf("channel %d has no default profile", ch.Number)
				}
			}
			for _, v := range Validate(show) {
				t.Errorf("violation: %v", v)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			fromBytes, err := OpenBytes(data)
			if err != nil {
				t.Fatal(err)
			}
			a, _ := json.Marshal(show)
			b, _ := json.Marshal(fromBytes)
			if !bytes.Equal(a, b) {
				t.Error("OpenFile and OpenBytes differ")
			}
		})
	}
}
