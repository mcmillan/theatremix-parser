package tmix

import (
	"path/filepath"
	"testing"

	"github.com/mcmillan/theatremix-parser/internal/fixture"
)

func TestValidateCleanFixture(t *testing.T) {
	for _, variant := range []string{"A", "B", "C"} {
		show := openFixture(t, variant)
		if v := Validate(show); len(v) != 0 {
			t.Errorf("variant %s: unexpected violations: %v", variant, v)
		}
	}
}

func TestValidateDetectsEachRule(t *testing.T) {
	cases := map[string][]string{
		"I1":  {`UPDATE cues SET dca01Channels = '1,42' WHERE number = 1 AND point = 0`},
		"I2":  {`UPDATE cues SET dca02Channels = '1' WHERE number = 1 AND point = 0`},
		"I3":  {`UPDATE cues SET channelLevels = '8=-20' WHERE number = 1 AND point = 0`},
		"I4":  {`UPDATE cues SET channelPositions = '3=0,4=77' WHERE number = 1 AND point = 0`},
		"I5":  {`UPDATE cues SET channelProfiles = '3=12' WHERE number = 1 AND point = 0`},
		"I6":  {`UPDATE cues SET channelFX = '1=4' WHERE number = 1 AND point = 0`},
		"I7":  {`UPDATE cues SET channelLevels = '3=-151,4=0' WHERE number = 1 AND point = 0`},
		"I8":  {`UPDATE cues SET fxMutes = '4' WHERE number = 1 AND point = 0`},
		"I9":  {`UPDATE profiles SET "default" = 0 WHERE id = 2`},
		"I10": {`INSERT INTO cues (number, point, name) VALUES (0, 0, 'line checks')`},
	}
	for rule, stmts := range cases {
		t.Run(rule, func(t *testing.T) {
			path := fixture.Create(t, filepath.Join(t.TempDir(), "show.tmix"), "C")
			fixture.Mutate(t, path, stmts...)
			show, err := OpenFile(path)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, v := range Validate(show) {
				if v.Rule == rule {
					found = true
				} else if rule != "I4" && rule != "I3" {
					t.Errorf("unexpected extra violation %v", v)
				}
			}
			if !found {
				t.Errorf("rule %s not reported", rule)
			}
		})
	}
}

func TestValidateDuplicateCueIDs(t *testing.T) {
	show := openFixture(t, "C")
	show.Cues = append(show.Cues, show.Cues[0])
	found := false
	for _, v := range Validate(show) {
		if v.Rule == "I11" {
			found = true
		}
	}
	if !found {
		t.Error("I11 not reported for duplicate cue ids")
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
