package handlers

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"music-server/auth"
	"music-server/database"
	"music-server/streaming"
	"music-server/utils"

	"github.com/gorilla/mux"
)

// MusicHandler handles music-related requests
type MusicHandler struct {
	db            *database.DB
	activeStreams atomic.Int64
}

// NewMusicHandler creates a new music handler
func NewMusicHandler(db *database.DB) *MusicHandler {
	return &MusicHandler{
		db: db,
	}
}

// GetAllMusic handles getting all music tracks
func (h *MusicHandler) GetAllMusic(w http.ResponseWriter, r *http.Request) {
	musicList, err := h.db.GetAllMusic()
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to retrieve music: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Music retrieved successfully",
		Data:    TransformMusicData(musicList),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// TransformMusicData adds frontend-compatible image URL fields to track data
func TransformMusicData(tracks []database.Music) []database.Music {
	transformed := make([]database.Music, len(tracks))
	for i, track := range tracks {
		transformedTrack := track
		imageURL := utils.NormalizeArtworkURL(track.ImageURL)
		transformedTrack.CoverArtURL = utils.NormalizeArtworkURL(track.CoverArtURL)
		transformedTrack.CoverArtSmallURL = utils.NormalizeArtworkURL(track.CoverArtSmallURL)
		transformedTrack.CoverArtMediumURL = utils.NormalizeArtworkURL(track.CoverArtMediumURL)
		transformedTrack.CoverArtLargeURL = utils.NormalizeArtworkURL(track.CoverArtLargeURL)

		if transformedTrack.CoverArtURL == "" {
			transformedTrack.CoverArtURL = imageURL
		}
		if transformedTrack.CoverArtSmallURL == "" {
			transformedTrack.CoverArtSmallURL = imageURL
		}
		transformed[i] = transformedTrack
	}
	return transformed
}

// AddMusic handles adding a new music track
func (h *MusicHandler) AddMusic(w http.ResponseWriter, r *http.Request) {
	var music database.Music
	err := json.NewDecoder(r.Body).Decode(&music)
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

	// Generate ID (in a real app, you'd use UUID or similar)
	music.ID = generateID()
	music.CreatedAt = time.Now()
	music.UpdatedAt = time.Now()

	if err := h.db.AddMusic(&music); err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to add music: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Music added successfully",
		Data:    music,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateMusic handles updating a music track
func (h *MusicHandler) UpdateMusic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updatedMusic database.Music
	err := json.NewDecoder(r.Body).Decode(&updatedMusic)
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

	// Check if music exists
	existingMusic, err := h.db.GetMusic(id)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Music not found",
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update fields
	updatedMusic.ID = id
	updatedMusic.CreatedAt = existingMusic.CreatedAt
	updatedMusic.UpdatedAt = time.Now()

	if err := h.db.UpdateMusic(&updatedMusic); err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to update music: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Music updated successfully",
		Data:    updatedMusic,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DeleteMusic handles deleting a music track
func (h *MusicHandler) DeleteMusic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.db.DeleteMusic(id); err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Music not found",
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Music deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// SearchMusic handles searching music tracks
func (h *MusicHandler) SearchMusic(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		response := auth.APIResponse{
			Success: false,
			Error:   "Search query is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	results, err := h.db.SearchMusic(query)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to search music: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Search completed successfully",
		Data:    results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ComprehensiveSearch handles comprehensive search across all content
func (h *MusicHandler) ComprehensiveSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		response := auth.APIResponse{
			Success: false,
			Error:   "Search query is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get all data
	musicList, err := h.db.GetAllMusic()
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to retrieve music: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	playlists, err := h.db.GetAllPlaylists()
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

	// Prepare search results
	searchQuery := strings.ToLower(query)
	results := map[string]interface{}{
		"songs":     []database.Music{},
		"albums":    []map[string]interface{}{},
		"artists":   []map[string]interface{}{},
		"playlists": []database.Playlist{},
	}

	// Search songs
	for _, track := range musicList {
		if strings.Contains(strings.ToLower(track.Title), searchQuery) ||
			strings.Contains(strings.ToLower(track.Artist), searchQuery) ||
			strings.Contains(strings.ToLower(track.Album), searchQuery) ||
			strings.Contains(strings.ToLower(track.Genre), searchQuery) {
			results["songs"] = append(results["songs"].([]database.Music), track)
		}
	}

	// Search albums (group by album name)
	albumMap := make(map[string]map[string]interface{})
	for _, track := range musicList {
		if track.Album != "" && strings.TrimSpace(track.Album) != "" && strings.TrimSpace(track.Album) != "Unknown Album" {
			if strings.Contains(strings.ToLower(track.Album), searchQuery) {
				if _, exists := albumMap[track.Album]; !exists {
					albumMap[track.Album] = map[string]interface{}{
						"name":                 track.Album,
						"artist":               track.Artist,
						"year":                 track.ReleaseDate.Year(),
						"cover_art_url":        track.CoverArtURL,
						"cover_art_small_url":  track.CoverArtSmallURL,
						"cover_art_medium_url": track.CoverArtMediumURL,
						"cover_art_large_url":  track.CoverArtLargeURL,
					}
				}
			}
		}
	}
	for _, album := range albumMap {
		results["albums"] = append(results["albums"].([]map[string]interface{}), album)
	}

	// Search artists
	artistMap := make(map[string]map[string]interface{})
	for _, track := range musicList {
		if strings.Contains(strings.ToLower(track.Artist), searchQuery) {
			if _, exists := artistMap[track.Artist]; !exists {
				// Count tracks and albums for this artist
				trackCount := 0
				albumSet := make(map[string]bool)
				for _, t := range musicList {
					if t.Artist == track.Artist {
						trackCount++
						if t.Album != "" && strings.TrimSpace(t.Album) != "" && strings.TrimSpace(t.Album) != "Unknown Album" {
							albumSet[t.Album] = true
						}
					}
				}

				artistMap[track.Artist] = map[string]interface{}{
					"name":        track.Artist,
					"track_count": trackCount,
					"album_count": len(albumSet),
				}
			}
		}
	}
	for _, artist := range artistMap {
		results["artists"] = append(results["artists"].([]map[string]interface{}), artist)
	}

	// Search playlists
	for _, playlist := range playlists {
		if strings.Contains(strings.ToLower(playlist.Name), searchQuery) ||
			strings.Contains(strings.ToLower(playlist.Description), searchQuery) {
			results["playlists"] = append(results["playlists"].([]database.Playlist), playlist)
		}
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Comprehensive search completed successfully",
		Data:    results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// StreamMusic handles streaming audio files
func (h *MusicHandler) StreamMusic(w http.ResponseWriter, r *http.Request) {
	h.activeStreams.Add(1)
	defer h.activeStreams.Add(-1)
	vars := mux.Vars(r)
	id := vars["id"]

	log.Printf("Streaming music request for ID: %s", id)

	// Get music track info
	music, err := h.db.GetMusic(id)
	if err != nil {
		log.Printf("Music not found: %v", err)
		response := auth.APIResponse{
			Success: false,
			Error:   "Music not found",
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Found music: %s, file_path: %s", music.Title, music.FilePath)

	// Get the file path from music record
	filePath := music.FilePath
	if filePath == "" {
		http.Error(w, "Track has no source file", http.StatusGone)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File does not exist: %s", filePath)
		http.Error(w, "Track source file is missing", http.StatusGone)
		return
	}

	log.Printf("File exists, streaming: %s", filePath)

	userID, _ := r.Context().Value("user_id").(string)
	profile, _ := h.db.GetPlaybackProfile(userID)
	properties, _ := h.db.GetTrackAudioProperties(music.ID)
	database.ApplyTrackAudioProperties(music, properties)
	gain := streaming.ReplayGainDB(profile, properties)
	forceTranscode := profile.TranscodeEnabled || gain != 0 || r.URL.Query().Get("transcode") == "true"
	if r.URL.Query().Get("transcode") == "false" {
		forceTranscode = false
	}
	if forceTranscode {
		format := profile.TranscodeFormat
		bitrate := profile.TranscodeBitrate
		if requested := r.URL.Query().Get("format"); requested != "" {
			format = requested
		}
		if requested, err := strconv.Atoi(r.URL.Query().Get("bitrate")); err == nil && requested > 0 {
			bitrate = requested
		}
		offset, _ := strconv.ParseFloat(r.URL.Query().Get("offset"), 64)
		if err := streaming.Serve(w, r, *music, streaming.Options{
			Format: format, Bitrate: bitrate, Offset: offset, GainDB: gain,
		}); err != nil && r.Context().Err() == nil {
			log.Printf("Transcoding failed for %s: %v", music.ID, err)
		}
		return
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		http.Error(w, "Track source file cannot be opened", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		http.Error(w, "Track source file cannot be read", http.StatusInternalServerError)
		return
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filePath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "private, no-cache")
	http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), file)
}

func (h *MusicHandler) ActiveStreams() int64 {
	return h.activeStreams.Load()
}

// generateSilentAudio creates a silent audio track for demo purposes
func generateSilentAudio(w http.ResponseWriter, r *http.Request, music *database.Music) {
	// Create a proper WAV file with silence
	// WAV header for 44.1kHz, 16-bit, mono silence
	duration := music.Duration
	if duration <= 0 {
		duration = 30 // Default to 30 seconds if no duration specified
	}

	sampleRate := 44100
	bitsPerSample := 16
	channels := 1
	bytesPerSample := bitsPerSample / 8
	blockAlign := channels * bytesPerSample
	byteRate := sampleRate * blockAlign
	totalSamples := sampleRate * duration
	dataSize := totalSamples * blockAlign
	fileSize := 36 + dataSize

	// Create WAV header
	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], uint32(fileSize))
	copy(header[8:12], []byte("WAVE"))
	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16) // Subchunk1Size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // AudioFormat (PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))

	// Set headers for audio streaming
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Length", strconv.Itoa(44+dataSize))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "no-cache")

	// Handle range requests for seeking
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		// Parse range header
		ranges := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
		if len(ranges) == 2 {
			start, err := strconv.ParseInt(ranges[0], 10, 64)
			if err != nil {
				start = 0
			}

			end := int64(44 + dataSize - 1)
			if ranges[1] != "" {
				end, err = strconv.ParseInt(ranges[1], 10, 64)
				if err != nil {
					end = int64(44 + dataSize - 1)
				}
			}

			// Validate range
			if start >= 0 && start < int64(44+dataSize) && end >= start && end < int64(44+dataSize) {
				// Set partial content headers
				w.WriteHeader(http.StatusPartialContent)
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, 44+dataSize))
				w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))

				// Write header if requested
				if start < 44 {
					w.Write(header[start:44])
					start = 44
				}

				// Write silent data
				if start >= 44 {
					silentData := make([]byte, end-start+1)
					// Already silent (all zeros)
					w.Write(silentData)
				}
				return
			}
		}
	}

	// If no range header or range parsing failed, write to entire file
	w.WriteHeader(http.StatusOK)

	// Write header
	w.Write(header)

	// Write silent data (all zeros)
	silentData := make([]byte, dataSize)
	// Already silent (all zeros)
	w.Write(silentData)
}

// generateID generates a simple unique ID (in production, use UUID)
func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// GetMusic handles getting a single music track by ID
func (h *MusicHandler) GetMusic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		response := auth.APIResponse{
			Success: false,
			Error:   "Music ID is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	music, err := h.db.GetMusic(id)
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Music not found: " + err.Error(),
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Music retrieved successfully",
		Data:    TransformMusicData([]database.Music{*music}),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
