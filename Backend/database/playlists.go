package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Playlist operations

func marshalTrackIDs(trackIDs []string) ([]byte, error) {
	if trackIDs == nil {
		trackIDs = []string{}
	}

	return json.Marshal(trackIDs)
}

func appendUniqueTrackID(trackIDs []string, trackID string) []string {
	for _, existingID := range trackIDs {
		if existingID == trackID {
			return trackIDs
		}
	}

	return append(trackIDs, trackID)
}

func removeTrackID(trackIDs []string, trackID string) []string {
	result := make([]string, 0, len(trackIDs))
	for _, existingID := range trackIDs {
		if existingID != trackID {
			result = append(result, existingID)
		}
	}

	return result
}

// GetAllPlaylists retrieves all playlists from the database
func (db *DB) GetAllPlaylists() ([]Playlist, error) {
	return db.getPlaylists("")
}

func (db *DB) GetUserPlaylists(userID string) ([]Playlist, error) {
	return db.getPlaylists(userID)
}

func (db *DB) getPlaylists(userID string) ([]Playlist, error) {
	query := `SELECT id, user_id, name, description, playlist_type, smart_rules, track_ids, created_at, updated_at FROM playlists`
	args := []interface{}{}
	if userID != "" {
		query += ` WHERE user_id = $1`
		args = append(args, userID)
	}
	query += ` ORDER BY name`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query playlists: %v", err)
	}
	defer rows.Close()

	var playlists []Playlist
	for rows.Next() {
		var playlist Playlist
		var smartRulesJSON []byte
		var trackIDsJSON []byte

		err := rows.Scan(&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Description, &playlist.Type, &smartRulesJSON, &trackIDsJSON, &playlist.CreatedAt, &playlist.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan playlist row: %v", err)
		}

		if len(trackIDsJSON) > 0 {
			if err := json.Unmarshal(trackIDsJSON, &playlist.TrackIDs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal track IDs: %v", err)
			}
		}
		if len(smartRulesJSON) > 0 {
			playlist.SmartRules = &SmartPlaylistRules{}
			if err := json.Unmarshal(smartRulesJSON, playlist.SmartRules); err != nil {
				return nil, fmt.Errorf("failed to unmarshal smart playlist rules: %v", err)
			}
		}
		if err := db.resolveSmartPlaylist(&playlist); err != nil {
			return nil, err
		}

		playlists = append(playlists, playlist)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating playlist rows: %v", err)
	}

	return playlists, nil
}

// GetPlaylist retrieves a playlist by ID
func (db *DB) GetPlaylist(id string) (*Playlist, error) {
	return db.getPlaylist(id, "")
}

func (db *DB) GetUserPlaylist(id, userID string) (*Playlist, error) {
	return db.getPlaylist(id, userID)
}

func (db *DB) getPlaylist(id, userID string) (*Playlist, error) {
	query := `SELECT id, user_id, name, description, playlist_type, smart_rules, track_ids, created_at, updated_at FROM playlists WHERE id = $1`
	args := []interface{}{id}
	if userID != "" {
		query += ` AND user_id = $2`
		args = append(args, userID)
	}

	var playlist Playlist
	var smartRulesJSON []byte
	var trackIDsJSON []byte

	err := db.conn.QueryRow(query, args...).Scan(&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Description, &playlist.Type, &smartRulesJSON, &trackIDsJSON, &playlist.CreatedAt, &playlist.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("playlist not found")
		}
		return nil, fmt.Errorf("failed to query playlist: %v", err)
	}

	if len(trackIDsJSON) > 0 {
		if err := json.Unmarshal(trackIDsJSON, &playlist.TrackIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal track IDs: %v", err)
		}
	}
	if len(smartRulesJSON) > 0 {
		playlist.SmartRules = &SmartPlaylistRules{}
		if err := json.Unmarshal(smartRulesJSON, playlist.SmartRules); err != nil {
			return nil, fmt.Errorf("failed to unmarshal smart playlist rules: %v", err)
		}
	}
	if err := db.resolveSmartPlaylist(&playlist); err != nil {
		return nil, err
	}

	return &playlist, nil
}

func (db *DB) resolveSmartPlaylist(playlist *Playlist) error {
	if playlist.Type == "" {
		playlist.Type = PlaylistTypeManual
	}
	if playlist.Type != PlaylistTypeSmart {
		return nil
	}
	if playlist.SmartRules == nil {
		return fmt.Errorf("smart playlist %s has no rules", playlist.ID)
	}
	tracks, err := db.EvaluateSmartPlaylist(playlist.UserID, *playlist.SmartRules)
	if err != nil {
		return fmt.Errorf("failed to evaluate smart playlist %s: %v", playlist.ID, err)
	}
	playlist.TrackIDs = make([]string, 0, len(tracks))
	for _, track := range tracks {
		playlist.TrackIDs = append(playlist.TrackIDs, track.ID)
		if track.UpdatedAt.After(playlist.UpdatedAt) {
			playlist.UpdatedAt = track.UpdatedAt
		}
	}
	stateRevision, err := db.smartPlaylistStateRevision(playlist.UserID)
	if err != nil {
		return err
	}
	if stateRevision.After(playlist.UpdatedAt) {
		playlist.UpdatedAt = stateRevision
	}
	return nil
}

func (db *DB) smartPlaylistStateRevision(userID string) (time.Time, error) {
	var revision time.Time
	err := db.conn.QueryRow(`
		SELECT GREATEST(
			COALESCE((SELECT MAX(updated_at) FROM media_ratings WHERE user_id = $1), TIMESTAMP 'epoch'),
			COALESCE((SELECT MAX(created_at) FROM liked_tracks WHERE user_id = $1), TIMESTAMP 'epoch'),
			COALESCE((SELECT MAX(played_at) FROM recently_played WHERE user_id = $1), TIMESTAMP 'epoch')
		)
	`, userID).Scan(&revision)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to determine smart playlist revision: %v", err)
	}
	return revision, nil
}

// AddPlaylist adds a new playlist to the database
func (db *DB) AddPlaylist(playlist *Playlist) error {
	// Generate ID if not provided
	if playlist.ID == "" {
		playlist.ID = fmt.Sprintf("playlist_%d", time.Now().UnixNano())
	}

	playlist.CreatedAt = time.Now()
	playlist.UpdatedAt = time.Now()

	if playlist.Type == "" {
		playlist.Type = PlaylistTypeManual
	}
	if playlist.Type != PlaylistTypeManual && playlist.Type != PlaylistTypeSmart {
		return fmt.Errorf("unsupported playlist type")
	}
	if playlist.Type == PlaylistTypeSmart {
		if err := ValidateSmartPlaylistRules(playlist.SmartRules); err != nil {
			return err
		}
		playlist.TrackIDs = []string{}
	}

	query := `
		INSERT INTO playlists (id, user_id, name, description, playlist_type, smart_rules, track_ids, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	trackIDsJSON, err := marshalTrackIDs(playlist.TrackIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal track IDs: %v", err)
	}

	var smartRulesJSON interface{}
	if playlist.SmartRules != nil {
		encodedRules, marshalErr := json.Marshal(playlist.SmartRules)
		err = marshalErr
		if err != nil {
			return fmt.Errorf("failed to marshal smart playlist rules: %v", err)
		}
		smartRulesJSON = encodedRules
	}

	_, err = db.conn.Exec(query, playlist.ID, playlist.UserID, playlist.Name, playlist.Description, playlist.Type, smartRulesJSON, trackIDsJSON, playlist.CreatedAt, playlist.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create playlist: %v", err)
	}

	return nil
}

// UpdatePlaylist updates an existing playlist
func (db *DB) UpdatePlaylist(playlist *Playlist) error {
	if playlist.Type == "" {
		playlist.Type = PlaylistTypeManual
	}
	if playlist.Type == PlaylistTypeSmart {
		if err := ValidateSmartPlaylistRules(playlist.SmartRules); err != nil {
			return err
		}
		playlist.TrackIDs = []string{}
	}
	query := `
		UPDATE playlists SET 
			name = $2, 
			description = $3, 
			playlist_type = $4,
			smart_rules = $5,
			track_ids = $6,
			updated_at = $7
		WHERE id = $1 AND user_id = $8
	`

	playlist.UpdatedAt = time.Now()

	trackIDsJSON, err := marshalTrackIDs(playlist.TrackIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal track IDs: %v", err)
	}
	var smartRulesJSON interface{}
	if playlist.SmartRules != nil {
		encodedRules, marshalErr := json.Marshal(playlist.SmartRules)
		err = marshalErr
		if err != nil {
			return fmt.Errorf("failed to marshal smart playlist rules: %v", err)
		}
		smartRulesJSON = encodedRules
	}

	result, err := db.conn.Exec(query, playlist.ID, playlist.Name, playlist.Description, playlist.Type, smartRulesJSON, trackIDsJSON, playlist.UpdatedAt, playlist.UserID)
	if err != nil {
		return fmt.Errorf("failed to update playlist: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("playlist not found")
	}

	return nil
}

// DeletePlaylist removes a playlist from the database
func (db *DB) DeletePlaylist(id, userID string) error {
	query := `DELETE FROM playlists WHERE id = $1 AND user_id = $2`

	result, err := db.conn.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete playlist: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("playlist not found")
	}

	return nil
}

// AddTrackToPlaylist adds a track once and returns the updated playlist.
func (db *DB) AddTrackToPlaylist(playlistID, trackID, userID string) (*Playlist, error) {
	return db.AddTracksToPlaylist(playlistID, []string{trackID}, userID)
}

// AddTracksToPlaylist validates and adds multiple tracks in one transaction.
func (db *DB) AddTracksToPlaylist(playlistID string, trackIDs []string, userID string) (*Playlist, error) {
	uniqueTrackIDs := make([]string, 0, len(trackIDs))
	seen := make(map[string]struct{}, len(trackIDs))
	for _, trackID := range trackIDs {
		if trackID == "" {
			continue
		}
		if _, exists := seen[trackID]; exists {
			continue
		}
		seen[trackID] = struct{}{}
		uniqueTrackIDs = append(uniqueTrackIDs, trackID)
	}
	if len(uniqueTrackIDs) == 0 {
		return nil, fmt.Errorf("at least one track is required")
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start playlist update: %v", err)
	}
	defer tx.Rollback()

	var trackCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM music WHERE id = ANY($1)`, pq.Array(uniqueTrackIDs)).Scan(&trackCount); err != nil {
		return nil, fmt.Errorf("failed to validate tracks: %v", err)
	}
	if trackCount != len(uniqueTrackIDs) {
		return nil, fmt.Errorf("one or more tracks were not found")
	}

	playlist, err := getPlaylistForUpdate(tx, playlistID, userID)
	if err != nil {
		return nil, err
	}
	if playlist.Type == PlaylistTypeSmart {
		return nil, fmt.Errorf("smart playlists are read-only")
	}

	for _, trackID := range uniqueTrackIDs {
		playlist.TrackIDs = appendUniqueTrackID(playlist.TrackIDs, trackID)
	}
	playlist.UpdatedAt = time.Now()
	trackIDsJSON, err := marshalTrackIDs(playlist.TrackIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal track IDs: %v", err)
	}

	if _, err := tx.Exec(
		`UPDATE playlists SET track_ids = $2, updated_at = $3 WHERE id = $1`,
		playlist.ID,
		trackIDsJSON,
		playlist.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to add tracks to playlist: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to save playlist: %v", err)
	}

	return playlist, nil
}

// RemoveTrackFromPlaylist removes a track and returns the updated playlist.
func (db *DB) RemoveTrackFromPlaylist(playlistID, trackID, userID string) (*Playlist, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start playlist update: %v", err)
	}
	defer tx.Rollback()

	playlist, err := getPlaylistForUpdate(tx, playlistID, userID)
	if err != nil {
		return nil, err
	}
	if playlist.Type == PlaylistTypeSmart {
		return nil, fmt.Errorf("smart playlists are read-only")
	}

	playlist.TrackIDs = removeTrackID(playlist.TrackIDs, trackID)
	playlist.UpdatedAt = time.Now()
	trackIDsJSON, err := marshalTrackIDs(playlist.TrackIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal track IDs: %v", err)
	}

	if _, err := tx.Exec(
		`UPDATE playlists SET track_ids = $2, updated_at = $3 WHERE id = $1`,
		playlist.ID,
		trackIDsJSON,
		playlist.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to remove track from playlist: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to save playlist: %v", err)
	}

	return playlist, nil
}

func getPlaylistForUpdate(tx *sql.Tx, playlistID, userID string) (*Playlist, error) {
	var playlist Playlist
	var smartRulesJSON []byte
	var trackIDsJSON []byte

	err := tx.QueryRow(`
		SELECT id, user_id, name, description, playlist_type, smart_rules, track_ids, created_at, updated_at
		FROM playlists
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, playlistID, userID).Scan(
		&playlist.ID,
		&playlist.UserID,
		&playlist.Name,
		&playlist.Description,
		&playlist.Type,
		&smartRulesJSON,
		&trackIDsJSON,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("playlist not found")
		}
		return nil, fmt.Errorf("failed to query playlist: %v", err)
	}

	if len(trackIDsJSON) > 0 {
		if err := json.Unmarshal(trackIDsJSON, &playlist.TrackIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal track IDs: %v", err)
		}
	}
	if len(smartRulesJSON) > 0 {
		playlist.SmartRules = &SmartPlaylistRules{}
		if err := json.Unmarshal(smartRulesJSON, playlist.SmartRules); err != nil {
			return nil, fmt.Errorf("failed to unmarshal smart playlist rules: %v", err)
		}
	}

	return &playlist, nil
}

// GetPlaylistTracks returns tracks in the order they were added.
func (db *DB) GetPlaylistTracks(playlistID, userID string) ([]Music, error) {
	playlist, err := db.GetUserPlaylist(playlistID, userID)
	if err != nil {
		return nil, err
	}

	tracks := make([]Music, 0, len(playlist.TrackIDs))
	for _, trackID := range playlist.TrackIDs {
		track, err := db.GetMusic(trackID)
		if err != nil {
			continue
		}
		tracks = append(tracks, *track)
	}

	return tracks, nil
}
