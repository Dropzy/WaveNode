package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Scan represents a library scan operation
type Scan struct {
	ID           int        `json:"id"`
	Status       string     `json:"status"`
	FilesFound   int        `json:"files_found"`
	FilesScanned int        `json:"files_scanned"`
	TotalFiles   int        `json:"total_files"`
	SongsAdded   int        `json:"songs_added"`
	Duplicates   int        `json:"duplicates"`
	CurrentFile  string     `json:"current_file"`
	Error        string     `json:"error,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// Song represents a music track (alias for Music to match scanner expectations)
type Song = Music

// CreateScan creates a new scan record
func (db *DB) CreateScan(status string) (*Scan, error) {
	// First create the scans table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS scans (
			id SERIAL PRIMARY KEY,
			status VARCHAR(50) NOT NULL,
			files_found INTEGER DEFAULT 0,
			files_scanned INTEGER DEFAULT 0,
			total_files INTEGER DEFAULT 0,
			songs_added INTEGER DEFAULT 0,
			duplicates INTEGER DEFAULT 0,
			current_file TEXT,
			error TEXT,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		)
	`

	if _, err := db.conn.Exec(createTableQuery); err != nil {
		return nil, fmt.Errorf("failed to create scans table: %v", err)
	}

	query := `
		INSERT INTO scans (status, started_at)
		VALUES ($1, $2)
		RETURNING id, status, files_found, files_scanned, total_files, songs_added, duplicates, current_file, error, started_at, completed_at
	`

	var scan Scan
	scan.Status = status
	scan.StartedAt = time.Now()

	var currentFile sql.NullString
	var errorStr sql.NullString

	scanErr := db.conn.QueryRow(query, scan.Status, scan.StartedAt).Scan(
		&scan.ID, &scan.Status, &scan.FilesFound, &scan.FilesScanned,
		&scan.TotalFiles, &scan.SongsAdded, &scan.Duplicates, &currentFile,
		&errorStr, &scan.StartedAt, &scan.CompletedAt,
	)

	if scanErr != nil {
		return nil, fmt.Errorf("failed to create scan: %v", scanErr)
	}

	if currentFile.Valid {
		scan.CurrentFile = currentFile.String
	}
	if errorStr.Valid {
		scan.Error = errorStr.String
	}

	return &scan, nil
}

// UpdateScan updates an existing scan record
func (db *DB) UpdateScan(scan *Scan) error {
	query := `
		UPDATE scans SET 
			status = $2, files_found = $3, files_scanned = $4, total_files = $5,
			songs_added = $6, duplicates = $7, current_file = $8, error = $9,
			completed_at = $10
		WHERE id = $1
	`

	_, err := db.conn.Exec(query,
		scan.ID, scan.Status, scan.FilesFound, scan.FilesScanned,
		scan.TotalFiles, scan.SongsAdded, scan.Duplicates, scan.CurrentFile,
		scan.Error, scan.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update scan: %v", err)
	}

	return nil
}

// ClearLibrary clears all songs from music table
func (db *DB) ClearLibrary() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin clearing library: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE playlists SET track_ids = '[]'::jsonb, updated_at = CURRENT_TIMESTAMP"); err != nil {
		return fmt.Errorf("failed to clear playlist tracks: %v", err)
	}
	if _, err := tx.Exec("DELETE FROM music"); err != nil {
		return fmt.Errorf("failed to clear music: %v", err)
	}
	if _, err := tx.Exec("DELETE FROM artists"); err != nil {
		return fmt.Errorf("failed to clear artists: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit library clear: %v", err)
	}
	return nil
}

// CreateSong creates a new song record (alias for AddMusic)
func (db *DB) CreateSong(song *Song) error {
	music := &Music{
		ID:                     song.ID,
		Title:                  song.Title,
		Artist:                 song.Artist,
		Album:                  song.Album,
		Genre:                  song.Genre,
		Duration:               song.Duration,
		ReleaseDate:            song.ReleaseDate,
		FilePath:               song.FilePath,
		FileName:               song.FileName,
		FileSize:               song.FileSize,
		Format:                 song.Format,
		Year:                   song.Year,
		TrackNumber:            song.TrackNumber,
		Featuring:              song.Featuring,
		HasMetadata:            song.HasMetadata,
		Confidence:             song.Confidence,
		Source:                 song.Source,
		ParsedFromFilename:     song.ParsedFromFilename,
		ArtistID:               song.ArtistID,
		PlayCount:              song.PlayCount,
		ImageURL:               song.ImageURL,
		CoverArtURL:            song.CoverArtURL,
		CoverArtSmallURL:       song.CoverArtSmallURL,
		CoverArtMediumURL:      song.CoverArtMediumURL,
		CoverArtLargeURL:       song.CoverArtLargeURL,
		CoverArtSource:         song.CoverArtSource,
		LastCoverArtEnrichedAt: song.LastCoverArtEnrichedAt,
		CreatedAt:              song.CreatedAt,
		UpdatedAt:              song.UpdatedAt,
	}

	return db.AddMusic(music)
}

// CheckDuplicate checks if a song already exists in database
func (db *DB) CheckDuplicate(song *Song) (*Song, error) {
	// Check for existing song by file path (primary duplicate check)
	// Secondary check by title+artist for files with different paths but same content
	query := `
		SELECT id, title, artist, album, genre, duration, release_date,
		       file_path, file_name, file_size, format, year, track_number,
		       featuring, has_metadata, confidence, source, parsed_from_filename,
		       artist_id, play_count, created_at, updated_at
		FROM music 
		WHERE file_path = $1
		UNION
		SELECT id, title, artist, album, genre, duration, release_date,
		       file_path, file_name, file_size, format, year, track_number,
		       featuring, has_metadata, confidence, source, parsed_from_filename,
		       artist_id, play_count, created_at, updated_at
		FROM music 
		WHERE file_path != $1 AND title = $2 AND artist = $3
		LIMIT 1
	`

	var existing Song
	var featuringJSON []byte
	var releaseDate sql.NullTime
	var artistID sql.NullString

	err := db.conn.QueryRow(query, song.FilePath, song.Title, song.Artist).Scan(
		&existing.ID, &existing.Title, &existing.Artist, &existing.Album,
		&existing.Genre, &existing.Duration, &releaseDate, &existing.FilePath,
		&existing.FileName, &existing.FileSize, &existing.Format, &existing.Year,
		&existing.TrackNumber, &featuringJSON, &existing.HasMetadata,
		&existing.Confidence, &existing.Source, &existing.ParsedFromFilename,
		&artistID, &existing.PlayCount, &existing.CreatedAt, &existing.UpdatedAt,
	)

	if artistID.Valid {
		existing.ArtistID = artistID.String
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No duplicate found
		}
		return nil, fmt.Errorf("failed to check duplicate: %v", err)
	}

	if releaseDate.Valid {
		existing.ReleaseDate = &releaseDate.Time
	}

	if len(featuringJSON) > 0 {
		if err := json.Unmarshal(featuringJSON, &existing.Featuring); err != nil {
			log.Printf("Warning: Failed to unmarshal featuring for duplicate check: %v", err)
		}
	}

	return &existing, nil
}

// GetArtistIDFromName gets artist ID from artist name
func (db *DB) GetArtistIDFromName(name string) (string, error) {
	artist, err := db.GetArtistByName(name)
	if err != nil {
		return "", fmt.Errorf("artist not found: %v", err)
	}
	return artist.ID, nil
}

// RemoveDuplicates removes duplicate tracks from database, keeping first occurrence
func (db *DB) RemoveDuplicates() (int, error) {
	query := `
		DELETE FROM music 
		WHERE id NOT IN (
			SELECT MIN(id) 
			FROM music 
			GROUP BY file_path, file_name 
			HAVING COUNT(*) > 1
		)
		AND (file_path, file_name) IN (
			SELECT file_path, file_name 
			FROM music 
			GROUP BY file_path, file_name 
			HAVING COUNT(*) > 1
		)
	`

	result, err := db.conn.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to remove duplicates: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %v", err)
	}

	return int(rowsAffected), nil
}

// FindDuplicates returns all duplicate tracks in database
func (db *DB) FindDuplicates() ([]Music, error) {
	query := `
		SELECT m.id, m.title, m.artist, m.album, m.genre, m.duration, m.release_date,
		       m.file_path, m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source, m.parsed_from_filename,
		       COALESCE(m.artist_id, ''), m.play_count, m.created_at, m.updated_at,
		       a.image_url as artist_image_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE (m.file_path, m.file_name) IN (
			SELECT file_path, file_name 
			FROM music 
			GROUP BY file_path, file_name 
			HAVING COUNT(*) > 1
		)
		ORDER BY m.file_path, m.file_name, m.created_at
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to find duplicates: %v", err)
	}
	defer rows.Close()

	var duplicates []Music
	for rows.Next() {
		var music Music
		var releaseDateNull sql.NullTime
		var featuringJSON []byte
		var artistImageURL sql.NullString

		err := rows.Scan(
			&music.ID,
			&music.Title,
			&music.Artist,
			&music.Album,
			&music.Genre,
			&music.Duration,
			&releaseDateNull,
			&music.FilePath,
			&music.FileName,
			&music.FileSize,
			&music.Format,
			&music.Year,
			&music.TrackNumber,
			&featuringJSON,
			&music.HasMetadata,
			&music.Confidence,
			&music.Source,
			&music.ParsedFromFilename,
			&music.ArtistID,
			&music.PlayCount,
			&music.CreatedAt,
			&music.UpdatedAt,
			&artistImageURL,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan duplicate music: %v", err)
		}

		if releaseDateNull.Valid {
			music.ReleaseDate = &releaseDateNull.Time
		}

		if artistImageURL.Valid {
			music.ArtistImageURL = artistImageURL.String
		}

		if len(featuringJSON) > 0 {
			if err := json.Unmarshal(featuringJSON, &music.Featuring); err != nil {
				log.Printf("Warning: Failed to unmarshal featuring for duplicate track %s: %v", music.ID, err)
			}
		}

		duplicates = append(duplicates, music)
	}

	return duplicates, nil
}

// IsSongUnchanged checks if a song's data is exactly the same as the existing record
func (db *DB) IsSongUnchanged(song *Song) (bool, error) {
	// Get the existing song by file path
	query := `
		SELECT title, artist, album, genre, duration, year, track_number,
		       file_size, format, has_metadata, confidence, source,
		       image_url, cover_art_url, cover_art_source
		FROM music 
		WHERE file_path = $1
	`

	var existing Song
	var coverArtURL, coverArtSource sql.NullString

	err := db.conn.QueryRow(query, song.FilePath).Scan(
		&existing.Title, &existing.Artist, &existing.Album,
		&existing.Genre, &existing.Duration, &existing.Year, &existing.TrackNumber,
		&existing.FileSize, &existing.Format, &existing.HasMetadata,
		&existing.Confidence, &existing.Source, &existing.ImageURL,
		&coverArtURL, &coverArtSource,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // Song doesn't exist, so it's not unchanged
		}
		return false, fmt.Errorf("failed to check if song is unchanged: %v", err)
	}

	// Handle nullable fields
	if coverArtURL.Valid {
		existing.CoverArtURL = coverArtURL.String
	}
	if coverArtSource.Valid {
		existing.CoverArtSource = coverArtSource.String
	}

	// Compare all relevant fields
	unchanged := song.Title == existing.Title &&
		song.Artist == existing.Artist &&
		song.Album == existing.Album &&
		song.Genre == existing.Genre &&
		song.Duration == existing.Duration &&
		song.Year == existing.Year &&
		song.TrackNumber == existing.TrackNumber &&
		song.FileSize == existing.FileSize &&
		song.Format == existing.Format &&
		song.HasMetadata == existing.HasMetadata &&
		song.Confidence == existing.Confidence &&
		song.Source == existing.Source &&
		song.ImageURL == existing.ImageURL &&
		song.CoverArtURL == existing.CoverArtURL &&
		song.CoverArtSource == existing.CoverArtSource

	return unchanged, nil
}

// UpdateSong updates an existing song record
func (db *DB) UpdateSong(song *Song) error {
	// Handle artist_id - if it's empty or invalid, set to NULL to avoid FK constraint
	var artistID interface{} = song.ArtistID
	if song.ArtistID == "" {
		artistID = nil // Set to NULL to avoid foreign key constraint
	} else {
		// Verify artist exists before using the ID
		var exists bool
		checkErr := db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM artists WHERE id = $1)", song.ArtistID).Scan(&exists)
		if checkErr != nil || !exists {
			artistID = nil // Artist doesn't exist, set to NULL
		}
	}

	query := `
		UPDATE music SET 
			title = $2, artist = $3, album = $4, genre = $5, duration = $6,
			release_date = $7, file_path = $8, file_name = $9, file_size = $10,
			format = $11, year = $12, track_number = $13, featuring = $14,
			has_metadata = $15, confidence = $16, source = $17,
			parsed_from_filename = $18, artist_id = $19, image_url = $20,
			cover_art_url = $21, cover_art_small_url = $22, cover_art_medium_url = $23,
			cover_art_large_url = $24, cover_art_source = $25, updated_at = $26
		WHERE id = $1
	`

	var featuringJSON []byte
	if song.Featuring != nil && len(song.Featuring) > 0 {
		var err error
		featuringJSON, err = json.Marshal(song.Featuring)
		if err != nil {
			return fmt.Errorf("failed to marshal featuring: %v", err)
		}
	} else {
		// Use empty JSON array instead of null
		featuringJSON = []byte("[]")
	}

	var releaseDate interface{}
	if song.ReleaseDate != nil {
		releaseDate = song.ReleaseDate
	}

	_, err := db.conn.Exec(query,
		song.ID, song.Title, song.Artist, song.Album, song.Genre, song.Duration,
		releaseDate, song.FilePath, song.FileName, song.FileSize, song.Format,
		song.Year, song.TrackNumber, featuringJSON, song.HasMetadata,
		song.Confidence, song.Source, song.ParsedFromFilename,
		artistID, song.ImageURL, song.CoverArtURL, song.CoverArtSmallURL,
		song.CoverArtMediumURL, song.CoverArtLargeURL, song.CoverArtSource,
		song.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update song: %v", err)
	}

	return nil
}

// GetArtistByMusicBrainzID gets artist by MusicBrainz ID
func (db *DB) GetArtistByMusicBrainzID(musicBrainzID string) (*Artist, error) {
	query := `
		SELECT id, name, musicbrainz_id, musicbrainz_url, image_url, 
		       image_small_url, image_medium_url, image_large_url,
		       country, tags, biography, created_at, updated_at, last_enriched_at
		FROM artists 
		WHERE musicbrainz_id = $1
	`

	var artist Artist
	var tagsJSON []byte
	var biography sql.NullString
	var lastEnrichedAt sql.NullTime

	err := db.conn.QueryRow(query, musicBrainzID).Scan(
		&artist.ID,
		&artist.Name,
		&artist.MusicBrainzID,
		&artist.MusicBrainzURL,
		&artist.ImageURL,
		&artist.ImageSmallURL,
		&artist.ImageMediumURL,
		&artist.ImageLargeURL,
		&artist.Country,
		&tagsJSON,
		&biography,
		&artist.CreatedAt,
		&artist.UpdatedAt,
		&lastEnrichedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artist with MusicBrainz ID %s not found", musicBrainzID)
		}
		return nil, fmt.Errorf("failed to get artist by MusicBrainz ID: %v", err)
	}

	if biography.Valid {
		artist.Biography = biography.String
	}

	if lastEnrichedAt.Valid {
		artist.LastEnrichedAt = &lastEnrichedAt.Time
	}

	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &artist.Tags); err != nil {
			log.Printf("Warning: Failed to unmarshal tags for artist %s: %v", artist.ID, err)
		}
	}

	return &artist, nil
}
