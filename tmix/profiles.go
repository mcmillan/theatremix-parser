package tmix

import (
	"database/sql"
	"fmt"
	"strings"
)

// ProfileData is a parsed profiles.data / actorProfiles.data value (spec §3.4):
// OSC-style parameter paths with raw string values, plus the lists of
// processing types the channel/actor stores.
type ProfileData struct {
	Params      map[string]string `json:"params"`
	StoreParams []string          `json:"storeParams"`
	ActorParams []string          `json:"actorParams"`
}

func emptyProfileData() ProfileData {
	return ProfileData{Params: map[string]string{}, StoreParams: []string{}, ActorParams: []string{}}
}

// ParseProfileData parses the ProfileData grammar. Values are kept as strings
// because units are console-native and the key set is open-ended.
func ParseProfileData(s sql.NullString) (ProfileData, error) {
	pd := emptyProfileData()
	v := text(s)
	if v == "" {
		return pd, nil
	}
	for _, tok := range strings.Split(v, ",") {
		if tok == "" {
			continue
		}
		k, val, ok := strings.Cut(tok, "=")
		if !ok {
			return pd, fmt.Errorf("profile data: token %q has no '='", tok)
		}
		switch k {
		case "storeParams":
			pd.StoreParams = ParseSemiList(val)
		case "actorParams":
			pd.ActorParams = ParseSemiList(val)
		default:
			pd.Params[k] = val
		}
	}
	return pd, nil
}
