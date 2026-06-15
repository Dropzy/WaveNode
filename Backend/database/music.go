package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Music operations

// GetAllMusic retrieves all music from the database
func (db *DB) GetAllMusic() ([]Music, error) {
	query := `
		SELECT m.id, m.title, m.artist, m.album, m.genre, m.duration, m.release_date, 
		       m.file_path, m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source, m.parsed_from_filename,
		       COALESCE(m.artist_id, ''), m.play_count, m.image_url, m.cover_art_url, m.cover_art_small_url,
		       m.cover_art_medium_url, m.cover_art_large_url, m.cover_art_source,
		       m.last_cover_art_enriched_at, m.upload_order, m.created_at, m.updated_at,
		       a.image_url as artist_image_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE BTRIM(COALESCE(m.title, '')) != ''
		ORDER BY m.upload_order DESC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query music: %v", err)
	}
	defer rows.Close()

	var musicList []Music
	for rows.Next() {
		var music Music
		var featuringJSON []byte
		var releaseDate sql.NullTime
		var artistImageURL sql.NullString
		var imageURL sql.NullString
		var coverArtURL sql.NullString
		var coverArtSmallURL sql.NullString
		var coverArtMediumURL sql.NullString
		var coverArtLargeURL sql.NullString
		var coverArtSource sql.NullString

		err := rows.Scan(
			&music.ID, &music.Title, &music.Artist, &music.Album, &music.Genre,
			&music.Duration, &releaseDate, &music.FilePath, &music.FileName,
			&music.FileSize, &music.Format, &music.Year, &music.TrackNumber,
			&featuringJSON, &music.HasMetadata, &music.Confidence, &music.Source,
			&music.ParsedFromFilename, &music.ArtistID, &music.PlayCount,
			&imageURL, &coverArtURL, &coverArtSmallURL, &coverArtMediumURL,
			&coverArtLargeURL, &coverArtSource, &music.LastCoverArtEnrichedAt,
			&music.UploadOrder, &music.CreatedAt, &music.UpdatedAt, &artistImageURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan music row: %v", err)
		}

		if releaseDate.Valid {
			music.ReleaseDate = &releaseDate.Time
		}

		if artistImageURL.Valid {
			music.ArtistImageURL = artistImageURL.String
		}

		if imageURL.Valid {
			music.ImageURL = imageURL.String
		}

		if coverArtURL.Valid {
			music.CoverArtURL = coverArtURL.String
		}

		if coverArtSmallURL.Valid {
			music.CoverArtSmallURL = coverArtSmallURL.String
		}

		if coverArtMediumURL.Valid {
			music.CoverArtMediumURL = coverArtMediumURL.String
		}

		if coverArtLargeURL.Valid {
			music.CoverArtLargeURL = coverArtLargeURL.String
		}

		if coverArtSource.Valid {
			music.CoverArtSource = coverArtSource.String
		}

		if len(featuringJSON) > 0 {
			if err := json.Unmarshal(featuringJSON, &music.Featuring); err != nil {
				log.Printf("Warning: Failed to unmarshal featuring for track %s: %v", music.ID, err)
			}
		}

		musicList = append(musicList, music)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating music rows: %v", err)
	}

	_ = db.EnrichTrackAudioProperties(musicList)
	return musicList, nil
}

// SyncMusicUploadOrder aligns upload_order with the scanner's chronological
// file order. It updates the full discovered set in one database operation so
// existing libraries are repaired without deleting and re-importing tracks.
func (db *DB) SyncMusicUploadOrder(filePaths []string) error {
	if len(filePaths) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin upload order sync: %v", err)
	}
	defer tx.Rollback()

	query := `
		WITH ordered_files AS (
			SELECT file_path, position
			FROM unnest($1::text[]) WITH ORDINALITY AS discovered(file_path, position)
		),
		order_base AS (
			SELECT COALESCE(MAX(upload_order), 0) AS max_order
			FROM music
		)
		UPDATE music AS track
		SET upload_order = order_base.max_order + ordered_files.position
		FROM ordered_files, order_base
		WHERE track.file_path = ordered_files.file_path
	`
	if _, err := tx.Exec(query, pq.Array(filePaths)); err != nil {
		return fmt.Errorf("failed to sync music upload order: %v", err)
	}

	if _, err := tx.Exec(`
		SELECT setval(
			'music_upload_order_seq',
			GREATEST(COALESCE((SELECT MAX(upload_order) FROM music), 0), 1),
			COALESCE((SELECT MAX(upload_order) FROM music), 0) > 0
		)
	`); err != nil {
		return fmt.Errorf("failed to sync music upload order sequence: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit upload order sync: %v", err)
	}
	return nil
}

// GetMusic retrieves a single music track by ID with artist image
func (db *DB) GetMusic(id string) (*Music, error) {
	query := `
		SELECT m.id, m.title, m.artist, m.album, m.genre, m.duration, m.release_date, 
		       m.file_path, m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source, m.parsed_from_filename,
		       COALESCE(m.artist_id, ''), m.play_count, m.image_url, m.cover_art_url, m.cover_art_small_url,
		       m.cover_art_medium_url, m.cover_art_large_url, m.cover_art_source,
		       m.last_cover_art_enriched_at, m.created_at, m.updated_at,
		       a.image_url as artist_image_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE m.id = $1`

	var music Music
	var featuringJSON []byte
	var releaseDate sql.NullTime
	var artistImageURL sql.NullString
	var imageURL sql.NullString
	var coverArtURL sql.NullString
	var coverArtSmallURL sql.NullString
	var coverArtMediumURL sql.NullString
	var coverArtLargeURL sql.NullString
	var coverArtSource sql.NullString

	err := db.conn.QueryRow(query, id).Scan(
		&music.ID, &music.Title, &music.Artist, &music.Album, &music.Genre,
		&music.Duration, &releaseDate, &music.FilePath, &music.FileName,
		&music.FileSize, &music.Format, &music.Year, &music.TrackNumber,
		&featuringJSON, &music.HasMetadata, &music.Confidence, &music.Source,
		&music.ParsedFromFilename, &music.ArtistID, &music.PlayCount,
		&imageURL, &coverArtURL, &coverArtSmallURL, &coverArtMediumURL,
		&coverArtLargeURL, &coverArtSource, &music.LastCoverArtEnrichedAt,
		&music.CreatedAt, &music.UpdatedAt, &artistImageURL,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("music not found")
		}
		return nil, fmt.Errorf("failed to query music: %v", err)
	}

	if releaseDate.Valid {
		music.ReleaseDate = &releaseDate.Time
	}

	if artistImageURL.Valid {
		music.ArtistImageURL = artistImageURL.String
	}

	if imageURL.Valid {
		music.ImageURL = imageURL.String
	}

	if coverArtURL.Valid {
		music.CoverArtURL = coverArtURL.String
	}

	if coverArtSmallURL.Valid {
		music.CoverArtSmallURL = coverArtSmallURL.String
	}

	if coverArtMediumURL.Valid {
		music.CoverArtMediumURL = coverArtMediumURL.String
	}

	if coverArtLargeURL.Valid {
		music.CoverArtLargeURL = coverArtLargeURL.String
	}

	if coverArtSource.Valid {
		music.CoverArtSource = coverArtSource.String
	}

	if len(featuringJSON) > 0 {
		if err := json.Unmarshal(featuringJSON, &music.Featuring); err != nil {
			log.Printf("Warning: Failed to unmarshal featuring for track %s: %v", music.ID, err)
		}
	}

	if properties, propertiesErr := db.GetTrackAudioProperties(music.ID); propertiesErr == nil {
		ApplyTrackAudioProperties(&music, properties)
	}
	return &music, nil
}

// AddMusic adds a new music track to the database
func (db *DB) AddMusic(music *Music) error {
	// Generate ID if not provided
	if music.ID == "" {
		music.ID = fmt.Sprintf("music_%d", time.Now().UnixNano())
	}

	query := `
		INSERT INTO music (
			id, title, artist, album, genre, duration, release_date,
			file_path, file_name, file_size, format, year, track_number,
			featuring, has_metadata, confidence, source, parsed_from_filename,
			artist_id, image_url, cover_art_url, cover_art_small_url,
			cover_art_medium_url, cover_art_large_url, cover_art_source,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24, $25, $26, $27
		)`

	music.CreatedAt = time.Now()
	music.UpdatedAt = time.Now()

	var featuringJSON []byte
	if len(music.Featuring) > 0 {
		var err error
		featuringJSON, err = json.Marshal(music.Featuring)
		if err != nil {
			return fmt.Errorf("failed to marshal featuring: %v", err)
		}
	} else {
		// Always provide a valid JSON value for JSONB column
		featuringJSON = []byte("null")
	}

	var releaseDate interface{}
	if music.ReleaseDate != nil && !music.ReleaseDate.IsZero() {
		releaseDate = music.ReleaseDate
	}

	_, err := db.conn.Exec(query,
		music.ID, music.Title, music.Artist, music.Album, music.Genre,
		music.Duration, releaseDate, music.FilePath, music.FileName,
		music.FileSize, music.Format, music.Year, music.TrackNumber,
		featuringJSON, music.HasMetadata, music.Confidence, music.Source,
		music.ParsedFromFilename, nullableString(music.ArtistID), music.ImageURL,
		music.CoverArtURL, music.CoverArtSmallURL, music.CoverArtMediumURL,
		music.CoverArtLargeURL, music.CoverArtSource, music.CreatedAt, music.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert music: %v", err)
	}

	return nil
}

// UpdateMusic updates an existing music track
func (db *DB) UpdateMusic(music *Music) error {
	query := `
		UPDATE music SET 
			title = $2, artist = $3, album = $4, genre = $5, duration = $6,
			release_date = $7, file_path = $8, file_name = $9, file_size = $10,
			format = $11, year = $12, track_number = $13, featuring = $14,
			has_metadata = $15, confidence = $16, source = $17,
			parsed_from_filename = $18, artist_id = $19, image_url = $20,
			cover_art_url = $21, cover_art_small_url = $22, cover_art_medium_url = $23, 
			cover_art_large_url = $24, cover_art_source = $25, last_cover_art_enriched_at = $26, 
			updated_at = $27
		WHERE id = $1`

	music.UpdatedAt = time.Now()

	var featuringJSON []byte
	if len(music.Featuring) > 0 {
		var err error
		featuringJSON, err = json.Marshal(music.Featuring)
		if err != nil {
			return fmt.Errorf("failed to marshal featuring: %v", err)
		}
	} else {
		// Always provide a valid JSON value for JSONB column
		featuringJSON = []byte("null")
	}

	var releaseDate interface{}
	if music.ReleaseDate != nil && !music.ReleaseDate.IsZero() {
		releaseDate = music.ReleaseDate
	}

	result, err := db.conn.Exec(query,
		music.ID, music.Title, music.Artist, music.Album, music.Genre,
		music.Duration, releaseDate, music.FilePath, music.FileName,
		music.FileSize, music.Format, music.Year, music.TrackNumber,
		featuringJSON, music.HasMetadata, music.Confidence, music.Source,
		music.ParsedFromFilename, music.ArtistID, music.ImageURL, music.CoverArtURL,
		music.CoverArtSmallURL, music.CoverArtMediumURL, music.CoverArtLargeURL,
		music.CoverArtSource, music.LastCoverArtEnrichedAt, music.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update music: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("music not found")
	}

	return db.touchSmartPlaylists("")
}

// DeleteMusic removes a music track from the database
func (db *DB) DeleteMusic(id string) error {
	query := "DELETE FROM music WHERE id = $1"

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete music: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("music not found")
	}

	return db.touchSmartPlaylists("")
}

// RemoveMissingMusic removes indexed tracks from active sources when their
// files are no longer present, and prunes their playlist references.
func (db *DB) RemoveMissingMusic(validPaths map[string]struct{}, sourceRoots []string) (int, error) {
	rows, err := db.conn.Query("SELECT id, file_path FROM music WHERE COALESCE(file_path, '') <> ''")
	if err != nil {
		return 0, fmt.Errorf("failed to load indexed music paths: %v", err)
	}
	defer rows.Close()

	missingIDs := make([]string, 0)
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			return 0, fmt.Errorf("failed to scan indexed music path: %v", err)
		}
		if _, exists := validPaths[path]; exists || !pathWithinSources(path, sourceRoots) {
			continue
		}
		missingIDs = append(missingIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to iterate indexed music paths: %v", err)
	}
	if len(missingIDs) == 0 {
		return 0, nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin missing-track cleanup: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM music WHERE id = ANY($1)", pq.Array(missingIDs)); err != nil {
		return 0, fmt.Errorf("failed to delete missing tracks: %v", err)
	}
	if _, err := tx.Exec(`
		UPDATE playlists
		SET track_ids = COALESCE((
			SELECT jsonb_agg(item)
			FROM jsonb_array_elements_text(COALESCE(track_ids, '[]'::jsonb)) AS item
			WHERE NOT (item = ANY($1))
		), '[]'::jsonb),
		updated_at = CURRENT_TIMESTAMP
	`, pq.Array(missingIDs)); err != nil {
		return 0, fmt.Errorf("failed to prune missing tracks from playlists: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit missing-track cleanup: %v", err)
	}
	return len(missingIDs), nil
}

func pathWithinSources(path string, sourceRoots []string) bool {
	cleanPath := filepath.Clean(path)
	for _, root := range sourceRoots {
		relative, err := filepath.Rel(filepath.Clean(root), cleanPath)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// SearchMusic searches for music tracks by title, artist, album, or genre
func (db *DB) SearchMusic(query string) ([]Music, error) {
	searchQuery := `
		SELECT m.id, m.title, m.artist, m.album, m.genre, m.duration, m.release_date, 
		       m.file_path, m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source, m.parsed_from_filename,
		       COALESCE(m.artist_id, ''), m.play_count, m.image_url, m.cover_art_url, m.cover_art_small_url,
		       m.cover_art_medium_url, m.cover_art_large_url, m.cover_art_source,
		       m.last_cover_art_enriched_at, m.created_at, m.updated_at,
		       a.image_url as artist_image_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE LOWER(m.title) LIKE LOWER($1) 
		   OR LOWER(m.artist) LIKE LOWER($1) 
		   OR LOWER(m.album) LIKE LOWER($1) 
		   OR LOWER(m.genre) LIKE LOWER($1)
		ORDER BY m.title, m.artist`

	searchPattern := "%" + strings.ToLower(query) + "%"

	rows, err := db.conn.Query(searchQuery, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search music: %v", err)
	}
	defer rows.Close()

	var results []Music
	for rows.Next() {
		var music Music
		var featuringJSON []byte
		var releaseDate sql.NullTime
		var artistImageURL sql.NullString
		var imageURL sql.NullString
		var coverArtURL sql.NullString
		var coverArtSmallURL sql.NullString
		var coverArtMediumURL sql.NullString
		var coverArtLargeURL sql.NullString
		var coverArtSource sql.NullString

		err := rows.Scan(
			&music.ID, &music.Title, &music.Artist, &music.Album, &music.Genre,
			&music.Duration, &releaseDate, &music.FilePath, &music.FileName,
			&music.FileSize, &music.Format, &music.Year, &music.TrackNumber,
			&featuringJSON, &music.HasMetadata, &music.Confidence, &music.Source,
			&music.ParsedFromFilename, &music.ArtistID, &music.PlayCount,
			&imageURL, &coverArtURL, &coverArtSmallURL, &coverArtMediumURL,
			&coverArtLargeURL, &coverArtSource, &music.LastCoverArtEnrichedAt,
			&music.CreatedAt, &music.UpdatedAt, &artistImageURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan music row: %v", err)
		}

		if releaseDate.Valid {
			music.ReleaseDate = &releaseDate.Time
		}

		if artistImageURL.Valid {
			music.ArtistImageURL = artistImageURL.String
		}

		if imageURL.Valid {
			music.ImageURL = imageURL.String
		}

		if coverArtURL.Valid {
			music.CoverArtURL = coverArtURL.String
		}

		if coverArtSmallURL.Valid {
			music.CoverArtSmallURL = coverArtSmallURL.String
		}

		if coverArtMediumURL.Valid {
			music.CoverArtMediumURL = coverArtMediumURL.String
		}

		if coverArtLargeURL.Valid {
			music.CoverArtLargeURL = coverArtLargeURL.String
		}

		if coverArtSource.Valid {
			music.CoverArtSource = coverArtSource.String
		}

		if len(featuringJSON) > 0 {
			if err := json.Unmarshal(featuringJSON, &music.Featuring); err != nil {
				log.Printf("Warning: Failed to unmarshal featuring for track %s: %v", music.ID, err)
			}
		}

		results = append(results, music)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating music rows: %v", err)
	}

	return results, nil
}

// GetMusicByFilePath retrieves a music track by its file path
func (db *DB) GetMusicByFilePath(filePath string) (*Music, error) {
	query := `
		SELECT m.id, m.title, m.artist, m.album, m.genre, m.duration, m.release_date, 
		       m.file_path, m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source, m.parsed_from_filename,
		       COALESCE(m.artist_id, ''), m.play_count, m.image_url, m.cover_art_url, m.cover_art_small_url,
		       m.cover_art_medium_url, m.cover_art_large_url, m.cover_art_source,
		       m.last_cover_art_enriched_at, m.created_at, m.updated_at,
		       a.image_url as artist_image_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE m.file_path = $1`

	var music Music
	var featuringJSON []byte
	var releaseDate sql.NullTime
	var artistImageURL sql.NullString
	var imageURL sql.NullString
	var coverArtURL sql.NullString
	var coverArtSmallURL sql.NullString
	var coverArtMediumURL sql.NullString
	var coverArtLargeURL sql.NullString
	var coverArtSource sql.NullString

	err := db.conn.QueryRow(query, filePath).Scan(
		&music.ID, &music.Title, &music.Artist, &music.Album, &music.Genre,
		&music.Duration, &releaseDate, &music.FilePath, &music.FileName,
		&music.FileSize, &music.Format, &music.Year, &music.TrackNumber,
		&featuringJSON, &music.HasMetadata, &music.Confidence, &music.Source,
		&music.ParsedFromFilename, &music.ArtistID, &music.PlayCount,
		&imageURL, &coverArtURL, &coverArtSmallURL, &coverArtMediumURL,
		&coverArtLargeURL, &coverArtSource, &music.LastCoverArtEnrichedAt,
		&music.CreatedAt, &music.UpdatedAt, &artistImageURL,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("music not found with file path: %s", filePath)
		}
		return nil, fmt.Errorf("failed to query music by file path: %v", err)
	}

	if releaseDate.Valid {
		music.ReleaseDate = &releaseDate.Time
	}

	if artistImageURL.Valid {
		music.ArtistImageURL = artistImageURL.String
	}

	if imageURL.Valid {
		music.ImageURL = imageURL.String
	}

	if coverArtURL.Valid {
		music.CoverArtURL = coverArtURL.String
	}

	if coverArtSmallURL.Valid {
		music.CoverArtSmallURL = coverArtSmallURL.String
	}

	if coverArtMediumURL.Valid {
		music.CoverArtMediumURL = coverArtMediumURL.String
	}

	if coverArtLargeURL.Valid {
		music.CoverArtLargeURL = coverArtLargeURL.String
	}

	if coverArtSource.Valid {
		music.CoverArtSource = coverArtSource.String
	}

	if len(featuringJSON) > 0 {
		if err := json.Unmarshal(featuringJSON, &music.Featuring); err != nil {
			log.Printf("Warning: Failed to unmarshal featuring for track %s: %v", music.ID, err)
		}
	}

	return &music, nil
}

// GetMusicByFilePathAndFileName retrieves a music track by both file path and file name
// This provides enhanced duplicate detection by checking both the full path and the file name
func (db *DB) GetMusicByFilePathAndFileName(filePath, fileName string) (*Music, error) {
	query := `
		SELECT m.id, m.title, m.artist, m.album, m.genre, m.duration, m.release_date, 
		       m.file_path, m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source, m.parsed_from_filename,
		       COALESCE(m.artist_id, ''), m.play_count, m.image_url, m.cover_art_url, m.cover_art_small_url,
		       m.cover_art_medium_url, m.cover_art_large_url, m.cover_art_source,
		       m.last_cover_art_enriched_at, m.created_at, m.updated_at,
		       a.image_url as artist_image_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE m.file_path = $1 AND m.file_name = $2`

	var music Music
	var featuringJSON []byte
	var releaseDate sql.NullTime
	var artistImageURL sql.NullString
	var imageURL sql.NullString
	var coverArtURL sql.NullString
	var coverArtSmallURL sql.NullString
	var coverArtMediumURL sql.NullString
	var coverArtLargeURL sql.NullString
	var coverArtSource sql.NullString

	err := db.conn.QueryRow(query, filePath, fileName).Scan(
		&music.ID, &music.Title, &music.Artist, &music.Album, &music.Genre,
		&music.Duration, &releaseDate, &music.FilePath, &music.FileName,
		&music.FileSize, &music.Format, &music.Year, &music.TrackNumber,
		&featuringJSON, &music.HasMetadata, &music.Confidence, &music.Source,
		&music.ParsedFromFilename, &music.ArtistID, &music.PlayCount,
		&imageURL, &coverArtURL, &coverArtSmallURL, &coverArtMediumURL,
		&coverArtLargeURL, &coverArtSource, &music.LastCoverArtEnrichedAt,
		&music.CreatedAt, &music.UpdatedAt, &artistImageURL,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("music not found with file path: %s and file name: %s", filePath, fileName)
		}
		return nil, fmt.Errorf("failed to query music by file path and file name: %v", err)
	}

	if releaseDate.Valid {
		music.ReleaseDate = &releaseDate.Time
	}

	if artistImageURL.Valid {
		music.ArtistImageURL = artistImageURL.String
	}

	if imageURL.Valid {
		music.ImageURL = imageURL.String
	}

	if coverArtURL.Valid {
		music.CoverArtURL = coverArtURL.String
	}

	if coverArtSmallURL.Valid {
		music.CoverArtSmallURL = coverArtSmallURL.String
	}

	if coverArtMediumURL.Valid {
		music.CoverArtMediumURL = coverArtMediumURL.String
	}

	if coverArtLargeURL.Valid {
		music.CoverArtLargeURL = coverArtLargeURL.String
	}

	if coverArtSource.Valid {
		music.CoverArtSource = coverArtSource.String
	}

	if len(featuringJSON) > 0 {
		if err := json.Unmarshal(featuringJSON, &music.Featuring); err != nil {
			log.Printf("Warning: Failed to unmarshal featuring for track %s: %v", music.ID, err)
		}
	}

	return &music, nil
}

// IncrementPlayCount increments the play count for a track
func (db *DB) IncrementPlayCount(trackID string) error {
	query := "UPDATE music SET play_count = play_count + 1 WHERE id = $1"

	result, err := db.conn.Exec(query, trackID)
	if err != nil {
		return fmt.Errorf("failed to increment play count: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("track not found")
	}

	return db.touchSmartPlaylists("")
}
