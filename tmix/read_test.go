package tmix

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/mcmillan/theatremix-parser/internal/fixture"
)

func openFixture(t *testing.T, variant string) *Show {
	t.Helper()
	path := fixture.Create(t, filepath.Join(t.TempDir(), "show.tmix"), variant)
	show, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile(%s): %v", variant, err)
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

func TestReadVariantC(t *testing.T) {
	show := openFixture(t, "C")

	if show.Format.SchemaVariant != "C" || show.Format.MinVersion != "3.1" || show.Format.ProfileSchemaVersion != 2 {
		t.Errorf("format = %+v", show.Format)
	}
	if !reflect.DeepEqual(show.Format.OptionalColumns["profiles"], []string{"label"}) ||
		len(show.Format.OptionalColumns["cues"]) != len(optionalColumns["cues"]) {
		t.Errorf("optionalColumns = %v", show.Format.OptionalColumns)
	}
	if show.Info.Designer != "Test Designer" || show.Info.Console.Model != "X32C" {
		t.Errorf("show info = %+v", show.Info)
	}

	cfg := show.Config
	if !reflect.DeepEqual(cfg.Channels, []int{1, 2, 3, 4, 5, 6, 7, 8, -1, -2}) || len(cfg.DCAs) != 10 {
		t.Errorf("config channels/dcas = %v / %v", cfg.Channels, cfg.DCAs)
	}
	if cfg.FX.Default != -1 || !reflect.DeepEqual(cfg.FX.BusMap, map[int]int{1: 13, 2: 14, 3: 15, 4: 16}) {
		t.Errorf("fx config = %+v", cfg.FX)
	}
	if !reflect.DeepEqual(cfg.ButtonMap, map[int]int{0: 12, 1: 11}) || cfg.Playback.Mode != "qlab" ||
		!cfg.Playback.SuppressBack || cfg.SpareBackup != 8 || cfg.GangLR.Colour != "11" ||
		!cfg.Features.CueZeroResetLevels || cfg.Features.SelectOnSpill {
		t.Errorf("config = %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.CueZero, CueZero{Snippets: []int{}, Scenes: []int{1}, ScenePoints: []int{0}}) {
		t.Errorf("cueZero = %+v", cfg.CueZero)
	}
	if cfg.Raw["designer"] != "Test Designer" || len(cfg.Raw) != len(fixture.Config) {
		t.Errorf("raw config has %d rows", len(cfg.Raw))
	}

	// Channels are synthesised from config.channels + default profiles.
	if len(show.Channels) != 10 {
		t.Fatalf("channels = %+v", show.Channels)
	}
	if c := show.Channels[2]; c.Number != 3 || c.Name != "Ch3" || c.Label != "C3" || c.DefaultProfileID != 3 || c.IsAuxIn {
		t.Errorf("channel 3 = %+v", c)
	}
	if c := show.Channels[8]; c.Number != -1 || c.Name != "Aux1" || !c.IsAuxIn || c.DefaultProfileID != 9 {
		t.Errorf("aux channel = %+v", c)
	}

	if len(show.Positions) != 3 || show.Positions[1].Pan != -30 || show.Positions[1].Buses == nil ||
		!reflect.DeepEqual(show.Positions[2].Buses, []int{1302}) || show.Positions[2].DelayMs != 5 {
		t.Errorf("positions = %+v", show.Positions)
	}
	if len(show.Profiles) != 12 || show.Profiles[10].IsDefault || show.Profiles[10].Data.Params["/eq/1/type"] != "PEQ" ||
		!reflect.DeepEqual(show.Profiles[10].Data.StoreParams, []string{"gain", "hpf", "eq"}) {
		t.Errorf("profiles = %+v", show.Profiles)
	}
	if len(show.Ensembles) != 3 || !reflect.DeepEqual(show.Ensembles[1].ChannelProfiles, map[int]int{4: 12}) ||
		show.Ensembles[0].ChannelProfiles == nil || !reflect.DeepEqual(show.Ensembles[2].Channels, []int{8, 7}) {
		t.Errorf("ensembles = %+v", show.Ensembles)
	}
	if len(show.Actors) != 2 || !show.Actors[0].Active || show.Actors[1].Order != 1 {
		t.Errorf("actors = %+v", show.Actors)
	}
	if len(show.ActorProfiles) != 1 || show.ActorProfiles[0].Data.Params["/gain/trim"] != "2" ||
		!reflect.DeepEqual(show.ActorProfiles[0].Data.ActorParams, []string{"gain"}) {
		t.Errorf("actorProfiles = %+v", show.ActorProfiles)
	}
	if len(show.ActorGroups) != 1 || !reflect.DeepEqual(show.ActorGroups[0].ChannelActors, map[int]int{3: 1, 4: 99}) {
		t.Errorf("actorGroups = %+v", show.ActorGroups)
	}
	if show.Caches.Snippets[0] != "Pit Mute" || show.Caches.FX[1] != "Hall" ||
		!reflect.DeepEqual(show.Caches.Scenes, []SceneCacheEntry{{1, 0, "Init"}, {2, 50, "Half"}}) {
		t.Errorf("caches = %+v", show.Caches)
	}

	// Cue order is numeric on (number, point): 0.1 < 0.10, 1.9 < 1.10.
	if got := cueIDs(show.Cues); !reflect.DeepEqual(got, []string{"0.1", "0.10", "1", "1.9", "1.10", "2", "3"}) {
		t.Errorf("cue order = %v", got)
	}

	c := findCue(t, show, "1")
	if c.Name != "Scene 1" || c.Colour != 5 || c.ColourName == nil || *c.ColourName != "purple" || c.Skip {
		t.Errorf("cue 1 header = %+v", c)
	}
	if len(c.DCAs) != 3 || !reflect.DeepEqual(c.DCAs[2], DcaAssign{DCA: 3, ConsoleDCA: 3, Channels: []int{3, 4, 5}, Label: "Trio"}) {
		t.Errorf("cue 1 dcas = %+v", c.DCAs)
	}
	if !reflect.DeepEqual(c.ChannelPositions, map[int]int{3: 1, 4: 2}) || !reflect.DeepEqual(c.ChannelProfiles, map[int]int{3: 11}) {
		t.Errorf("cue 1 positions/profiles = %v / %v", c.ChannelPositions, c.ChannelProfiles)
	}
	if !reflect.DeepEqual(c.ChannelFX, map[int]FxSpec{1: {Buses: []int{1}}, 3: {Buses: []int{1, 2}}, 5: {None: true}}) {
		t.Errorf("cue 1 fx = %+v", c.ChannelFX)
	}
	if !reflect.DeepEqual(c.ChannelLevels, map[int]int{3: -150, 4: 50}) || c.ChannelLevelsDb[3] != -15 || c.ChannelLevelsDb[4] != 5 {
		t.Errorf("cue 1 levels = %v / %v", c.ChannelLevels, c.ChannelLevelsDb)
	}
	if !reflect.DeepEqual(c.FxUnmuted, []int{3, 1}) || !reflect.DeepEqual(c.Snippets, []int{0}) ||
		!reflect.DeepEqual(c.Scenes, []int{2}) || !reflect.DeepEqual(c.ScenePoints, []int{50}) || c.PlaybackCue != "M01" {
		t.Errorf("cue 1 lists = %+v", c)
	}
	if !reflect.DeepEqual(c.MutedChannels, []int{-2, -1, 6, 7, 8}) {
		t.Errorf("cue 1 muted = %v", c.MutedChannels)
	}

	// Placeholder DCA (label without channels), dca10, NULL colour, skip.
	c = findCue(t, show, "2")
	if !c.Skip || c.Colour != 0 || c.ColourName != nil {
		t.Errorf("cue 2 header = %+v", c)
	}
	if !reflect.DeepEqual(c.DCAs, []DcaAssign{
		{DCA: 9, ConsoleDCA: 9, Channels: []int{}, Label: "Bunsen"},
		{DCA: 10, ConsoleDCA: 10, Channels: []int{-2, -1}, Label: "Aux"},
	}) {
		t.Errorf("cue 2 dcas = %+v", c.DCAs)
	}
	if c.ChannelFX == nil || len(c.ChannelFX) != 0 {
		t.Errorf("cue 2 fx should be empty map, got %#v", c.ChannelFX)
	}

	// NULL and "" are equivalent: neither DCA appears.
	c = findCue(t, show, "3")
	if len(c.DCAs) != 0 || len(c.MutedChannels) != 10 {
		t.Errorf("cue 3 = %+v", c)
	}
}

func TestReadVariantB(t *testing.T) {
	show := openFixture(t, "B")
	if show.Format.SchemaVariant != "B" {
		t.Errorf("variant = %s", show.Format.SchemaVariant)
	}
	for _, col := range show.Format.OptionalColumns["cues"] {
		if col == "skip" {
			t.Error("skip should be absent in variant B")
		}
	}
	// Missing config params fall back to spec defaults.
	if !show.Config.Features.SelectOnSpill || show.Config.DAW.IP != "" || show.Config.DAW.Remote {
		t.Errorf("defaults not applied: %+v %+v", show.Config.Features, show.Config.DAW)
	}
	if _, present := show.Config.Raw["selectOnSpill"]; present {
		t.Error("raw config must not contain defaulted params")
	}
	if c := findCue(t, show, "2"); c.Skip {
		t.Error("skip should read as false when the column is absent")
	}
}

func TestReadVariantA(t *testing.T) {
	show := openFixture(t, "A")
	if show.Format.SchemaVariant != "A" {
		t.Errorf("variant = %s", show.Format.SchemaVariant)
	}
	for _, tbl := range show.Format.Tables {
		if tbl == "actors" || tbl == "fxCache" {
			t.Errorf("table %s should be absent", tbl)
		}
	}
	if show.Actors == nil || len(show.Actors) != 0 || show.ActorProfiles == nil || show.ActorGroups == nil {
		t.Errorf("absent tables must decode to empty slices: %+v", show)
	}
	if show.Caches.Snippets == nil || len(show.Caches.Snippets) != 0 || show.Caches.FX == nil {
		t.Errorf("absent caches must be empty maps: %+v", show.Caches)
	}
	if len(show.Caches.Scenes) != 2 || show.Caches.Scenes[1].Point != 0 {
		t.Errorf("sceneCache without point column: %+v", show.Caches.Scenes)
	}
	if show.Channels[2].Label != "" || show.Profiles[2].Label != "" {
		t.Error("label must be empty when the column is absent")
	}
	c := findCue(t, show, "1")
	if len(c.ScenePoints) != 0 || c.ScenePoints == nil || !reflect.DeepEqual(c.Scenes, []int{2}) {
		t.Errorf("scenePoints absent: %+v", c)
	}
	if _, present := show.Config.Raw["cueZeroScenePoints"]; present || len(show.Config.CueZero.ScenePoints) != 0 {
		t.Errorf("cueZeroScenePoints default: %+v", show.Config.CueZero)
	}
}

// TestJSONNulls guarantees that only channelFX values and colourName may be
// null; every other collection must serialise as [] or {}.
func TestJSONNulls(t *testing.T) {
	show := openFixture(t, "A")
	b, err := json.Marshal(show)
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	var nulls []string
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case nil:
			nulls = append(nulls, path)
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
	for _, p := range nulls {
		if !strings.HasSuffix(p, ".colourName") && !strings.Contains(p, ".channelFX.") {
			t.Errorf("unexpected null at %s", p)
		}
	}
	if !strings.Contains(string(b), `"5":null`) {
		t.Error("explicit no-FX entry should serialise as null")
	}
}
