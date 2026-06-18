package database

import (
	"regexp"
	"strings"
)

var artistNameDelimiterPattern = regexp.MustCompile(`(?i)\s*(?:,|;|\s+\+\s+|\s+&\s+|\s+and\s+|\s+x\s+|\s+vs\.?\s+)\s*`)

// SplitArtistNames separates common multi-artist metadata strings while leaving
// the original track artist value available for display.
func SplitArtistNames(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	parts := artistNameDelimiterPattern.Split(name, -1)
	artists := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		clean := strings.Trim(strings.TrimSpace(part), `"'`)
		if clean == "" {
			continue
		}

		key := strings.ToLower(clean)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		artists = append(artists, clean)
	}

	if len(artists) == 0 {
		return []string{name}
	}
	return artists
}

func PrimaryArtistName(name string) string {
	artists := SplitArtistNames(name)
	if len(artists) == 0 {
		return strings.TrimSpace(name)
	}
	return artists[0]
}

func PrimaryArtistNameMatches(trackArtist, artistName string) bool {
	target := strings.ToLower(strings.TrimSpace(artistName))
	if target == "" {
		return false
	}
	return strings.ToLower(PrimaryArtistName(trackArtist)) == target
}
