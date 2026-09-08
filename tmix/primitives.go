// Package tmix reads TheatreMix .tmix show files (SQLite databases) into a
// decoded, JSON-friendly model.
//
// Every parsing rule implemented here is specified in
// docs/TMIX_FORMAT_SPEC.agent.md; section references in comments point there.
package tmix

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// text returns the value of a nullable column, treating NULL as "" (spec §0.4).
func text(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

// ParseIntList parses the IntList micro-format ("1,4,6"). NULL and "" yield an
// empty, non-nil slice. File order is preserved; most callers treat the result
// as a set (spec §2).
func ParseIntList(s sql.NullString) ([]int, error) {
	out := []int{}
	v := text(s)
	if v == "" {
		return out, nil
	}
	for _, tok := range strings.Split(v, ",") {
		if tok == "" {
			continue
		}
		n, err := strconv.Atoi(tok)
		if err != nil {
			return nil, fmt.Errorf("int list %q: bad token %q", v, tok)
		}
		out = append(out, n)
	}
	return out, nil
}

// ParseIntMap parses the IntMap micro-format ("k=v,k=v") keeping values as raw
// strings. Key order in the file is arbitrary and is not preserved (spec §2).
func ParseIntMap(s sql.NullString) (map[int]string, error) {
	out := map[int]string{}
	v := text(s)
	if v == "" {
		return out, nil
	}
	for _, tok := range strings.Split(v, ",") {
		if tok == "" {
			continue
		}
		k, val, ok := strings.Cut(tok, "=")
		if !ok {
			return nil, fmt.Errorf("int map %q: token %q has no '='", v, tok)
		}
		key, err := strconv.Atoi(k)
		if err != nil {
			return nil, fmt.Errorf("int map %q: bad key %q", v, k)
		}
		out[key] = val
	}
	return out, nil
}

// ParseIntMapInt is ParseIntMap with integer values.
func ParseIntMapInt(s sql.NullString) (map[int]int, error) {
	raw, err := ParseIntMap(s)
	if err != nil {
		return nil, err
	}
	out := make(map[int]int, len(raw))
	for k, v := range raw {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("int map %q: bad value %q for key %d", text(s), v, k)
		}
		out[k] = n
	}
	return out, nil
}

// FxSpec is a decoded channelFX value: either an explicit "no FX" (-1, which
// overrides config defaultFX) or one or more TheatreMix FX bus numbers
// ("1", "3+4") (spec §3.2).
type FxSpec struct {
	None  bool
	Buses []int
}

// ParseFxSpec parses a single channelFX map value.
func ParseFxSpec(v string) (FxSpec, error) {
	if v == "-1" {
		return FxSpec{None: true}, nil
	}
	if v == "" {
		return FxSpec{}, fmt.Errorf("fx spec: empty value")
	}
	buses := []int{}
	for _, tok := range strings.Split(v, "+") {
		n, err := strconv.Atoi(tok)
		if err != nil || n < 0 {
			return FxSpec{}, fmt.Errorf("fx spec %q: bad bus %q", v, tok)
		}
		buses = append(buses, n)
	}
	return FxSpec{Buses: buses}, nil
}

// MarshalJSON renders "no FX" as null and otherwise the list of buses.
func (f FxSpec) MarshalJSON() ([]byte, error) {
	if f.None {
		return []byte("null"), nil
	}
	if f.Buses == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(f.Buses)
}

// ParseSemiList parses the SemiList micro-format ("gain;hpf;eq").
func ParseSemiList(s string) []string {
	out := []string{}
	for _, tok := range strings.Split(s, ";") {
		if tok != "" {
			out = append(out, tok)
		}
	}
	return out
}

// parseFxMap parses a channelFX column into per-channel FxSpecs.
func parseFxMap(s sql.NullString) (map[int]FxSpec, error) {
	raw, err := ParseIntMap(s)
	if err != nil {
		return nil, err
	}
	out := make(map[int]FxSpec, len(raw))
	for ch, v := range raw {
		spec, err := ParseFxSpec(v)
		if err != nil {
			return nil, fmt.Errorf("channel %d: %w", ch, err)
		}
		out[ch] = spec
	}
	return out, nil
}
