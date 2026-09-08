package tmix

// buildChannels synthesises the channel list from config.channels and each
// channel's default profile (spec §0.7). Order follows config.channels.
func buildChannels(cfg Config, profiles []Profile) []Channel {
	defaults := map[int]Profile{}
	for _, p := range profiles {
		if p.IsDefault {
			if _, seen := defaults[p.Channel]; !seen {
				defaults[p.Channel] = p
			}
		}
	}
	out := make([]Channel, 0, len(cfg.Channels))
	for _, ch := range cfg.Channels {
		c := Channel{Number: ch, IsAuxIn: ch < 0}
		if p, ok := defaults[ch]; ok {
			c.Name, c.Label, c.DefaultProfileID = p.Name, p.Label, p.ID
		}
		out = append(out, c)
	}
	return out
}
