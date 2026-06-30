package router

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"music-server/auth"
	"music-server/database"
	"music-server/enrichment"
	"music-server/handlers"
	"music-server/middleware"
	"music-server/scanner"
	"music-server/utils"
	"music-server/websocket"

	"github.com/gorilla/mux"
)

// generateAlbumID generates a consistent hash for an album based on name and artist
// This function is deprecated - use database.generateAlbumID instead
// Keeping for backward compatibility but should match database implementation
func generateAlbumID(albumName, artistName string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(albumName))))
	return hex.EncodeToString(hash[:4])
}

// Router holds all the handlers and middleware
type Router struct {
	authHandler       *auth.AuthHandler
	musicHandler      *handlers.MusicHandler
	playlistHandler   *handlers.PlaylistHandler
	enrichmentHandler *EnrichmentHandler
	webSocketManager  *websocket.WebSocketManager
	db                *database.DB
	scanStore         *database.ScanStore
	autoUpdater       *scanner.AutoUpdater
	updateManager     *UpdateManager
	setupToken        string
	subsonicAuthCache sync.Map
	playbackHandoffs  sync.Map
	castTokens        sync.Map
	outputDevices     sync.Map
	corsConfig        struct {
		AllowedOrigins []string `json:"allowed_origins"`
		AllowedMethods []string `json:"allowed_methods"`
		AllowedHeaders []string `json:"allowed_headers"`
	}
}

// NewRouter creates a new router with all handlers
func NewRouter(
	authHandler *auth.AuthHandler,
	musicHandler *handlers.MusicHandler,
	playlistHandler *handlers.PlaylistHandler,
	webSocketManager *websocket.WebSocketManager,
	db *database.DB,
	setupToken string,
	corsConfig struct {
		AllowedOrigins []string `json:"allowed_origins"`
		AllowedMethods []string `json:"allowed_methods"`
		AllowedHeaders []string `json:"allowed_headers"`
	},
) *Router {
	scanStore := database.NewScanStore(db)
	enrichmentHandler := NewEnrichmentHandler(db, scanStore)

	router := &Router{
		authHandler:       authHandler,
		musicHandler:      musicHandler,
		playlistHandler:   playlistHandler,
		enrichmentHandler: enrichmentHandler,
		webSocketManager:  webSocketManager,
		db:                db,
		setupToken:        strings.TrimSpace(setupToken),
		scanStore:         scanStore,
		corsConfig:        corsConfig,
		updateManager:     NewUpdateManager(WaveNodeVersion),
	}
	router.startArtistMetadataRefreshLoop()
	return router
}

// SetEnrichmentScanner sets enrichment scanner on enrichment handler
func (r *Router) SetEnrichmentScanner(scanner *enrichment.EnrichmentScanner) {
	r.enrichmentHandler.SetEnrichmentScanner(scanner)
}

func (r *Router) SetAutoUpdater(updater *scanner.AutoUpdater) {
	r.autoUpdater = updater
}

// SetupRoutes configures all the routes
func (r *Router) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(middleware.CORSMiddleware(r.corsConfig))
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.LoggingMiddleware())

	// Health check endpoint
	router.HandleFunc("/health", r.healthCheck).Methods("GET")

	// API Documentation endpoint
	router.HandleFunc("/swagger/index.html", r.apiDocumentation).Methods("GET")

	// Subsonic/OpenSubsonic compatibility API.
	router.HandleFunc("/rest/{method}", r.subsonicAPI).Methods("GET", "POST")

	// Public routes (no authentication required)
	public := router.PathPrefix("/api").Subrouter()
	public.Handle("/auth/login", middleware.LoginRateLimit(10, 5*time.Minute)(http.HandlerFunc(r.authHandler.Login))).Methods("POST", "OPTIONS")
	public.HandleFunc("/auth/register", r.authHandler.Register).Methods("POST", "OPTIONS")
	public.HandleFunc("/setup/status", r.getSetupStatus).Methods("GET", "OPTIONS")
	setupRateLimit := middleware.LoginRateLimit(20, 5*time.Minute)
	public.Handle("/setup/directories", setupRateLimit(http.HandlerFunc(r.browseSetupDirectories))).Methods("GET", "OPTIONS")
	public.Handle("/setup/complete", setupRateLimit(http.HandlerFunc(r.completeSetup))).Methods("POST", "OPTIONS")
	public.HandleFunc("/cast/{token}/music/{id}", r.streamCastMusic).Methods("GET", "HEAD", "OPTIONS")

	// Protected routes (authentication required)
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(r.authHandler.AuthMiddleware())

	// Auth routes
	protected.HandleFunc("/auth/me", r.authHandler.GetCurrentUser).Methods("GET", "OPTIONS")
	protected.HandleFunc("/auth/password", r.authHandler.ChangePassword).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/auth/sessions", r.authHandler.GetSessions).Methods("GET", "OPTIONS")
	protected.HandleFunc("/auth/sessions/others", r.authHandler.RevokeOtherSessions).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/auth/sessions/{id}", r.authHandler.RevokeSession).Methods("DELETE", "OPTIONS")

	// Music routes
	protected.HandleFunc("/music", r.musicHandler.GetAllMusic).Methods("GET", "OPTIONS")
	protected.HandleFunc("/music/search", r.musicHandler.SearchMusic).Methods("GET", "OPTIONS")
	protected.HandleFunc("/music/search/comprehensive", r.musicHandler.ComprehensiveSearch).Methods("GET", "OPTIONS")
	protected.HandleFunc("/music/{id}", r.musicHandler.GetMusic).Methods("GET", "OPTIONS")
	protected.HandleFunc("/music/{id}/stream", r.musicHandler.StreamMusic).Methods("GET", "OPTIONS")
	protected.HandleFunc("/music/{id}/download", r.musicHandler.DownloadMusic).Methods("GET", "OPTIONS")
	// Artwork serving route (public - no auth required)
	public.HandleFunc("/artwork/{filename}", r.serveArtwork).Methods("GET", "OPTIONS")

	// Additional routes that frontend is calling
	protected.HandleFunc("/recently-played", r.recentlyPlayed).Methods("GET", "OPTIONS")
	protected.HandleFunc("/recently-played/{id}", r.addToRecentlyPlayed).Methods("POST", "OPTIONS")
	protected.HandleFunc("/albums", r.getAlbums).Methods("GET", "OPTIONS")
	protected.HandleFunc("/albums/{id}", r.getAlbumByID).Methods("GET", "OPTIONS")              // New ID-based endpoint
	protected.HandleFunc("/albums/{id}/tracks", r.getAlbumTracksByID).Methods("GET", "OPTIONS") // New ID-based endpoint
	protected.HandleFunc("/albums/{name}/tracks", r.getAlbumTracks).Methods("GET", "OPTIONS")   // Keep name-based for backward compatibility
	protected.HandleFunc("/artists", r.getArtists).Methods("GET", "OPTIONS")
	protected.HandleFunc("/artists/lookup", r.lookupArtistMetadata).Methods("GET", "OPTIONS")
	protected.HandleFunc("/artists/{id}", r.getArtistByID).Methods("GET", "OPTIONS") // New hash-based endpoint
	protected.HandleFunc("/artists/{id}/image", r.getArtistImage).Methods("GET", "OPTIONS")
	protected.HandleFunc("/artists/{id}/tracks", r.getArtistTracksByID).Methods("GET", "OPTIONS")
	protected.HandleFunc("/search", r.comprehensiveSearch).Methods("GET", "OPTIONS")
	protected.HandleFunc("/liked-tracks", r.getLikedTracks).Methods("GET", "OPTIONS")
	protected.HandleFunc("/liked-tracks/{id}", r.likeTrack).Methods("POST", "OPTIONS")
	protected.HandleFunc("/liked-tracks/{id}/check", r.checkTrackLiked).Methods("GET", "OPTIONS")
	protected.HandleFunc("/liked-tracks/{id}", r.unlikeTrack).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/ratings/{id}", r.getRating).Methods("GET", "OPTIONS")
	protected.HandleFunc("/ratings/{id}", r.setRating).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/playback-profile", r.getPlaybackProfile).Methods("GET", "OPTIONS")
	protected.HandleFunc("/playback-profile", r.updatePlaybackProfile).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/playback/connect", r.createPlaybackHandoff).Methods("POST", "OPTIONS")
	protected.HandleFunc("/playback/connect/pending", r.consumePlaybackHandoff).Methods("GET", "OPTIONS")
	protected.HandleFunc("/outputs/cast-url", r.createCastURL).Methods("POST", "OPTIONS")
	protected.HandleFunc("/outputs/devices", r.discoverOutputDevices).Methods("GET", "OPTIONS")
	protected.HandleFunc("/outputs/dlna/play", r.playOnDLNADevice).Methods("POST", "OPTIONS")
	protected.HandleFunc("/scrobble/settings", r.getScrobbleSettings).Methods("GET", "OPTIONS")
	protected.HandleFunc("/scrobble/settings", r.updateScrobbleSettings).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/scrobble/lastfm/start", r.startLastFMAuth).Methods("POST", "OPTIONS")
	protected.HandleFunc("/scrobble/lastfm/complete", r.completeLastFMAuth).Methods("POST", "OPTIONS")
	protected.HandleFunc("/scrobble/lastfm", r.disconnectLastFM).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/scrobble/now-playing/{id}", r.scrobbleNowPlaying).Methods("POST", "OPTIONS")
	protected.HandleFunc("/scrobble/listened/{id}", r.scrobbleListened).Methods("POST", "OPTIONS")
	protected.HandleFunc("/history", r.getListeningHistory).Methods("GET", "OPTIONS")
	protected.HandleFunc("/history/export", r.exportListeningHistory).Methods("GET", "OPTIONS")
	protected.HandleFunc("/history", r.clearListeningHistory).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/plugins/home-rows", r.getPluginHomeRows).Methods("GET", "OPTIONS")
	protected.HandleFunc("/plugins/radio-metadata", r.getPluginRadioMetadata).Methods("GET", "OPTIONS")
	protected.HandleFunc("/plugins/track-actions", r.getPluginTrackActions).Methods("GET", "OPTIONS")
	protected.HandleFunc("/podcasts/search", r.searchPodcasts).Methods("GET", "OPTIONS")
	protected.HandleFunc("/podcasts/home", r.getPodcastHome).Methods("GET", "OPTIONS")
	protected.HandleFunc("/podcasts/progress", r.updatePodcastProgress).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/podcasts/preferences", r.getPodcastPreferences).Methods("GET", "OPTIONS")
	protected.HandleFunc("/podcasts/preferences", r.updatePodcastPreferences).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/podcasts/subscriptions", r.listPodcastSubscriptions).Methods("GET", "OPTIONS")
	protected.HandleFunc("/podcasts/subscriptions", r.savePodcastSubscription).Methods("POST", "OPTIONS")
	protected.HandleFunc("/podcasts/subscriptions/{id}", r.deletePodcastSubscription).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/podcasts/queue", r.getPodcastQueue).Methods("GET", "OPTIONS")
	protected.HandleFunc("/podcasts/queue", r.updatePodcastQueue).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/podcasts/chapters", r.getPodcastChapters).Methods("GET", "OPTIONS")
	protected.HandleFunc("/podcasts/{id}/episodes", r.getPodcastEpisodes).Methods("GET", "OPTIONS")
	protected.HandleFunc("/discovery/settings", r.getDiscoverySettings).Methods("GET", "OPTIONS")
	protected.HandleFunc("/discovery/settings", r.updateDiscoverySettings).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/discovery/preview", r.previewDiscovery).Methods("GET", "OPTIONS")
	protected.HandleFunc("/discovery/import", r.importDiscoveryPlaylist).Methods("POST", "OPTIONS")

	// Playlist routes
	protected.HandleFunc("/playlists", r.playlistHandler.GetPlaylists).Methods("GET", "OPTIONS")
	protected.HandleFunc("/playlists", r.playlistHandler.CreatePlaylist).Methods("POST", "OPTIONS")
	protected.HandleFunc("/playlists/import", r.importPlaylistM3U).Methods("POST", "OPTIONS")
	protected.HandleFunc("/playlists/{id}", r.playlistHandler.GetPlaylist).Methods("GET", "OPTIONS")
	protected.HandleFunc("/playlists/{id}", r.playlistHandler.UpdatePlaylist).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/playlists/{id}", r.playlistHandler.DeletePlaylist).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/playlists/{id}/tracks", r.playlistHandler.GetPlaylistTracks).Methods("GET", "OPTIONS")
	protected.HandleFunc("/playlists/{id}/tracks", r.playlistHandler.AddToPlaylist).Methods("POST", "OPTIONS")
	protected.HandleFunc("/playlists/{id}/tracks/bulk", r.playlistHandler.AddManyToPlaylist).Methods("POST", "OPTIONS")
	protected.HandleFunc("/playlists/{id}/tracks/{music_id}", r.playlistHandler.RemoveFromPlaylist).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/playlists/{id}/export.m3u", r.exportPlaylistM3U).Methods("GET", "OPTIONS")
	protected.HandleFunc("/smart-playlists", r.playlistHandler.CreateSmartPlaylist).Methods("POST", "OPTIONS")
	protected.HandleFunc("/smart-playlists/preview", r.playlistHandler.PreviewSmartPlaylist).Methods("POST", "OPTIONS")
	protected.HandleFunc("/smart-playlists/{id}", r.playlistHandler.UpdateSmartPlaylist).Methods("PUT", "OPTIONS")

	// WebSocket endpoint
	router.HandleFunc("/ws", r.webSocketManager.HandleWebSocket).Methods("GET", "OPTIONS")

	// Admin WebSocket endpoint
	router.HandleFunc("/api/admin/ws/scan", r.webSocketManager.HandleWebSocket).Methods("GET", "OPTIONS")

	// Scan routes (accessible by all authenticated users)
	protected.HandleFunc("/scan/current", r.getCurrentScan).Methods("GET", "OPTIONS")

	// Admin routes (authentication + admin role required)
	admin := router.PathPrefix("/api/admin").Subrouter()
	admin.Use(r.authHandler.AuthMiddleware())
	admin.Use(r.authHandler.AdminMiddleware())
	admin.HandleFunc("/music", r.musicHandler.AddMusic).Methods("POST", "OPTIONS")
	admin.HandleFunc("/music/{id}", r.musicHandler.UpdateMusic).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/music/{id}", r.musicHandler.DeleteMusic).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/scan", r.scanLibrary).Methods("POST", "OPTIONS")
	admin.HandleFunc("/scan/library", r.scanLibrary).Methods("POST", "OPTIONS")
	admin.HandleFunc("/scan/library", handlers.StopScan).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/library/directories", r.browseMusicDirectories).Methods("GET", "OPTIONS")
	admin.HandleFunc("/library/sources", r.getMusicSources).Methods("GET", "OPTIONS")
	admin.HandleFunc("/library/sources", r.addMusicSource).Methods("POST", "OPTIONS")
	admin.HandleFunc("/library/sources/{id}", r.deleteMusicSource).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/library/automatic-updates", r.getAutomaticUpdates).Methods("GET", "OPTIONS")
	admin.HandleFunc("/library/automatic-updates", r.updateAutomaticUpdates).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/backup", r.downloadBackup).Methods("GET", "OPTIONS")
	admin.HandleFunc("/restore", r.restoreBackup).Methods("POST", "OPTIONS")
	admin.HandleFunc("/system/status", r.getSystemStatus).Methods("GET", "OPTIONS")
	admin.HandleFunc("/system/update", r.getUpdateStatus).Methods("GET", "OPTIONS")
	admin.HandleFunc("/system/update/check", r.checkForUpdate).Methods("POST", "OPTIONS")
	admin.HandleFunc("/system/update/run", r.runUpdate).Methods("POST", "OPTIONS")
	admin.HandleFunc("/library/diagnostics", r.getLibraryDiagnostics).Methods("GET", "OPTIONS")
	admin.HandleFunc("/library", r.clearLibrary).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/integrations/lastfm", r.getAdminLastFMIntegration).Methods("GET", "OPTIONS")
	admin.HandleFunc("/integrations/lastfm", r.updateAdminLastFMIntegration).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/stats", r.getStats).Methods("GET", "OPTIONS")
	admin.HandleFunc("/logs", r.getLogs).Methods("GET", "OPTIONS")
	admin.HandleFunc("/users", r.getUsers).Methods("GET", "OPTIONS")
	admin.HandleFunc("/users", r.createUser).Methods("POST", "OPTIONS")
	admin.HandleFunc("/users/{id}", r.updateUser).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/users/{id}", r.deleteUser).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/scans", r.clearScanHistory).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/scans", r.getScanHistory).Methods("GET", "OPTIONS")
	admin.HandleFunc("/scans/{id}", r.getScanDetails).Methods("GET", "OPTIONS")
	admin.HandleFunc("/artists/discover-images", r.discoverArtistImages).Methods("POST", "OPTIONS")
	admin.HandleFunc("/artists/{id}/refresh-metadata", r.refreshArtistMetadataEndpoint).Methods("POST", "OPTIONS")
	admin.HandleFunc("/artists/{id}/image-candidates", r.listArtistImageCandidates).Methods("GET", "OPTIONS")
	admin.HandleFunc("/artists/{id}/image-primary", r.setPrimaryArtistImage).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/artists/{id}/image-candidates/{imageId}", r.updateArtistImageAttribution).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/artists/{id}/image", r.uploadArtistImage).Methods("POST", "OPTIONS")
	admin.HandleFunc("/artists/{id}/image", r.deleteArtistImage).Methods("DELETE", "OPTIONS")
	admin.HandleFunc("/plugins", r.getAdminPlugins).Methods("GET", "OPTIONS")
	admin.HandleFunc("/plugins", r.installPlugin).Methods("POST", "OPTIONS")
	admin.HandleFunc("/plugins/{id}/enabled", r.setPluginEnabled).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/plugins/{id}", r.deletePlugin).Methods("DELETE", "OPTIONS")

	// Enrichment routes (admin only)
	admin.HandleFunc("/enrich/musicbrainz", r.enrichmentHandler.EnrichMusicBrainz).Methods("POST", "OPTIONS")
	admin.HandleFunc("/enrich/cover-art", r.enrichmentHandler.EnrichCoverArt).Methods("POST", "OPTIONS")

	return router
}

// healthCheck returns health status of server
func (r *Router) healthCheck(w http.ResponseWriter, req *http.Request) {
	if err := r.db.HealthCheck(); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database is unavailable")
		return
	}

	response := map[string]interface{}{
		"status":  "healthy",
		"message": "WaveNode is running",
		"version": WaveNodeVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// serveArtwork serves extracted artwork files
func (r *Router) serveArtwork(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	filename := vars["filename"]

	if filename == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filename = filepath.Base(filename)
	var artworkPath string
	var fileInfo os.FileInfo
	for _, directory := range utils.ArtworkSearchDirectories() {
		candidate := filepath.Join(directory, filename)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			artworkPath = candidate
			fileInfo = info
			break
		}
	}

	if artworkPath == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Open file
	file, err := os.Open(artworkPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Determine content type based on file extension
	ext := filepath.Ext(filename)
	contentType := "image/jpeg" // default
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	// Set headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	w.Header().Set("Access-Control-Allow-Origin", "*")          // CORS for artwork

	// Serve the file
	http.ServeContent(w, req, "", fileInfo.ModTime(), file)
}

// getArtistByID returns artist information by ID
func (r *Router) getArtistByID(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	artistID := vars["id"]

	if artistID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Artist ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	artist, err := r.db.GetArtistByHash(artistID)
	if err != nil {
		artist, err = r.db.GetLibraryArtistByID(artistID)
		if err != nil {
			response := map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	response := map[string]interface{}{
		"success": true,
		"data":    artist,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getArtistTracksByID returns tracks for a specific artist by ID
func (r *Router) getArtistTracksByID(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	artistID := vars["id"]

	if artistID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Artist ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Prefer the stored artist ID, but accept a URL-encoded name for links
	// created before artist IDs were available in every API response.
	artist, err := r.db.GetArtistByHash(artistID)
	if err != nil {
		decodedArtistName, decodeErr := url.QueryUnescape(artistID)
		if decodeErr != nil {
			decodedArtistName = artistID
		}

		artist, err = r.db.GetArtistByName(decodedArtistName)
		if err != nil {
			artist, err = r.db.GetLibraryArtistByID(artistID)
			if err != nil {
				response := map[string]interface{}{
					"success": false,
					"error":   "Artist not found",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(response)
				return
			}
		}
	}

	// Get tracks for artist using existing name-based method
	tracks, err := r.db.GetArtistTracks(artist.Name)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get unique albums for this artist and create album objects
	albumMap := make(map[string]database.Album)
	for _, track := range tracks {
		if track.Album != "" {
			// Try to get the album from the database
			albumPtr, err := r.db.GetAlbumByNameAndArtist(track.Album, artist.Name)
			if err == nil && albumPtr.ID != "" {
				albumMap[track.Album] = *albumPtr
			} else {
				// If not found in database, create a basic album object with generated ID
				generatedAlbum := database.Album{
					ID:     generateAlbumID(track.Album, artist.Name),
					Name:   track.Album,
					Artist: artist.Name,
				}
				albumMap[track.Album] = generatedAlbum
			}
		}
	}

	// Convert albums map to slice of album objects
	albumList := make([]database.Album, 0, len(albumMap))
	for _, album := range albumMap {
		albumList = append(albumList, album)
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"artist": map[string]interface{}{
				"id":               artist.ID,
				"name":             artist.Name,
				"track_count":      len(tracks),
				"album_count":      len(albumList),
				"image_url":        artist.ImageURL,
				"image_small_url":  artist.ImageSmallURL,
				"image_medium_url": artist.ImageMediumURL,
				"image_large_url":  artist.ImageLargeURL,
			},
			"tracks": tracks,
			"albums": albumList,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getCurrentScan returns current active scan status
func (r *Router) getCurrentScan(w http.ResponseWriter, req *http.Request) {
	// Use the library scanner instead of scanStore
	if handlers.ScannerInstance == nil {
		response := map[string]interface{}{
			"success": true,
			"data":    nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	scan, err := handlers.ScannerInstance.GetScanStatus()
	if err != nil {
		// No scan in progress
		response := map[string]interface{}{
			"success": true,
			"data":    nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    scan,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// clearScanHistory clears all scan history (admin only)
func (r *Router) clearScanHistory(w http.ResponseWriter, req *http.Request) {
	// Use the ClearScans handler from handlers package
	handlers.ClearScans(w, req)
}

// scanLibrary triggers a library scan (admin only)
func (r *Router) scanLibrary(w http.ResponseWriter, req *http.Request) {
	// Use handlers.ScanLibrary function with proper parameters
	handlers.ScanLibrary(w, req)
}

func (r *Router) getMusicSources(w http.ResponseWriter, req *http.Request) {
	sources, err := r.db.GetMusicSources()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    sources,
	})
}

func (r *Router) getAutomaticUpdates(w http.ResponseWriter, req *http.Request) {
	if r.autoUpdater == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Automatic library updates are unavailable")
		return
	}
	settings, err := r.autoUpdater.Settings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": settings})
}

func (r *Router) updateAutomaticUpdates(w http.ResponseWriter, req *http.Request) {
	if r.autoUpdater == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Automatic library updates are unavailable")
		return
	}
	var payload struct {
		Enabled         bool `json:"enabled"`
		IntervalMinutes int  `json:"interval_minutes"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid automatic update settings")
		return
	}
	settings, err := r.autoUpdater.UpdateSettings(payload.Enabled, payload.IntervalMinutes)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": settings})
}

type browsableDirectory struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (r *Router) browseMusicDirectories(w http.ResponseWriter, req *http.Request) {
	roots := serverFilesystemRoots()
	target := strings.TrimSpace(req.URL.Query().Get("path"))
	if target == "" {
		sources, err := r.db.GetMusicSources()
		if err == nil && len(sources) > 0 {
			target = sources[0].Path
		} else if len(roots) > 0 {
			target = roots[0]
		}
	}

	if !filepath.IsAbs(target) {
		writeJSONError(w, http.StatusBadRequest, "Directory path must be absolute")
		return
	}

	target = filepath.Clean(target)
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "Directory is not accessible")
		return
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		writeJSONError(w, http.StatusForbidden, "Directory cannot be opened")
		return
	}

	directories := make([]browsableDirectory, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		childPath := filepath.Join(target, entry.Name())
		if childInfo, statErr := os.Stat(childPath); statErr == nil && childInfo.IsDir() {
			directories = append(directories, browsableDirectory{
				Name: entry.Name(),
				Path: childPath,
			})
		}
	}

	parent := filepath.Dir(target)
	if parent == target {
		parent = ""
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"current_path": target,
			"parent_path":  parent,
			"directories":  directories,
			"roots":        roots,
		},
	})
}

func serverFilesystemRoots() []string {
	if os.PathSeparator != '\\' {
		return []string{string(os.PathSeparator)}
	}

	roots := make([]string, 0)
	for drive := 'A'; drive <= 'Z'; drive++ {
		root := fmt.Sprintf("%c:%c", drive, os.PathSeparator)
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			roots = append(roots, root)
		}
	}
	return roots
}

func (r *Router) addMusicSource(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "A valid music source path is required")
		return
	}

	source, err := r.db.AddMusicSource(payload.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    source,
	})
}

func (r *Router) deleteMusicSource(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "Music source ID is required")
		return
	}

	if err := r.db.DeleteMusicSource(id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSONError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Music source removed",
	})
}

func (r *Router) clearLibrary(w http.ResponseWriter, req *http.Request) {
	if handlers.ScannerInstance != nil && handlers.ScannerInstance.IsScanning() {
		writeJSONError(w, http.StatusConflict, "Wait for the current library scan to finish before clearing the library")
		return
	}

	if err := r.db.ClearLibrary(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Library cleared",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// getStats returns server statistics (admin only)
func (r *Router) getStats(w http.ResponseWriter, req *http.Request) {
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get enrichment statistics
	enrichmentStats, err := r.db.GetEnrichmentStatistics()
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get total counts using existing database methods
	music, _ := r.db.GetAllMusic()
	artists, _ := r.db.GetAllArtists()
	albums, _ := r.db.GetAllAlbums()
	playlists, _ := r.db.GetUserPlaylists(userID)

	totalTracks := len(music)
	totalArtists := len(artists)
	totalAlbums := len(albums)
	totalPlaylists := len(playlists)

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"total_tracks":    totalTracks,
			"total_artists":   totalArtists,
			"total_albums":    totalAlbums,
			"total_playlists": totalPlaylists,
			"connected_users": r.webSocketManager.GetClientCount(),
			"enrichment":      enrichmentStats,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getLogs returns server logs (admin only)
func (r *Router) getLogs(w http.ResponseWriter, req *http.Request) {
	// This would return actual server logs
	// For now, return a placeholder response
	response := map[string]interface{}{
		"success": true,
		"data":    []string{}, // Empty logs for now
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getUsers returns all users (admin only)
func (r *Router) getUsers(w http.ResponseWriter, req *http.Request) {
	users, err := r.db.GetAllUsers()
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Remove passwords from response
	var cleanUsers []map[string]interface{}
	for _, user := range users {
		cleanUser := map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		}
		cleanUsers = append(cleanUsers, cleanUser)
	}

	response := map[string]interface{}{
		"success": true,
		"data":    cleanUsers,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// createUser creates an account without exposing public registration.
func (r *Router) createUser(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	payload.Username = strings.TrimSpace(payload.Username)
	payload.Email = strings.TrimSpace(payload.Email)
	if len(payload.Username) < 3 {
		writeJSONError(w, http.StatusBadRequest, "Username must be at least 3 characters")
		return
	}
	if !strings.Contains(payload.Email, "@") {
		writeJSONError(w, http.StatusBadRequest, "Enter a valid email address")
		return
	}
	if len(payload.Password) < 8 {
		writeJSONError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}
	if payload.Role == "" {
		payload.Role = "user"
	}
	if payload.Role != "user" && payload.Role != "admin" {
		writeJSONError(w, http.StatusBadRequest, "Role must be either 'user' or 'admin'")
		return
	}

	user := &database.User{
		Username: payload.Username,
		Email:    payload.Email,
		Role:     payload.Role,
	}
	if err := r.db.CreateUser(user, payload.Password); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	user.Password = ""
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    user,
	})
}

// updateUser updates a user's role (admin only)
func (r *Router) updateUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID := vars["id"]

	if userID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "User ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var request struct {
		Role string `json:"role"`
	}

	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Invalid request body",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate role
	if request.Role != "user" && request.Role != "admin" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Role must be either 'user' or 'admin'",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update user role
	err = r.db.UpdateUserRole(userID, request.Role)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, database.ErrLastAdministrator) {
			status = http.StatusConflict
		}
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "User role updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// deleteUser deletes a user (admin only)
func (r *Router) deleteUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userID := vars["id"]

	if userID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "User ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Delete user
	err := r.db.DeleteUser(userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, database.ErrLastAdministrator) {
			status = http.StatusConflict
		}
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "User deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getScanHistory returns all scan history (admin only)
func (r *Router) getScanHistory(w http.ResponseWriter, req *http.Request) {
	scans, err := r.scanStore.GetAllScans()
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    scans,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getScanDetails returns details for a specific scan (admin only)
func (r *Router) getScanDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	scanID := vars["id"]

	if scanID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Scan ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	scan, err := r.scanStore.GetScan(scanID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    scan,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// recentlyPlayed returns recently played tracks
func (r *Router) recentlyPlayed(w http.ResponseWriter, req *http.Request) {
	// Get user ID from JWT token context
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to get user information",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get recently played tracks for user (limit to 20)
	tracks, err := r.db.GetRecentlyPlayedTracks(userID, 20)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get recently played tracks: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	for i := range tracks {
		normalizeTrackArtwork(&tracks[i])
		if tracks[i].CoverArtURL == "" &&
			tracks[i].CoverArtSmallURL == "" &&
			tracks[i].CoverArtMediumURL == "" &&
			tracks[i].CoverArtLargeURL == "" &&
			tracks[i].ImageURL == "" {
			album, albumErr := r.db.GetAlbumByNameAndArtist(tracks[i].Album, tracks[i].Artist)
			if albumErr == nil {
				normalizeAlbumArtwork(album)
				inheritAlbumArtwork(&tracks[i], album)
			}
		}
	}

	response := map[string]interface{}{
		"success": true,
		"data":    tracks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// addToRecentlyPlayed adds a track to recently played
func (r *Router) addToRecentlyPlayed(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	trackID := vars["id"]

	if trackID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Track ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the current user from JWT token
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to get user information",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	source := strings.ToLower(strings.TrimSpace(req.URL.Query().Get("source")))
	if source == "" {
		source = "web"
	}
	if source != "web" && source != "desktop" && source != "mobile" && source != "subsonic" {
		source = "web"
	}
	device := strings.TrimSpace(req.URL.Query().Get("device"))
	if len(device) > 80 {
		device = device[:80]
	}

	// Add track to their recently played list
	err = r.db.AddToRecentlyPlayedFrom(userID, trackID, source, device)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to add track to recently played: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Track added to recently played successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getAlbumByID returns album information by ID
func (r *Router) getAlbumByID(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	albumID := vars["id"]

	if albumID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Album ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	album, err := r.db.GetAlbumByID(albumID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}
	normalizeAlbumArtwork(album)

	response := map[string]interface{}{
		"success": true,
		"data":    album,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getAlbumTracksByID returns tracks for a specific album by ID
func (r *Router) getAlbumTracksByID(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	albumID := vars["id"]

	if albumID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Album ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get album info first
	album, err := r.db.GetAlbumByID(albumID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get tracks for the album
	tracks, err := r.db.GetAlbumTracksByID(albumID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	normalizeAlbumArtwork(album)
	for i := range tracks {
		normalizeTrackArtwork(&tracks[i])
		inheritAlbumArtwork(&tracks[i], album)
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"album":  album,
			"tracks": tracks,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getAlbums returns all albums
func (r *Router) getAlbums(w http.ResponseWriter, req *http.Request) {
	albums, err := r.db.GetAllAlbums()
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Transform album cover art URLs to use artwork endpoint.
	for i := range albums {
		normalizeAlbumArtwork(&albums[i])
	}

	response := map[string]interface{}{
		"success": true,
		"data":    albums,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func normalizeAlbumArtwork(album *database.Album) {
	album.CoverArtURL = utils.NormalizeArtworkURL(album.CoverArtURL)
	album.CoverArtSmallURL = utils.NormalizeArtworkURL(album.CoverArtSmallURL)
	album.CoverArtMediumURL = utils.NormalizeArtworkURL(album.CoverArtMediumURL)
	album.CoverArtLargeURL = utils.NormalizeArtworkURL(album.CoverArtLargeURL)
}

func normalizeTrackArtwork(track *database.Music) {
	track.ImageURL = utils.NormalizeArtworkURL(track.ImageURL)
	track.CoverArtURL = utils.NormalizeArtworkURL(track.CoverArtURL)
	track.CoverArtSmallURL = utils.NormalizeArtworkURL(track.CoverArtSmallURL)
	track.CoverArtMediumURL = utils.NormalizeArtworkURL(track.CoverArtMediumURL)
	track.CoverArtLargeURL = utils.NormalizeArtworkURL(track.CoverArtLargeURL)
}

func inheritAlbumArtwork(track *database.Music, album *database.Album) {
	if track.CoverArtURL == "" {
		track.CoverArtURL = album.CoverArtURL
	}
	if track.CoverArtSmallURL == "" {
		track.CoverArtSmallURL = firstArtworkURL(album.CoverArtSmallURL, album.CoverArtURL)
	}
	if track.CoverArtMediumURL == "" {
		track.CoverArtMediumURL = firstArtworkURL(album.CoverArtMediumURL, album.CoverArtURL)
	}
	if track.CoverArtLargeURL == "" {
		track.CoverArtLargeURL = firstArtworkURL(album.CoverArtLargeURL, album.CoverArtURL)
	}
	if track.CoverArtSource == "" {
		track.CoverArtSource = album.CoverArtSource
	}
}

func firstArtworkURL(preferred, fallback string) string {
	if preferred != "" {
		return preferred
	}
	return fallback
}

// getArtists returns all artists
func (r *Router) getArtists(w http.ResponseWriter, req *http.Request) {
	artists, err := r.db.GetAllArtistsForLibrary()
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    artists,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// apiDocumentation returns API documentation
func (r *Router) apiDocumentation(w http.ResponseWriter, req *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Music Server API Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .method { font-weight: bold; padding: 3px 8px; border-radius: 3px; color: white; }
        .get { background-color: #61affe; }
        .post { background-color: #49cc90; }
        .put { background-color: #fca130; }
        .delete { background-color: #f93e3e; }
        .path { font-family: monospace; background-color: #f5f5f5; padding: 2px 5px; }
    </style>
</head>
<body>
    <h1>Music Server API Documentation</h1>
    
    <h2>Authentication</h2>
    <div class="endpoint">
        <span class="method post">POST</span> <span class="path">/api/auth/login</span> - User login
    </div>
    <div class="endpoint">
        <span class="method post">POST</span> <span class="path">/api/auth/register</span> - User registration
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/auth/me</span> - Get current user
    </div>
    
    <h2>Music Management</h2>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/music</span> - Get all music
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/music/search</span> - Search music
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/music/{id}</span> - Get specific track
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/music/{id}/stream</span> - Stream audio
    </div>
    
    <h2>Playlist Management</h2>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/playlists</span> - Get all playlists
    </div>
    <div class="endpoint">
        <span class="method post">POST</span> <span class="path">/api/playlists</span> - Create playlist
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/playlists/{id}</span> - Get playlist details
    </div>
    
    <h2>Admin Functions</h2>
    <div class="endpoint">
        <span class="method post">POST</span> <span class="path">/api/admin/scan</span> - Scan library
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/admin/stats</span> - Get server statistics
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/admin/users</span> - Get all users
    </div>
    <div class="endpoint">
        <span class="method put">PUT</span> <span class="path">/api/admin/users/{id}</span> - Update user role
    </div>
    <div class="endpoint">
        <span class="method delete">DELETE</span> <span class="path">/api/admin/users/{id}</span> - Delete user
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/admin/scans</span> - Get scan history
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/admin/scans/{id}</span> - Get scan details
    </div>
    
    <h2>WebSocket</h2>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/ws</span> - WebSocket connection
    </div>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/api/admin/ws/scan</span> - Admin WebSocket for scanning
    </div>
    
    <h2>System</h2>
    <div class="endpoint">
        <span class="method get">GET</span> <span class="path">/health</span> - Health check
    </div>
    
    <p><strong>Note:</strong> All API endpoints except login, register, and health check require JWT authentication in Authorization header.</p>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// getAlbumTracks returns tracks for a specific album with fallback to similar albums
func (r *Router) getAlbumTracks(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	albumName := vars["name"]

	if albumName == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Album name is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// URL decode album name to handle special characters like /
	decodedAlbumName, err := url.QueryUnescape(albumName)
	if err != nil {
		// If decoding fails, try the original name
		decodedAlbumName = albumName
	}

	// Try exact match first, then fall back to similar albums
	tracks, similarAlbums, err := r.db.GetAlbumTracksWithFallback(decodedAlbumName, "")
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// If we have exact match tracks, return them
	if len(tracks) > 0 {
		// Get album info - determine artist from tracks
		var albumInfo map[string]interface{}
		if len(tracks) > 0 {
			// Count unique artists for this album
			artists := make(map[string]bool)
			var year int
			for _, track := range tracks {
				if track.Artist != "" {
					artists[track.Artist] = true
				}
				if track.Year > year {
					year = track.Year
				}
			}

			var artistName string
			if len(artists) > 1 {
				artistName = "Various Artists"
			} else if len(artists) == 1 {
				// Get the single artist name
				for artist := range artists {
					artistName = artist
					break
				}
			} else {
				artistName = "Unknown Artist"
			}

			albumInfo = map[string]interface{}{
				"name":   tracks[0].Album,
				"artist": artistName,
				"year":   year,
			}
		}

		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"album":  albumInfo,
				"tracks": tracks,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// If no exact match, return similar albums suggestions
	response := map[string]interface{}{
		"success": false,
		"error":   "Album not found",
		"data": map[string]interface{}{
			"message":        "Album not found. Did you mean one of these?",
			"searched_for":   decodedAlbumName,
			"similar_albums": similarAlbums,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(response)
}

// getArtistTracks returns tracks for a specific artist
func (r *Router) getArtistTracks(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	artistName := vars["name"]

	if artistName == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Artist name is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// URL decode artist name to handle special characters
	decodedArtistName, err := url.QueryUnescape(artistName)
	if err != nil {
		// If decoding fails, try the original name
		decodedArtistName = artistName
	}

	// Use existing GetArtistTracks method from database
	tracks, err := r.db.GetArtistTracks(decodedArtistName)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get unique albums for this artist
	albums := make(map[string]bool)
	for _, track := range tracks {
		if track.Album != "" {
			albums[track.Album] = true
		}
	}

	// Convert albums map to slice
	albumList := make([]string, 0, len(albums))
	for album := range albums {
		albumList = append(albumList, album)
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"artist": map[string]interface{}{
				"name":        decodedArtistName,
				"track_count": len(tracks),
				"album_count": len(albumList),
			},
			"tracks": tracks,
			"albums": albumList,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// comprehensiveSearch performs a comprehensive search across music, albums, artists, and playlists
func (r *Router) comprehensiveSearch(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query().Get("q")
	if query == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Search query is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Search for songs
	songs, err := r.db.SearchMusic(query)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to search songs: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Search for albums
	albums, err := r.searchAlbums(query)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to search albums: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Search for artists
	artists, err := r.searchArtists(query)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to search artists: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Search for playlists
	playlists, err := r.searchPlaylists(query)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to search playlists: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"songs":     songs,
			"albums":    albums,
			"artists":   artists,
			"playlists": playlists,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// searchAlbums searches for albums by name or artist
func (r *Router) searchAlbums(query string) ([]map[string]interface{}, error) {
	// Get all albums and filter them
	allAlbums, err := r.db.GetAllAlbums()
	if err != nil {
		return nil, fmt.Errorf("failed to get albums: %v", err)
	}

	var filteredAlbums []map[string]interface{}
	searchLower := strings.ToLower(query)

	for _, album := range allAlbums {
		if strings.Contains(strings.ToLower(album.Name), searchLower) ||
			strings.Contains(strings.ToLower(album.Artist), searchLower) {

			albumMap := map[string]interface{}{
				"id":                   album.ID,
				"name":                 album.Name,
				"artist":               album.Artist,
				"year":                 album.Year,
				"track_count":          album.TrackCount,
				"cover_art_url":        album.CoverArtURL,
				"cover_art_small_url":  album.CoverArtSmallURL,
				"cover_art_medium_url": album.CoverArtMediumURL,
				"cover_art_large_url":  album.CoverArtLargeURL,
				"cover_art_source":     album.CoverArtSource,
			}
			filteredAlbums = append(filteredAlbums, albumMap)
		}
	}

	// Limit to 50 results
	if len(filteredAlbums) > 50 {
		filteredAlbums = filteredAlbums[:50]
	}

	return filteredAlbums, nil
}

// searchArtists searches for artists by name
func (r *Router) searchArtists(query string) ([]map[string]interface{}, error) {
	// Get all artists and filter them
	allArtists, err := r.db.GetAllArtistsForLibrary()
	if err != nil {
		return nil, fmt.Errorf("failed to get artists: %v", err)
	}

	var filteredArtists []map[string]interface{}
	searchLower := strings.ToLower(query)

	for _, artist := range allArtists {
		name, _ := artist["name"].(string)
		if strings.Contains(strings.ToLower(name), searchLower) {
			filteredArtists = append(filteredArtists, artist)
		}
	}

	// Limit to 50 results
	if len(filteredArtists) > 50 {
		filteredArtists = filteredArtists[:50]
	}

	return filteredArtists, nil
}

// searchPlaylists searches for playlists by name or description
func (r *Router) searchPlaylists(query string) ([]database.Playlist, error) {
	// Get all playlists and filter them
	allPlaylists, err := r.db.GetAllPlaylists()
	if err != nil {
		return nil, fmt.Errorf("failed to get playlists: %v", err)
	}

	var filteredPlaylists []database.Playlist
	searchLower := strings.ToLower(query)

	for _, playlist := range allPlaylists {
		if strings.Contains(strings.ToLower(playlist.Name), searchLower) ||
			strings.Contains(strings.ToLower(playlist.Description), searchLower) {
			filteredPlaylists = append(filteredPlaylists, playlist)
		}
	}

	// Limit to 50 results
	if len(filteredPlaylists) > 50 {
		filteredPlaylists = filteredPlaylists[:50]
	}

	return filteredPlaylists, nil
}

// getLikedTracks returns liked tracks for current user
func (r *Router) getLikedTracks(w http.ResponseWriter, req *http.Request) {
	// Get user ID from JWT token context
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to get user information",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get liked tracks for user
	tracks, err := r.db.GetLikedTracks(userID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get liked tracks: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    tracks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// likeTrack adds a track to user's liked tracks
func (r *Router) likeTrack(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	trackID := vars["id"]

	if trackID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Track ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the current user from JWT token
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to get user information",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Add track to user's liked tracks
	err = r.db.LikeTrack(userID, trackID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to like track: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Track liked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// checkTrackLiked checks if a track is liked by current user
func (r *Router) checkTrackLiked(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	trackID := vars["id"]

	if trackID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Track ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the current user from JWT token
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to get user information",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if track is liked
	isLiked, err := r.db.IsTrackLiked(userID, trackID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to check if track is liked: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"is_liked": isLiked,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// unlikeTrack removes a track from user's liked tracks
func (r *Router) unlikeTrack(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	trackID := vars["id"]

	if trackID == "" {
		response := map[string]interface{}{
			"success": false,
			"error":   "Track ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the current user from JWT token
	userID, err := auth.GetUserFromContext(req)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to get user information",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Remove track from user's liked tracks
	err = r.db.UnlikeTrack(userID, trackID)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to unlike track: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Track unliked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
