package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// Enrichment operations

// GetEnrichmentStatistics retrieves enrichment statistics
func (db *DB) GetEnrichmentStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total tracks
	var totalTracks int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM music").Scan(&totalTracks)
	if err != nil {
		return nil, fmt.Errorf("failed to get total tracks: %v", err)
	}
	stats["total_tracks"] = totalTracks

	// Get tracks with metadata
	var tracksWithMetadata int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM music WHERE has_metadata = true").Scan(&tracksWithMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks with metadata: %v", err)
	}
	stats["tracks_with_metadata"] = tracksWithMetadata

	// Get total artists
	var totalArtists int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM artists").Scan(&totalArtists)
	if err != nil {
		return nil, fmt.Errorf("failed to get total artists: %v", err)
	}
	stats["total_artists"] = totalArtists

	// Get artists with images
	var artistsWithImages int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM artists WHERE image_url IS NOT NULL AND image_url != ''").Scan(&artistsWithImages)
	if err != nil {
		return nil, fmt.Errorf("failed to get artists with images: %v", err)
	}
	stats["artists_with_images"] = artistsWithImages

	// Get tracks with cover art
	var tracksWithCoverArt int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM music WHERE cover_art_url IS NOT NULL AND cover_art_url != ''").Scan(&tracksWithCoverArt)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks with cover art: %v", err)
	}
	stats["tracks_with_cover_art"] = tracksWithCoverArt

	return stats, nil
}

// ScanStore represents a scan store interface
type ScanStore struct {
	db *DB
}

// NewScanStore creates a new scan store
func NewScanStore(db *DB) *ScanStore {
	return &ScanStore{db: db}
}

// CreateScan creates a new scan record
func (s *ScanStore) CreateScan(scanType string) (*ScanStatus, error) {
	scan := &ScanStatus{
		ID:            fmt.Sprintf("scan_%d", time.Now().UnixNano()),
		Type:          scanType,
		Status:        "running",
		Progress:      0,
		TotalFiles:    0,
		Processed:     0,
		Errors:        []string{},
		SongsAdded:    0,
		SongsUpdated:  0,
		TracksSkipped: 0,
		Duplicates:    0,
		StartedAt:     time.Now(),
	}

	query := `
		INSERT INTO scan_status (id, type, status, progress, total_files, processed, current_file, errors, started_at, songs_added, songs_updated, tracks_skipped, duplicates)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	errorsJSON, _ := json.Marshal(scan.Errors)
	_, err := s.db.conn.Exec(query, scan.ID, scan.Type, scan.Status, scan.Progress,
		scan.TotalFiles, scan.Processed, scan.CurrentFile, errorsJSON, scan.StartedAt,
		scan.SongsAdded, scan.SongsUpdated, scan.TracksSkipped, scan.Duplicates)
	if err != nil {
		return nil, fmt.Errorf("failed to create scan: %v", err)
	}

	return scan, nil
}

// GetScan retrieves a scan by ID
func (s *ScanStore) GetScan(scanID string) (*ScanStatus, error) {
	query := `
		SELECT id, type, status, progress, total_files, processed, current_file, errors, started_at, completed_at, songs_added, songs_updated, tracks_skipped, duplicates
		FROM scan_status WHERE id = $1
	`

	var scan ScanStatus
	var errorsJSON []byte
	var completedAt sql.NullTime
	var songsAdded, songsUpdated, tracksSkipped, duplicates sql.NullInt64

	err := s.db.conn.QueryRow(query, scanID).Scan(&scan.ID, &scan.Type, &scan.Status,
		&scan.Progress, &scan.TotalFiles, &scan.Processed, &scan.CurrentFile,
		&errorsJSON, &scan.StartedAt, &completedAt, &songsAdded, &songsUpdated, &tracksSkipped, &duplicates)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("scan not found")
		}
		return nil, fmt.Errorf("failed to get scan: %v", err)
	}

	if len(errorsJSON) > 0 {
		json.Unmarshal(errorsJSON, &scan.Errors)
	}

	if completedAt.Valid {
		scan.CompletedAt = &completedAt.Time
	}

	if songsAdded.Valid {
		scan.SongsAdded = int(songsAdded.Int64)
	}

	if songsUpdated.Valid {
		scan.SongsUpdated = int(songsUpdated.Int64)
	}

	if tracksSkipped.Valid {
		scan.TracksSkipped = int(tracksSkipped.Int64)
	}

	if duplicates.Valid {
		scan.Duplicates = int(duplicates.Int64)
	}

	return &scan, nil
}

// SetScanCompleted sets the scan status to completed with completion timestamp
func (s *ScanStore) SetScanCompleted(scanID string) error {
	query := `UPDATE scan_status SET status = $2, completed_at = $3 WHERE id = $1`

	_, err := s.db.conn.Exec(query, scanID, "completed", time.Now())
	if err != nil {
		return fmt.Errorf("failed to set scan completed: %v", err)
	}

	return nil
}

// SetScanStopped records a cooperative cancellation without claiming completion.
func (s *ScanStore) SetScanStopped(scanID string) error {
	query := `
		UPDATE scan_status
		SET status = 'stopped', completed_at = $2, current_file = ''
		WHERE id = $1
	`

	if _, err := s.db.conn.Exec(query, scanID, time.Now()); err != nil {
		return fmt.Errorf("failed to set scan stopped: %v", err)
	}
	return nil
}

// StopInterruptedLibraryScans recovers scans left active by a server restart.
func (s *ScanStore) StopInterruptedLibraryScans() error {
	_, err := s.db.conn.Exec(`
		UPDATE scan_status
		SET status = 'stopped', completed_at = COALESCE(completed_at, $1), current_file = ''
		WHERE type = 'library' AND status IN ('running', 'stopping')
	`, time.Now())
	if err != nil {
		return fmt.Errorf("failed to recover interrupted library scans: %v", err)
	}
	return nil
}

// ClearAllScans deletes all scan records
func (s *ScanStore) ClearAllScans() error {
	query := `DELETE FROM scan_status`

	_, err := s.db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear all scans: %v", err)
	}

	return nil
}

// GetAllScans retrieves all scans
func (s *ScanStore) GetAllScans() ([]ScanStatus, error) {
	query := `
		SELECT id, type, status, progress, total_files, processed, current_file, errors, started_at, completed_at, songs_added, songs_updated, tracks_skipped, duplicates
		FROM scan_status ORDER BY started_at DESC
	`

	rows, err := s.db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scans: %v", err)
	}
	defer rows.Close()

	var scans []ScanStatus
	for rows.Next() {
		var scan ScanStatus
		var errorsJSON []byte
		var completedAt sql.NullTime
		var songsAdded, songsUpdated, tracksSkipped, duplicates sql.NullInt64

		err := rows.Scan(&scan.ID, &scan.Type, &scan.Status, &scan.Progress,
			&scan.TotalFiles, &scan.Processed, &scan.CurrentFile, &errorsJSON,
			&scan.StartedAt, &completedAt, &songsAdded, &songsUpdated, &tracksSkipped, &duplicates)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scan row: %v", err)
		}

		if len(errorsJSON) > 0 {
			json.Unmarshal(errorsJSON, &scan.Errors)
		}

		if completedAt.Valid {
			scan.CompletedAt = &completedAt.Time
		}

		if songsAdded.Valid {
			scan.SongsAdded = int(songsAdded.Int64)
		}

		if songsUpdated.Valid {
			scan.SongsUpdated = int(songsUpdated.Int64)
		}

		if tracksSkipped.Valid {
			scan.TracksSkipped = int(tracksSkipped.Int64)
		}

		if duplicates.Valid {
			scan.Duplicates = int(duplicates.Int64)
		}

		scans = append(scans, scan)
	}

	return scans, nil
}

// UpdateScan updates a scan status
func (s *ScanStore) UpdateScan(scanID, status string, progress int) error {
	query := `UPDATE scan_status SET status = $2, progress = $3 WHERE id = $1`

	_, err := s.db.conn.Exec(query, scanID, status, progress)
	if err != nil {
		return fmt.Errorf("failed to update scan: %v", err)
	}

	return nil
}

// UpdateScanStatus changes lifecycle state without overwriting progress.
func (s *ScanStore) UpdateScanStatus(scanID, status string) error {
	if _, err := s.db.conn.Exec(`
		UPDATE scan_status
		SET status = $2
		WHERE id = $1 AND status = 'running'
	`, scanID, status); err != nil {
		return fmt.Errorf("failed to update scan status: %v", err)
	}
	return nil
}

// UpdateScanProgress changes discovery progress without overwriting lifecycle state.
func (s *ScanStore) UpdateScanProgress(scanID string, progress int) error {
	if _, err := s.db.conn.Exec(`UPDATE scan_status SET progress = $2 WHERE id = $1`, scanID, progress); err != nil {
		return fmt.Errorf("failed to update scan progress: %v", err)
	}
	return nil
}

// UpdateScanWithFile updates a scan with current file and progress
func (s *ScanStore) UpdateScanWithFile(scanID, currentFile string, processed, total int) error {
	progress := calculateScanProgress(processed, total)
	query := `
		UPDATE scan_status
		SET current_file = $2, processed = $3, total_files = $4, progress = $5
		WHERE id = $1
	`

	_, err := s.db.conn.Exec(query, scanID, currentFile, processed, total, progress)
	if err != nil {
		return fmt.Errorf("failed to update scan with file: %v", err)
	}

	return nil
}

// UpdateScanCurrentFile updates only the current item without resetting progress counters.
func (s *ScanStore) UpdateScanCurrentFile(scanID, currentFile string) error {
	query := `UPDATE scan_status SET current_file = $2 WHERE id = $1`

	if _, err := s.db.conn.Exec(query, scanID, currentFile); err != nil {
		return fmt.Errorf("failed to update current scan file: %v", err)
	}

	return nil
}

// UpdateScanWithFileAndStats updates a scan with current file, progress, songs added, and duplicates
func (s *ScanStore) UpdateScanWithFileAndStats(scanID, currentFile string, processed, total, songsAdded, duplicates int) error {
	progress := calculateScanProgress(processed, total)

	query := `
		UPDATE scan_status 
		SET current_file = $2, processed = $3, total_files = $4, songs_added = $5, duplicates = $6, progress = $7
		WHERE id = $1
	`

	_, err := s.db.conn.Exec(query, scanID, currentFile, processed, total, songsAdded, duplicates, progress)
	if err != nil {
		return fmt.Errorf("failed to update scan with file and stats: %v", err)
	}

	return nil
}

func calculateScanProgress(processed, total int) int {
	if total <= 0 || processed <= 0 {
		return 0
	}

	progress := int(math.Round((float64(processed) / float64(total)) * 100))
	if progress > 100 {
		return 100
	}

	return progress
}

// UpdateScanWithStats updates a scan with songs added and duplicates
func (s *ScanStore) UpdateScanWithStats(scanID, status string, progress, songsAdded, duplicates int) error {
	// Get current tracks_skipped value to preserve it
	var currentTracksSkipped int
	err := s.db.conn.QueryRow("SELECT COALESCE(tracks_skipped, 0) FROM scan_status WHERE id = $1", scanID).Scan(&currentTracksSkipped)
	if err != nil {
		// If query fails, we'll proceed with 0 for tracks_skipped
		currentTracksSkipped = 0
	}

	// Check if status is being set to completed - if so, include completion timestamp
	if status == "completed" {
		query := `
			UPDATE scan_status 
			SET status = $2, progress = $3, songs_added = $4, duplicates = $5, tracks_skipped = $6, completed_at = $7
			WHERE id = $1
		`
		_, err = s.db.conn.Exec(query, scanID, status, progress, songsAdded, duplicates, currentTracksSkipped, time.Now())
	} else {
		query := `
			UPDATE scan_status 
			SET status = $2, progress = $3, songs_added = $4, duplicates = $5, tracks_skipped = $6
			WHERE id = $1
		`
		_, err = s.db.conn.Exec(query, scanID, status, progress, songsAdded, duplicates, currentTracksSkipped)
	}

	if err != nil {
		return fmt.Errorf("failed to update scan with stats: %v", err)
	}

	return nil
}

// UpdateScanWithTracksSkipped updates only the tracks_skipped field
func (s *ScanStore) UpdateScanWithTracksSkipped(scanID string, tracksSkipped int) error {
	query := `UPDATE scan_status SET tracks_skipped = $2 WHERE id = $1`

	_, err := s.db.conn.Exec(query, scanID, tracksSkipped)
	if err != nil {
		return fmt.Errorf("failed to update tracks skipped: %v", err)
	}

	return nil
}

// UpdateScanWithSkippedCount updates the scan with skip count (unchanged songs)
func (s *ScanStore) UpdateScanWithSkippedCount(scanID string, skippedCount int) error {
	query := `UPDATE scan_status SET tracks_skipped = tracks_skipped + $2 WHERE id = $1`

	_, err := s.db.conn.Exec(query, scanID, skippedCount)
	if err != nil {
		return fmt.Errorf("failed to update skipped count: %v", err)
	}

	return nil
}

// UpdateScanResult records progress, result counters, errors, and completion state.
func (s *ScanStore) UpdateScanResult(scanID, status, currentFile string, processed, total, songsAdded, songsUpdated, tracksSkipped int, errors []string, completed bool) error {
	progress := calculateScanProgress(processed, total)
	if completed {
		progress = 100
	}

	errorsJSON, err := json.Marshal(errors)
	if err != nil {
		return fmt.Errorf("failed to encode scan errors: %v", err)
	}

	var completedAt interface{}
	if completed {
		completedAt = time.Now()
	}

	_, err = s.db.conn.Exec(`
		UPDATE scan_status
		SET status = $2,
		    current_file = $3,
		    processed = $4,
		    total_files = $5,
		    progress = $6,
		    songs_added = $7,
		    songs_updated = $8,
		    tracks_skipped = $9,
		    errors = $10,
		    completed_at = $11
		WHERE id = $1
	`, scanID, status, currentFile, processed, total, progress, songsAdded, songsUpdated, tracksSkipped, errorsJSON, completedAt)
	if err != nil {
		return fmt.Errorf("failed to update scan result: %v", err)
	}

	return nil
}

// GetCurrentScan retrieves the currently active scan
func (s *ScanStore) GetCurrentScan() (*ScanStatus, error) {
	query := `
		SELECT id, type, status, progress, total_files, processed, current_file, errors, started_at, completed_at, songs_added, songs_updated, tracks_skipped, duplicates
		FROM scan_status 
		WHERE status IN ('running', 'stopping')
		ORDER BY started_at DESC 
		LIMIT 1
	`

	var scan ScanStatus
	var errorsJSON []byte
	var completedAt sql.NullTime
	var songsAdded, songsUpdated, tracksSkipped, duplicates sql.NullInt64

	err := s.db.conn.QueryRow(query).Scan(&scan.ID, &scan.Type, &scan.Status,
		&scan.Progress, &scan.TotalFiles, &scan.Processed, &scan.CurrentFile,
		&errorsJSON, &scan.StartedAt, &completedAt, &songsAdded, &songsUpdated, &tracksSkipped, &duplicates)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No active scan
		}
		return nil, fmt.Errorf("failed to get current scan: %v", err)
	}

	if len(errorsJSON) > 0 {
		json.Unmarshal(errorsJSON, &scan.Errors)
	}

	if completedAt.Valid {
		scan.CompletedAt = &completedAt.Time
	}

	if songsAdded.Valid {
		scan.SongsAdded = int(songsAdded.Int64)
	}

	if songsUpdated.Valid {
		scan.SongsUpdated = int(songsUpdated.Int64)
	}

	if tracksSkipped.Valid {
		scan.TracksSkipped = int(tracksSkipped.Int64)
	}

	if duplicates.Valid {
		scan.Duplicates = int(duplicates.Int64)
	}

	return &scan, nil
}
