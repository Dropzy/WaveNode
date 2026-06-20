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
	TargetSessionID string   `json:"target_session_id"`
	TrackIDs        []string `json:"track_ids"`
	StartIndex      int      `json:"start_index"`
	Action          string   `json:"action"`
	PositionMs      int64    `json:"position_ms"`
}

type playbackHandoffCommand struct {
	ID              string           `json:"id"`
	SourceSessionID string           `json:"source_session_id"`
	TargetSessionID string           `json:"target_session_id"`
	TrackIDs        []string         `json:"track_ids"`
	Tracks          []database.Music `json:"tracks,omitempty"`
	StartIndex      int              `json:"start_index"`
	Action          string           `json:"action"`
	PositionMs      int64            `json:"position_ms,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
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
	if payload.Action != "play_queue" && payload.Action != "toggle_play_pause" && payload.Action != "seek" {
		writeJSONError(w, http.StatusBadRequest, "Unsupported playback command")
		return
	}
	if payload.Action == "play_queue" {
		if len(payload.TrackIDs) == 0 {
			writeJSONError(w, http.StatusBadRequest, "No tracks were provided")
			return
		}
		if payload.StartIndex < 0 || payload.StartIndex >= len(payload.TrackIDs) {
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
	tracks := make([]database.Music, 0, len(payload.TrackIDs))
	if payload.Action == "play_queue" {
		for _, trackID := range payload.TrackIDs {
			track, err := r.db.GetMusic(trackID)
			if err == nil && track != nil {
				tracks = append(tracks, *track)
			}
		}
		if len(tracks) == 0 {
			writeJSONError(w, http.StatusBadRequest, "No playable library tracks were provided")
			return
		}
		if payload.StartIndex >= len(tracks) {
			payload.StartIndex = 0
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
	trackIDs, err := r.resolveM3UTrackIDs(file)
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

func (r *Router) resolveM3UTrackIDs(reader io.Reader) ([]string, error) {
	tracks, err := r.db.GetAllMusic()
	if err != nil {
		return nil, err
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

func normalizeM3UPath(value string) string {
	value, _ = url.PathUnescape(strings.TrimSpace(value))
	return strings.ToLower(filepath.Clean(filepath.FromSlash(value)))
}
