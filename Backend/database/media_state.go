package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (db *DB) SetMediaRating(userID, mediaID, mediaType string, rating int) error {
	if rating < 0 || rating > 5 {
		return fmt.Errorf("rating must be between 0 and 5")
	}
	if rating == 0 {
		_, err := db.conn.Exec(`DELETE FROM media_ratings WHERE user_id = $1 AND media_id = $2`, userID, mediaID)
		if err != nil {
			return err
		}
		return db.touchSmartPlaylists(userID)
	}
	if mediaType == "" {
		mediaType = "song"
	}
	_, err := db.conn.Exec(`
		INSERT INTO media_ratings (user_id, media_id, media_type, rating, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, media_id)
		DO UPDATE SET media_type = EXCLUDED.media_type, rating = EXCLUDED.rating, updated_at = CURRENT_TIMESTAMP
	`, userID, mediaID, mediaType, rating)
	if err != nil {
		return fmt.Errorf("failed to save rating: %v", err)
	}
	return db.touchSmartPlaylists(userID)
}

func (db *DB) GetMediaRating(userID, mediaID string) (int, error) {
	var rating int
	err := db.conn.QueryRow(`SELECT rating FROM media_ratings WHERE user_id = $1 AND media_id = $2`, userID, mediaID).Scan(&rating)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to load rating: %v", err)
	}
	return rating, nil
}

func (db *DB) GetMediaRatings(userID string) (map[string]int, error) {
	rows, err := db.conn.Query(`SELECT media_id, rating FROM media_ratings WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load ratings: %v", err)
	}
	defer rows.Close()
	ratings := make(map[string]int)
	for rows.Next() {
		var id string
		var rating int
		if err := rows.Scan(&id, &rating); err != nil {
			return nil, err
		}
		ratings[id] = rating
	}
	return ratings, rows.Err()
}

func (db *DB) GetMediaAverageRatings() (map[string]float64, error) {
	rows, err := db.conn.Query(`
		SELECT media_id, AVG(rating)::DOUBLE PRECISION
		FROM media_ratings
		GROUP BY media_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load average ratings: %v", err)
	}
	defer rows.Close()

	ratings := make(map[string]float64)
	for rows.Next() {
		var id string
		var rating float64
		if err := rows.Scan(&id, &rating); err != nil {
			return nil, err
		}
		ratings[id] = rating
	}
	return ratings, rows.Err()
}

func (db *DB) StarMedia(userID, mediaID, mediaType string) error {
	if mediaType != "artist" && mediaType != "album" {
		return fmt.Errorf("unsupported starred media type")
	}
	_, err := db.conn.Exec(`
		INSERT INTO media_stars (user_id, media_id, media_type)
		VALUES ($1, $2, $3) ON CONFLICT (user_id, media_id) DO NOTHING
	`, userID, mediaID, mediaType)
	return err
}

func (db *DB) UnstarMedia(userID, mediaID string) error {
	_, err := db.conn.Exec(`DELETE FROM media_stars WHERE user_id = $1 AND media_id = $2`, userID, mediaID)
	return err
}

func (db *DB) GetStarredMedia(userID, mediaType string) ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT media_id FROM media_stars
		WHERE user_id = $1 AND media_type = $2 ORDER BY created_at DESC
	`, userID, mediaType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (db *DB) SaveBookmark(userID, trackID string, positionMS int64, comment string) error {
	if positionMS < 0 {
		return fmt.Errorf("bookmark position cannot be negative")
	}
	_, err := db.conn.Exec(`
		INSERT INTO media_bookmarks (user_id, track_id, position_ms, comment, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, track_id)
		DO UPDATE SET position_ms = EXCLUDED.position_ms, comment = EXCLUDED.comment, updated_at = CURRENT_TIMESTAMP
	`, userID, trackID, positionMS, comment)
	if err != nil {
		return fmt.Errorf("failed to save bookmark: %v", err)
	}
	return nil
}

func (db *DB) DeleteBookmark(userID, trackID string) error {
	_, err := db.conn.Exec(`DELETE FROM media_bookmarks WHERE user_id = $1 AND track_id = $2`, userID, trackID)
	return err
}

func (db *DB) GetBookmarks(userID string) ([]MediaBookmark, error) {
	rows, err := db.conn.Query(`
		SELECT track_id, position_ms, comment, updated_at
		FROM media_bookmarks WHERE user_id = $1 ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load bookmarks: %v", err)
	}
	defer rows.Close()
	bookmarks := make([]MediaBookmark, 0)
	for rows.Next() {
		var bookmark MediaBookmark
		if err := rows.Scan(&bookmark.TrackID, &bookmark.PositionMS, &bookmark.Comment, &bookmark.UpdatedAt); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, bookmark)
	}
	return bookmarks, rows.Err()
}

func (db *DB) SavePlayQueue(userID string, queue PlayQueue) error {
	trackIDs, err := json.Marshal(queue.TrackIDs)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec(`
		INSERT INTO user_play_queues (user_id, track_ids, current_track_id, position_ms, changed_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id)
		DO UPDATE SET track_ids = EXCLUDED.track_ids, current_track_id = EXCLUDED.current_track_id,
			position_ms = EXCLUDED.position_ms, changed_at = CURRENT_TIMESTAMP
	`, userID, trackIDs, queue.CurrentTrackID, queue.PositionMS)
	if err != nil {
		return fmt.Errorf("failed to save play queue: %v", err)
	}
	return nil
}

func (db *DB) GetPlayQueue(userID string) (*PlayQueue, error) {
	var raw []byte
	var current sql.NullString
	var queue PlayQueue
	err := db.conn.QueryRow(`
		SELECT track_ids, current_track_id, position_ms, changed_at
		FROM user_play_queues WHERE user_id = $1
	`, userID).Scan(&raw, &current, &queue.PositionMS, &queue.ChangedAt)
	if err == sql.ErrNoRows {
		return &PlayQueue{TrackIDs: []string{}, ChangedAt: time.Unix(0, 0)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load play queue: %v", err)
	}
	if err := json.Unmarshal(raw, &queue.TrackIDs); err != nil {
		return nil, fmt.Errorf("failed to decode play queue: %v", err)
	}
	queue.CurrentTrackID = current.String
	return &queue, nil
}
