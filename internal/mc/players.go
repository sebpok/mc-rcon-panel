package mc

import "strings"

func ParsePlayers(resp string) []string {
	// resp = "There are 2 of a max 20 players online: Player1, Player2, Player3, Player4, Player2, Player3, Player4, Player2, Player3, Player4"

	if !strings.Contains(resp, ":") {
		return nil
	}

	parts := strings.SplitN(resp, ":", 2)
	if len(parts) < 2 {
		return nil
	}

	raw := strings.TrimSpace(parts[1])
	if raw == "" {
		return nil
	}

	names := strings.Split(raw, ",")
	var players []string

	for _, n := range names {
		name := strings.TrimSpace(n)
		if name != "" {
			players = append(players, name)
		}
	}

	return players
}

func DiffAdded(old, new []string) []string {
	set := make(map[string]struct{})
	for _, o := range old {
		set[o] = struct{}{}
	}

	var out []string
	for _, n := range new {
		if _, ok := set[n]; !ok {
			out = append(out, n)
		}
	}
	return out
}

func DiffRemoved(old, new []string) []string {
	set := make(map[string]struct{})
	for _, n := range new {
		set[n] = struct{}{}
	}

	var out []string
	for _, o := range old {
		if _, ok := set[o]; !ok {
			out = append(out, o)
		}
	}
	return out
}
