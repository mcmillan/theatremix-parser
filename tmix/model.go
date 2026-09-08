package tmix

import "fmt"

// Show is the decoded content of a .tmix file. Field order here is the JSON
// output order.
type Show struct {
	Format        Format         `json:"format"`
	Info          ShowInfo       `json:"show"`
	Config        Config         `json:"config"`
	Channels      []Channel      `json:"channels"`
	Positions     []Position     `json:"positions"`
	Profiles      []Profile      `json:"profiles"`
	Ensembles     []Ensemble     `json:"ensembles"`
	Actors        []Actor        `json:"actors"`
	ActorProfiles []ActorProfile `json:"actorProfiles"`
	ActorGroups   []ActorGroup   `json:"actorGroups"`
	Caches        Caches         `json:"caches"`
	Cues          []Cue          `json:"cues"`
}

// Format describes the file itself: which schema generation wrote it and
// which optional tables/columns are present (spec §2.2, §9).
type Format struct {
	SchemaVariant        string              `json:"schemaVariant"`
	MinVersion           string              `json:"minVersion"`
	ProfileSchemaVersion int                 `json:"profileSchemaVersion"`
	Tables               []string            `json:"tables"`
	OptionalColumns      map[string][]string `json:"optionalColumns"`
}

// ShowInfo carries identity fields from config.
type ShowInfo struct {
	Designer      string      `json:"designer"`
	Venue         string      `json:"venue"`
	TargetConsole string      `json:"targetConsole"`
	Console       ConsoleInfo `json:"console"`
}

// ConsoleInfo is the last-connected console (informational).
type ConsoleInfo struct {
	Model   string `json:"model"`
	Version string `json:"version"`
	IP      string `json:"ip"`
	MAC     string `json:"mac"`
}

// Config is the typed view of the config table (spec §3.1). Raw holds every
// config row verbatim; values the spec marks [?] are passed through untouched.
type Config struct {
	Channels             []int             `json:"channels"`
	DCAs                 []int             `json:"dcas"`
	FX                   FXConfig          `json:"fx"`
	SpareBackup          int               `json:"spareBackup"`
	BackupChannels       string            `json:"backupChannels"`
	Features             Features          `json:"features"`
	CueZero              CueZero           `json:"cueZero"`
	Playback             Playback          `json:"playback"`
	DAW                  DAW               `json:"daw"`
	GangLR               GangLR            `json:"gangLR"`
	LabelLR              int               `json:"labelLR"`
	LabelTargetBus       int               `json:"labelTargetBus"`
	ButtonMap            map[int]int       `json:"buttonMap"`
	MuteButtonMap        map[int]int       `json:"muteButtonMap"`
	MuteButtonAssignKeys map[int]int       `json:"muteButtonAssignKeys"`
	Raw                  map[string]string `json:"raw"`
}

// FXConfig groups the FX-related config params.
type FXConfig struct {
	Assigns []int       `json:"assigns"`
	Mutes   []int       `json:"mutes"`
	Default int         `json:"default"`
	BusMap  map[int]int `json:"busMap"`
}

// Features holds every boolean config switch.
type Features struct {
	AutoConnect                 bool `json:"autoConnect"`
	ChannelLevels               bool `json:"channelLevels"`
	CueZeroResetLevels          bool `json:"cueZeroResetLevels"`
	CueZeroActorLabels          bool `json:"cueZeroActorLabels"`
	SnippetRecall               bool `json:"snippetRecall"`
	SceneRecall                 bool `json:"sceneRecall"`
	ConsoleMuteDCAUnassign      bool `json:"consoleMuteDCAUnassign"`
	SuppressDCAMuteBackupSwitch bool `json:"suppressDCAMuteBackupSwitch"`
	SelectOnSpill               bool `json:"selectOnSpill"`
	EnableChannelMonitoring     bool `json:"enableChannelMonitoring"`
	ActiveChannelHighlight      bool `json:"activeChannelHighlight"`
	DimDCAFaders                bool `json:"dimDCAFaders"`
	DimDCAFadersSuppressColours bool `json:"dimDCAFadersSuppressColours"`
	QLCLDyn1                    bool `json:"qlclDyn1"`
}

// CueZero is what the implicit line-checks cue 0 recalls.
type CueZero struct {
	Snippets    []int `json:"snippets"`
	Scenes      []int `json:"scenes"`
	ScenePoints []int `json:"scenePoints"`
}

// Playback is the playback-software cue recall configuration.
type Playback struct {
	Mode         string `json:"mode"` // off | qlab | scs | cueplayer | unknown
	Raw          int    `json:"raw"`
	Passcode     string `json:"passcode"`
	SuppressBack bool   `json:"suppressBack"`
}

// DAW is the REAPER link configuration.
type DAW struct {
	Remote  bool   `json:"remote"`
	IP      string `json:"ip"`
	PreRoll int    `json:"preRoll"`
}

// GangLR is the LR-fader ganging configuration; Channels and Colour are raw.
type GangLR struct {
	Enabled  bool   `json:"enabled"`
	Channels string `json:"channels"`
	Name     string `json:"name"`
	Colour   string `json:"colour"`
}

// Channel is synthesised from config.channels and the default profiles.
type Channel struct {
	Number           int    `json:"number"`
	Name             string `json:"name"`
	Label            string `json:"label"`
	IsAuxIn          bool   `json:"isAuxIn"`
	DefaultProfileID int    `json:"defaultProfileId"`
}

// Position is an acting-position preset (spec §3.3).
type Position struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	ShortName string  `json:"shortName"`
	DelayMs   float64 `json:"delayMs"`
	Pan       float64 `json:"pan"`
	Buses     []int   `json:"buses"`
}

// Profile is a channel's default or alternate processing profile (spec §3.4).
type Profile struct {
	ID        int         `json:"id"`
	Channel   int         `json:"channel"`
	Name      string      `json:"name"`
	Label     string      `json:"label"`
	IsDefault bool        `json:"isDefault"`
	Data      ProfileData `json:"data"`
}

// Ensemble is a named channel group (spec §3.5).
type Ensemble struct {
	ID              int         `json:"id"`
	Name            string      `json:"name"`
	Channels        []int       `json:"channels"`
	ChannelProfiles map[int]int `json:"channelProfiles"`
}

// Actor is a performer attached to a channel (spec §3.6).
type Actor struct {
	ID      int    `json:"id"`
	Channel int    `json:"channel"`
	Name    string `json:"name"`
	Order   int    `json:"order"`
	Active  bool   `json:"active"`
}

// ActorProfile is an actor's stored values for one profile.
type ActorProfile struct {
	Actor   int         `json:"actor"`
	Profile int         `json:"profile"`
	Data    ProfileData `json:"data"`
}

// ActorGroup is a cast: the active actor per channel.
type ActorGroup struct {
	ID            int         `json:"id"`
	Name          string      `json:"name"`
	ChannelActors map[int]int `json:"channelActors"`
}

// Caches are console name caches (spec §3.7).
type Caches struct {
	Snippets map[int]string    `json:"snippets"`
	FX       map[int]string    `json:"fx"`
	Scenes   []SceneCacheEntry `json:"scenes"`
}

// SceneCacheEntry is one sceneCache row.
type SceneCacheEntry struct {
	Scene int    `json:"scene"`
	Point int    `json:"point"`
	Name  string `json:"name"`
}

// Cue is one stored cue (spec §3.2) plus derived fields.
type Cue struct {
	Number           int             `json:"number"`
	Point            int             `json:"point"`
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Skip             bool            `json:"skip"`
	Colour           int             `json:"colour"`
	ColourName       *string         `json:"colourName"`
	DCAs             []DcaAssign     `json:"dcas"`
	ChannelPositions map[int]int     `json:"channelPositions"`
	ChannelProfiles  map[int]int     `json:"channelProfiles"`
	ChannelFX        map[int]FxSpec  `json:"channelFX"`
	ChannelLevels    map[int]int     `json:"channelLevels"`
	ChannelLevelsDb  map[int]float64 `json:"channelLevelsDb"`
	FxUnmuted        []int           `json:"fxUnmuted"`
	Snippets         []int           `json:"snippets"`
	Scenes           []int           `json:"scenes"`
	ScenePoints      []int           `json:"scenePoints"`
	PlaybackCue      string          `json:"playbackCue"`
	MutedChannels    []int           `json:"mutedChannels"`
}

// DcaAssign is one DCA's assignment in a cue. DCA is the column index
// (1-based, dca01…dca12); ConsoleDCA is the console DCA number it controls
// (config.dcas[DCA-1]) when known, else equal to DCA.
type DcaAssign struct {
	DCA        int    `json:"dca"`
	ConsoleDCA int    `json:"consoleDca"`
	Channels   []int  `json:"channels"`
	Label      string `json:"label"`
}

// DisplayID renders a cue id the way the app does: "5" or "5.20", never
// zero-padded (spec §3.2).
func (c *Cue) DisplayID() string {
	return cueID(c.Number, c.Point)
}

func cueID(number, point int) string {
	if point == 0 {
		return fmt.Sprint(number)
	}
	return fmt.Sprintf("%d.%d", number, point)
}
