package tmix

import "testing"

func TestValidateCleanFixture(t *testing.T) {
	if v := Validate(openReal(t)); len(v) != 0 {
		t.Errorf("unexpected violations: %v", v)
	}
}

// TestValidateDetectsEachRule breaks one invariant at a time in the decoded
// model (cue 1 has channels 11, 7, 14 and 4 on DCAs 1, 3 and 5; config has
// fxAssigns=[2], fxMutes=[3]; profile ids equal their channel numbers).
func TestValidateDetectsEachRule(t *testing.T) {
	cases := map[string]func(s *Show){
		"I1": func(s *Show) { c := findCue(t, s, "1"); c.DCAs[0].Channels = append(c.DCAs[0].Channels, 42) },
		"I2": func(s *Show) { c := findCue(t, s, "1"); c.DCAs[1].Channels = append(c.DCAs[1].Channels, 11) },
		"I3": func(s *Show) { findCue(t, s, "1").ChannelLevels[8] = -20 },
		"I4": func(s *Show) { c := findCue(t, s, "1"); c.ChannelPositions[4] = 0; c.ChannelPositions[11] = 77 },
		"I5": func(s *Show) {
			c := findCue(t, s, "1")
			c.ChannelProfiles[4] = 4
			c.ChannelProfiles[11] = 7
			c.ChannelProfiles[7] = 99
		},
		"I6":  func(s *Show) { findCue(t, s, "1").ChannelFX[4] = FxSpec{Buses: []int{1}} },
		"I7":  func(s *Show) { c := findCue(t, s, "1"); c.ChannelLevels[4] = -151; c.ChannelLevels[11] = 0 },
		"I8":  func(s *Show) { findCue(t, s, "1").FxUnmuted = []int{4} },
		"I9":  func(s *Show) { s.Profiles[0].IsDefault = false },
		"I10": func(s *Show) { s.Cues = append(s.Cues, Cue{Number: 0, Point: 0, ID: "0"}) },
		"I11": func(s *Show) { s.Cues = append(s.Cues, s.Cues[0]) },
	}
	for rule, mutate := range cases {
		t.Run(rule, func(t *testing.T) {
			show := openReal(t)
			mutate(show)
			found := false
			for _, v := range Validate(show) {
				if v.Rule == rule {
					found = true
				} else if rule != "I11" { // the duplicated cue re-reports nothing new
					t.Errorf("unexpected extra violation %v", v)
				}
			}
			if !found {
				t.Errorf("rule %s not reported", rule)
			}
		})
	}
}

func TestViolationString(t *testing.T) {
	if s := (Violation{Rule: "I2", Cue: "5.20", Message: "x"}).String(); s != "I2: cue 5.20: x" {
		t.Errorf("got %q", s)
	}
	if s := (Violation{Rule: "I9", Message: "y"}).String(); s != "I9: y" {
		t.Errorf("got %q", s)
	}
}
