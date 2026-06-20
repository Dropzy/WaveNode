package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"music-server/auth"
	"music-server/database"

	"github.com/gorilla/mux"
)

// PlaylistHandler handles playlist-related requests
type PlaylistHandler struct {
	db *database.DB
}

// NewPlaylistHandler creates a new playlist handler
func NewPlaylistHandler(db *database.DB) *PlaylistHandler {
	return &PlaylistHandler{
		db: db,
	}
}

// GetPlaylists handles getting all playlists
func (h *PlaylistHandler) GetPlaylists(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}
	playlists, err := h.db.GetUserPlaylists(userID)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to retrieve playlists: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Playlists retrieved successfully",
		Data:    playlists,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPlaylist handles getting a specific playlist
func (h *PlaylistHandler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}

	playlist, err := h.db.GetUserPlaylist(id, userID)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Playlist not found",
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Playlist retrieved successfully",
		Data:    playlist,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreatePlaylist handles creating a new playlist
func (h *PlaylistHandler) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	var playlist database.Playlist
	err := json.NewDecoder(r.Body).Decode(&playlist)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}
	playlist.UserID = userID
	playlist.Type = database.PlaylistTypeManual
	playlist.SmartRules = nil

	// Generate ID
	playlist.ID = generateID()
	playlist.CreatedAt = time.Now()
	playlist.UpdatedAt = time.Now()

	if err := h.db.AddPlaylist(&playlist); err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to create playlist: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Playlist created successfully",
		Data:    playlist,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdatePlaylist handles updating a playlist
func (h *PlaylistHandler) UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updatedPlaylist database.Playlist
	err := json.NewDecoder(r.Body).Decode(&updatedPlaylist)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Check if playlist exists
	existingPlaylist, err := h.db.GetUserPlaylist(id, userID)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Playlist not found",
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	if existingPlaylist.Type == database.PlaylistTypeSmart {
		writePlaylistError(w, http.StatusConflict, "Use the smart playlist editor to update this playlist")
		return
	}

	// Update fields
	updatedPlaylist.ID = id
	updatedPlaylist.UserID = userID
	updatedPlaylist.CreatedAt = existingPlaylist.CreatedAt
	updatedPlaylist.UpdatedAt = time.Now()
	updatedPlaylist.Type = database.PlaylistTypeManual
	updatedPlaylist.SmartRules = nil
	if updatedPlaylist.TrackIDs == nil {
		updatedPlaylist.TrackIDs = existingPlaylist.TrackIDs
	}
	if updatedPlaylist.ImageURL == "" {
		updatedPlaylist.ImageURL = existingPlaylist.ImageURL
	}

	if err := h.db.UpdatePlaylist(&updatedPlaylist); err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to update playlist: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Playlist updated successfully",
		Data:    updatedPlaylist,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateSmartPlaylist creates a dynamic playlist whose membership is rule-driven.
func (h *PlaylistHandler) CreateSmartPlaylist(w http.ResponseWriter, r *http.Request) {
	var playlist database.Playlist
	if err := json.NewDecoder(r.Body).Decode(&playlist); err != nil {
		writePlaylistError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if playlist.Name == "" {
		writePlaylistError(w, http.StatusBadRequest, "Playlist name is required")
		return
	}
	playlist.ID = generateID()
	playlist.UserID = userID
	playlist.Type = database.PlaylistTypeSmart
	playlist.TrackIDs = []string{}
	if err := database.ValidateSmartPlaylistRules(playlist.SmartRules); err != nil {
		writePlaylistError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.db.AddPlaylist(&playlist); err != nil {
		writePlaylistError(w, http.StatusInternalServerError, "Failed to create smart playlist: "+err.Error())
		return
	}
	resolved, err := h.db.GetUserPlaylist(playlist.ID, userID)
	if err != nil {
		writePlaylistError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(auth.APIResponse{Success: true, Message: "Smart playlist created successfully", Data: resolved})
}

// UpdateSmartPlaylist updates the rules and metadata for a dynamic playlist.
func (h *PlaylistHandler) UpdateSmartPlaylist(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}
	id := mux.Vars(r)["id"]
	existing, err := h.db.GetUserPlaylist(id, userID)
	if err != nil {
		writePlaylistError(w, http.StatusNotFound, "Playlist not found")
		return
	}
	if existing.Type != database.PlaylistTypeSmart {
		writePlaylistError(w, http.StatusConflict, "This is not a smart playlist")
		return
	}
	var updated database.Playlist
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writePlaylistError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if updated.Name == "" {
		writePlaylistError(w, http.StatusBadRequest, "Playlist name is required")
		return
	}
	if err := database.ValidateSmartPlaylistRules(updated.SmartRules); err != nil {
		writePlaylistError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated.ID = id
	updated.UserID = userID
	updated.Type = database.PlaylistTypeSmart
	updated.TrackIDs = []string{}
	updated.CreatedAt = existing.CreatedAt
	if updated.ImageURL == "" {
		updated.ImageURL = existing.ImageURL
	}
	if err := h.db.UpdatePlaylist(&updated); err != nil {
		writePlaylistError(w, http.StatusInternalServerError, "Failed to update smart playlist: "+err.Error())
		return
	}
	resolved, err := h.db.GetUserPlaylist(id, userID)
	if err != nil {
		writePlaylistError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auth.APIResponse{Success: true, Message: "Smart playlist updated successfully", Data: resolved})
}

// PreviewSmartPlaylist evaluates rules without saving them.
func (h *PlaylistHandler) PreviewSmartPlaylist(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var rules database.SmartPlaylistRules
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		writePlaylistError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	tracks, err := h.db.EvaluateSmartPlaylist(userID, rules)
	if err != nil {
		writePlaylistError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auth.APIResponse{Success: true, Message: "Smart playlist preview generated", Data: tracks})
}

// DeletePlaylist handles deleting a playlist
func (h *PlaylistHandler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if err := h.db.DeletePlaylist(id, userID); err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Playlist not found",
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Playlist deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AddToPlaylist handles adding a track to a playlist
func (h *PlaylistHandler) AddToPlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]
	userID, contextErr := auth.GetUserFromContext(r)
	if contextErr != nil {
		writePlaylistError(w, http.StatusUnauthorized, contextErr.Error())
		return
	}

	var request struct {
		TrackID string `json:"track_id"`
		MusicID string `json:"music_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	trackID := request.TrackID
	if trackID == "" {
		trackID = request.MusicID
	}
	if trackID == "" {
		response := auth.APIResponse{
			Success: false,
			Error:   "Track ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	playlist, err := h.db.AddTrackToPlaylist(playlistID, trackID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "playlist not found" || err.Error() == "track not found" {
			status = http.StatusNotFound
		} else if err.Error() == "smart playlists are read-only" {
			status = http.StatusConflict
		}
		response := auth.APIResponse{
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Track added to playlist successfully",
		Data:    playlist,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AddManyToPlaylist handles adding multiple tracks atomically.
func (h *PlaylistHandler) AddManyToPlaylist(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserFromContext(r)
	if err != nil {
		writePlaylistError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var request struct {
		TrackIDs []string `json:"track_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writePlaylistError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	playlist, err := h.db.AddTracksToPlaylist(mux.Vars(r)["id"], request.TrackIDs, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "playlist not found" || err.Error() == "one or more tracks were not found" {
			status = http.StatusNotFound
		} else if err.Error() == "at least one track is required" {
			status = http.StatusBadRequest
		} else if err.Error() == "smart playlists are read-only" {
			status = http.StatusConflict
		}
		writePlaylistError(w, status, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auth.APIResponse{
		Success: true,
		Message: "Tracks added to playlist successfully",
		Data:    playlist,
	})
}

// RemoveFromPlaylist handles removing a track from a playlist
func (h *PlaylistHandler) RemoveFromPlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, contextErr := auth.GetUserFromContext(r)
	if contextErr != nil {
		writePlaylistError(w, http.StatusUnauthorized, contextErr.Error())
		return
	}
	playlist, err := h.db.RemoveTrackFromPlaylist(vars["id"], vars["music_id"], userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "playlist not found" {
			status = http.StatusNotFound
		} else if err.Error() == "smart playlists are read-only" {
			status = http.StatusConflict
		}
		response := auth.APIResponse{
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Track removed from playlist successfully",
		Data:    playlist,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPlaylistTracks handles getting tracks in a playlist
func (h *PlaylistHandler) GetPlaylistTracks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, contextErr := auth.GetUserFromContext(r)
	if contextErr != nil {
		writePlaylistError(w, http.StatusUnauthorized, contextErr.Error())
		return
	}
	tracks, err := h.db.GetPlaylistTracks(vars["id"], userID)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "playlist not found" {
			status = http.StatusNotFound
		}
		response := auth.APIResponse{
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Playlist tracks retrieved successfully",
		Data:    tracks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func writePlaylistError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(auth.APIResponse{Success: false, Error: message})
}
