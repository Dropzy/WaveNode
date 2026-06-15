package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// Library operations

// GetAllArtistsForLibrary retrieves all artists for library view
func (db *DB) GetAllArtistsForLibrary() ([]map[string]interface{}, error) {
	query := `
		SELECT DISTINCT 
			a.id as id,
			m.artist as name,
			COUNT(DISTINCT m.album) as album_count,
			COUNT(m.id) as track_count,
			a.image_url as image_url,
			a.image_small_url,
			a.image_medium_url,
			a.image_large_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE m.artist IS NOT NULL AND m.artist != ''
		GROUP BY m.artist, a.id, a.image_url, a.image_small_url, a.image_medium_url, a.image_large_url
		ORDER BY m.artist
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query artists for library: %v", err)
	}
	defer rows.Close()

	var artists []map[string]interface{}
	for rows.Next() {
		var id sql.NullString
		var name sql.NullString
		var albumCount int
		var trackCount int
		var imageURL sql.NullString
		var imageSmallURL sql.NullString
		var imageMediumURL sql.NullString
		var imageLargeURL sql.NullString

		err := rows.Scan(&id, &name, &albumCount, &trackCount, &imageURL, &imageSmallURL, &imageMediumURL, &imageLargeURL)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artist row: %v", err)
		}

		// If no artist record exists, generate a hash for backwards compatibility
		artistID := id.String
		if !id.Valid || artistID == "" {
			artistID = GenerateArtistHash(name.String)
		}

		artist := map[string]interface{}{
			"id":               artistID,
			"name":             name.String,
			"album_count":      albumCount,
			"track_count":      trackCount,
			"image_url":        imageURL.String,
			"image_small_url":  imageSmallURL.String,
			"image_medium_url": imageMediumURL.String,
			"image_large_url":  imageLargeURL.String,
		}

		artists = append(artists, artist)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artist rows: %v", err)
	}

	return artists, nil
}

// GetArtistTracks retrieves all tracks for a specific artist
func (db *DB) GetArtistTracks(artistName string) ([]Music, error) {
	query := `
		SELECT m.id, m.title, m.artist, m.album, m.genre, m.duration, m.release_date, 
		       m.file_path, m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source, m.parsed_from_filename,
		       COALESCE(m.artist_id, ''), m.play_count, m.image_url,
		       m.cover_art_url, m.cover_art_small_url,
		       m.cover_art_medium_url, m.cover_art_large_url, m.cover_art_source,
		       m.last_cover_art_enriched_at, m.created_at, m.updated_at,
		       a.image_url as artist_image_url
		FROM music m
		LEFT JOIN artists a ON m.artist_id = a.id
		WHERE m.artist = $1
		ORDER BY m.album, m.track_number, m.title
	`

	rows, err := db.conn.Query(query, artistName)
	if err != nil {
		return nil, fmt.Errorf("failed to query artist tracks: %v", err)
	}
	defer rows.Close()

	var tracks []Music
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
			&imageURL,
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

		tracks = append(tracks, music)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating music rows: %v", err)
	}

	_ = db.EnrichTrackAudioProperties(tracks)
	return tracks, nil
}
