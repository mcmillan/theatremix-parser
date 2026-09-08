package tmix

import (
	"fmt"
	"sort"
)

// Violation is one failed invariant from spec §6.
type Violation struct {
	Rule    string `json:"rule"`
	Cue     string `json:"cue,omitempty"`
	Message string `json:"message"`
}

func (v Violation) String() string {
	if v.Cue != "" {
		return fmt.Sprintf("%s: cue %s: %s", v.Rule, v.Cue, v.Message)
	}
	return fmt.Sprintf("%s: %s", v.Rule, v.Message)
}

// Validate checks the spec §6 invariants (I1–I11). Every rule held on every
// app-written file examined, so violations indicate a corrupted or hand-edited
// file — or a spec gap worth reporting.
func Validate(s *Show) []Violation {
	var out []Violation
	add := func(rule, cue, format string, args ...any) {
		out = append(out, Violation{Rule: rule, Cue: cue, Message: fmt.Sprintf(format, args...)})
	}
	controlled := intSet(s.Config.Channels)
	fxAssigns := intSet(s.Config.FX.Assigns)
	fxMutes := intSet(s.Config.FX.Mutes)
	positions := map[int]bool{}
	for _, p := range s.Positions {
		positions[p.ID] = true
	}
	profiles := map[int]Profile{}
	defaultsPerChannel := map[int]int{}
	for _, p := range s.Profiles {
		profiles[p.ID] = p
		if p.IsDefault {
			defaultsPerChannel[p.Channel]++
		}
	}

	seenIDs := map[[2]int]bool{}
	for i := range s.Cues {
		c := &s.Cues[i]
		id := c.DisplayID()
		key := [2]int{c.Number, c.Point}
		if seenIDs[key] {
			add("I11", id, "duplicate cue id")
		}
		seenIDs[key] = true
		if c.Number == 0 && c.Point == 0 {
			add("I10", id, "cue 0.0 must not be stored")
		}

		assigned := map[int]int{} // channel -> dca
		for _, d := range c.DCAs {
			for _, ch := range d.Channels {
				if !controlled[ch] {
					add("I1", id, "DCA %d channel %d is not in config.channels", d.DCA, ch)
				}
				if prev, dup := assigned[ch]; dup {
					add("I2", id, "channel %d assigned to DCA %d and DCA %d", ch, prev, d.DCA)
				}
				assigned[ch] = d.DCA
			}
		}
		checkKeys := func(name string, keys []int) {
			for _, ch := range keys {
				if _, ok := assigned[ch]; !ok {
					add("I3", id, "%s key %d is not assigned to a DCA", name, ch)
				}
			}
		}
		checkKeys("channelPositions", sortedKeys(c.ChannelPositions))
		checkKeys("channelProfiles", sortedKeys(c.ChannelProfiles))
		checkKeys("channelFX", sortedFxKeys(c.ChannelFX))
		checkKeys("channelLevels", sortedKeys(c.ChannelLevels))

		for _, ch := range sortedKeys(c.ChannelPositions) {
			pos := c.ChannelPositions[ch]
			if pos == 0 {
				add("I4", id, "channel %d stores default position 0", ch)
			} else if !positions[pos] {
				add("I4", id, "channel %d position %d does not exist", ch, pos)
			}
		}
		for _, ch := range sortedKeys(c.ChannelProfiles) {
			pid := c.ChannelProfiles[ch]
			p, ok := profiles[pid]
			switch {
			case !ok:
				add("I5", id, "channel %d profile %d does not exist", ch, pid)
			case p.IsDefault:
				add("I5", id, "channel %d stores default profile %d", ch, pid)
			case p.Channel != ch:
				add("I5", id, "channel %d profile %d belongs to channel %d", ch, pid, p.Channel)
			}
		}
		for _, ch := range sortedFxKeys(c.ChannelFX) {
			spec := c.ChannelFX[ch]
			if spec.None {
				continue
			}
			for _, bus := range spec.Buses {
				if !fxAssigns[bus] {
					add("I6", id, "channel %d FX bus %d is not in config.fxAssigns", ch, bus)
				}
			}
		}
		for _, ch := range sortedKeys(c.ChannelLevels) {
			v := c.ChannelLevels[ch]
			if v == 0 || v < -150 || v > 50 {
				add("I7", id, "channel %d level offset %d outside -150..50 or zero", ch, v)
			}
		}
		for _, bus := range c.FxUnmuted {
			if !fxMutes[bus] {
				add("I8", id, "fxMutes bus %d is not in config.fxMutes", bus)
			}
		}
	}

	for _, ch := range s.Config.Channels {
		if n := defaultsPerChannel[ch]; n != 1 {
			add("I9", "", "channel %d has %d default profiles (want 1)", ch, n)
		}
	}
	return out
}

func intSet(l []int) map[int]bool {
	m := make(map[int]bool, len(l))
	for _, v := range l {
		m[v] = true
	}
	return m
}

func sortedKeys(m map[int]int) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func sortedFxKeys(m map[int]FxSpec) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}
