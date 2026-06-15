package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// generateUserID generates a simple unique ID for users
func generateUserID() string {
	return fmt.Sprintf("user_%d", time.Now().UnixNano())
}

// AddToRecentlyPlayed adds a track to the user's recently played list
func (db *DB) AddToRecentlyPlayed(userID, trackID string) error {
	return db.AddToRecentlyPlayedFrom(userID, trackID, "web", "")
}

func (db *DB) AddToRecentlyPlayedFrom(userID, trackID, source, device string) error {
	// Generate unique ID for the recently played entry
	id := fmt.Sprintf("rp_%d_%s", time.Now().UnixNano(), userID)

	// Insert the recently played track
	query := `
		INSERT INTO recently_played (id, user_id, track_id, played_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, track_id) 
		DO UPDATE SET played_at = $4, id = $1
	`

	_, err := db.conn.Exec(query, id, userID, trackID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add track to recently played: %v", err)
	}
	if err := db.AddListeningHistory(userID, trackID, source, device); err != nil {
		log.Printf("Warning: failed to append listening history: %v", err)
	}

	// Clean up old recently played tracks (keep only last 50 per user)
	cleanupQuery := `
		DELETE FROM recently_played 
		WHERE user_id = $1 AND id NOT IN (
			SELECT id FROM recently_played 
			WHERE user_id = $1 
			ORDER BY played_at DESC 
			LIMIT 50
		)
	`

	_, err = db.conn.Exec(cleanupQuery, userID)
	if err != nil {
		log.Printf("Warning: failed to cleanup old recently played tracks: %v", err)
	}

	return nil
}

// GetRecentlyPlayedTracks gets the recently played tracks for a user
func (db *DB) GetRecentlyPlayedTracks(userID string, limit int) ([]Music, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}

	query := `
		SELECT m.id, m.title, m.artist, COALESCE(m.artist_id, ''),
		       m.album, m.genre, m.duration, m.release_date, m.file_path,
		       m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source,
		       m.parsed_from_filename, m.play_count, m.image_url, m.cover_art_url,
		       m.cover_art_small_url, m.cover_art_medium_url, m.cover_art_large_url,
		       m.cover_art_source, m.last_cover_art_enriched_at, m.created_at, m.updated_at
		FROM recently_played rp
		JOIN music m ON rp.track_id = m.id
		WHERE rp.user_id = $1
		ORDER BY rp.played_at DESC
		LIMIT $2
	`

	rows, err := db.conn.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recently played tracks: %v", err)
	}
	defer rows.Close()

	var tracks []Music
	for rows.Next() {
		var track Music
		var featuringArray []byte
		var releaseDate sql.NullTime
		var imageURL sql.NullString
		var coverArtURL sql.NullString
		var coverArtSmallURL sql.NullString
		var coverArtMediumURL sql.NullString
		var coverArtLargeURL sql.NullString
		var coverArtSource sql.NullString

		err := rows.Scan(
			&track.ID, &track.Title, &track.Artist, &track.ArtistID,
			&track.Album, &track.Genre, &track.Duration, &releaseDate, &track.FilePath,
			&track.FileName, &track.FileSize, &track.Format, &track.Year, &track.TrackNumber,
			&featuringArray, &track.HasMetadata, &track.Confidence, &track.Source,
			&track.ParsedFromFilename, &track.PlayCount, &imageURL, &coverArtURL,
			&coverArtSmallURL, &coverArtMediumURL, &coverArtLargeURL,
			&coverArtSource, &track.LastCoverArtEnrichedAt, &track.CreatedAt, &track.UpdatedAt,
		)

		if releaseDate.Valid {
			track.ReleaseDate = &releaseDate.Time
		}

		if imageURL.Valid {
			track.ImageURL = imageURL.String
		}

		if coverArtURL.Valid {
			track.CoverArtURL = coverArtURL.String
		}

		if coverArtSmallURL.Valid {
			track.CoverArtSmallURL = coverArtSmallURL.String
		}

		if coverArtMediumURL.Valid {
			track.CoverArtMediumURL = coverArtMediumURL.String
		}

		if coverArtLargeURL.Valid {
			track.CoverArtLargeURL = coverArtLargeURL.String
		}

		if coverArtSource.Valid {
			track.CoverArtSource = coverArtSource.String
		}

		if err != nil {
			return nil, fmt.Errorf("failed to scan recently played track: %v", err)
		}

		// Parse featuring array if present
		if len(featuringArray) > 0 {
			if err := json.Unmarshal(featuringArray, &track.Featuring); err != nil {
				log.Printf("Warning: failed to parse featuring array for track %s: %v", track.ID, err)
			}
		}

		tracks = append(tracks, track)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recently played tracks: %v", err)
	}

	return tracks, nil
}

// LikeTrack adds a track to user's liked tracks
func (db *DB) LikeTrack(userID, trackID string) error {
	// Generate unique ID for liked track entry
	id := fmt.Sprintf("lt_%d_%s", time.Now().UnixNano(), userID)

	query := `
		INSERT INTO liked_tracks (id, user_id, track_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, track_id) DO NOTHING
	`

	_, err := db.conn.Exec(query, id, userID, trackID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to like track: %v", err)
	}

	return db.touchSmartPlaylists(userID)
}

// UnlikeTrack removes a track from user's liked tracks
func (db *DB) UnlikeTrack(userID, trackID string) error {
	query := `
		DELETE FROM liked_tracks 
		WHERE user_id = $1 AND track_id = $2
	`

	result, err := db.conn.Exec(query, userID, trackID)
	if err != nil {
		return fmt.Errorf("failed to unlike track: %v", err)
	}

	// Check if any row was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("track was not in liked tracks")
	}

	return db.touchSmartPlaylists(userID)
}

// GetLikedTracks gets all liked tracks for a user
func (db *DB) GetLikedTracks(userID string) ([]Music, error) {
	query := `
		SELECT m.id, m.title, m.artist, COALESCE(m.artist_id, ''),
		       m.album, m.genre, m.duration, m.release_date, m.file_path,
		       m.file_name, m.file_size, m.format, m.year, m.track_number,
		       m.featuring, m.has_metadata, m.confidence, m.source,
		       m.parsed_from_filename, m.play_count, m.cover_art_url,
		       m.cover_art_small_url, m.cover_art_medium_url, m.cover_art_large_url,
		       m.cover_art_source, m.last_cover_art_enriched_at, m.created_at, m.updated_at
		FROM liked_tracks lt
		JOIN music m ON lt.track_id = m.id
		WHERE lt.user_id = $1
		ORDER BY lt.created_at DESC
	`

	rows, err := db.conn.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get liked tracks: %v", err)
	}
	defer rows.Close()

	var tracks []Music
	for rows.Next() {
		var track Music
		var featuringArray []byte
		var releaseDate sql.NullTime
		var coverArtURL sql.NullString
		var coverArtSmallURL sql.NullString
		var coverArtMediumURL sql.NullString
		var coverArtLargeURL sql.NullString
		var coverArtSource sql.NullString

		err := rows.Scan(
			&track.ID, &track.Title, &track.Artist, &track.ArtistID,
			&track.Album, &track.Genre, &track.Duration, &releaseDate, &track.FilePath,
			&track.FileName, &track.FileSize, &track.Format, &track.Year, &track.TrackNumber,
			&featuringArray, &track.HasMetadata, &track.Confidence, &track.Source,
			&track.ParsedFromFilename, &track.PlayCount, &coverArtURL,
			&coverArtSmallURL, &coverArtMediumURL, &coverArtLargeURL,
			&coverArtSource, &track.LastCoverArtEnrichedAt, &track.CreatedAt, &track.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan liked track: %v", err)
		}

		// Handle nullable fields
		if releaseDate.Valid {
			track.ReleaseDate = &releaseDate.Time
		}

		if coverArtURL.Valid {
			track.CoverArtURL = coverArtURL.String
		}

		if coverArtSmallURL.Valid {
			track.CoverArtSmallURL = coverArtSmallURL.String
		}

		if coverArtMediumURL.Valid {
			track.CoverArtMediumURL = coverArtMediumURL.String
		}

		if coverArtLargeURL.Valid {
			track.CoverArtLargeURL = coverArtLargeURL.String
		}

		if coverArtSource.Valid {
			track.CoverArtSource = coverArtSource.String
		}

		// Parse featuring array if present
		if len(featuringArray) > 0 {
			if err := json.Unmarshal(featuringArray, &track.Featuring); err != nil {
				log.Printf("Warning: failed to parse featuring array for track %s: %v", track.ID, err)
			}
		}

		tracks = append(tracks, track)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating liked tracks: %v", err)
	}

	return tracks, nil
}

// IsTrackLiked checks if a track is liked by a user
func (db *DB) IsTrackLiked(userID, trackID string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM liked_tracks 
		WHERE user_id = $1 AND track_id = $2
	`

	var count int
	err := db.conn.QueryRow(query, userID, trackID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if track is liked: %v", err)
	}

	return count > 0, nil
}
