package tmix

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// kitchenSink is the committed real show file (see testdata/real/README.md).
// Every end-to-end test derives from it: as-is, copied under an awkward path,
// or copied and altered with SQL to reproduce the older schema variants and
// invalid states that the app itself never writes.
const kitchenSink = "../testdata/real/kitchen_sink.tmix"

func openReal(t *testing.T) *Show {
	t.Helper()
	show, err := OpenFile(filepath.FromSlash(kitchenSink))
	if err != nil {
		t.Fatal(err)
	}
	return show
}

// copyReal copies the real fixture to dst and returns dst.
func copyReal(t *testing.T, dst string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.FromSlash(kitchenSink))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dst
}

// mutatedCopy copies the real fixture into a temp dir and runs stmts on it.
func mutatedCopy(t *testing.T, stmts ...string) string {
	t.Helper()
	path := copyReal(t, filepath.Join(t.TempDir(), "show.tmix"))
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", (&url.URL{Scheme: "file", Path: abs}).String())
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

// downgrades strip a current file down to the older schema variants (spec §9).
var downgrades = map[string][]string{
	"B": {
		`ALTER TABLE cues DROP COLUMN skip`,
		`DELETE FROM config WHERE param IN ('cueZeroActorLabels','dawRemote','dawIP','dawPreRoll','selectOnSpill')`,
	},
	"A": {
		`ALTER TABLE cues DROP COLUMN skip`,
		`ALTER TABLE cues DROP COLUMN scenePoints`,
		`ALTER TABLE profiles DROP COLUMN label`,
		`ALTER TABLE sceneCache DROP COLUMN point`,
		`DROP TABLE actors`, `DROP TABLE actorProfiles`, `DROP TABLE actorGroups`,
		`DROP TABLE snippetCache`, `DROP TABLE fxCache`,
		`DELETE FROM config WHERE param IN ('cueZeroActorLabels','dawRemote','dawIP','dawPreRoll','selectOnSpill',
		  'cueZeroScenePoints','dimDCAFadersSuppressColours')`,
	},
}

// openVariant decodes the real fixture downgraded to schema variant "A" or "B".
func openVariant(t *testing.T, variant string) *Show {
	t.Helper()
	stmts, ok := downgrades[variant]
	if !ok {
		t.Fatalf("unknown variant %q", variant)
	}
	show, err := OpenFile(mutatedCopy(t, stmts...))
	if err != nil {
		t.Fatalf("variant %s: %v", variant, err)
	}
	return show
}

func cueIDs(cues []Cue) []string {
	ids := make([]string, len(cues))
	for i, c := range cues {
		ids[i] = c.ID
	}
	return ids
}

func findCue(t *testing.T, show *Show, id string) *Cue {
	t.Helper()
	for i := range show.Cues {
		if show.Cues[i].ID == id {
			return &show.Cues[i]
		}
	}
	t.Fatalf("cue %s not found in %v", id, cueIDs(show.Cues))
	return nil
}
