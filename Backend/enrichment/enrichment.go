package enrichment

import (
	"fmt"
	"log"
	"music-server/database"
	"music-server/metadata"
	"music-server/musicbrainz"
	"music-server/websocket"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// EnrichmentScanner handles artist image enrichment using MusicBrainz
type EnrichmentScanner struct {
	db          *database.DB
	musicBrainz *musicbrainz.MusicBrainzClient
	scanStore   *database.ScanStore
}

// EnrichmentResult holds the result of an enrichment operation
type EnrichmentResult struct {
	TrackID      string `json:"track_id"`
	ArtistName   string `json:"artist_name"`
	Success      bool   `json:"success"`
	ImageURL     string `json:"image_url,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// NewEnrichmentScanner creates a new enrichment scanner
func NewEnrichmentScanner(db *database.DB, scanStore *database.ScanStore) *EnrichmentScanner {
	musicBrainzClient := musicbrainz.NewMusicBrainzClient()

	// Set user agent for MusicBrainz API (required)
	musicBrainzClient.SetUserAgent("MusicServer", "1.0", "musicserver@example.com")

	return &EnrichmentScanner{
		db:          db,
		musicBrainz: musicBrainzClient,
		scanStore:   scanStore,
	}
}

// StartEnrichmentScan starts a new enrichment scan
func (es *EnrichmentScanner) StartEnrichmentScan() *database.ScanStatus {
	scan, err := es.scanStore.CreateScan("enrichment")
	if err != nil {
		log.Printf("Failed to create enrichment scan: %v", err)
		return nil
	}

	go es.performEnrichmentScan(scan.ID)

	return scan
}

// StartCoverArtEnrichment starts a new cover art enrichment scan
func (es *EnrichmentScanner) StartCoverArtEnrichment() *database.ScanStatus {
	scan, err := es.scanStore.CreateScan("cover-art-enrichment")
	if err != nil {
		log.Printf("Failed to create cover art enrichment scan: %v", err)
		return nil
	}

	go es.performCoverArtEnrichment(scan.ID)

	return scan
}

// broadcastScanUpdate broadcasts current scan status via WebSocket
func (es *EnrichmentScanner) broadcastScanUpdate(scanID string) {
	// Get current scan status
	scan, err := es.scanStore.GetScan(scanID)
	if err != nil {
		log.Printf("Failed to get scan status for broadcast: %v", err)
		return
	}

	// Broadcast to all connected clients
	websocket.BroadcastScanUpdate(scan)

	log.Printf("Broadcasted enrichment scan update: ID=%s, Status=%s, Progress=%d%%",
		scan.ID, scan.Status, scan.Progress)
}

// performEnrichmentScan performs actual enrichment scan
func (es *EnrichmentScanner) performEnrichmentScan(scanID string) {
	log.Printf("Starting enrichment scan with ID: %s", scanID)

	// Test connection
	if err := es.musicBrainz.TestConnection(); err != nil {
		es.scanStore.UpdateScan(scanID, "failed", 0)
		return
	}

	// Get all music tracks
	tracks, err := es.db.GetAllMusic()
	if err != nil {
		es.scanStore.UpdateScan(scanID, "failed", 0)
		return
	}

	log.Printf("Retrieved %d tracks from database", len(tracks))

	// Get tracks that specifically need enrichment
	tracksToEnrich := es.getTracksNeedingEnrichment(tracks)

	totalTracks := len(tracksToEnrich)
	if totalTracks == 0 {
		es.scanStore.UpdateScan(scanID, "completed", 100)
		return
	}

	log.Printf("Found %d tracks that need enrichment", totalTracks)

	// Get unique artists from tracks that need enrichment
	uniqueArtists := es.getUniqueArtists(tracksToEnrich)
	totalArtists := len(uniqueArtists)

	log.Printf("Found %d unique artists to enrich", totalArtists)

	// Process each artist
	results := make([]EnrichmentResult, 0)
	errors := make([]string, 0)
	processedArtists := 0
	var mutex sync.Mutex

	// Use a semaphore to limit concurrent requests
	semaphore := make(chan struct{}, 3) // Max 3 concurrent requests (MusicBrainz has stricter rate limits)
	var wg sync.WaitGroup

	for artistName := range uniqueArtists {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Get artist info from MusicBrainz with retry logic
			var artistInfo *musicbrainz.ArtistInfo
			err := es.retryWithBackoff(func() error {
				var retryErr error
				artistInfo, retryErr = es.musicBrainz.GetBestMatchForArtist(name)
				return retryErr
			}, fmt.Sprintf("GetBestMatchForArtist(%s)", name))

			mutex.Lock()
			defer mutex.Unlock()

			if err != nil {
				errorMsg := fmt.Sprintf("Failed to get info for artist '%s': %v", name, err)
				log.Print(errorMsg)
				errors = append(errors, errorMsg)

				// Create failure result for all tracks by this artist
				for _, track := range uniqueArtists[name] {
					results = append(results, EnrichmentResult{
						TrackID:      track.ID,
						ArtistName:   name,
						Success:      false,
						ErrorMessage: err.Error(),
					})
				}
			} else {
				log.Printf("Found MusicBrainz info for artist '%s': %s (ID: %s)", name, artistInfo.Name, artistInfo.ID)

				// Check if this MusicBrainz ID is already used by another artist
				existingArtistByMBID, err := es.db.GetArtistByMusicBrainzID(artistInfo.ID)
				if err == nil && existingArtistByMBID != nil {
					// MusicBrainz ID already exists, use that artist instead
					log.Printf("MusicBrainz ID %s already exists for artist '%s', using existing artist", artistInfo.ID, existingArtistByMBID.Name)
					artist := existingArtistByMBID

					// Update all tracks by this artist with existing artist ID
					for _, track := range uniqueArtists[name] {
						track.ArtistID = artist.ID

						// Update track in database
						if err := es.db.UpdateMusic(&track); err != nil {
							errorMsg := fmt.Sprintf("Failed to update track '%s' with artist info: %v", track.ID, err)
							log.Print(errorMsg)
							errors = append(errors, errorMsg)

							results = append(results, EnrichmentResult{
								TrackID:      track.ID,
								ArtistName:   name,
								Success:      false,
								ErrorMessage: fmt.Sprintf("Track database update failed: %v", err),
							})
						} else {
							results = append(results, EnrichmentResult{
								TrackID:    track.ID,
								ArtistName: name,
								Success:    true,
								ImageURL:   artist.ImageURL,
							})
						}
					}
				} else {
					// Use proper ArtistOrStoreArtist function to handle duplicates correctly
					existingArtist, err := es.db.ArtistOrStoreArtist(name)
					if err != nil {
						errorMsg := fmt.Sprintf("Failed to get or create artist '%s': %v", name, err)
						log.Print(errorMsg)
						errors = append(errors, errorMsg)

						// Create failure result for all tracks by this artist
						for _, track := range uniqueArtists[name] {
							results = append(results, EnrichmentResult{
								TrackID:      track.ID,
								ArtistName:   name,
								Success:      false,
								ErrorMessage: fmt.Sprintf("Artist database operation failed: %v", err),
							})
						}
						processedArtists++
						es.scanStore.UpdateScanWithFile(scanID, name, processedArtists, totalArtists)
						return
					}

					// Now update existing/new artist with MusicBrainz info if needed
					artist := existingArtist
					needsUpdate := false

					if artist.MusicBrainzID == "" {
						// No MusicBrainz ID, need to add all MusicBrainz info
						needsUpdate = true
						artist.MusicBrainzID = artistInfo.ID
						artist.ImageURL = artistInfo.ImageURL
						artist.Country = artistInfo.Country
						artist.Tags = artistInfo.Tags
						now := time.Now()
						artist.LastEnrichedAt = &now
						log.Printf("Updating artist '%s' with missing MusicBrainz info", artist.Name)
					} else if artist.ImageURL == "" {
						// Has MusicBrainz ID but missing image URL, update with image info
						needsUpdate = true
						artist.ImageURL = artistInfo.ImageURL
						artist.Country = artistInfo.Country
						artist.Tags = artistInfo.Tags
						now := time.Now()
						artist.LastEnrichedAt = &now
						log.Printf("Updating artist '%s' with missing image data", artist.Name)
					}

					if needsUpdate {
						artist.UpdatedAt = time.Now()
						err = es.db.UpdateArtist(artist)
						if err != nil {
							errorMsg := fmt.Sprintf("Failed to update artist '%s' with MusicBrainz info: %v", name, err)
							log.Print(errorMsg)
							errors = append(errors, errorMsg)

							// Create failure result for all tracks by this artist
							for _, track := range uniqueArtists[name] {
								results = append(results, EnrichmentResult{
									TrackID:      track.ID,
									ArtistName:   name,
									Success:      false,
									ErrorMessage: fmt.Sprintf("Artist update failed: %v", err),
								})
							}
							processedArtists++
							es.scanStore.UpdateScanWithFile(scanID, name, processedArtists, totalArtists)
							return
						}
						log.Printf("Successfully updated artist '%s' with MusicBrainz data", artist.Name)
					} else {
						log.Printf("Artist '%s' already has complete MusicBrainz info, skipping update", artist.Name)
					}

					// Update all tracks by this artist with artist ID
					for _, track := range uniqueArtists[name] {
						track.ArtistID = artist.ID

						// Update track in database
						if err := es.db.UpdateMusic(&track); err != nil {
							errorMsg := fmt.Sprintf("Failed to update track '%s' with artist info: %v", track.ID, err)
							log.Print(errorMsg)
							errors = append(errors, errorMsg)

							results = append(results, EnrichmentResult{
								TrackID:      track.ID,
								ArtistName:   name,
								Success:      false,
								ErrorMessage: fmt.Sprintf("Track database update failed: %v", err),
							})
						} else {
							results = append(results, EnrichmentResult{
								TrackID:    track.ID,
								ArtistName: name,
								Success:    true,
								ImageURL:   artistInfo.ImageURL,
							})
						}
					}
				}
			}

			processedArtists++
			es.scanStore.UpdateScanWithFile(scanID, name, processedArtists, totalArtists)

		}(artistName)
	}

	wg.Wait()

	// Final update
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	status := "completed"
	if len(errors) > 0 && successCount == 0 {
		status = "failed"
	} else if len(errors) > 0 {
		status = "completed_with_errors"
	}

	es.scanStore.UpdateScan(scanID, status, 100)

	log.Printf("Enrichment scan completed. Successfully processed %d/%d artists (%d tracks), %d errors",
		successCount, totalArtists, successCount, len(errors))
}

// performCoverArtEnrichment performs actual cover art enrichment scan
func (es *EnrichmentScanner) performCoverArtEnrichment(scanID string) {
	log.Printf("Starting cover art enrichment scan with ID: %s", scanID)

	// Get all music tracks that need cover art
	tracks, err := es.db.GetAllMusic()
	if err != nil {
		es.scanStore.UpdateScan(scanID, "failed", 0)
		return
	}

	log.Printf("Retrieved %d tracks from database", len(tracks))

	// Get tracks that specifically need cover art
	tracksToEnrich := es.getTracksNeedingCoverArt(tracks)

	totalTracks := len(tracksToEnrich)
	if totalTracks == 0 {
		es.scanStore.UpdateScan(scanID, "completed", 100)
		return
	}

	log.Printf("Found %d tracks that need cover art enrichment", totalTracks)

	// Get unique albums from tracks that need cover art
	uniqueAlbums := es.getUniqueAlbums(tracksToEnrich)
	totalAlbums := len(uniqueAlbums)

	log.Printf("Found %d unique albums to enrich with cover art", totalAlbums)

	// Process each album
	results := make([]EnrichmentResult, 0)
	errors := make([]string, 0)
	processedAlbums := 0
	var mutex sync.Mutex

	// Use a semaphore to limit concurrent requests
	semaphore := make(chan struct{}, 3) // Max 3 concurrent requests (MusicBrainz has stricter rate limits)
	var wg sync.WaitGroup

	for albumKey, albumInfo := range uniqueAlbums {
		wg.Add(1)
		go func(key string, info AlbumInfo) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			coverArt, err := es.findAlbumCoverArt(info)

			mutex.Lock()
			defer mutex.Unlock()

			if err != nil {
				errorMsg := fmt.Sprintf("Failed to get cover art for album '%s' by '%s': %v", info.Name, info.Artist, err)
				log.Print(errorMsg)
				errors = append(errors, errorMsg)

				// Create failure result for all tracks in this album
				for _, track := range info.Tracks {
					results = append(results, EnrichmentResult{
						TrackID:      track.ID,
						ArtistName:   track.Artist,
						Success:      false,
						ErrorMessage: err.Error(),
					})
				}
			} else {
				log.Printf("Found %s cover art for album '%s' by '%s'", coverArt.Source, info.Name, info.Artist)
				log.Printf("Cover art URLs - Small: %s, Medium: %s, Large: %s",
					coverArt.SmallURL, coverArt.MediumURL, coverArt.LargeURL)

				// Update all tracks in this album with cover art URLs in different sizes
				for _, track := range info.Tracks {
					track.CoverArtURL = coverArt.LargeURL
					track.CoverArtSmallURL = coverArt.SmallURL
					track.CoverArtMediumURL = coverArt.MediumURL
					track.CoverArtLargeURL = coverArt.LargeURL
					track.CoverArtSource = coverArt.Source
					track.LastCoverArtEnrichedAt = &coverArt.EnrichedAt
					if coverArt.Source == "embedded" {
						track.ImageURL = coverArt.LargeURL
					}

					log.Printf("Updating track %s with cover art - Main: %s, Small: %s, Medium: %s, Large: %s",
						track.ID, track.CoverArtURL, track.CoverArtSmallURL, track.CoverArtMediumURL, track.CoverArtLargeURL)

					// Update track in database
					if err := es.db.UpdateMusic(&track); err != nil {
						errorMsg := fmt.Sprintf("Failed to update track '%s' with cover art: %v", track.ID, err)
						log.Print(errorMsg)
						errors = append(errors, errorMsg)

						results = append(results, EnrichmentResult{
							TrackID:      track.ID,
							ArtistName:   track.Artist,
							Success:      false,
							ErrorMessage: fmt.Sprintf("Track database update failed: %v", err),
						})
					} else {
						results = append(results, EnrichmentResult{
							TrackID:    track.ID,
							ArtistName: track.Artist,
							Success:    true,
							ImageURL:   coverArt.LargeURL,
						})
					}
				}
			}

			processedAlbums++
			es.scanStore.UpdateScanWithFile(scanID, fmt.Sprintf("%s - %s", info.Artist, info.Name), processedAlbums, totalAlbums)

		}(albumKey, albumInfo)
	}

	wg.Wait()

	// Final update
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	status := "completed"
	if len(errors) > 0 && successCount == 0 {
		status = "failed"
	} else if len(errors) > 0 {
		status = "completed_with_errors"
	}

	es.scanStore.UpdateScan(scanID, status, 100)

	log.Printf("Cover art enrichment scan completed. Successfully processed %d/%d albums (%d tracks), %d errors",
		successCount, totalAlbums, successCount, len(errors))
}

type albumCoverArt struct {
	SmallURL   string
	MediumURL  string
	LargeURL   string
	Source     string
	EnrichedAt time.Time
}

func (es *EnrichmentScanner) findAlbumCoverArt(info AlbumInfo) (*albumCoverArt, error) {
	parser := metadata.NewMetadataParser()

	for _, track := range info.Tracks {
		if strings.TrimSpace(track.FilePath) == "" {
			continue
		}

		artworkData, artworkFormat, err := parser.ExtractArtwork(track.FilePath)
		if err != nil {
			log.Printf("Could not inspect embedded artwork in %s: %v", track.FilePath, err)
			continue
		}
		if len(artworkData) == 0 {
			continue
		}

		artworkPath, err := parser.SaveArtwork(artworkData, info.Artist, info.Name, artworkFormat)
		if err != nil {
			log.Printf("Could not save embedded artwork from %s: %v", track.FilePath, err)
			continue
		}

		artworkURL := "/artwork/" + filepath.Base(artworkPath)
		return &albumCoverArt{
			SmallURL:   artworkURL,
			MediumURL:  artworkURL,
			LargeURL:   artworkURL,
			Source:     "embedded",
			EnrichedAt: time.Now(),
		}, nil
	}

	var musicBrainzArt *musicbrainz.CoverArtInfo
	err := es.retryWithBackoff(func() error {
		var retryErr error
		musicBrainzArt, retryErr = es.musicBrainz.GetAlbumCoverArtSizes(info.Name, info.Artist)
		return retryErr
	}, fmt.Sprintf("GetAlbumCoverArtSizes(%s, %s)", info.Name, info.Artist))
	if err != nil {
		return nil, err
	}

	return &albumCoverArt{
		SmallURL:   musicBrainzArt.SmallURL,
		MediumURL:  musicBrainzArt.MediumURL,
		LargeURL:   musicBrainzArt.LargeURL,
		Source:     "musicbrainz",
		EnrichedAt: time.Now(),
	}, nil
}

// AlbumInfo holds album information for cover art enrichment
type AlbumInfo struct {
	Name   string
	Artist string
	Tracks []database.Music
}

// getTracksNeedingCoverArt filters tracks that need cover art
func (es *EnrichmentScanner) getTracksNeedingCoverArt(tracks []database.Music) []database.Music {
	tracksToEnrich := make([]database.Music, 0)

	for _, track := range tracks {
		needsEnrichment := false

		// Check if track is missing new cover art URL fields
		// We don't check the old CoverArt field since we want to upgrade all tracks to use new fields
		if track.CoverArtLargeURL == "" || track.CoverArtMediumURL == "" || track.CoverArtSmallURL == "" {
			// Track has missing cover art size URLs, needs enrichment
			log.Printf("Track %s: Missing cover art size URLs (large:%s, medium:%s, small:%s), needs enrichment",
				track.ID,
				boolToStr(track.CoverArtLargeURL != ""),
				boolToStr(track.CoverArtMediumURL != ""),
				boolToStr(track.CoverArtSmallURL != ""))
			needsEnrichment = true
		} else {
			log.Printf("Track %s: Already has all cover art sizes, skipping", track.ID)
		}

		if needsEnrichment {
			tracksToEnrich = append(tracksToEnrich, track)
		}
	}

	return tracksToEnrich
}

// Helper function to convert bool to string for logging
func boolToStr(b bool) string {
	if b {
		return "✓"
	}
	return "✗"
}

// getUniqueAlbums extracts unique albums from tracks and groups tracks by album
func (es *EnrichmentScanner) getUniqueAlbums(tracks []database.Music) map[string]AlbumInfo {
	uniqueAlbums := make(map[string]AlbumInfo)

	for _, track := range tracks {
		// Normalize album and artist names
		albumName := strings.TrimSpace(track.Album)
		artistName := strings.TrimSpace(track.Artist)

		if albumName == "" || artistName == "" {
			continue
		}

		// Create a unique key for the album (artist + album name)
		albumKey := fmt.Sprintf("%s - %s", artistName, albumName)

		// Check if this album already exists
		if albumInfo, exists := uniqueAlbums[albumKey]; exists {
			// Add track to existing album
			albumInfo.Tracks = append(albumInfo.Tracks, track)
			uniqueAlbums[albumKey] = albumInfo
		} else {
			// Create new album entry
			uniqueAlbums[albumKey] = AlbumInfo{
				Name:   albumName,
				Artist: artistName,
				Tracks: []database.Music{track},
			}
		}
	}

	return uniqueAlbums
}

// getTracksNeedingEnrichment filters tracks that need MusicBrainz enrichment
func (es *EnrichmentScanner) getTracksNeedingEnrichment(tracks []database.Music) []database.Music {
	tracksToEnrich := make([]database.Music, 0)

	for _, track := range tracks {
		needsEnrichment := false

		if track.ArtistID == "" {
			// Track has no artist ID, needs enrichment
			log.Printf("Track %s: No artist ID, needs enrichment", track.ID)
			needsEnrichment = true
		} else {
			// Track has artist ID, check if artist has complete MusicBrainz data including images
			artist, err := es.db.GetArtistByID(track.ArtistID)
			if err != nil {
				// Artist not found in artists table, this track needs enrichment
				log.Printf("Track %s: Artist ID %s not found in artists table, needs enrichment", track.ID, track.ArtistID)
				needsEnrichment = true
			} else if artist.MusicBrainzID == "" {
				// Artist exists but no MusicBrainz ID, needs enrichment
				log.Printf("Track %s: Artist %s exists but no MusicBrainz ID, needs enrichment", track.ID, artist.Name)
				needsEnrichment = true
			} else if artist.ImageURL == "" {
				// Artist has MusicBrainz ID but no image URL, needs enrichment
				log.Printf("Track %s: Artist %s has MusicBrainz ID %s but no image URL, needs enrichment", track.ID, artist.Name, artist.MusicBrainzID)
				needsEnrichment = true
			} else {
				log.Printf("Track %s: Artist %s already has MusicBrainz ID %s and image URL, skipping", track.ID, artist.Name, artist.MusicBrainzID)
			}
		}

		if needsEnrichment {
			tracksToEnrich = append(tracksToEnrich, track)
		}
	}

	return tracksToEnrich
}

// getUniqueArtists extracts unique artists from tracks and groups tracks by artist
func (es *EnrichmentScanner) getUniqueArtists(tracks []database.Music) map[string][]database.Music {
	uniqueArtists := make(map[string][]database.Music)

	for _, track := range tracks {
		// Normalize artist name
		artistName := strings.TrimSpace(track.Artist)
		if artistName == "" {
			continue
		}

		// Check if this artist already exists
		found := false
		for existingArtist := range uniqueArtists {
			// Simple name comparison - could be enhanced with fuzzy matching
			if strings.EqualFold(existingArtist, artistName) {
				uniqueArtists[existingArtist] = append(uniqueArtists[existingArtist], track)
				found = true
				break
			}
		}

		if !found {
			uniqueArtists[artistName] = []database.Music{track}
		}
	}

	return uniqueArtists
}

// GetEnrichmentResults gets detailed results of an enrichment scan
func (es *EnrichmentScanner) GetEnrichmentResults(scanID string) ([]EnrichmentResult, error) {
	// This would typically be stored during scan
	// For now, return a placeholder
	return []EnrichmentResult{}, nil
}

// TestMusicBrainzConnection tests MusicBrainz configuration
func (es *EnrichmentScanner) TestMusicBrainzConnection() error {
	if es.musicBrainz == nil {
		return fmt.Errorf("MusicBrainz client not initialized")
	}

	return es.musicBrainz.TestConnection()
}

// GetArtistInfo gets artist information for a specific artist name
func (es *EnrichmentScanner) GetArtistInfo(artistName string) (*musicbrainz.ArtistInfo, error) {
	if es.musicBrainz == nil {
		return nil, fmt.Errorf("MusicBrainz client not initialized")
	}

	return es.musicBrainz.GetBestMatchForArtist(artistName)
}

// StartArtistEnrichment starts artist enrichment from MusicBrainz
func (es *EnrichmentScanner) StartArtistEnrichment() *database.ScanStatus {
	scan, err := es.scanStore.CreateScan("artist-enrichment")
	if err != nil {
		log.Printf("Failed to create artist enrichment scan: %v", err)
		return nil
	}

	go func() {
		// Broadcast initial scan status
		es.broadcastScanUpdate(scan.ID)

		// Use existing performEnrichmentScan logic
		es.performEnrichmentScan(scan.ID)
	}()

	return scan
}

// StartMetadataEnrichment starts metadata enrichment
func (es *EnrichmentScanner) StartMetadataEnrichment() *database.ScanStatus {
	scan, err := es.scanStore.CreateScan("metadata-enrichment")
	if err != nil {
		log.Printf("Failed to create metadata enrichment scan: %v", err)
		return nil
	}

	go func() {
		// Use existing performEnrichmentScan logic for now
		// This could be enhanced to handle different types of metadata enrichment
		es.performEnrichmentScan(scan.ID)
	}()

	return scan
}

// retryWithBackoff executes a function with exponential backoff retry logic
func (es *EnrichmentScanner) retryWithBackoff(operation func() error, operationName string) error {
	maxRetries := 3
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		// Check if it's a rate limit error (HTTP 429 or 503)
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "503") ||
			strings.Contains(err.Error(), "Too Many Requests") || strings.Contains(err.Error(), "Service Unavailable") {

			if attempt < maxRetries {
				// Calculate exponential backoff delay
				delay := baseDelay * time.Duration(1<<uint(attempt))
				if delay > maxDelay {
					delay = maxDelay
				}

				log.Printf("Rate limit hit for %s, retrying in %v (attempt %d/%d)", operationName, delay, attempt+1, maxRetries+1)
				time.Sleep(delay)
				continue
			}
		}

		// For other errors or max retries exceeded, return error
		if attempt == maxRetries {
			log.Printf("Max retries exceeded for %s: %v", operationName, err)
		}
		return err
	}

	return nil
}
