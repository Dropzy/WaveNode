package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type PodcastSubscription struct {
	UserID        string    `json:"-"`
	PodcastID     string    `json:"podcast_id"`
	Title         string    `json:"title"`
	Publisher     string    `json:"publisher"`
	Description   string    `json:"description"`
	ImageURL      string    `json:"image_url"`
	ThumbnailURL  string    `json:"thumbnail_url"`
	WebsiteURL    string    `json:"website_url"`
	FeedURL       string    `json:"feed_url"`
	AutoDownload  bool      `json:"auto_download"`
	PlaybackSpeed float64   `json:"playback_speed"`
	SubscribedAt  time.Time `json:"subscribed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PodcastPreferences struct {
	UserID               string    `json:"-"`
	DefaultPlaybackSpeed float64   `json:"default_playback_speed"`
	SkipBackSeconds      int       `json:"skip_back_seconds"`
	SkipForwardSeconds   int       `json:"skip_forward_seconds"`
	AutoDeletePlayed     bool      `json:"auto_delete_played"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type PodcastQueue struct {
	UserID          string          `json:"-"`
	Items           json.RawMessage `json:"items"`
	CurrentIndex    int             `json:"current_index"`
	PositionSeconds int             `json:"position_seconds"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (db *DB) ListPodcastSubscriptions(userID string) ([]PodcastSubscription, error) {
	rows, err := db.conn.Query(`SELECT user_id, podcast_id, title, publisher, description, image_url,
		thumbnail_url, website_url, feed_url, auto_download, playback_speed, subscribed_at, updated_at
		FROM podcast_subscriptions WHERE user_id = $1 ORDER BY subscribed_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list podcast subscriptions: %v", err)
	}
	defer rows.Close()
	result := make([]PodcastSubscription, 0)
	for rows.Next() {
		var item PodcastSubscription
		if err := rows.Scan(&item.UserID, &item.PodcastID, &item.Title, &item.Publisher, &item.Description,
			&item.ImageURL, &item.ThumbnailURL, &item.WebsiteURL, &item.FeedURL, &item.AutoDownload,
			&item.PlaybackSpeed, &item.SubscribedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan podcast subscription: %v", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (db *DB) SavePodcastSubscription(item PodcastSubscription) (PodcastSubscription, error) {
	err := db.conn.QueryRow(`INSERT INTO podcast_subscriptions (
		user_id, podcast_id, title, publisher, description, image_url, thumbnail_url, website_url,
		feed_url, auto_download, playback_speed) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (user_id, podcast_id) DO UPDATE SET title=EXCLUDED.title, publisher=EXCLUDED.publisher,
		description=EXCLUDED.description, image_url=EXCLUDED.image_url, thumbnail_url=EXCLUDED.thumbnail_url,
		website_url=EXCLUDED.website_url, feed_url=EXCLUDED.feed_url, auto_download=EXCLUDED.auto_download,
		playback_speed=EXCLUDED.playback_speed, updated_at=CURRENT_TIMESTAMP
		RETURNING subscribed_at, updated_at`, item.UserID, item.PodcastID, item.Title, item.Publisher,
		item.Description, item.ImageURL, item.ThumbnailURL, item.WebsiteURL, item.FeedURL,
		item.AutoDownload, item.PlaybackSpeed).Scan(&item.SubscribedAt, &item.UpdatedAt)
	if err != nil {
		return PodcastSubscription{}, fmt.Errorf("failed to save podcast subscription: %v", err)
	}
	return item, nil
}

func (db *DB) DeletePodcastSubscription(userID, podcastID string) error {
	_, err := db.conn.Exec(`DELETE FROM podcast_subscriptions WHERE user_id=$1 AND podcast_id=$2`, userID, podcastID)
	if err != nil {
		return fmt.Errorf("failed to delete podcast subscription: %v", err)
	}
	return nil
}

func (db *DB) GetPodcastPreferences(userID string) (PodcastPreferences, error) {
	preferences := PodcastPreferences{UserID: userID, DefaultPlaybackSpeed: 1, SkipBackSeconds: 15, SkipForwardSeconds: 30, AutoDeletePlayed: true}
	err := db.conn.QueryRow(`SELECT default_playback_speed, skip_back_seconds, skip_forward_seconds,
		auto_delete_played, updated_at FROM podcast_preferences WHERE user_id=$1`, userID).Scan(
		&preferences.DefaultPlaybackSpeed, &preferences.SkipBackSeconds, &preferences.SkipForwardSeconds,
		&preferences.AutoDeletePlayed, &preferences.UpdatedAt)
	if err == sql.ErrNoRows {
		return preferences, nil
	}
	if err != nil {
		return PodcastPreferences{}, fmt.Errorf("failed to load podcast preferences: %v", err)
	}
	return preferences, nil
}

func (db *DB) SavePodcastPreferences(preferences PodcastPreferences) (PodcastPreferences, error) {
	err := db.conn.QueryRow(`INSERT INTO podcast_preferences (user_id, default_playback_speed,
		skip_back_seconds, skip_forward_seconds, auto_delete_played) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id) DO UPDATE SET default_playback_speed=EXCLUDED.default_playback_speed,
		skip_back_seconds=EXCLUDED.skip_back_seconds, skip_forward_seconds=EXCLUDED.skip_forward_seconds,
		auto_delete_played=EXCLUDED.auto_delete_played, updated_at=CURRENT_TIMESTAMP RETURNING updated_at`,
		preferences.UserID, preferences.DefaultPlaybackSpeed, preferences.SkipBackSeconds,
		preferences.SkipForwardSeconds, preferences.AutoDeletePlayed).Scan(&preferences.UpdatedAt)
	if err != nil {
		return PodcastPreferences{}, fmt.Errorf("failed to save podcast preferences: %v", err)
	}
	return preferences, nil
}

func (db *DB) GetPodcastQueue(userID string) (PodcastQueue, error) {
	queue := PodcastQueue{UserID: userID, Items: json.RawMessage("[]")}
	err := db.conn.QueryRow(`SELECT items, current_index, position_seconds, updated_at
		FROM podcast_queues WHERE user_id=$1`, userID).Scan(&queue.Items, &queue.CurrentIndex, &queue.PositionSeconds, &queue.UpdatedAt)
	if err == sql.ErrNoRows {
		return queue, nil
	}
	if err != nil {
		return PodcastQueue{}, fmt.Errorf("failed to load podcast queue: %v", err)
	}
	return queue, nil
}

func (db *DB) SavePodcastQueue(queue PodcastQueue) (PodcastQueue, error) {
	err := db.conn.QueryRow(`INSERT INTO podcast_queues (user_id, items, current_index, position_seconds)
		VALUES ($1,$2,$3,$4) ON CONFLICT (user_id) DO UPDATE SET items=EXCLUDED.items,
		current_index=EXCLUDED.current_index, position_seconds=EXCLUDED.position_seconds,
		updated_at=CURRENT_TIMESTAMP RETURNING updated_at`, queue.UserID, queue.Items,
		queue.CurrentIndex, queue.PositionSeconds).Scan(&queue.UpdatedAt)
	if err != nil {
		return PodcastQueue{}, fmt.Errorf("failed to save podcast queue: %v", err)
	}
	return queue, nil
}
