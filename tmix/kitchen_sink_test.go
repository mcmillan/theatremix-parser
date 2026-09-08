package tmix

import (
	"path/filepath"
	"reflect"
	"testing"
)

// kitchenSink is the committed real show file (see testdata/real/README.md).
const kitchenSink = "../testdata/real/kitchen_sink.tmix"

// TestKitchenSink pins the decode of the real fixture, which exercises
// settings no synthetic fixture does: a Yamaha DM7 target, non-contiguous
// controlled DCAs, populated backupChannels and actors, an explicit channelFX
// entry equal to defaultFX, and a merge of the implicit "All" ensemble.
func TestKitchenSink(t *testing.T) {
	show, err := OpenFile(filepath.FromSlash(kitchenSink))
	if err != nil {
		t.Fatal(err)
	}
	if v := Validate(show); len(v) != 0 {
		t.Errorf("violations: %v", v)
	}
	if show.Format.SchemaVariant != "C" || show.Format.MinVersion != "3.0" || show.Format.ProfileSchemaVersion != 2 {
		t.Errorf("format = %+v", show.Format)
	}
	if show.Info.TargetConsole != "DM7" || show.Info.Designer != "designer" || show.Info.Venue != "venue" ||
		show.Info.Console != (ConsoleInfo{}) {
		t.Errorf("info = %+v", show.Info)
	}

	cfg := show.Config
	if !reflect.DeepEqual(cfg.Channels, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}) {
		t.Errorf("channels = %v", cfg.Channels)
	}
	if !reflect.DeepEqual(cfg.DCAs, []int{1, 2, 3, 4, 5, 6, 7, 8, 11}) {
		t.Errorf("dcas = %v (non-contiguous list expected)", cfg.DCAs)
	}
	if cfg.BackupChannels != "44=1,46=8" || cfg.SpareBackup != 60 || cfg.LabelTargetBus != 1421 || cfg.LabelLR != 1 {
		t.Errorf("backup/label config = %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.FX, FXConfig{Assigns: []int{2}, Mutes: []int{3}, Default: 2,
		BusMap: map[int]int{1: 37, 2: 38, 3: 39, 4: 40}}) {
		t.Errorf("fx = %+v", cfg.FX)
	}
	if cfg.Playback != (Playback{Mode: "qlab", Raw: 1, Passcode: "6969", SuppressBack: true}) {
		t.Errorf("playback = %+v", cfg.Playback)
	}
	if cfg.DAW != (DAW{Remote: true, IP: "127.0.0.1"}) {
		t.Errorf("daw = %+v", cfg.DAW)
	}
	if f := cfg.Features; !f.ActiveChannelHighlight || !f.DimDCAFaders || !f.CueZeroResetLevels ||
		!f.SceneRecall || !f.ChannelLevels || f.SnippetRecall || f.AutoConnect || f.QLCLDyn1 {
		t.Errorf("features = %+v", f)
	}
	if !reflect.DeepEqual(cfg.MuteButtonMap, map[int]int{0: 8, 1: 7}) ||
		!reflect.DeepEqual(cfg.MuteButtonAssignKeys, map[int]int{8: 19, 7: 18}) || len(cfg.ButtonMap) != 0 {
		t.Errorf("button maps = %v %v %v", cfg.ButtonMap, cfg.MuteButtonMap, cfg.MuteButtonAssignKeys)
	}

	if len(show.Channels) != 16 || show.Channels[9].Name != "Channel 10" || show.Channels[9].DefaultProfileID != 10 {
		t.Errorf("channels = %+v", show.Channels)
	}
	if len(show.Positions) != 2 || !reflect.DeepEqual(show.Positions[1],
		Position{ID: 1, Name: "Outer Space", ShortName: "OS", DelayMs: 69, Pan: 33, Buses: []int{5}}) {
		t.Errorf("positions = %+v", show.Positions)
	}
	if len(show.Ensembles) != 3 || show.Ensembles[2].Name != "ensemble" ||
		!reflect.DeepEqual(show.Ensembles[2].Channels, []int{2, 5, 8}) {
		t.Errorf("ensembles = %+v", show.Ensembles)
	}
	wantActors := []Actor{
		{ID: 1, Channel: 10, Name: "Dave", Order: 0, Active: true},
		{ID: 2, Channel: 10, Name: "Brenda", Order: 1, Active: false},
	}
	if !reflect.DeepEqual(show.Actors, wantActors) {
		t.Errorf("actors = %+v", show.Actors)
	}
	if len(show.ActorProfiles)+len(show.ActorGroups)+len(show.Caches.Snippets)+len(show.Caches.FX)+len(show.Caches.Scenes) != 0 {
		t.Errorf("expected empty actorProfiles/actorGroups/caches")
	}

	if got := cueIDs(show.Cues); !reflect.DeepEqual(got, []string{"1", "2"}) {
		t.Fatalf("cues = %v", got)
	}
	c := findCue(t, show, "1")
	wantDCAs := []DcaAssign{
		{DCA: 1, ConsoleDCA: 1, Channels: []int{11}},
		{DCA: 3, ConsoleDCA: 3, Channels: []int{7, 14}, Label: "Banana"},
		{DCA: 5, ConsoleDCA: 5, Channels: []int{4}},
	}
	if c.Name != "foo" || !reflect.DeepEqual(c.DCAs, wantDCAs) {
		t.Errorf("cue 1 = %+v", c)
	}
	if !reflect.DeepEqual(c.ChannelPositions, map[int]int{4: 1}) {
		t.Errorf("cue 1 positions = %v", c.ChannelPositions)
	}
	// channelFX holds explicit settings, stored even when equal to defaultFX.
	if !reflect.DeepEqual(c.ChannelFX, map[int]FxSpec{4: {Buses: []int{2}}}) {
		t.Errorf("cue 1 fx = %+v", c.ChannelFX)
	}
	if !reflect.DeepEqual(c.MutedChannels, []int{1, 2, 3, 5, 6, 8, 9, 10, 12, 13, 15, 16}) {
		t.Errorf("cue 1 muted = %v", c.MutedChannels)
	}

	// Typing the implicit "All" ensemble into a DCA stores the label and the
	// expanded channel list minus channels already assigned to other DCAs.
	c = findCue(t, show, "2")
	if c.Name != "" || len(c.DCAs) != 3 || c.DCAs[0].Channels[0] != 14 || c.DCAs[1].Channels[0] != 1 ||
		c.DCAs[2].DCA != 8 || c.DCAs[2].Label != "All" ||
		!reflect.DeepEqual(c.DCAs[2].Channels, []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 16}) {
		t.Errorf("cue 2 = %+v", c)
	}
	if len(c.MutedChannels) != 0 {
		t.Errorf("cue 2 muted = %v, want none", c.MutedChannels)
	}
}
