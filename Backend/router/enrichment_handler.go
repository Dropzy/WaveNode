package router

import (
	"encoding/json"
	"log"
	"net/http"

	"music-server/database"
	"music-server/enrichment"
)

// EnrichmentHandler handles enrichment-related HTTP requests
type EnrichmentHandler struct {
	db                *database.DB
	scanStore         *database.ScanStore
	enrichmentScanner *enrichment.EnrichmentScanner
}

// NewEnrichmentHandler creates a new enrichment handler
func NewEnrichmentHandler(db *database.DB, scanStore *database.ScanStore) *EnrichmentHandler {
	enrichmentScanner := enrichment.NewEnrichmentScanner(db, scanStore)
	return &EnrichmentHandler{
		db:                db,
		scanStore:         scanStore,
		enrichmentScanner: enrichmentScanner,
	}
}

// SetEnrichmentScanner sets the enrichment scanner for the handler
func (h *EnrichmentHandler) SetEnrichmentScanner(scanner *enrichment.EnrichmentScanner) {
	h.enrichmentScanner = scanner
}

// EnrichMusicBrainz starts MusicBrainz enrichment (admin only)
func (h *EnrichmentHandler) EnrichMusicBrainz(w http.ResponseWriter, r *http.Request) {
	// Check if enrichment scanner is available
	if h.enrichmentScanner == nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Enrichment scanner not initialized",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Start MusicBrainz enrichment scan
	scan := h.enrichmentScanner.StartArtistEnrichment()

	response := map[string]interface{}{
		"success": true,
		"message": "MusicBrainz enrichment scan started",
		"data":    scan,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// EnrichCoverArt starts cover art enrichment (admin only)
func (h *EnrichmentHandler) EnrichCoverArt(w http.ResponseWriter, r *http.Request) {
	// Check if enrichment scanner is available
	if h.enrichmentScanner == nil {
		response := map[string]interface{}{
			"success": false,
			"error":   "Enrichment scanner not initialized",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Starting cover art enrichment scan")

	// Start cover art enrichment scan
	scan := h.enrichmentScanner.StartCoverArtEnrichment()

	response := map[string]interface{}{
		"success": true,
		"message": "Cover art enrichment scan started",
		"data":    scan,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
