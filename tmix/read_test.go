package tmix

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// TestVariantB reads the real fixture downgraded to schema variant B: no
// cues.skip column and several config params absent.
func TestVariantB(t *testing.T) {
	show := openVariant(t, "B")
	if show.Format.SchemaVariant != "B" {
		t.Errorf("variant = %s", show.Format.SchemaVariant)
	}
	for _, col := range show.Format.OptionalColumns["cues"] {
		if col == "skip" {
			t.Error("skip should be reported absent")
		}
	}
	// Deleted params fall back to spec defaults (the real file has dawRemote=1
	// and dawIP=127.0.0.1) and do not appear in the raw map.
	if show.Config.DAW.Remote || show.Config.DAW.IP != "" || !show.Config.Features.SelectOnSpill {
		t.Errorf("defaults not applied: %+v %+v", show.Config.DAW, show.Config.Features)
	}
	if _, present := show.Config.Raw["dawRemote"]; present {
		t.Error("raw config must not contain defaulted params")
	}
	if c := findCue(t, show, "1"); c.Skip {
		t.Error("skip should read as false when the column is absent")
	}
	if v := Validate(show); len(v) != 0 {
		t.Errorf("violations: %v", v)
	}
}

// TestVariantA reads the real fixture downgraded to the oldest schema: no
// scenePoints/skip/label/point columns and no actor or cache tables.
func TestVariantA(t *testing.T) {
	show := openVariant(t, "A")
	if show.Format.SchemaVariant != "A" {
		t.Errorf("variant = %s", show.Format.SchemaVariant)
	}
	for _, tbl := range show.Format.Tables {
		if tbl == "actors" || tbl == "fxCache" || tbl == "snippetCache" {
			t.Errorf("table %s should be absent", tbl)
		}
	}
	if !reflect.DeepEqual(show.Format.OptionalColumns["profiles"], []string{}) ||
		!reflect.DeepEqual(show.Format.OptionalColumns["sceneCache"], []string{}) {
		t.Errorf("optionalColumns = %v", show.Format.OptionalColumns)
	}
	if show.Actors == nil || len(show.Actors) != 0 || show.ActorProfiles == nil || show.ActorGroups == nil {
		t.Errorf("absent tables must decode to empty slices: %+v", show)
	}
	if show.Caches.Snippets == nil || len(show.Caches.Snippets) != 0 || show.Caches.FX == nil || show.Caches.Scenes == nil {
		t.Errorf("absent caches must be empty collections: %+v", show.Caches)
	}
	for _, p := range show.Profiles {
		if p.Label != "" {
			t.Errorf("profile %d label = %q with the column absent", p.ID, p.Label)
		}
	}
	c := findCue(t, show, "1")
	if c.ScenePoints == nil || len(c.ScenePoints) != 0 || c.Skip {
		t.Errorf("absent columns: %+v", c)
	}
	if _, present := show.Config.Raw["cueZeroScenePoints"]; present || len(show.Config.CueZero.ScenePoints) != 0 {
		t.Errorf("cueZeroScenePoints default: %+v", show.Config.CueZero)
	}
	if v := Validate(show); len(v) != 0 {
		t.Errorf("violations: %v", v)
	}
}

// TestJSONNulls guarantees that only channelFX values and colourName may be
// null; every other collection must serialise as [] or {}.
func TestJSONNulls(t *testing.T) {
	for _, variant := range []string{"A", "B", "C"} {
		var show *Show
		if variant == "C" {
			show = openReal(t)
		} else {
			show = openVariant(t, variant)
		}
		b, err := json.Marshal(show)
		if err != nil {
			t.Fatal(err)
		}
		var doc any
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatal(err)
		}
		var walk func(path string, v any)
		walk = func(path string, v any) {
			switch x := v.(type) {
			case nil:
				if !strings.HasSuffix(path, ".colourName") && !strings.Contains(path, ".channelFX.") {
					t.Errorf("variant %s: unexpected null at %s", variant, path)
				}
			case map[string]any:
				for k, vv := range x {
					walk(path+"."+k, vv)
				}
			case []any:
				for i, vv := range x {
					walk(path+"["+strconv.Itoa(i)+"]", vv)
				}
			}
		}
		walk("$", doc)
	}
}
