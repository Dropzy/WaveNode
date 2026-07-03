package router

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"music-server/database"

	"github.com/gorilla/mux"
)

func requestUserID(req *http.Request) string {
	userID, _ := req.Context().Value("user_id").(string)
	return userID
}

type playbackHandoffRequest struct {
	TargetSessionID string                 `json:"target_session_id"`
	TrackIDs        []string               `json:"track_ids"`
	Tracks          []playbackHandoffTrack `json:"tracks"`
	StartIndex      int                    `json:"start_index"`
	Action          string                 `json:"action"`
	PositionMs      int64                  `json:"position_ms"`
}

type playbackHandoffCommand struct {
	ID              string                 `json:"id"`
	SourceSessionID string                 `json:"source_session_id"`
	TargetSessionID string                 `json:"target_session_id"`
	TrackIDs        []string               `json:"track_ids"`
	Tracks          []playbackHandoffTrack `json:"tracks,omitempty"`
	StartIndex      int                    `json:"start_index"`
	Action          string                 `json:"action"`
	PositionMs      int64                  `json:"position_ms,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

type playbackHandoffTrack struct {
	ID                     string  `json:"id"`
	Title                  string  `json:"title"`
	Artist                 string  `json:"artist"`
	Album                  string  `json:"album"`
	Genre                  string  `json:"genre"`
	Duration               int     `json:"duration"`
	TrackNumber            int     `json:"track_number,omitempty"`
	DiscNumber             int     `json:"disc_number,omitempty"`
	DiscTotal              int     `json:"disc_total,omitempty"`
	ReplayGainTrackDB      float64 `json:"replaygain_track_db,omitempty"`
	ReplayGainAlbumDB      float64 `json:"replaygain_album_db,omitempty"`
	ReleaseDate            string  `json:"release_date,omitempty"`
	FilePath               string  `json:"file_path,omitempty"`
	FileName               string  `json:"file_name,omitempty"`
	Format                 string  `json:"format,omitempty"`
	ImageURL               string  `json:"image_url,omitempty"`
	CoverArtURL            string  `json:"cover_art_url,omitempty"`
	CoverArtSmallURL       string  `json:"cover_art_small_url,omitempty"`
	CoverArtMediumURL      string  `json:"cover_art_medium_url,omitempty"`
	CoverArtLargeURL       string  `json:"cover_art_large_url,omitempty"`
	StreamURL              string  `json:"stream_url,omitempty"`
	IsExternal             bool    `json:"is_external,omitempty"`
	ExternalKind           string  `json:"external_kind,omitempty"`
	RadioStationID         string  `json:"radio_station_id,omitempty"`
	PodcastID              string  `json:"podcast_id,omitempty"`
	PodcastTitle           string  `json:"podcast_title,omitempty"`
	PodcastPublisher       string  `json:"podcast_publisher,omitempty"`
	PodcastEpisodeID       string  `json:"podcast_episode_id,omitempty"`
	PodcastDescription     string  `json:"podcast_description,omitempty"`
	PodcastWebsiteURL      string  `json:"podcast_website_url,omitempty"`
	PodcastChaptersURL     string  `json:"podcast_chapters_url,omitempty"`
	PodcastChaptersType    string  `json:"podcast_chapters_type,omitempty"`
	PodcastAudioURL        string  `json:"podcast_audio_url,omitempty"`
	PodcastProgressSeconds int     `json:"podcast_progress_seconds,omitempty"`
	PodcastCompleted       bool    `json:"podcast_completed,omitempty"`
	AudiobookID            string  `json:"audiobook_id,omitempty"`
	AudiobookTitle         string  `json:"audiobook_title,omitempty"`
	AudiobookAuthor        string  `json:"audiobook_author,omitempty"`
	AudiobookChapterID     string  `json:"audiobook_chapter_id,omitempty"`
	AudiobookChapterNumber int     `json:"audiobook_chapter_number,omitempty"`
	AudiobookDescription   string  `json:"audiobook_description,omitempty"`
	AudiobookWebsiteURL    string  `json:"audiobook_website_url,omitempty"`
	AudiobookProgress      int     `json:"audiobook_progress_seconds,omitempty"`
	AudiobookCompleted     bool    `json:"audiobook_completed,omitempty"`
	UploadOrder            int64   `json:"upload_order,omitempty"`
	CreatedAt              string  `json:"created_at,omitempty"`
	UpdatedAt              string  `json:"updated_at,omitempty"`
}

const maxPlaybackHandoffTracks = 500

func playbackTrackFromMusic(track database.Music) playbackHandoffTrack {
	releaseDate := ""
	if track.ReleaseDate != nil {
		releaseDate = track.ReleaseDate.Format(time.RFC3339)
	}
	return playbackHandoffTrack{
		ID: track.ID, Title: track.Title, Artist: track.Artist, Album: track.Album, Genre: track.Genre,
		Duration: track.Duration, TrackNumber: track.TrackNumber, DiscNumber: track.DiscNumber, DiscTotal: track.DiscTotal,
		ReplayGainTrackDB: track.ReplayGainTrackDB, ReplayGainAlbumDB: track.ReplayGainAlbumDB,
		ReleaseDate: releaseDate, FilePath: track.FilePath, FileName: track.FileName,
		Format: track.Format, ImageURL: track.ImageURL, CoverArtURL: track.CoverArtURL,
		CoverArtSmallURL: track.CoverArtSmallURL, CoverArtMediumURL: track.CoverArtMediumURL,
		CoverArtLargeURL: track.CoverArtLargeURL, UploadOrder: track.UploadOrder,
		CreatedAt: track.CreatedAt.Format(time.RFC3339), UpdatedAt: track.UpdatedAt.Format(time.RFC3339),
	}
}

func validateExternalHandoffTrack(track playbackHandoffTrack) (playbackHandoffTrack, error) {
	track.ID = strings.TrimSpace(track.ID)
	track.Title = strings.TrimSpace(track.Title)
	track.ExternalKind = strings.ToLower(strings.TrimSpace(track.ExternalKind))
	if track.ID == "" || len(track.ID) > 512 || track.Title == "" || len(track.Title) > 1000 {
		return playbackHandoffTrack{}, fmt.Errorf("external item has invalid identity")
	}
	if track.ExternalKind != "podcast" && track.ExternalKind != "audiobook" && track.ExternalKind != "radio" {
		return playbackHandoffTrack{}, fmt.Errorf("unsupported external playback type")
	}
	streamURL := strings.TrimSpace(track.StreamURL)
	if track.ExternalKind == "podcast" && !strings.HasPrefix(streamURL, "https://") {
		streamURL = strings.TrimSpace(track.PodcastAudioURL)
	}
	parsed, err := url.Parse(streamURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || len(streamURL) > 4096 {
		return playbackHandoffTrack{}, fmt.Errorf("external item requires a secure stream URL")
	}
	track.StreamURL = parsed.String()
	track.PodcastAudioURL = track.StreamURL
	track.IsExternal = true
	track.FilePath = ""
	track.Duration = max(0, track.Duration)
	track.PodcastProgressSeconds = max(0, track.PodcastProgressSeconds)
	track.AudiobookProgress = max(0, track.AudiobookProgress)
	for _, field := range []struct {
		value *string
		limit int
	}{
		{&track.Artist, 1000}, {&track.Album, 1000}, {&track.Genre, 500},
		{&track.PodcastID, 512}, {&track.PodcastTitle, 1000}, {&track.PodcastPublisher, 1000},
		{&track.PodcastEpisodeID, 512}, {&track.PodcastDescription, 10000},
		{&track.AudiobookID, 512}, {&track.AudiobookTitle, 1000}, {&track.AudiobookAuthor, 1000},
		{&track.AudiobookChapterID, 512}, {&track.AudiobookDescription, 10000}, {&track.RadioStationID, 128},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if len(*field.value) > field.limit {
			*field.value = (*field.value)[:field.limit]
		}
	}
	track.ImageURL = secureOptionalURL(track.ImageURL)
	track.CoverArtURL = secureOptionalURL(track.CoverArtURL)
	track.CoverArtSmallURL = secureOptionalURL(track.CoverArtSmallURL)
	track.CoverArtMediumURL = secureOptionalURL(track.CoverArtMediumURL)
	track.CoverArtLargeURL = secureOptionalURL(track.CoverArtLargeURL)
	track.PodcastWebsiteURL = secureOptionalURL(track.PodcastWebsiteURL)
	track.PodcastChaptersURL = secureOptionalURL(track.PodcastChaptersURL)
	track.AudiobookWebsiteURL = secureOptionalURL(track.AudiobookWebsiteURL)
	return track, nil
}

func (r *Router) getPlaybackProfile(w http.ResponseWriter, req *http.Request) {
	profile, err := r.db.GetPlaybackProfile(requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": profile})
}

func (r *Router) updatePlaybackProfile(w http.ResponseWriter, req *http.Request) {
	profile := database.DefaultPlaybackProfile(requestUserID(req))
	if err := json.NewDecoder(req.Body).Decode(&profile); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid playback profile")
		return
	}
	profile.UserID = requestUserID(req)
	if err := r.db.SavePlaybackProfile(profile); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	saved, _ := r.db.GetPlaybackProfile(profile.UserID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": saved})
}

func (r *Router) createPlaybackHandoff(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	sourceSessionID, _ := req.Context().Value("session_id").(string)
	var payload playbackHandoffRequest
	req.Body = http.MaxBytesReader(w, req.Body, 8<<20)
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid playback handoff request")
		return
	}
	payload.TargetSessionID = strings.TrimSpace(payload.TargetSessionID)
	if payload.TargetSessionID == "" || payload.TargetSessionID == sourceSessionID {
		writeJSONError(w, http.StatusBadRequest, "Select another WaveNode device")
		return
	}
	payload.Action = strings.TrimSpace(payload.Action)
	if payload.Action == "" {
		payload.Action = "play_queue"
	}
	if payload.Action != "play_queue" && payload.Action != "toggle_play_pause" && payload.Action != "seek" && payload.Action != "stop" {
		writeJSONError(w, http.StatusBadRequest, "Unsupported playback command")
		return
	}
	if payload.Action == "play_queue" {
		if len(payload.Tracks) == 0 {
			payload.Tracks = make([]playbackHandoffTrack, 0, len(payload.TrackIDs))
			for _, trackID := range payload.TrackIDs {
				payload.Tracks = append(payload.Tracks, playbackHandoffTrack{ID: trackID})
			}
		}
		if len(payload.Tracks) == 0 {
			writeJSONError(w, http.StatusBadRequest, "No tracks were provided")
			return
		}
		if len(payload.Tracks) > maxPlaybackHandoffTracks {
			writeJSONError(w, http.StatusBadRequest, "Playback queue is too large")
			return
		}
		if payload.StartIndex < 0 || payload.StartIndex >= len(payload.Tracks) {
			payload.StartIndex = 0
		}
	}
	sessions, err := r.db.GetUserSessions(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Could not load connected devices")
		return
	}
	targetActive := false
	activeCutoff := time.Now().Add(-15 * time.Minute)
	for _, session := range sessions {
		if session.ID == payload.TargetSessionID && session.LastSeenAt.After(activeCutoff) {
			targetActive = true
			break
		}
	}
	if !targetActive {
		writeJSONError(w, http.StatusNotFound, "That WaveNode device is not available")
		return
	}
	tracks := make([]playbackHandoffTrack, 0, len(payload.Tracks))
	if payload.Action == "play_queue" {
		localTracks := make([]database.Music, 0, len(payload.Tracks))
		for _, requested := range payload.Tracks {
			if requested.IsExternal {
				continue
			}
			track, trackErr := r.db.GetMusic(strings.TrimSpace(requested.ID))
			if trackErr == nil && track != nil {
				localTracks = append(localTracks, *track)
			}
		}
		localTracks, err = r.db.FilterMusicForUser(userID, localTracks)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Could not apply library permissions")
			return
		}
		allowedLocal := make(map[string]database.Music, len(localTracks))
		for _, track := range localTracks {
			allowedLocal[track.ID] = track
		}
		selectedID := payload.Tracks[payload.StartIndex].ID
		for _, requested := range payload.Tracks {
			if requested.IsExternal {
				external, externalErr := validateExternalHandoffTrack(requested)
				if externalErr != nil {
					continue
				}
				tracks = append(tracks, external)
				continue
			}
			if local, allowed := allowedLocal[strings.TrimSpace(requested.ID)]; allowed {
				tracks = append(tracks, playbackTrackFromMusic(local))
			}
		}
		if len(tracks) == 0 {
			writeJSONError(w, http.StatusBadRequest, "No playable items were provided")
			return
		}
		payload.StartIndex = 0
		for index, track := range tracks {
			if track.ID == selectedID {
				payload.StartIndex = index
				break
			}
		}
		payload.TrackIDs = make([]string, len(tracks))
		for index, track := range tracks {
			payload.TrackIDs[index] = track.ID
		}
	}
	command := playbackHandoffCommand{
		ID:              fmt.Sprintf("handoff_%d", time.Now().UnixNano()),
		SourceSessionID: sourceSessionID,
		TargetSessionID: payload.TargetSessionID,
		TrackIDs:        payload.TrackIDs,
		Tracks:          tracks,
		StartIndex:      payload.StartIndex,
		Action:          payload.Action,
		PositionMs:      payload.PositionMs,
		CreatedAt:       time.Now(),
	}
	r.playbackHandoffs.Store(payload.TargetSessionID, command)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": command})
}

func (r *Router) consumePlaybackHandoff(w http.ResponseWriter, req *http.Request) {
	sessionID, _ := req.Context().Value("session_id").(string)
	if sessionID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Session is required")
		return
	}
	value, ok := r.playbackHandoffs.LoadAndDelete(sessionID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": value})
}

func (r *Router) getListeningHistory(w http.ResponseWriter, req *http.Request) {
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(req.URL.Query().Get("offset"))
	entries, err := r.db.GetListeningHistory(requestUserID(req), req.URL.Query().Get("search"), limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	entries, err = r.filterListeningHistory(req, entries)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to apply library permissions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": entries})
}

func (r *Router) clearListeningHistory(w http.ResponseWriter, req *http.Request) {
	if err := r.db.ClearListeningHistory(requestUserID(req)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (r *Router) exportListeningHistory(w http.ResponseWriter, req *http.Request) {
	entries, err := r.db.GetListeningHistory(requestUserID(req), req.URL.Query().Get("search"), 500, 0)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	entries, err = r.filterListeningHistory(req, entries)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to apply library permissions")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="wavenode-listening-history.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"played_at", "title", "artist", "album", "source", "device"})
	for _, entry := range entries {
		_ = writer.Write([]string{
			entry.PlayedAt.Format(time.RFC3339), entry.Track.Title, entry.Track.Artist,
			entry.Track.Album, entry.Source, entry.Device,
		})
	}
	writer.Flush()
}

func (r *Router) exportPlaylistM3U(w http.ResponseWriter, req *http.Request) {
	playlist, err := r.db.GetUserPlaylist(mux.Vars(req)["id"], requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Playlist not found")
		return
	}
	tracks, err := r.db.GetPlaylistTracks(playlist.ID, requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tracks, err = r.filterMusicForRequest(req, tracks)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to apply library permissions")
		return
	}
	filename := strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(playlist.Name)
	w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.m3u8"`, filename))
	fmt.Fprintln(w, "#EXTM3U")
	fmt.Fprintf(w, "#PLAYLIST:%s\n", playlist.Name)
	for _, track := range tracks {
		fmt.Fprintf(w, "#EXTINF:%d,%s - %s\n", track.Duration, track.Artist, track.Title)
		fmt.Fprintf(w, "#WAVENODE:%s\n", track.ID)
		fmt.Fprintf(w, "/api/music/%s/stream\n", url.PathEscape(track.ID))
	}
}

func (r *Router) importPlaylistM3U(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseMultipartForm(8 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Upload a valid M3U playlist")
		return
	}
	file, header, err := req.FormFile("playlist")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Playlist file is required")
		return
	}
	defer file.Close()
	name := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	if supplied := strings.TrimSpace(req.FormValue("name")); supplied != "" {
		name = supplied
	}
	trackIDs, err := r.resolveM3UTrackIDs(file, requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	playlist := &database.Playlist{
		UserID: requestUserID(req), Name: name, Description: "Imported from M3U",
		Type: database.PlaylistTypeManual, TrackIDs: trackIDs,
	}
	if err := r.db.AddPlaylist(playlist); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"success": true, "data": playlist})
}

func (r *Router) resolveM3UTrackIDs(reader io.Reader, userIDs ...string) ([]string, error) {
	tracks, err := r.db.GetAllMusic()
	if err != nil {
		return nil, err
	}
	if len(userIDs) > 0 && userIDs[0] != "" {
		tracks, err = r.db.FilterMusicForUser(userIDs[0], tracks)
		if err != nil {
			return nil, err
		}
	}
	byPath := make(map[string]string)
	byBase := make(map[string][]string)
	knownIDs := make(map[string]bool)
	for _, track := range tracks {
		knownIDs[track.ID] = true
		byPath[normalizeM3UPath(track.FilePath)] = track.ID
		base := strings.ToLower(filepath.Base(track.FilePath))
		byBase[base] = append(byBase[base], track.ID)
	}
	var result []string
	var pendingID string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if strings.HasPrefix(line, "#WAVENODE:") {
			pendingID = strings.TrimSpace(strings.TrimPrefix(line, "#WAVENODE:"))
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		id := pendingID
		pendingID = ""
		if !knownIDs[id] {
			if parsed, parseErr := url.Parse(line); parseErr == nil && strings.Contains(parsed.Path, "/music/") {
				segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
				for index, segment := range segments {
					if segment == "music" && index+1 < len(segments) {
						id, _ = url.PathUnescape(segments[index+1])
						break
					}
				}
			}
		}
		if !knownIDs[id] {
			id = byPath[normalizeM3UPath(line)]
		}
		if !knownIDs[id] {
			matches := byBase[strings.ToLower(filepath.Base(filepath.FromSlash(line)))]
			if len(matches) == 1 {
				id = matches[0]
			}
		}
		if knownIDs[id] {
			result = append(result, id)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("none of the playlist entries matched this library")
	}
	return result, scanner.Err()
}

func (r *Router) filterListeningHistory(req *http.Request, entries []database.ListeningHistoryEntry) ([]database.ListeningHistoryEntry, error) {
	tracks := make([]database.Music, len(entries))
	for index := range entries {
		tracks[index] = entries[index].Track
	}
	allowed, err := r.filterMusicForRequest(req, tracks)
	if err != nil {
		return nil, err
	}
	allowedIDs := make(map[string]bool, len(allowed))
	for _, track := range allowed {
		allowedIDs[track.ID] = true
	}
	filtered := make([]database.ListeningHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		if allowedIDs[entry.Track.ID] {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

func normalizeM3UPath(value string) string {
	value, _ = url.PathUnescape(strings.TrimSpace(value))
	return strings.ToLower(filepath.Clean(filepath.FromSlash(value)))
}
