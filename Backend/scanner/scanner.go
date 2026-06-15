package scanner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"music-server/database"
	"music-server/metadata"
	"music-server/websocket"
)

// Scanner handles music library scanning process
type Scanner struct {
	db          *database.DB
	musicPath   string
	scanStore   *database.ScanStore
	isScanning  bool
	scanMutex   sync.Mutex
	cancelScan  context.CancelFunc
	stopPending bool
	currentScan *database.ScanStatus
}

var errScanStopped = errors.New("scan stopped")

// NewScanner creates a new scanner instance
func NewScanner(db *database.DB, musicPath string) *Scanner {
	scanner := &Scanner{
		db:        db,
		musicPath: musicPath,
		scanStore: database.NewScanStore(db),
	}
	if err := scanner.scanStore.StopInterruptedLibraryScans(); err != nil {
		log.Printf("Warning: Failed to recover interrupted library scans: %v", err)
	}
	return scanner
}

// SetMusicPath sets the path to the music library
func (s *Scanner) SetMusicPath(path string) {
	s.musicPath = path
}

// StartScan starts a new scan of the music library
func (s *Scanner) StartScan() (*database.ScanStatus, error) {
	s.scanMutex.Lock()
	defer s.scanMutex.Unlock()

	if s.isScanning {
		return nil, fmt.Errorf("scan already in progress")
	}

	// Create new scan record using the scan store
	scan, err := s.scanStore.CreateScan("library")
	if err != nil {
		return nil, fmt.Errorf("failed to create scan record: %v", err)
	}

	// Ensure scan has proper ID with scan_ prefix
	if !strings.HasPrefix(scan.ID, "scan_") {
		scan.ID = fmt.Sprintf("scan_%s", scan.ID)
		log.Printf("Fixed scan ID to: %s", scan.ID)
	}

	s.currentScan = scan
	s.isScanning = true
	s.stopPending = false
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelScan = cancel

	// Send immediate WebSocket update
	s.broadcastScanUpdate(scan)

	// Start scanning in a goroutine
	go s.performScan(ctx, scan.ID)

	return scan, nil
}

// performScan executes the actual scanning process
func (s *Scanner) performScan(ctx context.Context, scanID string) {
	defer func() {
		s.scanMutex.Lock()
		s.isScanning = false
		s.stopPending = false
		s.cancelScan = nil
		s.currentScan = nil
		s.scanMutex.Unlock()
	}()

	log.Printf("Starting scan with ID: %s", scanID)

	// Get list of all music files
	files, activeSources, err := s.discoverMusicFiles(ctx, scanID)
	if err != nil {
		if errors.Is(err, errScanStopped) || errors.Is(err, context.Canceled) {
			s.finishStoppedScan(scanID)
			return
		}
		s.handleScanError(scanID, fmt.Sprintf("Failed to discover music files: %v", err))
		return
	}
	if s.scanCancelled(ctx) {
		s.finishStoppedScan(scanID)
		return
	}

	// Update scan with total files
	err = s.scanStore.UpdateScanWithFile(scanID, "", 0, len(files))
	if err != nil {
		log.Printf("Warning: Failed to update scan progress: %v", err)
	}

	validPaths := make(map[string]struct{}, len(files))
	for _, path := range files {
		validPaths[path] = struct{}{}
	}
	// Cleanup is only safe after a complete discovery pass. An incomplete file
	// list could otherwise cause valid indexed tracks to be deleted.
	if removed, cleanupErr := s.db.RemoveMissingMusic(validPaths, activeSources); cleanupErr != nil {
		log.Printf("Warning: Failed to remove missing tracks: %v", cleanupErr)
	} else if removed > 0 {
		log.Printf("Removed %d tracks whose source files no longer exist", removed)
	}
	if s.scanCancelled(ctx) {
		s.finishStoppedScan(scanID)
		return
	}

	// Scan files in batches
	const batchSize = 25
	var songsAdded, duplicates, skipped int

	for i := 0; i < len(files); i += batchSize {
		if s.scanCancelled(ctx) {
			s.finishStoppedScan(scanID)
			return
		}

		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		processed, added, dup, skip, batchErr := s.processBatch(ctx, scanID, batch)
		songsAdded += added
		duplicates += dup
		skipped += skip

		if batchErr != nil {
			s.updateScanProgress(scanID, batch, i+processed, len(files), songsAdded, duplicates, skipped)
			if errors.Is(batchErr, errScanStopped) || errors.Is(batchErr, context.Canceled) {
				s.finishStoppedScan(scanID)
				return
			}
			s.handleScanError(scanID, fmt.Sprintf("Error processing batch: %v", batchErr))
			return
		}

		s.updateScanProgress(scanID, batch, end, len(files), songsAdded, duplicates, skipped)
		progress := int((float64(end) / float64(len(files))) * 100)
		log.Printf("Scan progress: %d/%d files (%d%%), %d songs added, %d duplicates, %d skipped",
			end, len(files), progress, songsAdded, duplicates, skipped)
	}

	if err := s.db.SyncMusicUploadOrder(files); err != nil {
		log.Printf("Warning: Failed to synchronize upload order: %v", err)
	}

	if !s.claimCompletion(ctx) {
		s.finishStoppedScan(scanID)
		return
	}

	// Mark scan as completed - this is the key fix
	log.Printf("All batches processed, marking scan as completed")
	progress := 100
	totalSkipped := duplicates + skipped

	// Set completion time first to ensure timestamp is set
	err = s.scanStore.SetScanCompleted(scanID)
	if err != nil {
		log.Printf("Warning: Failed to set completion time: %v", err)
	}

	// Update final stats and mark as completed
	err = s.scanStore.UpdateScanWithStats(scanID, "completed", progress, songsAdded, duplicates)
	if err != nil {
		log.Printf("Warning: Failed to mark scan as completed: %v", err)
		// Fallback: try to set status separately
		err = s.scanStore.UpdateScan(scanID, "completed", progress)
		if err != nil {
			log.Printf("Error: Failed to set scan status to completed: %v", err)
		}
	}

	// Update final skip count
	err = s.scanStore.UpdateScanWithTracksSkipped(scanID, totalSkipped)
	if err != nil {
		log.Printf("Warning: Failed to update final skip count: %v", err)
	}

	// Wait a moment to ensure database is updated, then send final update
	time.Sleep(100 * time.Millisecond)

	// Get the completed scan directly to broadcast it
	completedScan, err := s.scanStore.GetScan(scanID)
	if err != nil {
		log.Printf("Warning: Failed to get completed scan for broadcast: %v", err)
		// Fallback to current scan method
		for i := 0; i < 3; i++ {
			s.broadcastCurrentScan()
			time.Sleep(50 * time.Millisecond)
		}
	} else {
		// Broadcast the completed scan directly multiple times
		for i := 0; i < 3; i++ {
			s.broadcastScanUpdate(completedScan)
			time.Sleep(50 * time.Millisecond)
		}
	}

	log.Printf("✅ Scan completed: %d files processed, %d songs added, %d duplicates, %d skipped",
		len(files), songsAdded, duplicates, skipped)
}

// discoverMusicFiles recursively finds all music files in the given directory
func (s *Scanner) discoverMusicFiles(ctx context.Context, scanID string) ([]string, []string, error) {
	fileSet := make(map[string]struct{})
	files := make([]string, 0)
	var fileCount int

	sources, err := s.db.GetMusicSources()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load music sources: %v", err)
	}
	if len(sources) == 0 && strings.TrimSpace(s.musicPath) != "" {
		sources = []database.MusicSource{{Path: s.musicPath}}
	}
	if len(sources) == 0 {
		return nil, nil, fmt.Errorf("no music sources configured")
	}

	validSources := 0
	activeSources := make([]string, 0, len(sources))
	for _, source := range sources {
		if s.scanCancelled(ctx) {
			return nil, nil, errScanStopped
		}
		log.Printf("Starting music file discovery in: %s", source.Path)

		info, statErr := os.Stat(source.Path)
		if statErr != nil || !info.IsDir() {
			log.Printf("Skipping unavailable music source %s: %v", source.Path, statErr)
			continue
		}
		validSources++
		activeSources = append(activeSources, source.Path)

		walkErr := filepath.WalkDir(source.Path, func(path string, d fs.DirEntry, walkErr error) error {
			if s.scanCancelled(ctx) {
				return errScanStopped
			}
			if walkErr != nil {
				log.Printf("Error accessing path %s: %v", path, walkErr)
				return nil
			}

			if d.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if s.isMusicFile(ext) {
				if _, exists := fileSet[path]; !exists {
					fileSet[path] = struct{}{}
					files = append(files, path)
					fileCount++

					if fileCount%10 == 0 {
						progress := int((float64(fileCount) / 1000.0) * 20)
						if progress > 20 {
							progress = 20
						}

						s.scanStore.UpdateScanProgress(scanID, progress)
						s.broadcastCurrentScan()
						log.Printf("Discovery progress: found %d music files so far", fileCount)
					}
				}
			}

			return nil
		})
		if walkErr != nil {
			if errors.Is(walkErr, errScanStopped) {
				return nil, nil, errScanStopped
			}
			log.Printf("Error walking music source %s: %v", source.Path, walkErr)
		}
	}

	if validSources == 0 {
		return nil, nil, fmt.Errorf("none of the configured music sources are accessible")
	}

	log.Printf("Discovery complete. Found %d music files", len(files))

	return files, activeSources, nil
}

// isMusicFile checks if the file extension indicates a music file
func (s *Scanner) isMusicFile(ext string) bool {
	musicExtensions := map[string]bool{
		".mp3":  true,
		".flac": true,
		".wav":  true,
		".m4a":  true,
		".aac":  true,
		".ogg":  true,
		".wma":  true,
		".opus": true,
	}
	return musicExtensions[ext]
}

// processBatch processes a batch of music files
func (s *Scanner) processBatch(ctx context.Context, scanID string, files []string) (int, int, int, int, error) {
	var processed int
	var songsAdded, duplicates, skipped int
	parser := metadata.NewMetadataParser()

	for index, filePath := range files {
		if s.scanCancelled(ctx) {
			return processed, songsAdded, duplicates, skipped, errScanStopped
		}

		// Keep the live current-file display useful without writing to the
		// database for every track.
		if index == 0 || index%5 == 0 {
			if err := s.scanStore.UpdateScanCurrentFile(scanID, filePath); err != nil {
				log.Printf("Warning: Failed to update current file: %v", err)
			}
		}

		// Parse metadata
		trackInfo, err := parser.ExtractMetadata(filePath)
		if err != nil {
			log.Printf("Warning: Failed to parse metadata for %s: %v", filePath, err)
			processed++
			continue
		}

		artworkData := trackInfo.ArtworkData
		artworkFormat := trackInfo.ArtworkFormat
		if s.scanCancelled(ctx) {
			return processed, songsAdded, duplicates, skipped, errScanStopped
		}

		// Store embedded artwork and save its URL with the track record.
		var artworkURL string
		if len(artworkData) > 0 {
			artworkPath, err := parser.SaveArtwork(
				artworkData,
				trackInfo.Artist,
				trackInfo.Title,
				artworkFormat,
			)
			if err != nil {
				log.Printf("Warning: Failed to save artwork for %s: %v", filePath, err)
			} else {
				// Convert file path to URL for serving
				artworkURL = "/artwork/" + filepath.Base(artworkPath)
			}
		}

		// Create song record
		song := &database.Song{
			Title:               trackInfo.Title,
			Artist:              trackInfo.Artist,
			Album:               trackInfo.Album,
			Genre:               trackInfo.Genre,
			Year:                trackInfo.Year,
			TrackNumber:         trackInfo.TrackNumber,
			DiscNumber:          trackInfo.DiscNumber,
			DiscTotal:           trackInfo.DiscTotal,
			ReplayGainTrackDB:   trackInfo.ReplayGainTrackDB,
			ReplayGainAlbumDB:   trackInfo.ReplayGainAlbumDB,
			ReplayGainTrackPeak: trackInfo.ReplayGainTrackPeak,
			ReplayGainAlbumPeak: trackInfo.ReplayGainAlbumPeak,
			Duration:            trackInfo.Duration,
			FileSize:            trackInfo.FileSize,
			FilePath:            filePath,
			FileName:            filepath.Base(filePath),
			Format:              trackInfo.Format,
			HasMetadata:         trackInfo.HasMetadata,
			Confidence:          trackInfo.Confidence,
			Source:              trackInfo.Source,
		}
		applyScannedArtwork(song, artworkURL)
		artist, artistErr := s.db.ArtistOrStoreArtist(song.Artist)
		if artistErr != nil {
			log.Printf("Warning: Failed to link artist %q for %s: %v", song.Artist, filePath, artistErr)
		} else {
			song.ArtistID = artist.ID
		}

		// Check for existing record
		existing, err := s.db.CheckDuplicate(song)
		if err != nil {
			log.Printf("Warning: Failed to check duplicate for %s: %v", filePath, err)
			// Continue processing even if duplicate check fails - let database handle constraint
		}

		if existing != nil {
			// Found existing record - check if it's unchanged
			if existing.FilePath == song.FilePath {
				// Same file path - check if song data is unchanged
				unchanged, err := s.db.IsSongUnchanged(song)
				if err != nil {
					log.Printf("Warning: Failed to check if song is unchanged for %s: %v", filePath, err)
					// Continue with update if check fails
				} else if unchanged {
					song.ID = existing.ID
					s.storeTrackAudioProperties(song)
					// Song is exactly the same - skip it
					skipped++
					processed++
					log.Printf("Skipped unchanged song: %s - %s by %s", filePath, song.Title, song.Artist)
					continue
				}

				// Song has changed - update existing record
				song.ID = existing.ID               // Keep same ID
				song.CreatedAt = existing.CreatedAt // Preserve creation time
				song.UpdatedAt = time.Now()         // Update timestamp
				if song.ArtistID == "" {
					song.ArtistID = existing.ArtistID
				}

				err = s.db.UpdateSong(song)
				if err != nil {
					log.Printf("Warning: Failed to update existing song %s: %v", filePath, err)
					duplicates++
				} else {
					s.storeTrackAudioProperties(song)
					songsAdded++ // Count as processed/updated
					log.Printf("Updated existing song: %s - %s by %s (ID: %s)", filePath, song.Title, song.Artist, existing.ID)
				}
			} else {
				// Different file path but same title+artist - this might be the same file with different path format
				// Let's check if this is actually the same file by comparing metadata
				log.Printf("Found same title+artist with different path: %s vs %s", filePath, existing.FilePath)

				// For now, treat this as a duplicate but log it clearly
				duplicates++
				processed++
				log.Printf("Duplicate found: %s - %s by %s (existing ID: %s, existing path: %s)", filePath, song.Title, song.Artist, existing.ID, existing.FilePath)
				continue
			}
		} else {
			// No existing record - create new song
			err = s.db.CreateSong(song)
			if err != nil {
				log.Printf("Warning: Failed to save song %s: %v", filePath, err)
				// Check if it's a constraint violation (duplicate)
				if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
					log.Printf("Constraint violation detected - treating as duplicate: %s", filePath)
					duplicates++
				}
				processed++
				continue
			}

			songsAdded++
			s.storeTrackAudioProperties(song)
			log.Printf("Added new song: %s - %s by %s", filePath, song.Title, song.Artist)
		}
		processed++
	}

	return processed, songsAdded, duplicates, skipped, nil
}

func (s *Scanner) storeTrackAudioProperties(song *database.Song) {
	if err := s.db.UpsertTrackAudioProperties(database.TrackAudioProperties{
		TrackID: song.ID, DiscNumber: song.DiscNumber, DiscTotal: song.DiscTotal,
		ReplayGainTrackDB: song.ReplayGainTrackDB, ReplayGainAlbumDB: song.ReplayGainAlbumDB,
		ReplayGainTrackPeak: song.ReplayGainTrackPeak, ReplayGainAlbumPeak: song.ReplayGainAlbumPeak,
	}); err != nil {
		log.Printf("Warning: failed to store technical audio metadata for %s: %v", song.FilePath, err)
	}
}

func (s *Scanner) scanCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// claimCompletion closes the cancellation window before the final completed
// status is written. A stop accepted before this point always wins.
func (s *Scanner) claimCompletion(ctx context.Context) bool {
	s.scanMutex.Lock()
	defer s.scanMutex.Unlock()

	if s.stopPending || s.scanCancelled(ctx) {
		return false
	}

	s.isScanning = false
	s.cancelScan = nil
	return true
}

func (s *Scanner) updateScanProgress(scanID string, batch []string, processed, total, songsAdded, duplicates, skipped int) {
	currentFile := ""
	if len(batch) > 0 {
		currentFile = batch[len(batch)-1]
	}

	if err := s.scanStore.UpdateScanWithTracksSkipped(scanID, duplicates+skipped); err != nil {
		log.Printf("Warning: Failed to update tracks skipped: %v", err)
	}
	if err := s.scanStore.UpdateScanWithFileAndStats(scanID, currentFile, processed, total, songsAdded, duplicates); err != nil {
		log.Printf("Warning: Failed to update scan progress: %v", err)
	}
	s.broadcastCurrentScan()
}

func (s *Scanner) finishStoppedScan(scanID string) {
	log.Println("Scan stopped by user")
	if err := s.scanStore.SetScanStopped(scanID); err != nil {
		log.Printf("Warning: Failed to mark scan as stopped: %v", err)
	}
	if scan, err := s.scanStore.GetScan(scanID); err == nil {
		s.broadcastScanUpdate(scan)
	}
}

func applyScannedArtwork(song *database.Song, artworkURL string) {
	if artworkURL == "" {
		return
	}

	song.ImageURL = artworkURL
	song.CoverArtURL = artworkURL
	song.CoverArtSmallURL = artworkURL
	song.CoverArtMediumURL = artworkURL
	song.CoverArtLargeURL = artworkURL
	song.CoverArtSource = "embedded"
}

// handleScanError handles errors during scanning
func (s *Scanner) handleScanError(scanID string, errorMsg string) {
	err := s.scanStore.UpdateScan(scanID, "failed", 0)
	if err != nil {
		log.Printf("Warning: Failed to update scan status to failed: %v", err)
	}

	// Broadcast error update
	s.broadcastCurrentScan()

	log.Printf("Scan failed: %s", errorMsg)
}

// broadcastCurrentScan gets the current scan and broadcasts it
func (s *Scanner) broadcastCurrentScan() {
	scan, err := s.scanStore.GetCurrentScan()
	if err != nil {
		log.Printf("Warning: Failed to get current scan: %v", err)
		return
	}

	if scan != nil {
		// Ensure scan has a valid ID with scan_ prefix
		if scan.ID == "" || !strings.HasPrefix(scan.ID, "scan_") {
			// If ID is missing or invalid, generate a new one
			scan.ID = fmt.Sprintf("scan_%d", time.Now().UnixNano())
			log.Printf("Generated new scan ID: %s", scan.ID)

			// Note: We can't update the ID directly due to database constraints
			// The scan will be updated with the correct ID on the next cycle
			log.Printf("Scan ID will be corrected on next update cycle")
		}

		s.broadcastScanUpdate(scan)
	}
}

// broadcastScanUpdate sends a scan update via WebSocket
func (s *Scanner) broadcastScanUpdate(scan *database.ScanStatus) {
	// Log update for debugging
	log.Printf("Broadcasting scan update: ID=%s, Status=%s, Progress=%d%%, Files=%d/%d, CurrentFile=%s, SongsAdded=%d, Duplicates=%d, TracksSkipped=%d",
		scan.ID, scan.Status, scan.Progress, scan.Processed, scan.TotalFiles, scan.CurrentFile, scan.SongsAdded, scan.Duplicates, scan.TracksSkipped)

	// Send scan data directly - WebSocket manager will wrap it properly
	websocket.BroadcastScanUpdate(scan)
}

// StopScan stops the current scan
func (s *Scanner) StopScan() error {
	s.scanMutex.Lock()
	if !s.isScanning {
		s.scanMutex.Unlock()
		return fmt.Errorf("no scan in progress")
	}
	if s.stopPending {
		s.scanMutex.Unlock()
		return nil
	}

	s.stopPending = true
	cancel := s.cancelScan
	scanID := s.currentScan.ID
	s.currentScan.Status = "stopping"
	s.scanMutex.Unlock()

	if cancel != nil {
		cancel()
	}
	if err := s.scanStore.UpdateScanStatus(scanID, "stopping"); err != nil {
		log.Printf("Warning: Failed to mark scan as stopping: %v", err)
	}
	if scan, err := s.scanStore.GetScan(scanID); err == nil {
		s.broadcastScanUpdate(scan)
	}
	return nil
}

// GetScanStatus returns the current scan status
func (s *Scanner) GetScanStatus() (*database.ScanStatus, error) {
	s.scanMutex.Lock()
	defer s.scanMutex.Unlock()

	if !s.isScanning {
		return nil, fmt.Errorf("no scan in progress")
	}

	return s.currentScan, nil
}

// IsScanning returns true if a scan is currently in progress
func (s *Scanner) IsScanning() bool {
	s.scanMutex.Lock()
	defer s.scanMutex.Unlock()
	return s.isScanning
}

// GetDB returns the database instance
func (s *Scanner) GetDB() *database.DB {
	return s.db
}
