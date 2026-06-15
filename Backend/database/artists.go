package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// GenerateArtistHash creates a consistent hash for an artist based on name only
func GenerateArtistHash(artistName string) string {
	// Use SHA256 to match database format (32-character hex strings)
	// This ensures consistency with existing artist IDs in database
	hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(artistName))))
	return hex.EncodeToString(hash[:]) // Use full hash (32 characters)
}

func nullableArtistIdentifier(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

// GetArtistByHash retrieves an artist by hash (supports both prefixed and non-prefixed for backward compatibility)
func (db *DB) GetArtistByHash(hash string) (*Artist, error) {
	// First try to query without spotify_url to handle schema inconsistencies
	query := `
		SELECT id, name, musicbrainz_id, musicbrainz_url, image_url, image_small_url, 
		       image_medium_url, image_large_url, country, tags, biography, 
		       spotify_id, followers, popularity, genres, 
		       external_urls, uri, href, type, api_data,
		       last_enriched_at, created_at, updated_at
		FROM artists 
		WHERE id = $1 OR id = $2
		ORDER BY id LIMIT 1`

	var artist Artist
	var genresJSON []byte
	var externalURLsJSON []byte
	var tagsJSON []byte
	var lastEnrichedAt sql.NullTime
	var spotifyID sql.NullString
	var imageURL sql.NullString
	var imageSmallURL sql.NullString
	var imageMediumURL sql.NullString
	var imageLargeURL sql.NullString
	var biography sql.NullString
	var country sql.NullString
	var musicbrainzID sql.NullString
	var musicbrainzURL sql.NullString
	var uri sql.NullString
	var href sql.NullString
	var apiData sql.NullString

	// Try both clean hash and prefixed hash for backward compatibility
	prefixedHash := "artist_" + hash
	err := db.conn.QueryRow(query, hash, prefixedHash).Scan(
		&artist.ID, &artist.Name, &musicbrainzID, &musicbrainzURL,
		&imageURL, &imageSmallURL, &imageMediumURL,
		&imageLargeURL, &country, &tagsJSON, &biography,
		&spotifyID, &artist.Followers, &artist.Popularity,
		&genresJSON, &externalURLsJSON,
		&uri, &href, &artist.Type, &apiData,
		&lastEnrichedAt, &artist.CreatedAt, &artist.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artist not found")
		}
		return nil, fmt.Errorf("failed to query artist: %v", err)
	}

	// Handle nullable fields
	if musicbrainzID.Valid {
		artist.MusicBrainzID = musicbrainzID.String
	}
	if musicbrainzURL.Valid {
		artist.MusicBrainzURL = musicbrainzURL.String
	}
	if spotifyID.Valid {
		artist.SpotifyID = spotifyID.String
	}
	if imageURL.Valid {
		artist.ImageURL = imageURL.String
	}
	if imageSmallURL.Valid {
		artist.ImageSmallURL = imageSmallURL.String
	}
	if imageMediumURL.Valid {
		artist.ImageMediumURL = imageMediumURL.String
	}
	if imageLargeURL.Valid {
		artist.ImageLargeURL = imageLargeURL.String
	}
	if biography.Valid {
		artist.Biography = biography.String
	}
	if country.Valid {
		artist.Country = country.String
	}
	if uri.Valid {
		artist.URI = uri.String
	}
	if href.Valid {
		artist.HREF = href.String
	}
	if apiData.Valid {
		artist.APIData = apiData.String
	}
	if lastEnrichedAt.Valid {
		artist.LastEnrichedAt = &lastEnrichedAt.Time
	}

	if len(genresJSON) > 0 {
		if err := json.Unmarshal(genresJSON, &artist.Genres); err != nil {
			log.Printf("Warning: Failed to unmarshal genres for artist %s: %v", artist.ID, err)
		}
	}

	if len(externalURLsJSON) > 0 {
		if err := json.Unmarshal(externalURLsJSON, &artist.ExternalURLs); err != nil {
			log.Printf("Warning: Failed to unmarshal external URLs for artist %s: %v", artist.ID, err)
		}
	}

	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &artist.Tags); err != nil {
			log.Printf("Warning: Failed to unmarshal tags for artist %s: %v", artist.ID, err)
		}
	}

	return &artist, nil
}

// GetOrCreateArtistByHash gets an existing artist by hash or creates a new one
func (db *DB) GetOrCreateArtistByHash(name string) (*Artist, error) {
	// First try to find artist by name
	artists, err := db.GetAllArtists()
	if err != nil {
		return nil, fmt.Errorf("failed to get artists: %v", err)
	}

	for _, artist := range artists {
		if strings.EqualFold(artist.Name, name) {
			// Found existing artist, return it
			return &artist, nil
		}
	}

	// Artist not found, create new one with hash
	newArtist := &Artist{
		ID:   GenerateArtistHash(name),
		Name: name,
		Type: "artist",
	}

	err = db.AddArtist(newArtist)
	if err != nil {
		return nil, fmt.Errorf("failed to create new artist: %v", err)
	}

	return newArtist, nil
}

// Artist operations

// GetAllArtists retrieves all artists from database
func (db *DB) GetAllArtists() ([]Artist, error) {
	query := `
		SELECT id, name, musicbrainz_id, musicbrainz_url, image_url, image_small_url, 
		       image_medium_url, image_large_url, country, tags, biography,
		       spotify_id, followers, popularity, genres, 
		       external_urls, uri, href, type, api_data,
		       last_enriched_at, created_at, updated_at
		FROM artists 
		ORDER BY name`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query artists: %v", err)
	}
	defer rows.Close()

	var artists []Artist
	for rows.Next() {
		var artist Artist
		var genresJSON []byte
		var externalURLsJSON []byte
		var tagsJSON []byte
		var lastEnrichedAt sql.NullTime
		var spotifyID sql.NullString
		var imageURL sql.NullString
		var imageSmallURL sql.NullString
		var imageMediumURL sql.NullString
		var imageLargeURL sql.NullString
		var biography sql.NullString
		var country sql.NullString
		var musicbrainzID sql.NullString
		var musicbrainzURL sql.NullString
		var uri sql.NullString
		var href sql.NullString
		var apiData sql.NullString

		err := rows.Scan(
			&artist.ID, &artist.Name, &musicbrainzID, &musicbrainzURL,
			&imageURL, &imageSmallURL, &imageMediumURL,
			&imageLargeURL, &country, &tagsJSON, &biography,
			&spotifyID, &artist.Followers, &artist.Popularity,
			&genresJSON, &externalURLsJSON,
			&uri, &href, &artist.Type, &apiData,
			&lastEnrichedAt, &artist.CreatedAt, &artist.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artist row: %v", err)
		}

		// Handle nullable fields
		if musicbrainzID.Valid {
			artist.MusicBrainzID = musicbrainzID.String
		}
		if musicbrainzURL.Valid {
			artist.MusicBrainzURL = musicbrainzURL.String
		}
		if spotifyID.Valid {
			artist.SpotifyID = spotifyID.String
		}
		if imageURL.Valid {
			artist.ImageURL = imageURL.String
		}
		if imageSmallURL.Valid {
			artist.ImageSmallURL = imageSmallURL.String
		}
		if imageMediumURL.Valid {
			artist.ImageMediumURL = imageMediumURL.String
		}
		if imageLargeURL.Valid {
			artist.ImageLargeURL = imageLargeURL.String
		}
		if biography.Valid {
			artist.Biography = biography.String
		}
		if country.Valid {
			artist.Country = country.String
		}
		if uri.Valid {
			artist.URI = uri.String
		}
		if href.Valid {
			artist.HREF = href.String
		}
		if apiData.Valid {
			artist.APIData = apiData.String
		}
		if lastEnrichedAt.Valid {
			artist.LastEnrichedAt = &lastEnrichedAt.Time
		}

		if len(genresJSON) > 0 {
			if err := json.Unmarshal(genresJSON, &artist.Genres); err != nil {
				log.Printf("Warning: Failed to unmarshal genres for artist %s: %v", artist.ID, err)
			}
		}

		if len(externalURLsJSON) > 0 {
			if err := json.Unmarshal(externalURLsJSON, &artist.ExternalURLs); err != nil {
				log.Printf("Warning: Failed to unmarshal external URLs for artist %s: %v", artist.ID, err)
			}
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &artist.Tags); err != nil {
				log.Printf("Warning: Failed to unmarshal tags for artist %s: %v", artist.ID, err)
			}
		}

		artists = append(artists, artist)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artist rows: %v", err)
	}

	return artists, nil
}

// GetArtistByID retrieves an artist by ID
func (db *DB) GetArtistByID(id string) (*Artist, error) {
	query := `
		SELECT id, name, musicbrainz_id, musicbrainz_url, image_url, image_small_url, 
		       image_medium_url, image_large_url, country, tags, biography,
		       spotify_id, followers, popularity, genres, 
		       external_urls, uri, href, type, api_data,
		       last_enriched_at, created_at, updated_at
		FROM artists 
		WHERE id = $1`

	var artist Artist
	var genresJSON []byte
	var externalURLsJSON []byte
	var tagsJSON []byte
	var lastEnrichedAt sql.NullTime
	var spotifyID sql.NullString
	var imageURL sql.NullString
	var imageSmallURL sql.NullString
	var imageMediumURL sql.NullString
	var imageLargeURL sql.NullString
	var biography sql.NullString
	var country sql.NullString
	var musicbrainzID sql.NullString
	var musicbrainzURL sql.NullString
	var uri sql.NullString
	var href sql.NullString
	var apiData sql.NullString

	err := db.conn.QueryRow(query, id).Scan(
		&artist.ID, &artist.Name, &musicbrainzID, &musicbrainzURL,
		&imageURL, &imageSmallURL, &imageMediumURL,
		&imageLargeURL, &country, &tagsJSON, &biography,
		&spotifyID, &artist.Followers, &artist.Popularity,
		&genresJSON, &externalURLsJSON,
		&uri, &href, &artist.Type, &apiData,
		&lastEnrichedAt, &artist.CreatedAt, &artist.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artist not found")
		}
		return nil, fmt.Errorf("failed to query artist: %v", err)
	}

	// Handle nullable fields
	if musicbrainzID.Valid {
		artist.MusicBrainzID = musicbrainzID.String
	}
	if musicbrainzURL.Valid {
		artist.MusicBrainzURL = musicbrainzURL.String
	}
	if spotifyID.Valid {
		artist.SpotifyID = spotifyID.String
	}
	if imageURL.Valid {
		artist.ImageURL = imageURL.String
	}
	if imageSmallURL.Valid {
		artist.ImageSmallURL = imageSmallURL.String
	}
	if imageMediumURL.Valid {
		artist.ImageMediumURL = imageMediumURL.String
	}
	if imageLargeURL.Valid {
		artist.ImageLargeURL = imageLargeURL.String
	}
	if biography.Valid {
		artist.Biography = biography.String
	}
	if country.Valid {
		artist.Country = country.String
	}
	if uri.Valid {
		artist.URI = uri.String
	}
	if href.Valid {
		artist.HREF = href.String
	}
	if apiData.Valid {
		artist.APIData = apiData.String
	}
	if lastEnrichedAt.Valid {
		artist.LastEnrichedAt = &lastEnrichedAt.Time
	}

	if len(genresJSON) > 0 {
		if err := json.Unmarshal(genresJSON, &artist.Genres); err != nil {
			log.Printf("Warning: Failed to unmarshal genres for artist %s: %v", artist.ID, err)
		}
	}

	if len(externalURLsJSON) > 0 {
		if err := json.Unmarshal(externalURLsJSON, &artist.ExternalURLs); err != nil {
			log.Printf("Warning: Failed to unmarshal external URLs for artist %s: %v", artist.ID, err)
		}
	}

	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &artist.Tags); err != nil {
			log.Printf("Warning: Failed to unmarshal tags for artist %s: %v", artist.ID, err)
		}
	}

	return &artist, nil
}

// GetArtistByName retrieves an artist by name
func (db *DB) GetArtistByName(name string) (*Artist, error) {
	query := `
		SELECT id, name, musicbrainz_id, musicbrainz_url, image_url, image_small_url, 
		       image_medium_url, image_large_url, country, tags, biography,
		       spotify_id, followers, popularity, genres, 
		       external_urls, uri, href, type, api_data,
		       last_enriched_at, created_at, updated_at
		FROM artists 
		WHERE name = $1`

	var artist Artist
	var genresJSON []byte
	var externalURLsJSON []byte
	var tagsJSON []byte
	var lastEnrichedAt sql.NullTime
	var spotifyID sql.NullString
	var imageURL sql.NullString
	var imageSmallURL sql.NullString
	var imageMediumURL sql.NullString
	var imageLargeURL sql.NullString
	var biography sql.NullString
	var country sql.NullString
	var musicbrainzID sql.NullString
	var musicbrainzURL sql.NullString
	var uri sql.NullString
	var href sql.NullString
	var apiData sql.NullString

	err := db.conn.QueryRow(query, name).Scan(
		&artist.ID, &artist.Name, &musicbrainzID, &musicbrainzURL,
		&imageURL, &imageSmallURL, &imageMediumURL,
		&imageLargeURL, &country, &tagsJSON, &biography,
		&spotifyID, &artist.Followers, &artist.Popularity,
		&genresJSON, &externalURLsJSON,
		&uri, &href, &artist.Type, &apiData,
		&lastEnrichedAt, &artist.CreatedAt, &artist.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artist not found")
		}
		return nil, fmt.Errorf("failed to query artist: %v", err)
	}

	// Handle nullable fields
	if musicbrainzID.Valid {
		artist.MusicBrainzID = musicbrainzID.String
	}
	if musicbrainzURL.Valid {
		artist.MusicBrainzURL = musicbrainzURL.String
	}
	if spotifyID.Valid {
		artist.SpotifyID = spotifyID.String
	}
	if imageURL.Valid {
		artist.ImageURL = imageURL.String
	}
	if imageSmallURL.Valid {
		artist.ImageSmallURL = imageSmallURL.String
	}
	if imageMediumURL.Valid {
		artist.ImageMediumURL = imageMediumURL.String
	}
	if imageLargeURL.Valid {
		artist.ImageLargeURL = imageLargeURL.String
	}
	if biography.Valid {
		artist.Biography = biography.String
	}
	if country.Valid {
		artist.Country = country.String
	}
	if uri.Valid {
		artist.URI = uri.String
	}
	if href.Valid {
		artist.HREF = href.String
	}
	if apiData.Valid {
		artist.APIData = apiData.String
	}
	if lastEnrichedAt.Valid {
		artist.LastEnrichedAt = &lastEnrichedAt.Time
	}

	if len(genresJSON) > 0 {
		if err := json.Unmarshal(genresJSON, &artist.Genres); err != nil {
			log.Printf("Warning: Failed to unmarshal genres for artist %s: %v", artist.ID, err)
		}
	}

	if len(externalURLsJSON) > 0 {
		if err := json.Unmarshal(externalURLsJSON, &artist.ExternalURLs); err != nil {
			log.Printf("Warning: Failed to unmarshal external URLs for artist %s: %v", artist.ID, err)
		}
	}

	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &artist.Tags); err != nil {
			log.Printf("Warning: Failed to unmarshal tags for artist %s: %v", artist.ID, err)
		}
	}

	return &artist, nil
}

// GetArtistBySpotifyID retrieves an artist by Spotify ID
func (db *DB) GetArtistBySpotifyID(spotifyID string) (*Artist, error) {
	query := `
		SELECT id, name, musicbrainz_id, musicbrainz_url, image_url, image_small_url, 
		       image_medium_url, image_large_url, country, tags, biography,
		       spotify_id, followers, popularity, genres, 
		       external_urls, uri, href, type, api_data,
		       last_enriched_at, created_at, updated_at
		FROM artists 
		WHERE spotify_id = $1`

	var artist Artist
	var genresJSON []byte
	var externalURLsJSON []byte
	var tagsJSON []byte
	var lastEnrichedAt sql.NullTime
	var spotifyIDFromDB sql.NullString
	var imageURL sql.NullString
	var imageSmallURL sql.NullString
	var imageMediumURL sql.NullString
	var imageLargeURL sql.NullString
	var biography sql.NullString
	var country sql.NullString
	var musicbrainzID sql.NullString
	var musicbrainzURL sql.NullString
	var uri sql.NullString
	var href sql.NullString
	var apiData sql.NullString

	err := db.conn.QueryRow(query, spotifyID).Scan(
		&artist.ID, &artist.Name, &musicbrainzID, &musicbrainzURL,
		&imageURL, &imageSmallURL, &imageMediumURL,
		&imageLargeURL, &country, &tagsJSON, &biography,
		&spotifyIDFromDB, &artist.Followers, &artist.Popularity,
		&genresJSON, &externalURLsJSON,
		&uri, &href, &artist.Type, &apiData,
		&lastEnrichedAt, &artist.CreatedAt, &artist.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artist not found")
		}
		return nil, fmt.Errorf("failed to query artist: %v", err)
	}

	// Handle nullable fields
	if musicbrainzID.Valid {
		artist.MusicBrainzID = musicbrainzID.String
	}
	if musicbrainzURL.Valid {
		artist.MusicBrainzURL = musicbrainzURL.String
	}
	if imageURL.Valid {
		artist.ImageURL = imageURL.String
	}
	if imageSmallURL.Valid {
		artist.ImageSmallURL = imageSmallURL.String
	}
	if imageMediumURL.Valid {
		artist.ImageMediumURL = imageMediumURL.String
	}
	if imageLargeURL.Valid {
		artist.ImageLargeURL = imageLargeURL.String
	}
	if biography.Valid {
		artist.Biography = biography.String
	}
	if country.Valid {
		artist.Country = country.String
	}
	if uri.Valid {
		artist.URI = uri.String
	}
	if href.Valid {
		artist.HREF = href.String
	}
	if apiData.Valid {
		artist.APIData = apiData.String
	}
	if lastEnrichedAt.Valid {
		artist.LastEnrichedAt = &lastEnrichedAt.Time
	}

	if len(genresJSON) > 0 {
		if err := json.Unmarshal(genresJSON, &artist.Genres); err != nil {
			log.Printf("Warning: Failed to unmarshal genres for artist %s: %v", artist.ID, err)
		}
	}

	if len(externalURLsJSON) > 0 {
		if err := json.Unmarshal(externalURLsJSON, &artist.ExternalURLs); err != nil {
			log.Printf("Warning: Failed to unmarshal external URLs for artist %s: %v", artist.ID, err)
		}
	}

	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &artist.Tags); err != nil {
			log.Printf("Warning: Failed to unmarshal tags for artist %s: %v", artist.ID, err)
		}
	}

	return &artist, nil
}

// AddArtist adds a new artist to database
func (db *DB) AddArtist(artist *Artist) error {
	// Set default values for fields that might not be provided
	artist.CreatedAt = time.Now()
	artist.UpdatedAt = time.Now()

	// Set default values for optional fields if not provided
	if artist.Type == "" {
		artist.Type = "artist"
	}
	if artist.SpotifyURL == "" && artist.SpotifyID != "" {
		artist.SpotifyURL = "https://open.spotify.com/artist/" + artist.SpotifyID
	}
	if artist.APIData == "" {
		artist.APIData = "{}"
	}

	query := `
		INSERT INTO artists (
			id, name, musicbrainz_id, musicbrainz_url, image_url, image_small_url,
			image_medium_url, image_large_url, country, tags, biography,
			spotify_id, followers, popularity, genres,
			external_urls, uri, href, type, api_data,
			last_enriched_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)`

	var genresJSON []byte
	if len(artist.Genres) > 0 {
		var err error
		genresJSON, err = json.Marshal(artist.Genres)
		if err != nil {
			return fmt.Errorf("failed to marshal genres: %v", err)
		}
	} else {
		genresJSON = []byte("[]")
	}

	var externalURLsJSON []byte
	if len(artist.ExternalURLs) > 0 {
		var err error
		externalURLsJSON, err = json.Marshal(artist.ExternalURLs)
		if err != nil {
			return fmt.Errorf("failed to marshal external URLs: %v", err)
		}
	} else {
		externalURLsJSON = []byte("{}")
	}

	var tagsJSON []byte
	if len(artist.Tags) > 0 {
		var err error
		tagsJSON, err = json.Marshal(artist.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %v", err)
		}
	} else {
		tagsJSON = []byte("[]")
	}

	var lastEnrichedAt interface{}
	if artist.LastEnrichedAt != nil {
		lastEnrichedAt = artist.LastEnrichedAt
	}

	_, err := db.conn.Exec(query,
		artist.ID, artist.Name, nullableArtistIdentifier(artist.MusicBrainzID), artist.MusicBrainzURL,
		artist.ImageURL, artist.ImageSmallURL, artist.ImageMediumURL,
		artist.ImageLargeURL, artist.Country, tagsJSON, artist.Biography,
		nullableArtistIdentifier(artist.SpotifyID),
		artist.Followers, artist.Popularity,
		genresJSON, externalURLsJSON,
		artist.URI, artist.HREF, artist.Type, artist.APIData,
		lastEnrichedAt, artist.CreatedAt, artist.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert artist: %v", err)
	}

	return nil
}

// UpdateArtist updates an existing artist
func (db *DB) UpdateArtist(artist *Artist) error {
	query := `
		UPDATE artists SET 
			name = $2, musicbrainz_id = $3, musicbrainz_url = $4, image_url = $5,
			image_small_url = $6, image_medium_url = $7, image_large_url = $8,
			country = $9, tags = $10, biography = $11,
			spotify_id = $12, followers = $13, popularity = $14,
			genres = $15, external_urls = $16, uri = $17, href = $18,
			type = $19, api_data = $20, last_enriched_at = $21, updated_at = $22
		WHERE id = $1`

	artist.UpdatedAt = time.Now()

	var genresJSON []byte
	if len(artist.Genres) > 0 {
		var err error
		genresJSON, err = json.Marshal(artist.Genres)
		if err != nil {
			return fmt.Errorf("failed to marshal genres: %v", err)
		}
	} else {
		genresJSON = []byte("[]")
	}

	var externalURLsJSON []byte
	if len(artist.ExternalURLs) > 0 {
		var err error
		externalURLsJSON, err = json.Marshal(artist.ExternalURLs)
		if err != nil {
			return fmt.Errorf("failed to marshal external URLs: %v", err)
		}
	} else {
		externalURLsJSON = []byte("{}")
	}

	var lastEnrichedAt interface{}
	if artist.LastEnrichedAt != nil {
		lastEnrichedAt = artist.LastEnrichedAt
	}

	result, err := db.conn.Exec(query,
		artist.ID, artist.Name, nullableArtistIdentifier(artist.MusicBrainzID), artist.MusicBrainzURL,
		artist.ImageURL, artist.ImageSmallURL, artist.ImageMediumURL,
		artist.ImageLargeURL, artist.Country, artist.Tags, artist.Biography,
		nullableArtistIdentifier(artist.SpotifyID),
		artist.Followers, artist.Popularity,
		genresJSON, externalURLsJSON,
		artist.URI, artist.HREF, artist.Type, artist.APIData,
		lastEnrichedAt, artist.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update artist: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("artist not found")
	}

	return nil
}

// UpdateArtistImage updates every artist image size to the same locally stored image.
func (db *DB) UpdateArtistImage(id, imageURL string) error {
	result, err := db.conn.Exec(`
		UPDATE artists
		SET image_url = $2,
		    image_small_url = $2,
		    image_medium_url = $2,
		    image_large_url = $2,
		    updated_at = $3
		WHERE id = $1
	`, id, imageURL, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update artist image: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("artist not found")
	}

	return nil
}

// DeleteArtist removes an artist from database
func (db *DB) DeleteArtist(id string) error {
	query := "DELETE FROM artists WHERE id = $1"

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete artist: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("artist not found")
	}

	return nil
}

// ArtistOrStoreArtist gets an existing artist by name or creates a new one
func (db *DB) ArtistOrStoreArtist(name string) (*Artist, error) {
	// First try to get existing artist
	artist, err := db.GetArtistByName(name)
	if err == nil {
		return artist, nil
	}

	// If artist not found, create a new one
	if err.Error() == "artist not found" {
		newArtist := &Artist{
			ID:   GenerateArtistHash(name),
			Name: name,
		}

		err = db.AddArtist(newArtist)
		if err != nil {
			return nil, fmt.Errorf("failed to create new artist: %v", err)
		}

		return newArtist, nil
	}

	// Some other error occurred
	return nil, fmt.Errorf("failed to get artist: %v", err)
}
