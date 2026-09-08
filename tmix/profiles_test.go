package tmix

import (
	"reflect"
	"testing"
)

func TestParseProfileData(t *testing.T) {
	pd, err := ParseProfileData(ns("/gain/headamp=5,/eq/1/type=PEQ,/eq/1/g=-9.4,storeParams=gain;hpf;eq,actorParams=gain"))
	if err != nil {
		t.Fatal(err)
	}
	want := ProfileData{
		Params:      map[string]string{"/gain/headamp": "5", "/eq/1/type": "PEQ", "/eq/1/g": "-9.4"},
		StoreParams: []string{"gain", "hpf", "eq"},
		ActorParams: []string{"gain"},
	}
	if !reflect.DeepEqual(pd, want) {
		t.Errorf("got %+v want %+v", pd, want)
	}
	for _, empty := range []string{"", "NULL"} {
		in := ns(empty)
		if empty == "NULL" {
			in = null
		}
		pd, err := ParseProfileData(in)
		if err != nil || pd.Params == nil || pd.StoreParams == nil || pd.ActorParams == nil ||
			len(pd.Params)+len(pd.StoreParams)+len(pd.ActorParams) != 0 {
			t.Errorf("empty %q: got %+v, %v", empty, pd, err)
		}
	}
	if _, err := ParseProfileData(ns("/hpf/on")); err == nil {
		t.Error("expected error for pair without '='")
	}
}
