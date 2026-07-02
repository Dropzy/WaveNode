package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"music-server/database"
)

const (
	listenBrainzBaseURL             = "https://api.listenbrainz.org/1"
	discoveryDefaultSource          = "created-for-you"
	discoveryWeeklyExploration      = "weekly-exploration"
	discoveryWeeklyJams             = "weekly-jams"
	discoveryDailyJams              = "daily-jams"
	discoveryListenBrainzSettingKey = "discovery.listenbrainz_user."
)

var discoveryTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

var (
	errDiscoveryUsernameRequired = errors.New("Set a ListenBrainz username first")
	errDiscoveryUserNotFound     = errors.New("ListenBrainz user was not found")
	errDiscoveryNoPlaylists      = errors.New("ListenBrainz has not generated any discovery playlists for this user yet")
	errDiscoveryNoMatches        = errors.New("None of the recommended tracks are currently in your WaveNode library")
)

type discoverySettings struct {
	ListenBrainzUser string `json:"listenbrainz_user"`
}

type discoveryRecommendation struct {
	Title          string `json:"title"`
	Artist         string `json:"artist"`
	Album          string `json:"album"`
	SourcePlaylist string `json:"source_playlist"`
	MatchedTrackID string `json:"matched_track_id,omitempty"`
}

type discoveryPreview struct {
	Source          string                    `json:"source"`
	ListenBrainzURL string                    `json:"listenbrainz_url"`
	Total           int                       `json:"total"`
	Matched         []database.Music          `json:"matched"`
	Missing         []discoveryRecommendation `json:"missing"`
	Recommendations []discoveryRecommendation `json:"recommendations"`
}

type discoveryImportRequest struct {
	Source       string `json:"source"`
	PlaylistName string `json:"playlist_name"`
}

type listenBrainzPlaylistList struct {
	Playlists []listenBrainzPlaylistRef `json:"playlists"`
}

type listenBrainzPlaylistRef struct {
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	Date       string `json:"date"`
}

type listenBrainzPlaylistEnvelope struct {
	Playlist listenBrainzPlaylist `json:"playlist"`
}

type listenBrainzPlaylist struct {
	Title string                   `json:"title"`
	Track []listenBrainzTrackEntry `json:"track"`
}

type listenBrainzTrackEntry struct {
	Title      string                 `json:"title"`
	Creator    string                 `json:"creator"`
	Album      string                 `json:"album"`
	Extension  map[string]interface{} `json:"extension"`
	Additional map[string]interface{} `json:"additional_metadata"`
}

func (r *Router) getDiscoverySettings(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}

	username, err := r.db.GetSetting(discoveryListenBrainzSettingKey + userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": discoverySettings{
			ListenBrainzUser: username,
		},
	})
}

func (r *Router) updateDiscoverySettings(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}

	var settings discoverySettings
	if err := json.NewDecoder(req.Body).Decode(&settings); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid discovery settings")
		return
	}

	settings.ListenBrainzUser = strings.TrimSpace(settings.ListenBrainzUser)
	if strings.ContainsAny(settings.ListenBrainzUser, "/?#") {
		writeJSONError(w, http.StatusBadRequest, "ListenBrainz username is invalid")
		return
	}

	if err := r.db.SetSetting(discoveryListenBrainzSettingKey+userID, settings.ListenBrainzUser); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": settings})
}

func (r *Router) previewDiscovery(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}

	source := discoverySource(req.URL.Query().Get("source"))
	preview, err := r.buildDiscoveryPreview(userID, source)
	if err != nil {
		writeJSONError(w, discoveryHTTPStatus(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": preview})
}

func (r *Router) importDiscoveryPlaylist(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}

	var payload discoveryImportRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid discovery import request")
		return
	}

	source := discoverySource(payload.Source)
	preview, err := r.buildDiscoveryPreview(userID, source)
	if err != nil {
		writeJSONError(w, discoveryHTTPStatus(err), err.Error())
		return
	}
	if len(preview.Matched) == 0 {
		writeJSONError(w, http.StatusNotFound, errDiscoveryNoMatches.Error())
		return
	}

	trackIDs := make([]string, 0, len(preview.Matched))
	for _, track := range preview.Matched {
		trackIDs = append(trackIDs, track.ID)
	}

	playlistName := strings.TrimSpace(payload.PlaylistName)
	if playlistName == "" {
		playlistName = discoveryPlaylistName(source)
	}

	playlist := &database.Playlist{
		UserID:      userID,
		Name:        playlistName,
		Description: fmt.Sprintf("Imported from ListenBrainz. %d matched, %d missing.", len(preview.Matched), len(preview.Missing)),
		Type:        database.PlaylistTypeManual,
		TrackIDs:    trackIDs,
	}
	if err := r.db.AddPlaylist(playlist); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"playlist": playlist,
			"preview":  preview,
		},
	})
}

func (r *Router) buildDiscoveryPreview(userID, source string) (*discoveryPreview, error) {
	username, err := r.db.GetSetting(discoveryListenBrainzSettingKey + userID)
	if err != nil {
		return nil, err
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errDiscoveryUsernameRequired
	}

	recommendations, playlistURL, err := fetchListenBrainzRecommendations(username, source)
	if err != nil {
		return nil, err
	}

	tracks, err := r.db.GetAllMusic()
	if err != nil {
		return nil, err
	}
	tracks, err = r.db.FilterMusicForUser(userID, tracks)
	if err != nil {
		return nil, err
	}

	matches, missing := matchDiscoveryRecommendations(recommendations, tracks)
	return &discoveryPreview{
		Source:          source,
		ListenBrainzURL: playlistURL,
		Total:           len(recommendations),
		Matched:         matches,
		Missing:         missing,
		Recommendations: recommendations,
	}, nil
}

func fetchListenBrainzRecommendations(username, source string) ([]discoveryRecommendation, string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	endpoint := fmt.Sprintf("%s/user/%s/playlists/createdfor", listenBrainzBaseURL, url.PathEscape(username))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "WaveNode/1.0 (+https://github.com/)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("Could not contact ListenBrainz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, "", errDiscoveryUserNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("ListenBrainz returned %s", resp.Status)
	}

	var list listenBrainzPlaylistList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, "", fmt.Errorf("Could not read ListenBrainz playlists: %v", err)
	}
	if len(list.Playlists) == 0 {
		return nil, "", errDiscoveryNoPlaylists
	}

	ref := chooseListenBrainzPlaylist(list.Playlists, source)
	if ref.Identifier == "" {
		return nil, "", fmt.Errorf("No matching ListenBrainz playlist was found")
	}

	playlistID := listenBrainzPlaylistID(ref.Identifier)
	if playlistID == "" {
		return nil, "", fmt.Errorf("ListenBrainz playlist did not include a usable identifier")
	}

	playlistEndpoint := fmt.Sprintf("%s/playlist/%s", listenBrainzBaseURL, url.PathEscape(playlistID))
	playlistReq, err := http.NewRequest(http.MethodGet, playlistEndpoint, nil)
	if err != nil {
		return nil, "", err
	}
	playlistReq.Header.Set("Accept", "application/json")
	playlistReq.Header.Set("User-Agent", "WaveNode/1.0 (+https://github.com/)")

	playlistResp, err := client.Do(playlistReq)
	if err != nil {
		return nil, "", fmt.Errorf("Could not load ListenBrainz playlist: %v", err)
	}
	defer playlistResp.Body.Close()
	if playlistResp.StatusCode < 200 || playlistResp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("ListenBrainz playlist returned %s", playlistResp.Status)
	}

	var envelope listenBrainzPlaylistEnvelope
	if err := json.NewDecoder(playlistResp.Body).Decode(&envelope); err != nil {
		return nil, "", fmt.Errorf("Could not read ListenBrainz playlist: %v", err)
	}

	recommendations := make([]discoveryRecommendation, 0, len(envelope.Playlist.Track))
	seen := make(map[string]struct{})
	playlistTitle := firstNonEmptyDiscovery(envelope.Playlist.Title, ref.Title)
	for _, entry := range envelope.Playlist.Track {
		title := strings.TrimSpace(entry.Title)
		artist := strings.TrimSpace(entry.Creator)
		album := strings.TrimSpace(entry.Album)
		if title == "" || artist == "" {
			title, artist, album = listenBrainzTrackMetadata(entry, title, artist, album)
		}
		if title == "" || artist == "" {
			continue
		}
		key := normalizeDiscoveryText(artist) + "|" + normalizeDiscoveryText(title)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		recommendations = append(recommendations, discoveryRecommendation{
			Title:          title,
			Artist:         artist,
			Album:          album,
			SourcePlaylist: playlistTitle,
		})
	}
	if len(recommendations) == 0 {
		return nil, "", fmt.Errorf("ListenBrainz playlist did not contain usable track recommendations")
	}

	return recommendations, ref.Identifier, nil
}

func chooseListenBrainzPlaylist(playlists []listenBrainzPlaylistRef, source string) listenBrainzPlaylistRef {
	preferred := discoverySource(source)
	fallbacks := map[string][]string{
		discoveryWeeklyExploration: {"weekly exploration", "exploration"},
		discoveryWeeklyJams:        {"weekly jams", "jams"},
		discoveryDailyJams:         {"daily jams"},
		discoveryDefaultSource:     {"weekly exploration", "weekly jams", "daily jams", "exploration", "jams"},
	}
	needles := fallbacks[preferred]
	if len(needles) == 0 {
		needles = fallbacks[discoveryDefaultSource]
	}

	sort.SliceStable(playlists, func(i, j int) bool {
		return playlists[i].Date > playlists[j].Date
	})

	for _, needle := range needles {
		for _, playlist := range playlists {
			if strings.Contains(strings.ToLower(playlist.Title), needle) {
				return playlist
			}
		}
	}
	return playlists[0]
}

func listenBrainzPlaylistID(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}
	parsed, err := url.Parse(identifier)
	if err == nil && parsed.Path != "" {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return strings.Trim(identifier, "/")
}

func listenBrainzTrackMetadata(entry listenBrainzTrackEntry, title, artist, album string) (string, string, string) {
	readString := func(value interface{}) string {
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed)
		case []interface{}:
			parts := make([]string, 0, len(typed))
			for _, item := range typed {
				if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
					parts = append(parts, strings.TrimSpace(text))
				}
			}
			return strings.Join(parts, ", ")
		default:
			return ""
		}
	}
	for _, source := range []map[string]interface{}{entry.Additional, entry.Extension} {
		if source == nil {
			continue
		}
		title = firstNonEmptyDiscovery(title, readString(source["track_name"]), readString(source["recording_name"]), readString(source["title"]))
		artist = firstNonEmptyDiscovery(artist, readString(source["artist_name"]), readString(source["artist_credit_name"]), readString(source["creator"]))
		album = firstNonEmptyDiscovery(album, readString(source["release_name"]), readString(source["album"]))
	}
	return title, artist, album
}

func matchDiscoveryRecommendations(recommendations []discoveryRecommendation, tracks []database.Music) ([]database.Music, []discoveryRecommendation) {
	exact := make(map[string]database.Music)
	titleOnly := make(map[string][]database.Music)
	for _, track := range tracks {
		titleKey := normalizeDiscoveryText(track.Title)
		artistKey := normalizeDiscoveryText(track.Artist)
		if titleKey == "" || artistKey == "" {
			continue
		}
		key := artistKey + "|" + titleKey
		if _, exists := exact[key]; !exists {
			exact[key] = track
		}
		titleOnly[titleKey] = append(titleOnly[titleKey], track)
	}

	matched := make([]database.Music, 0)
	missing := make([]discoveryRecommendation, 0)
	usedTracks := make(map[string]struct{})
	for _, recommendation := range recommendations {
		titleKey := normalizeDiscoveryText(recommendation.Title)
		artistKey := normalizeDiscoveryText(recommendation.Artist)
		var track database.Music
		found := false

		if candidate, exists := exact[artistKey+"|"+titleKey]; exists {
			track = candidate
			found = true
		} else if candidates := titleOnly[titleKey]; len(candidates) == 1 {
			track = candidates[0]
			found = true
		}

		if !found {
			missing = append(missing, recommendation)
			continue
		}
		if _, exists := usedTracks[track.ID]; exists {
			continue
		}
		usedTracks[track.ID] = struct{}{}
		recommendation.MatchedTrackID = track.ID
		matched = append(matched, track)
	}
	return matched, missing
}

func normalizeDiscoveryText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	tokens := discoveryTokenPattern.FindAllString(value, -1)
	return strings.Join(tokens, "")
}

func discoverySource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case discoveryWeeklyExploration, discoveryWeeklyJams, discoveryDailyJams:
		return source
	default:
		return discoveryDefaultSource
	}
}

func discoveryPlaylistName(source string) string {
	switch discoverySource(source) {
	case discoveryWeeklyJams:
		return "ListenBrainz Weekly Jams"
	case discoveryDailyJams:
		return "ListenBrainz Daily Jams"
	default:
		return "ListenBrainz Weekly Exploration"
	}
}

func firstNonEmptyDiscovery(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func discoveryHTTPStatus(err error) int {
	switch {
	case errors.Is(err, errDiscoveryUsernameRequired):
		return http.StatusBadRequest
	case errors.Is(err, errDiscoveryUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, errDiscoveryNoPlaylists), errors.Is(err, errDiscoveryNoMatches):
		return http.StatusConflict
	default:
		return http.StatusBadGateway
	}
}
