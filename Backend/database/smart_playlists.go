package database

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	PlaylistTypeManual = "manual"
	PlaylistTypeSmart  = "smart"
)

func (db *DB) touchSmartPlaylists(userID string) error {
	query := `UPDATE playlists SET updated_at = CURRENT_TIMESTAMP WHERE playlist_type = $1`
	args := []interface{}{PlaylistTypeSmart}
	if userID != "" {
		query += ` AND user_id = $2`
		args = append(args, userID)
	}
	_, err := db.conn.Exec(query, args...)
	return err
}

func ValidateSmartPlaylistRules(rules *SmartPlaylistRules) error {
	if rules == nil {
		return fmt.Errorf("smart playlist rules are required")
	}
	if rules.Match == "" {
		rules.Match = "all"
	}
	if rules.Match != "all" && rules.Match != "any" {
		return fmt.Errorf("match must be all or any")
	}
	totalConditions := len(rules.Conditions)
	for index := range rules.Groups {
		count, err := validateSmartGroup(&rules.Groups[index], 1)
		if err != nil {
			return fmt.Errorf("group %d: %v", index+1, err)
		}
		totalConditions += count
	}
	if totalConditions > 50 {
		return fmt.Errorf("smart playlists support up to 50 conditions")
	}
	if rules.SortBy == "" {
		rules.SortBy = "date_added"
	}
	validSortFields := map[string]bool{
		"title": true, "artist": true, "album": true, "genre": true, "year": true,
		"duration": true, "date_added": true, "play_count": true, "rating": true, "random": true,
	}
	if !validSortFields[rules.SortBy] {
		return fmt.Errorf("unsupported sort field %q", rules.SortBy)
	}
	if rules.SortDirection == "" {
		rules.SortDirection = "desc"
	}
	if rules.SortDirection != "asc" && rules.SortDirection != "desc" {
		return fmt.Errorf("sort direction must be asc or desc")
	}
	if rules.Limit == 0 {
		rules.Limit = 100
	}
	if rules.Limit < 1 || rules.Limit > 500 {
		return fmt.Errorf("limit must be between 1 and 500")
	}

	for index := range rules.Conditions {
		condition := &rules.Conditions[index]
		if err := validateSmartCondition(condition); err != nil {
			return fmt.Errorf("condition %d: %v", index+1, err)
		}
	}
	return nil
}

func validateSmartGroup(group *SmartPlaylistGroup, depth int) (int, error) {
	if depth > 3 {
		return 0, fmt.Errorf("rule groups can be nested up to 3 levels")
	}
	if group.Match == "" {
		group.Match = "all"
	}
	if group.Match != "all" && group.Match != "any" {
		return 0, fmt.Errorf("match must be all or any")
	}
	count := len(group.Conditions)
	for index := range group.Conditions {
		if err := validateSmartCondition(&group.Conditions[index]); err != nil {
			return 0, fmt.Errorf("condition %d: %v", index+1, err)
		}
	}
	for index := range group.Groups {
		nestedCount, err := validateSmartGroup(&group.Groups[index], depth+1)
		if err != nil {
			return 0, fmt.Errorf("group %d: %v", index+1, err)
		}
		count += nestedCount
	}
	return count, nil
}

func validateSmartCondition(condition *SmartPlaylistCondition) error {
	stringFields := map[string]bool{"title": true, "artist": true, "album": true, "genre": true}
	numberFields := map[string]bool{"year": true, "duration": true, "play_count": true, "rating": true}
	booleanFields := map[string]bool{"liked": true, "has_artwork": true}

	switch {
	case stringFields[condition.Field]:
		if !map[string]bool{"contains": true, "not_contains": true, "equals": true, "not_equals": true}[condition.Operator] {
			return fmt.Errorf("unsupported text operator %q", condition.Operator)
		}
		if strings.TrimSpace(condition.Value) == "" {
			return fmt.Errorf("a value is required")
		}
	case numberFields[condition.Field]:
		if !map[string]bool{"equals": true, "not_equals": true, "greater_than": true, "at_least": true, "less_than": true, "at_most": true}[condition.Operator] {
			return fmt.Errorf("unsupported number operator %q", condition.Operator)
		}
		if _, err := strconv.ParseFloat(strings.TrimSpace(condition.Value), 64); err != nil {
			return fmt.Errorf("value must be a number")
		}
	case condition.Field == "date_added":
		if condition.Operator == "within_last_days" || condition.Operator == "not_within_last_days" {
			days, err := strconv.Atoi(strings.TrimSpace(condition.Value))
			if err != nil || days < 1 || days > 36500 {
				return fmt.Errorf("relative date must be between 1 and 36500 days")
			}
			return nil
		}
		if condition.Operator != "before" && condition.Operator != "after" {
			return fmt.Errorf("date operator must be before, after, or within a number of days")
		}
		if _, err := parseSmartDate(condition.Value); err != nil {
			return fmt.Errorf("value must be a date")
		}
	case booleanFields[condition.Field]:
		if condition.Operator != "is_true" && condition.Operator != "is_false" {
			return fmt.Errorf("boolean operator must be is_true or is_false")
		}
	default:
		return fmt.Errorf("unsupported field %q", condition.Field)
	}
	return nil
}

func (db *DB) EvaluateSmartPlaylist(userID string, rules SmartPlaylistRules) ([]Music, error) {
	if err := ValidateSmartPlaylistRules(&rules); err != nil {
		return nil, err
	}
	tracks, err := db.GetAllMusic()
	if err != nil {
		return nil, err
	}
	likedTracks, err := db.GetLikedTracks(userID)
	if err != nil {
		return nil, err
	}
	liked := make(map[string]bool, len(likedTracks))
	for _, track := range likedTracks {
		liked[track.ID] = true
	}
	ratings, err := db.GetMediaRatings(userID)
	if err != nil {
		return nil, err
	}

	matches := make([]Music, 0, len(tracks))
	for _, track := range tracks {
		if smartTrackMatches(track, rules, liked[track.ID], ratings[track.ID]) {
			matches = append(matches, track)
		}
	}
	sortSmartTracks(matches, rules, ratings)
	if len(matches) > rules.Limit {
		matches = matches[:rules.Limit]
	}
	return matches, nil
}

func smartTrackMatches(track Music, rules SmartPlaylistRules, liked bool, rating int) bool {
	if len(rules.Conditions) == 0 && len(rules.Groups) == 0 {
		return true
	}
	group := SmartPlaylistGroup{Match: rules.Match, Conditions: rules.Conditions, Groups: rules.Groups}
	return smartGroupMatches(track, group, liked, rating)
}

func smartGroupMatches(track Music, group SmartPlaylistGroup, liked bool, rating int) bool {
	results := make([]bool, 0, len(group.Conditions)+len(group.Groups))
	for _, condition := range group.Conditions {
		results = append(results, smartConditionMatches(track, condition, liked, rating))
	}
	for _, nested := range group.Groups {
		results = append(results, smartGroupMatches(track, nested, liked, rating))
	}
	if len(results) == 0 {
		return true
	}
	if group.Match == "any" {
		for _, matched := range results {
			if matched {
				return true
			}
		}
		return false
	}
	for _, matched := range results {
		if !matched {
			return false
		}
	}
	return true
}

func smartConditionMatches(track Music, condition SmartPlaylistCondition, liked bool, rating int) bool {
	var text string
	switch condition.Field {
	case "title":
		text = track.Title
	case "artist":
		text = track.Artist
	case "album":
		text = track.Album
	case "genre":
		text = track.Genre
	}
	if text != "" || map[string]bool{"title": true, "artist": true, "album": true, "genre": true}[condition.Field] {
		left := strings.ToLower(strings.TrimSpace(text))
		right := strings.ToLower(strings.TrimSpace(condition.Value))
		switch condition.Operator {
		case "contains":
			return strings.Contains(left, right)
		case "not_contains":
			return !strings.Contains(left, right)
		case "equals":
			return left == right
		case "not_equals":
			return left != right
		}
	}

	if condition.Field == "liked" || condition.Field == "has_artwork" {
		value := liked
		if condition.Field == "has_artwork" {
			value = firstArtwork(track) != ""
		}
		return value == (condition.Operator == "is_true")
	}
	if condition.Field == "date_added" {
		if condition.Operator == "within_last_days" || condition.Operator == "not_within_last_days" {
			days, err := strconv.Atoi(strings.TrimSpace(condition.Value))
			if err != nil {
				return false
			}
			matched := !track.CreatedAt.Before(time.Now().AddDate(0, 0, -days))
			if condition.Operator == "not_within_last_days" {
				return !matched
			}
			return matched
		}
		value, err := parseSmartDate(condition.Value)
		if err != nil {
			return false
		}
		if condition.Operator == "before" {
			return track.CreatedAt.Before(value)
		}
		return track.CreatedAt.After(value)
	}

	var left float64
	switch condition.Field {
	case "year":
		left = float64(track.Year)
	case "duration":
		left = float64(track.Duration)
	case "play_count":
		left = float64(track.PlayCount)
	case "rating":
		left = float64(rating)
	}
	right, _ := strconv.ParseFloat(strings.TrimSpace(condition.Value), 64)
	switch condition.Operator {
	case "equals":
		return left == right
	case "not_equals":
		return left != right
	case "greater_than":
		return left > right
	case "at_least":
		return left >= right
	case "less_than":
		return left < right
	case "at_most":
		return left <= right
	}
	return false
}

func sortSmartTracks(tracks []Music, rules SmartPlaylistRules, ratings map[string]int) {
	if rules.SortBy == "random" {
		rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(tracks), func(i, j int) {
			tracks[i], tracks[j] = tracks[j], tracks[i]
		})
		return
	}
	sort.SliceStable(tracks, func(i, j int) bool {
		left, right := tracks[i], tracks[j]
		less := false
		switch rules.SortBy {
		case "title":
			less = strings.ToLower(left.Title) < strings.ToLower(right.Title)
		case "artist":
			less = strings.ToLower(left.Artist) < strings.ToLower(right.Artist)
		case "album":
			less = strings.ToLower(left.Album) < strings.ToLower(right.Album)
		case "genre":
			less = strings.ToLower(left.Genre) < strings.ToLower(right.Genre)
		case "year":
			less = left.Year < right.Year
		case "duration":
			less = left.Duration < right.Duration
		case "play_count":
			less = left.PlayCount < right.PlayCount
		case "rating":
			less = ratings[left.ID] < ratings[right.ID]
		default:
			less = left.CreatedAt.Before(right.CreatedAt)
		}
		if rules.SortDirection == "desc" {
			return !less && !smartSortEqual(left, right, rules.SortBy, ratings)
		}
		return less
	})
}

func smartSortEqual(left, right Music, field string, ratings map[string]int) bool {
	switch field {
	case "title":
		return strings.EqualFold(left.Title, right.Title)
	case "artist":
		return strings.EqualFold(left.Artist, right.Artist)
	case "album":
		return strings.EqualFold(left.Album, right.Album)
	case "genre":
		return strings.EqualFold(left.Genre, right.Genre)
	case "year":
		return left.Year == right.Year
	case "duration":
		return left.Duration == right.Duration
	case "play_count":
		return left.PlayCount == right.PlayCount
	case "rating":
		return ratings[left.ID] == ratings[right.ID]
	default:
		return left.CreatedAt.Equal(right.CreatedAt)
	}
}

func parseSmartDate(value string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}

func firstArtwork(track Music) string {
	for _, value := range []string{track.ImageURL, track.CoverArtURL, track.CoverArtLargeURL, track.CoverArtMediumURL, track.CoverArtSmallURL} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
