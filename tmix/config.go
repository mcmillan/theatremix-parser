package tmix

import (
	"database/sql"
	"fmt"
	"strconv"
)

// configDefaults are the values TheatreMix 3.5.0 uses for a param that is
// absent from the file (spec §3.1). Console-specific params default to the
// neutral value noted in the spec.
var configDefaults = map[string]string{
	"designer": "", "venue": "", "minVersion": "", "profileSchemaVersion": "2", "targetConsole": "",
	"consoleModel": "", "consoleVersion": "", "consoleIP": "", "consoleMAC": "", "autoConnect": "0",
	"channels": "", "dcas": "", "backupChannels": "", "spareBackup": "0",
	"fxAssigns": "", "fxMutes": "", "defaultFX": "-1", "fxBusMap": "",
	"channelLevels": "0", "cueZeroResetLevels": "0", "cueZeroActorLabels": "0",
	"snippetRecall": "0", "sceneRecall": "0",
	"cueZeroSnippets": "", "cueZeroScenes": "", "cueZeroScenePoints": "",
	"qLabCues": "0", "qLabPasscode": "", "qLabSuppressBack": "0",
	"dawRemote": "0", "dawIP": "", "dawPreRoll": "0",
	"gangLR": "0", "gangLRChannels": "", "gangLRName": "", "gangLRColour": "",
	"labelLR": "0", "labelTargetBus": "1000",
	"consoleMuteDCAUnassign": "1", "suppressDCAMuteBackupSwitch": "0", "selectOnSpill": "1",
	"enableChannelMonitoring": "1", "activeChannelHighlight": "0",
	"dimDCAFaders": "0", "dimDCAFadersSuppressColours": "0", "qlclDyn1": "0",
	"buttonMap": "", "muteButtonMap": "", "muteButtonAssignKeys": "",
}

var playbackModes = map[int]string{0: "off", 1: "qlab", 2: "scs", 3: "cueplayer"}

// readRawConfig returns the config table verbatim.
func readRawConfig(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT "param", "value" FROM "config"`)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	defer rows.Close()
	raw := map[string]string{}
	for rows.Next() {
		var param, value sql.NullString
		if err := rows.Scan(&param, &value); err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
		raw[text(param)] = text(value)
	}
	return raw, rows.Err()
}

// cfgReader reads typed values out of the raw config map, applying defaults
// and remembering the first conversion error.
type cfgReader struct {
	raw map[string]string
	err error
}

func (r *cfgReader) fail(param string, err error) {
	if r.err == nil {
		r.err = fmt.Errorf("config %s: %w", param, err)
	}
}

func (r *cfgReader) str(param string) string {
	if v, ok := r.raw[param]; ok {
		return v
	}
	return configDefaults[param]
}

func (r *cfgReader) boolean(param string) bool {
	v := r.str(param)
	if v == "" {
		return false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		r.fail(param, fmt.Errorf("bad boolean %q", v))
		return false
	}
	return n != 0
}

func (r *cfgReader) integer(param string) int {
	v := r.str(param)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		r.fail(param, fmt.Errorf("bad integer %q", v))
		return 0
	}
	return n
}

func (r *cfgReader) intList(param string) []int {
	l, err := ParseIntList(sql.NullString{String: r.str(param), Valid: true})
	if err != nil {
		r.fail(param, err)
		return []int{}
	}
	return l
}

func (r *cfgReader) intMap(param string) map[int]int {
	m, err := ParseIntMapInt(sql.NullString{String: r.str(param), Valid: true})
	if err != nil {
		r.fail(param, err)
		return map[int]int{}
	}
	return m
}

// buildConfig converts the raw config rows into the typed Config plus the
// identity and format fields that live in the same table.
func buildConfig(raw map[string]string) (Config, ShowInfo, string, int, error) {
	r := &cfgReader{raw: raw}
	playbackRaw := r.integer("qLabCues")
	mode, ok := playbackModes[playbackRaw]
	if !ok {
		mode = "unknown"
	}
	cfg := Config{
		Channels:       r.intList("channels"),
		DCAs:           r.intList("dcas"),
		SpareBackup:    r.integer("spareBackup"),
		BackupChannels: r.str("backupChannels"),
		FX: FXConfig{
			Assigns: r.intList("fxAssigns"),
			Mutes:   r.intList("fxMutes"),
			Default: r.integer("defaultFX"),
			BusMap:  r.intMap("fxBusMap"),
		},
		Features: Features{
			AutoConnect:                 r.boolean("autoConnect"),
			ChannelLevels:               r.boolean("channelLevels"),
			CueZeroResetLevels:          r.boolean("cueZeroResetLevels"),
			CueZeroActorLabels:          r.boolean("cueZeroActorLabels"),
			SnippetRecall:               r.boolean("snippetRecall"),
			SceneRecall:                 r.boolean("sceneRecall"),
			ConsoleMuteDCAUnassign:      r.boolean("consoleMuteDCAUnassign"),
			SuppressDCAMuteBackupSwitch: r.boolean("suppressDCAMuteBackupSwitch"),
			SelectOnSpill:               r.boolean("selectOnSpill"),
			EnableChannelMonitoring:     r.boolean("enableChannelMonitoring"),
			ActiveChannelHighlight:      r.boolean("activeChannelHighlight"),
			DimDCAFaders:                r.boolean("dimDCAFaders"),
			DimDCAFadersSuppressColours: r.boolean("dimDCAFadersSuppressColours"),
			QLCLDyn1:                    r.boolean("qlclDyn1"),
		},
		CueZero: CueZero{
			Snippets:    r.intList("cueZeroSnippets"),
			Scenes:      r.intList("cueZeroScenes"),
			ScenePoints: r.intList("cueZeroScenePoints"),
		},
		Playback: Playback{
			Mode:         mode,
			Raw:          playbackRaw,
			Passcode:     r.str("qLabPasscode"),
			SuppressBack: r.boolean("qLabSuppressBack"),
		},
		DAW: DAW{
			Remote:  r.boolean("dawRemote"),
			IP:      r.str("dawIP"),
			PreRoll: r.integer("dawPreRoll"),
		},
		GangLR: GangLR{
			Enabled:  r.boolean("gangLR"),
			Channels: r.str("gangLRChannels"),
			Name:     r.str("gangLRName"),
			Colour:   r.str("gangLRColour"),
		},
		LabelLR:              r.integer("labelLR"),
		LabelTargetBus:       r.integer("labelTargetBus"),
		ButtonMap:            r.intMap("buttonMap"),
		MuteButtonMap:        r.intMap("muteButtonMap"),
		MuteButtonAssignKeys: r.intMap("muteButtonAssignKeys"),
		Raw:                  raw,
	}
	info := ShowInfo{
		Designer:      r.str("designer"),
		Venue:         r.str("venue"),
		TargetConsole: r.str("targetConsole"),
		Console: ConsoleInfo{
			Model:   r.str("consoleModel"),
			Version: r.str("consoleVersion"),
			IP:      r.str("consoleIP"),
			MAC:     r.str("consoleMAC"),
		},
	}
	minVersion := r.str("minVersion")
	profileSchemaVersion := r.integer("profileSchemaVersion")
	return cfg, info, minVersion, profileSchemaVersion, r.err
}
