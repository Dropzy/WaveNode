package enrichment

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"music-server/database"
)

// Handler handles enrichment-related HTTP requests
type Handler struct {
	db         *database.DB
	scanStore  *database.ScanStore
	scanner    *EnrichmentScanner
	scannerMux sync.Mutex
}

// NewHandler creates a new enrichment handler
func NewHandler(db *database.DB, scanStore *database.ScanStore) *Handler {
	return &Handler{
		db:        db,
		scanStore: scanStore,
	}
}

// EnrichArtistsFromSpotify starts artist enrichment from Spotify
func (h *Handler) EnrichArtistsFromSpotify(w http.ResponseWriter, r *http.Request) {
	// Get Spotify credentials from the enrichment scanner
	h.scannerMux.Lock()
	if h.scanner == nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Enrichment scanner not initialized",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		h.scannerMux.Unlock()
		return
	}
	h.scannerMux.Unlock()

	// Start artist enrichment scan
	scan, err := h.scanStore.CreateScan("artist-enrichment")
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to create scan: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Start scan in background
	go func() {
		scanResult := h.scanner.StartArtistEnrichment()
		// Broadcast updates as scan progresses
		for {
			scanStatus, err := h.scanStore.GetScan(scanResult.ID)
			if err != nil {
				log.Printf("Error getting scan status: %v", err)
				break
			}

			// Broadcast to WebSocket clients would go here
			// For now, just log progress
			log.Printf("Artist enrichment progress: %s - %d%%", scanStatus.Status, scanStatus.Progress)

			if scanStatus.Status == "completed" || scanStatus.Status == "failed" || scanStatus.Status == "completed_with_errors" {
				break
			}

			// In a real implementation, you'd have a proper notification system
			// For now, just wait a bit between checks
			select {
			case <-r.Context().Done():
				return
			default:
			}
		}
	}()

	response := map[string]interface{}{
		"success": true,
		"message": "Artist enrichment scan started",
		"data":    scan,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// EnrichMissingMetadata starts metadata enrichment
func (h *Handler) EnrichMissingMetadata(w http.ResponseWriter, r *http.Request) {
	// Get Spotify credentials from the enrichment scanner
	h.scannerMux.Lock()
	if h.scanner == nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Enrichment scanner not initialized",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		h.scannerMux.Unlock()
		return
	}
	h.scannerMux.Unlock()

	// Start metadata enrichment scan
	scan, err := h.scanStore.CreateScan("metadata-enrichment")
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to create scan: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Start scan in background
	go func() {
		scanResult := h.scanner.StartMetadataEnrichment()
		// Broadcast updates as scan progresses
		for {
			scanStatus, err := h.scanStore.GetScan(scanResult.ID)
			if err != nil {
				log.Printf("Error getting scan status: %v", err)
				break
			}

			// Broadcast to WebSocket clients would go here
			// For now, just log progress
			log.Printf("Metadata enrichment progress: %s - %d%%", scanStatus.Status, scanStatus.Progress)

			if scanStatus.Status == "completed" || scanStatus.Status == "failed" || scanStatus.Status == "completed_with_errors" {
				break
			}

			// In a real implementation, you'd have a proper notification system
			// For now, just wait a bit between checks
			select {
			case <-r.Context().Done():
				return
			default:
			}
		}
	}()

	response := map[string]interface{}{
		"success": true,
		"message": "Metadata enrichment scan started",
		"data":    scan,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetEnrichmentStatus returns the current enrichment status
func (h *Handler) GetEnrichmentStatus(w http.ResponseWriter, r *http.Request) {
	h.scannerMux.Lock()
	defer h.scannerMux.Unlock()

	if h.scanner == nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Enrichment scanner not initialized",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get enrichment statistics
	statistics, err := h.db.GetEnrichmentStatistics()
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Failed to get enrichment statistics: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Enrichment status retrieved successfully",
		"data":    statistics,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// SetScanner sets the enrichment scanner for the handler
func (h *Handler) SetScanner(scanner *EnrichmentScanner) {
	h.scannerMux.Lock()
	defer h.scannerMux.Unlock()
	h.scanner = scanner
}
