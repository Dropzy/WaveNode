package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
)

// Library operations

// GetAllArtistsForLibrary retrieves all artists for library view
func (db *DB) GetAllArtistsForLibrary() ([]map[string]interface{}, error) {
	query := `
		SELECT artist, album
		FROM music
		WHERE artist IS NOT NULL AND BTRIM(artist) != ''
		ORDER BY artist
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query artists for library: %v", err)
	}
	defer rows.Close()

	type libraryArtistTrackRow struct {
		artist string
		album  string
	}
	type artistAggregate struct {
		name       string
		albums     map[string]struct{}
		trackCount int
	}

	trackRows := make([]libraryArtistTrackRow, 0)
	for rows.Next() {
		var rawArtist string
		var album sql.NullString

		if err := rows.Scan(&rawArtist, &album); err != nil {
			return nil, fmt.Errorf("failed to scan artist row: %v", err)
		}

		albumName := ""
		if album.Valid {
			albumName = album.String
		}
		trackRows = append(trackRows, libraryArtistTrackRow{artist: rawArtist, album: albumName})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artist rows: %v", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("failed to close artist row query: %v", err)
	}

	storedArtists, err := db.GetAllArtists()
	if err != nil {
		return nil, fmt.Errorf("failed to query stored artists for library: %v", err)
	}

	storedByName := make(map[string]Artist, len(storedArtists))
	for _, artist := range storedArtists {
		storedByName[strings.ToLower(strings.TrimSpace(artist.Name))] = artist
	}

	aggregates := make(map[string]*artistAggregate)
	for _, row := range trackRows {
		name := PrimaryArtistName(row.artist)
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			continue
		}

		aggregate, exists := aggregates[key]
		if !exists {
			aggregate = &artistAggregate{
				name:   name,
				albums: make(map[string]struct{}),
			}
			aggregates[key] = aggregate
		}
		if storedArtist, exists := storedByName[key]; exists {
			aggregate.name = storedArtist.Name
		}
		if strings.TrimSpace(row.album) != "" {
			aggregate.albums[row.album] = struct{}{}
		}
		aggregate.trackCount++
	}

	var artists []map[string]interface{}
	for key, aggregate := range aggregates {
		storedArtist, exists := storedByName[key]
		artistID := GenerateArtistHash(aggregate.name)
		imageURL := ""
		imageSmallURL := ""
		imageMediumURL := ""
		imageLargeURL := ""
		if exists {
			artistID = storedArtist.ID
			imageURL = storedArtist.ImageURL
			imageSmallURL = storedArtist.ImageSmallURL
			imageMediumURL = storedArtist.ImageMediumURL
			imageLargeURL = storedArtist.ImageLargeURL
		}

		artist := map[string]interface{}{
			"id":               artistID,
			"name":             aggregate.name,
			"album_count":      len(aggregate.albums),
			"track_count":      aggregate.trackCount,
			"image_url":        imageURL,
			"image_small_url":  imageSmallURL,
			"image_medium_url": imageMediumURL,
			"image_large_url":  imageLargeURL,
		}

		artists = append(artists, artist)
	}

	sort.Slice(artists, func(i, j int) bool {
		return strings.ToLower(fmt.Sprint(artists[i]["name"])) < strings.ToLower(fmt.Sprint(artists[j]["name"]))
	})

	return artists, nil
}

func (db *DB) GetLibraryArtistByID(artistID string) (*Artist, error) {
	artists, err := db.GetAllArtistsForLibrary()
	if err != nil {
		return nil, err
	}

	for _, item := range artists {
		if fmt.Sprint(item["id"]) != artistID {
			continue
		}

		return &Artist{
			ID:             artistID,
			Name:           fmt.Sprint(item["name"]),
			ImageURL:       fmt.Sprint(item["image_url"]),
			ImageSmallURL:  fmt.Sprint(item["image_small_url"]),
			ImageMediumURL: fmt.Sprint(item["image_medium_url"]),
			ImageLargeURL:  fmt.Sprint(item["image_large_url"]),
			Type:           "artist",
		}, nil
	}

	return nil, fmt.Errorf("artist not found")
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
		WHERE m.artist IS NOT NULL AND BTRIM(m.artist) != ''
		ORDER BY m.album, m.track_number, m.title
	`

	rows, err := db.conn.Query(query)
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

		if PrimaryArtistNameMatches(music.Artist, artistName) {
			tracks = append(tracks, music)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating music rows: %v", err)
	}

	_ = db.EnrichTrackAudioProperties(tracks)
	return tracks, nil
}
