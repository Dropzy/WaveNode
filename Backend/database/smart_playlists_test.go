package database

import (
	"testing"
	"time"
)

func TestValidateSmartPlaylistRulesDefaults(t *testing.T) {
	rules := &SmartPlaylistRules{}
	if err := ValidateSmartPlaylistRules(rules); err != nil {
		t.Fatalf("ValidateSmartPlaylistRules() error = %v", err)
	}
	if rules.Match != "all" || rules.SortBy != "date_added" || rules.SortDirection != "desc" || rules.Limit != 100 {
		t.Fatalf("unexpected defaults: %#v", rules)
	}
}

func TestValidateSmartPlaylistRulesRejectsInvalidCondition(t *testing.T) {
	rules := &SmartPlaylistRules{
		Match: "all", SortBy: "title", SortDirection: "asc", Limit: 25,
		Conditions: []SmartPlaylistCondition{{Field: "rating", Operator: "contains", Value: "5"}},
	}
	if err := ValidateSmartPlaylistRules(rules); err == nil {
		t.Fatal("expected invalid rating operator to be rejected")
	}
}

func TestSmartTrackMatchesAllAndAny(t *testing.T) {
	track := Music{
		Title: "Midnight Drive", Artist: "WaveNode", Genre: "Drum & Bass",
		PlayCount: 12, CreatedAt: time.Now().Add(-24 * time.Hour),
	}
	conditions := []SmartPlaylistCondition{
		{Field: "genre", Operator: "contains", Value: "bass"},
		{Field: "play_count", Operator: "at_least", Value: "10"},
	}
	if !smartTrackMatches(track, SmartPlaylistRules{Match: "all", Conditions: conditions}, false, 0) {
		t.Fatal("expected track to match all conditions")
	}
	conditions[1].Value = "100"
	if smartTrackMatches(track, SmartPlaylistRules{Match: "all", Conditions: conditions}, false, 0) {
		t.Fatal("expected track not to match all conditions")
	}
	if !smartTrackMatches(track, SmartPlaylistRules{Match: "any", Conditions: conditions}, false, 0) {
		t.Fatal("expected track to match one condition")
	}
}

func TestSortSmartTracks(t *testing.T) {
	tracks := []Music{
		{ID: "older", Title: "A", CreatedAt: time.Unix(100, 0)},
		{ID: "newer", Title: "B", CreatedAt: time.Unix(200, 0)},
	}
	sortSmartTracks(tracks, SmartPlaylistRules{SortBy: "date_added", SortDirection: "desc"}, nil)
	if tracks[0].ID != "newer" {
		t.Fatalf("descending date sort started with %q", tracks[0].ID)
	}
	sortSmartTracks(tracks, SmartPlaylistRules{SortBy: "title", SortDirection: "asc"}, nil)
	if tracks[0].Title != "A" {
		t.Fatalf("ascending title sort started with %q", tracks[0].Title)
	}
}

func TestSmartPlaylistNestedGroupsAndRelativeDates(t *testing.T) {
	rules := SmartPlaylistRules{
		Match: "all", SortBy: "date_added", SortDirection: "desc", Limit: 100,
		Groups: []SmartPlaylistGroup{{
			Match: "any",
			Conditions: []SmartPlaylistCondition{
				{Field: "genre", Operator: "contains", Value: "ambient"},
				{Field: "date_added", Operator: "within_last_days", Value: "30"},
			},
			Groups: []SmartPlaylistGroup{{
				Match: "all",
				Conditions: []SmartPlaylistCondition{
					{Field: "liked", Operator: "is_true"},
					{Field: "rating", Operator: "at_least", Value: "4"},
				},
			}},
		}},
	}
	if err := ValidateSmartPlaylistRules(&rules); err != nil {
		t.Fatalf("nested rules were rejected: %v", err)
	}
	recent := Music{Genre: "Rock", CreatedAt: time.Now().Add(-10 * 24 * time.Hour)}
	if !smartTrackMatches(recent, rules, false, 0) {
		t.Fatal("recent track should match relative-date branch")
	}
	oldLiked := Music{Genre: "Rock", CreatedAt: time.Now().Add(-90 * 24 * time.Hour)}
	if !smartTrackMatches(oldLiked, rules, true, 5) {
		t.Fatal("liked, highly-rated track should match nested branch")
	}
	if smartTrackMatches(oldLiked, rules, false, 2) {
		t.Fatal("old unliked track should not match")
	}
}

func TestSmartPlaylistRejectsExcessiveNesting(t *testing.T) {
	rules := SmartPlaylistRules{
		Groups: []SmartPlaylistGroup{{Groups: []SmartPlaylistGroup{{Groups: []SmartPlaylistGroup{{Groups: []SmartPlaylistGroup{{}}}}}}}},
	}
	if err := ValidateSmartPlaylistRules(&rules); err == nil {
		t.Fatal("expected excessive nesting to be rejected")
	}
}
