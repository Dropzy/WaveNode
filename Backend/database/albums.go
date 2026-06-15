package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// Album operations

// GetAllAlbums retrieves all albums from database with cover art from tracks
func (db *DB) GetAllAlbums() ([]Album, error) {
	query := `
		SELECT DISTINCT album, 
		       CASE 
		         WHEN COUNT(DISTINCT artist) > 1 THEN 'Various Artists'
		         WHEN MIN(artist) IS NULL OR MIN(artist) = '' THEN 'Unknown Artist'
		         ELSE MIN(artist)
		       END as artist,
		       COUNT(*) as track_count,
		       MIN(year) as year
		FROM music
		WHERE album IS NOT NULL AND album != ''
		GROUP BY album
		ORDER BY album
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query albums: %v", err)
	}
	defer rows.Close()

	var albums []Album
	for rows.Next() {
		var album Album
		var year sql.NullInt32

		err := rows.Scan(&album.Name, &album.Artist, &album.TrackCount, &year)
		if err != nil {
			return nil, fmt.Errorf("failed to scan album row: %v", err)
		}

		// Handle nullable year field
		if year.Valid {
			album.Year = int(year.Int32)
		}

		// Generate ID from album name using hash
		album.ID = generateAlbumID(album.Name, album.Artist)

		// Get cover art from tracks for this album
		coverArt, err := db.getAlbumCoverArt(album.Name, album.Artist)
		if err == nil {
			album.CoverArtURL = coverArt.CoverArtURL
			album.CoverArtSmallURL = coverArt.CoverArtSmallURL
			album.CoverArtMediumURL = coverArt.CoverArtMediumURL
			album.CoverArtLargeURL = coverArt.CoverArtLargeURL
			album.CoverArtSource = coverArt.CoverArtSource
		}

		// Only add album if we successfully scanned it
		albums = append(albums, album)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating album rows: %v", err)
	}

	return albums, nil
}

// GetAlbumByID retrieves an album by its ID
func (db *DB) GetAlbumByID(albumID string) (*Album, error) {
	albums, err := db.GetAllAlbums()
	if err != nil {
		return nil, err
	}

	for _, album := range albums {
		if album.ID == albumID {
			return &album, nil
		}
	}

	return nil, fmt.Errorf("album with ID %s not found", albumID)
}

// GetAlbumTracksByID retrieves all tracks for a specific album by ID
func (db *DB) GetAlbumTracksByID(albumID string) ([]Music, error) {
	// First find album by ID
	album, err := db.GetAlbumByID(albumID)
	if err != nil {
		return nil, err
	}

	// Then get tracks for that album name
	return db.GetAlbumTracks(album.Name, album.Artist)
}

// generateAlbumID creates a consistent hash for an album based on name and artist
func generateAlbumID(name, artist string) string {
	// Albums are grouped by name throughout the database, including
	// multi-artist releases. IDs must use the same identity rule.
	hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name))))
	// Take first 8 characters of hex hash for a clean, short ID
	return hex.EncodeToString(hash[:4])
}

// GetAlbumByNameAndArtist retrieves an album by name and artist name
func (db *DB) GetAlbumByNameAndArtist(albumName, artistName string) (*Album, error) {
	query := `
		SELECT DISTINCT album, 
		       CASE 
		         WHEN COUNT(DISTINCT artist) > 1 THEN 'Various Artists'
		         WHEN MIN(artist) IS NULL OR MIN(artist) = '' THEN 'Unknown Artist'
		         ELSE MIN(artist)
		       END as artist,
		       COUNT(*) as track_count,
		       MIN(year) as year
		FROM music 
		WHERE album IS NOT NULL AND album != '' AND album = $1
		GROUP BY album
		LIMIT 1
	`

	var album Album
	var year sql.NullInt32

	err := db.conn.QueryRow(query, albumName).Scan(&album.Name, &album.Artist, &album.TrackCount, &year)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("album '%s' not found", albumName)
		}
		return nil, fmt.Errorf("failed to query album: %v", err)
	}

	// Handle nullable year field
	if year.Valid {
		album.Year = int(year.Int32)
	}

	// Generate ID from album name using hash
	album.ID = generateAlbumID(album.Name, album.Artist)

	// Get cover art from tracks for this album
	coverArt, err := db.getAlbumCoverArt(album.Name, album.Artist)
	if err == nil {
		album.CoverArtURL = coverArt.CoverArtURL
		album.CoverArtSmallURL = coverArt.CoverArtSmallURL
		album.CoverArtMediumURL = coverArt.CoverArtMediumURL
		album.CoverArtLargeURL = coverArt.CoverArtLargeURL
		album.CoverArtSource = coverArt.CoverArtSource
	}

	return &album, nil
}

// GetAlbumTracks retrieves all tracks for a specific album
func (db *DB) GetAlbumTracks(albumName, artistName string) ([]Music, error) {
	query := `
		SELECT m.id, m.title, m.artist, m.album, m.genre, m.duration, m.release_date, 
		       m.file_path, m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source, m.parsed_from_filename,
		       COALESCE(m.artist_id, ''), m.play_count, m.cover_art_url, m.cover_art_small_url,
		       m.cover_art_medium_url, m.cover_art_large_url, m.cover_art_source,
		       m.last_cover_art_enriched_at, m.created_at, m.updated_at,
		       a.image_url as artist_image_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE m.album = $1
		ORDER BY COALESCE((SELECT disc_number FROM track_audio_properties WHERE track_id = m.id), 1),
		         m.track_number, m.title
	`

	rows, err := db.conn.Query(query, albumName)
	if err != nil {
		return nil, fmt.Errorf("failed to query album tracks: %v", err)
	}
	defer rows.Close()

	var tracks []Music
	for rows.Next() {
		var music Music
		var featuringJSON []byte
		var releaseDate sql.NullTime
		var artistImageURL sql.NullString
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
			&coverArtURL, &coverArtSmallURL, &coverArtMediumURL,
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

		tracks = append(tracks, music)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating music rows: %v", err)
	}

	_ = db.EnrichTrackAudioProperties(tracks)
	return tracks, nil
}

// getAlbumCoverArt retrieves cover art information for an album from its tracks
// It prioritizes non-default cover art from tracks in the following order:
// 1. Tracks with CoverArtLargeURL (best quality)
// 2. Tracks with CoverArtMediumURL
// 3. Tracks with CoverArtSmallURL
// 4. Tracks with ImageURL (embedded metadata)
func (db *DB) getAlbumCoverArt(albumName, artistName string) (Album, error) {
	query := `
		SELECT cover_art_url, cover_art_small_url, cover_art_medium_url, cover_art_large_url, cover_art_source, image_url
		FROM music
		WHERE album = $1 AND (
			cover_art_large_url IS NOT NULL AND cover_art_large_url != '' OR
			cover_art_medium_url IS NOT NULL AND cover_art_medium_url != '' OR
			cover_art_small_url IS NOT NULL AND cover_art_small_url != '' OR
			image_url IS NOT NULL AND image_url != ''
		)
		ORDER BY
			CASE
				WHEN cover_art_large_url IS NOT NULL AND cover_art_large_url != '' THEN 1
				WHEN cover_art_medium_url IS NOT NULL AND cover_art_medium_url != '' THEN 2
				WHEN cover_art_small_url IS NOT NULL AND cover_art_small_url != '' THEN 3
				WHEN image_url IS NOT NULL AND image_url != '' THEN 4
				ELSE 5
			END,
			track_number ASC,
			title ASC
		LIMIT 1
	`

	var album Album
	var coverArtURL, coverArtSmallURL, coverArtMediumURL, coverArtLargeURL, coverArtSource, imageURL sql.NullString

	err := db.conn.QueryRow(query, albumName).Scan(
		&coverArtURL, &coverArtSmallURL, &coverArtMediumURL, &coverArtLargeURL, &coverArtSource, &imageURL,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// No cover art found for this album
			return album, fmt.Errorf("no cover art found for album '%s'", albumName)
		}
		return album, fmt.Errorf("failed to query album cover art: %v", err)
	}

	// Set cover art URLs based on available data
	// Prioritize the best quality available
	if coverArtLargeURL.Valid && coverArtLargeURL.String != "" {
		album.CoverArtURL = coverArtLargeURL.String
		album.CoverArtLargeURL = coverArtLargeURL.String
	} else if coverArtMediumURL.Valid && coverArtMediumURL.String != "" {
		album.CoverArtURL = coverArtMediumURL.String
		album.CoverArtMediumURL = coverArtMediumURL.String
	} else if coverArtSmallURL.Valid && coverArtSmallURL.String != "" {
		album.CoverArtURL = coverArtSmallURL.String
		album.CoverArtSmallURL = coverArtSmallURL.String
	} else if imageURL.Valid && imageURL.String != "" {
		album.CoverArtURL = imageURL.String
		album.CoverArtSource = "embedded"
	}

	// Set all available size URLs
	if coverArtSmallURL.Valid {
		album.CoverArtSmallURL = coverArtSmallURL.String
	}
	if coverArtMediumURL.Valid {
		album.CoverArtMediumURL = coverArtMediumURL.String
	}
	if coverArtLargeURL.Valid {
		album.CoverArtLargeURL = coverArtLargeURL.String
	}
	if coverArtSource.Valid {
		album.CoverArtSource = coverArtSource.String
	} else if imageURL.Valid && imageURL.String != "" {
		album.CoverArtSource = "embedded"
	}

	return album, nil
}
