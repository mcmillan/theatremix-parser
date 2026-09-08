package tmix

import (
	"database/sql"
	"fmt"
	"sort"
)

// maxDCAs is the number of dcaNN column pairs the schema can hold.
const maxDCAs = 12

// colourNames maps cues.colour to the app's highlight colours (spec §4;
// 0/NULL = none is confirmed, the 1–5 order is inferred).
var colourNames = map[int]string{1: "red", 2: "yellow", 3: "green", 4: "blue", 5: "purple"}

func cueColumns() []string {
	cols := []string{"number", "point", "name"}
	for n := 1; n <= maxDCAs; n++ {
		cols = append(cols, dcaColumn(n, "Channels"))
	}
	for n := 1; n <= maxDCAs; n++ {
		cols = append(cols, dcaColumn(n, "Label"))
	}
	return append(cols, "channelPositions", "channelProfiles", "channelFX", "fxMutes",
		"snippets", "scenes", "scenePoints", "qLabCue", "channelLevels", "colour", "skip")
}

func dcaColumn(n int, suffix string) string {
	return fmt.Sprintf("dca%02d%s", n, suffix)
}

func readCues(db *sql.DB, sc *schema, cfg *Config) ([]Cue, error) {
	rows, err := selectExisting(db, sc, "cues", cueColumns(), `"number", "point"`)
	if err != nil {
		return nil, err
	}
	// Columns beyond len(config.dcas) are unused (spec §3.2). A file with no
	// dcas param is malformed; fall back to every column rather than hide data.
	ndcas := len(cfg.DCAs)
	if ndcas == 0 || ndcas > maxDCAs {
		ndcas = maxDCAs
	}
	out := make([]Cue, 0, len(rows))
	for _, r := range rows {
		c, err := decodeCue(r, ndcas, cfg)
		if err != nil {
			return nil, fmt.Errorf("cue %s: %w", cueID(mustInt(r, "number"), mustInt(r, "point")), err)
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Number != out[j].Number {
			return out[i].Number < out[j].Number
		}
		return out[i].Point < out[j].Point
	})
	return out, nil
}

// mustInt is for error messages only: a column that fails to parse reads as 0.
func mustInt(r row, col string) int {
	n, _ := r.integer(col)
	return n
}

// decodeCue applies the spec §3.2 rules to one cues row.
func decodeCue(r row, ndcas int, cfg *Config) (Cue, error) {
	var c Cue
	var err error
	if c.Number, err = r.integer("number"); err != nil {
		return c, err
	}
	if c.Point, err = r.integer("point"); err != nil {
		return c, err
	}
	c.ID = c.DisplayID()
	c.Name = r.str("name")
	if c.Skip, err = r.boolean("skip"); err != nil {
		return c, err
	}
	if c.Colour, err = r.integer("colour"); err != nil {
		return c, err
	}
	if name, ok := colourNames[c.Colour]; ok {
		c.ColourName = &name
	}

	c.DCAs = []DcaAssign{}
	assigned := map[int]bool{}
	for n := 1; n <= ndcas; n++ {
		chs, err := ParseIntList(r[dcaColumn(n, "Channels")])
		if err != nil {
			return c, fmt.Errorf("%s: %w", dcaColumn(n, "Channels"), err)
		}
		label := r.str(dcaColumn(n, "Label"))
		if len(chs) == 0 && label == "" {
			continue
		}
		sort.Ints(chs)
		for _, ch := range chs {
			assigned[ch] = true
		}
		consoleDCA := n
		if n <= len(cfg.DCAs) {
			consoleDCA = cfg.DCAs[n-1]
		}
		c.DCAs = append(c.DCAs, DcaAssign{DCA: n, ConsoleDCA: consoleDCA, Channels: chs, Label: label})
	}

	if c.ChannelPositions, err = ParseIntMapInt(r["channelPositions"]); err != nil {
		return c, fmt.Errorf("channelPositions: %w", err)
	}
	if c.ChannelProfiles, err = ParseIntMapInt(r["channelProfiles"]); err != nil {
		return c, fmt.Errorf("channelProfiles: %w", err)
	}
	if c.ChannelFX, err = parseFxMap(r["channelFX"]); err != nil {
		return c, fmt.Errorf("channelFX: %w", err)
	}
	if c.ChannelLevels, err = ParseIntMapInt(r["channelLevels"]); err != nil {
		return c, fmt.Errorf("channelLevels: %w", err)
	}
	c.ChannelLevelsDb = make(map[int]float64, len(c.ChannelLevels))
	for ch, tenths := range c.ChannelLevels {
		c.ChannelLevelsDb[ch] = float64(tenths) / 10
	}
	lists := []struct {
		col string
		dst *[]int
	}{
		{"fxMutes", &c.FxUnmuted}, {"snippets", &c.Snippets},
		{"scenes", &c.Scenes}, {"scenePoints", &c.ScenePoints},
	}
	for _, l := range lists {
		if *l.dst, err = ParseIntList(r[l.col]); err != nil {
			return c, fmt.Errorf("%s: %w", l.col, err)
		}
	}
	c.PlaybackCue = r.str("qLabCue")

	c.MutedChannels = []int{}
	for _, ch := range cfg.Channels {
		if !assigned[ch] {
			c.MutedChannels = append(c.MutedChannels, ch)
		}
	}
	sort.Ints(c.MutedChannels)
	return c, nil
}
