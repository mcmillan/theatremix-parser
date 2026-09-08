package tmix

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
)

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

var null = sql.NullString{}

func TestParseIntList(t *testing.T) {
	cases := []struct {
		in      sql.NullString
		want    []int
		wantErr bool
	}{
		{null, []int{}, false},
		{ns(""), []int{}, false},
		{ns("1"), []int{1}, false},
		{ns("5,3,2,20"), []int{5, 3, 2, 20}, false},
		{ns("1,-1,-2"), []int{1, -1, -2}, false},
		{ns("1,,2"), []int{1, 2}, false},
		{ns("1,x"), nil, true},
		{ns("1, 2"), nil, true},
		{ns("1=2"), nil, true},
	}
	for _, c := range cases {
		got, err := ParseIntList(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseIntList(%q): err=%v wantErr=%v", c.in.String, err, c.wantErr)
			continue
		}
		if !c.wantErr && !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseIntList(%q) = %v, want %v", c.in.String, got, c.want)
		}
	}
}

func TestParseIntMap(t *testing.T) {
	got, err := ParseIntMap(ns("22=5,20=5,21=abc"))
	if err != nil {
		t.Fatal(err)
	}
	if want := map[int]string{22: "5", 20: "5", 21: "abc"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
	for _, empty := range []sql.NullString{null, ns("")} {
		m, err := ParseIntMap(empty)
		if err != nil || m == nil || len(m) != 0 {
			t.Errorf("empty input: got %v, %v", m, err)
		}
	}
	for _, bad := range []string{"1", "a=1", "1=2,3"} {
		if _, err := ParseIntMap(ns(bad)); err == nil {
			t.Errorf("ParseIntMap(%q): expected error", bad)
		}
	}
	im, err := ParseIntMapInt(ns("3=-150,4=50"))
	if err != nil || !reflect.DeepEqual(im, map[int]int{3: -150, 4: 50}) {
		t.Errorf("ParseIntMapInt: got %v, %v", im, err)
	}
	if _, err := ParseIntMapInt(ns("1=x")); err == nil {
		t.Error("ParseIntMapInt: expected error for non-integer value")
	}
}

func TestParseFxSpec(t *testing.T) {
	cases := []struct {
		in       string
		want     FxSpec
		wantJSON string
		wantErr  bool
	}{
		{"-1", FxSpec{None: true}, "null", false},
		{"1", FxSpec{Buses: []int{1}}, "[1]", false},
		{"3+4", FxSpec{Buses: []int{3, 4}}, "[3,4]", false},
		{"1+2+3", FxSpec{Buses: []int{1, 2, 3}}, "[1,2,3]", false},
		{"", FxSpec{}, "", true},
		{"1+-1", FxSpec{}, "", true},
		{"a", FxSpec{}, "", true},
	}
	for _, c := range cases {
		got, err := ParseFxSpec(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseFxSpec(%q): err=%v wantErr=%v", c.in, err, c.wantErr)
			continue
		}
		if c.wantErr {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseFxSpec(%q) = %+v, want %+v", c.in, got, c.want)
		}
		b, _ := json.Marshal(got)
		if string(b) != c.wantJSON {
			t.Errorf("ParseFxSpec(%q) JSON = %s, want %s", c.in, b, c.wantJSON)
		}
	}
	if b, _ := json.Marshal(FxSpec{}); string(b) != "[]" {
		t.Errorf("zero FxSpec JSON = %s, want []", b)
	}
}

func TestParseSemiList(t *testing.T) {
	if got := ParseSemiList("gain;hpf;eq"); !reflect.DeepEqual(got, []string{"gain", "hpf", "eq"}) {
		t.Errorf("got %v", got)
	}
	if got := ParseSemiList(""); len(got) != 0 || got == nil {
		t.Errorf("empty: got %#v", got)
	}
	if got := ParseSemiList("dyn;"); !reflect.DeepEqual(got, []string{"dyn"}) {
		t.Errorf("trailing separator: got %v", got)
	}
}
