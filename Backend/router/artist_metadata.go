package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"music-server/artistmeta"
	"music-server/database"

	"github.com/gorilla/mux"
)

func (r *Router) lookupArtistMetadata(w http.ResponseWriter, req *http.Request) {
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "Artist name is required")
		return
	}

	artist, err := r.db.GetOrCreateArtistByHash(name)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result, err := r.refreshArtistMetadata(req.Context(), artist)
	if err != nil && result == nil {
		if primary, cachedErr := r.db.GetPrimaryArtistImage(artist.ID); cachedErr == nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"artist": artist,
					"image":  primary,
				},
			})
			return
		}
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": result})
}

func (r *Router) getArtistImage(w http.ResponseWriter, req *http.Request) {
	artistID := mux.Vars(req)["id"]
	artist, err := r.db.GetArtistByHash(artistID)
	if err != nil {
		artist, err = r.db.GetLibraryArtistByID(artistID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
	}
	tracks, trackErr := r.db.GetArtistTracks(artist.Name)
	if trackErr == nil {
		tracks, trackErr = r.filterMusicForRequest(req, tracks)
	}
	if trackErr != nil || len(tracks) == 0 {
		writeJSONError(w, http.StatusNotFound, "Artist not found")
		return
	}
	image, err := r.db.GetPrimaryArtistImage(artist.ID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"artist_id": artist.ID,
				"image":     nil,
				"fallback":  true,
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": image})
}

func (r *Router) refreshArtistMetadataEndpoint(w http.ResponseWriter, req *http.Request) {
	artist, err := r.db.GetArtistByHash(mux.Vars(req)["id"])
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	result, err := r.refreshArtistMetadata(req.Context(), artist)
	if err != nil && result == nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": result})
}

func (r *Router) listArtistImageCandidates(w http.ResponseWriter, req *http.Request) {
	artist, err := r.db.GetArtistByHash(mux.Vars(req)["id"])
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	images, err := r.db.ListArtistImages(artist.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": images})
}

func (r *Router) setPrimaryArtistImage(w http.ResponseWriter, req *http.Request) {
	artist, err := r.db.GetArtistByHash(mux.Vars(req)["id"])
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	var payload struct {
		ImageID int64 `json:"image_id"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload.ImageID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "image_id is required")
		return
	}
	if err := r.db.SetPrimaryArtistImage(artist.ID, payload.ImageID); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	image, _ := r.db.GetPrimaryArtistImage(artist.ID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": image})
}

func (r *Router) updateArtistImageAttribution(w http.ResponseWriter, req *http.Request) {
	artist, err := r.db.GetArtistByHash(mux.Vars(req)["id"])
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	imageID, err := strconv.ParseInt(mux.Vars(req)["imageId"], 10, 64)
	if err != nil || imageID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Image ID is required")
		return
	}
	var payload struct {
		AuthorName      string `json:"author_name"`
		AttributionText string `json:"attribution_text"`
		LicenseName     string `json:"license_name"`
		LicenseURL      string `json:"license_url"`
		SourcePageURL   string `json:"source_page_url"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid attribution details")
		return
	}
	images, err := r.db.ListArtistImages(artist.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, image := range images {
		if image.ID == imageID {
			image.AuthorName = payload.AuthorName
			image.AttributionText = payload.AttributionText
			image.LicenseName = payload.LicenseName
			image.LicenseURL = payload.LicenseURL
			image.SourcePageURL = payload.SourcePageURL
			if image.AttributionText == "" {
				image.AttributionText = artistmeta.BuildAttribution(image.AuthorName, image.LicenseName, image.SourcePageURL)
			}
			if err := r.db.UpsertArtistImage(&image); err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": image})
			return
		}
	}
	writeJSONError(w, http.StatusNotFound, "Artist image not found")
}

func (r *Router) refreshArtistMetadata(ctx context.Context, artist *database.Artist) (*artistmeta.LookupResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	pipeline := artistmeta.NewPipeline(r.db)
	result, err := pipeline.Lookup(ctx, artist.Name)
	if result == nil {
		return nil, err
	}

	if result.Artist.MBID != "" {
		artist.MusicBrainzID = result.Artist.MBID
		artist.MusicBrainzURL = "https://musicbrainz.org/artist/" + result.Artist.MBID
		artist.Country = result.Artist.Country
		artist.Tags = result.Artist.Tags
		now := time.Now()
		artist.LastEnrichedAt = &now
		if updateErr := r.db.UpdateArtist(artist); updateErr != nil {
			log.Printf("artist_metadata update_artist artist_id=%s error=%v", artist.ID, updateErr)
		}
		_ = r.db.UpsertArtistExternalID(artist.ID, "musicbrainz", result.Artist.MBID, artist.MusicBrainzURL)
	}
	if result.Artist.WikidataID != "" {
		_ = r.db.UpsertArtistExternalID(artist.ID, "wikidata", result.Artist.WikidataID, "https://www.wikidata.org/wiki/"+result.Artist.WikidataID)
	}

	existingPrimary, primaryErr := r.db.GetPrimaryArtistImage(artist.ID)
	for i := range result.Candidates {
		result.Candidates[i].ArtistID = artist.ID
		result.Candidates[i].IsPrimary = primaryErr != nil && i == 0
		dbImage := artistMetaCandidateToDB(result.Candidates[i])
		if saveErr := r.db.UpsertArtistImage(&dbImage); saveErr != nil {
			log.Printf("artist_metadata save_image artist_id=%s source=%s error=%v", artist.ID, result.Candidates[i].Source, saveErr)
			continue
		}
		result.Candidates[i] = dbArtistImageToCandidate(dbImage)
		if result.Candidates[i].IsPrimary {
			result.Image = &result.Candidates[i]
		}
	}
	if result.Image == nil && existingPrimary != nil {
		candidate := dbArtistImageToCandidate(*existingPrimary)
		result.Image = &candidate
	}
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return result, fmt.Errorf("artist metadata lookup timed out")
	}
	return result, err
}

func (r *Router) startArtistMetadataRefreshLoop() {
	if strings.ToLower(strings.TrimSpace(os.Getenv("ARTIST_METADATA_REFRESH_ENABLED"))) != "true" {
		return
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		r.refreshStaleArtistMetadataBatch()
		for range ticker.C {
			r.refreshStaleArtistMetadataBatch()
		}
	}()
}

func (r *Router) refreshStaleArtistMetadataBatch() {
	artists, err := r.db.GetArtistsNeedingMetadataRefresh(25, 30*24*time.Hour)
	if err != nil {
		log.Printf("artist_metadata_refresh status=failed error=%v", err)
		return
	}
	for index := range artists {
		artist := artists[index]
		if _, err := r.refreshArtistMetadata(context.Background(), &artist); err != nil {
			log.Printf("artist_metadata_refresh artist_id=%s artist=%q status=failed error=%v", artist.ID, artist.Name, err)
			continue
		}
		log.Printf("artist_metadata_refresh artist_id=%s artist=%q status=completed", artist.ID, artist.Name)
	}
}

func artistMetaCandidateToDB(candidate artistmeta.ImageCandidate) database.ArtistImage {
	return database.ArtistImage{
		ArtistID:        candidate.ArtistID,
		Source:          candidate.Source,
		ImageURL:        candidate.ImageURL,
		ThumbnailURL:    candidate.ThumbnailURL,
		SourcePageURL:   candidate.SourcePageURL,
		LicenseName:     candidate.LicenseName,
		LicenseURL:      candidate.LicenseURL,
		AuthorName:      candidate.AuthorName,
		AttributionText: candidate.AttributionText,
		Width:           candidate.Width,
		Height:          candidate.Height,
		MimeType:        candidate.MimeType,
		ConfidenceScore: candidate.ConfidenceScore,
		IsPrimary:       candidate.IsPrimary,
	}
}

func dbArtistImageToCandidate(image database.ArtistImage) artistmeta.ImageCandidate {
	return artistmeta.ImageCandidate{
		ArtistID:        image.ArtistID,
		Source:          image.Source,
		ImageURL:        image.ImageURL,
		ThumbnailURL:    image.ThumbnailURL,
		SourcePageURL:   image.SourcePageURL,
		LicenseName:     image.LicenseName,
		LicenseURL:      image.LicenseURL,
		AuthorName:      image.AuthorName,
		AttributionText: image.AttributionText,
		Width:           image.Width,
		Height:          image.Height,
		MimeType:        image.MimeType,
		ConfidenceScore: image.ConfidenceScore,
		IsPrimary:       image.IsPrimary,
		CreatedAt:       image.CreatedAt,
		UpdatedAt:       image.UpdatedAt,
	}
}
