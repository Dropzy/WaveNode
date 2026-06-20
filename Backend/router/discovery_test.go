package router

import (
	"testing"

	"music-server/database"
)

func TestMatchDiscoveryRecommendations(t *testing.T) {
	recommendations := []discoveryRecommendation{
		{Title: "The Nine", Artist: "Bad Company UK", Album: "The Nine (Remastered)"},
		{Title: "Atlantis (original mix)", Artist: "Gravit-e", Album: "Atlantis Ep"},
		{Title: "Missing Track", Artist: "Missing Artist"},
	}
	tracks := []database.Music{
		{ID: "track_1", Title: "The Nine", Artist: "Bad Company Uk", Album: "The Nine (remastered)"},
		{ID: "track_2", Title: "Atlantis (Original Mix)", Artist: "Gravit-e", Album: "Atlantis Ep"},
		{ID: "track_3", Title: "Different", Artist: "Artist"},
	}

	matched, missing := matchDiscoveryRecommendations(recommendations, tracks)
	if len(matched) != 2 {
		t.Fatalf("expected 2 matched tracks, got %d", len(matched))
	}
	if matched[0].ID != "track_1" || matched[1].ID != "track_2" {
		t.Fatalf("unexpected matched tracks: %#v", matched)
	}
	if len(missing) != 1 || missing[0].Title != "Missing Track" {
		t.Fatalf("unexpected missing recommendations: %#v", missing)
	}
}

func TestNormalizeDiscoveryText(t *testing.T) {
	got := normalizeDiscoveryText(" Atlantis (Original Mix) ")
	if got != "atlantisoriginalmix" {
		t.Fatalf("unexpected normalized value %q", got)
	}
}
