package database

import (
	"database/sql"
	"fmt"
	"time"
)

type PodcastProgress struct {
	UserID          string     `json:"-"`
	PodcastID       string     `json:"podcast_id"`
	EpisodeID       string     `json:"episode_id"`
	PodcastTitle    string     `json:"podcast_title"`
	Publisher       string     `json:"publisher"`
	EpisodeTitle    string     `json:"episode_title"`
	Description     string     `json:"description"`
	ImageURL        string     `json:"image_url"`
	AudioURL        string     `json:"audio_url"`
	WebsiteURL      string     `json:"website_url"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	DurationSeconds int        `json:"duration_seconds"`
	PositionSeconds int        `json:"position_seconds"`
	Completed       bool       `json:"completed"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (db *DB) SavePodcastProgress(progress PodcastProgress) (PodcastProgress, error) {
	query := `
		INSERT INTO podcast_progress (
			user_id, podcast_id, episode_id, podcast_title, publisher, episode_title,
			description, image_url, audio_url, website_url, published_at,
			duration_seconds, position_seconds, completed, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, podcast_id, episode_id) DO UPDATE SET
			podcast_title = EXCLUDED.podcast_title,
			publisher = EXCLUDED.publisher,
			episode_title = EXCLUDED.episode_title,
			description = EXCLUDED.description,
			image_url = EXCLUDED.image_url,
			audio_url = EXCLUDED.audio_url,
			website_url = EXCLUDED.website_url,
			published_at = EXCLUDED.published_at,
			duration_seconds = EXCLUDED.duration_seconds,
			position_seconds = EXCLUDED.position_seconds,
			completed = EXCLUDED.completed,
			updated_at = CURRENT_TIMESTAMP
		RETURNING updated_at`
	if err := db.conn.QueryRow(query,
		progress.UserID, progress.PodcastID, progress.EpisodeID, progress.PodcastTitle,
		progress.Publisher, progress.EpisodeTitle, progress.Description, progress.ImageURL,
		progress.AudioURL, progress.WebsiteURL, progress.PublishedAt, progress.DurationSeconds,
		progress.PositionSeconds, progress.Completed,
	).Scan(&progress.UpdatedAt); err != nil {
		return PodcastProgress{}, fmt.Errorf("failed to save podcast progress: %v", err)
	}
	return progress, nil
}

func (db *DB) GetPodcastProgress(userID, podcastID string) (map[string]PodcastProgress, error) {
	rows, err := db.conn.Query(`
		SELECT user_id, podcast_id, episode_id, podcast_title, publisher, episode_title,
			description, image_url, audio_url, website_url, published_at,
			duration_seconds, position_seconds, completed, updated_at
		FROM podcast_progress WHERE user_id = $1 AND podcast_id = $2`, userID, podcastID)
	if err != nil {
		return nil, fmt.Errorf("failed to load podcast progress: %v", err)
	}
	defer rows.Close()

	result := make(map[string]PodcastProgress)
	for rows.Next() {
		progress, err := scanPodcastProgress(rows)
		if err != nil {
			return nil, err
		}
		result[progress.EpisodeID] = progress
	}
	return result, rows.Err()
}

func (db *DB) GetContinueListeningPodcasts(userID string, limit int) ([]PodcastProgress, error) {
	if limit <= 0 || limit > 25 {
		limit = 12
	}
	rows, err := db.conn.Query(`
		SELECT user_id, podcast_id, episode_id, podcast_title, publisher, episode_title,
			description, image_url, audio_url, website_url, published_at,
			duration_seconds, position_seconds, completed, updated_at
		FROM (
			SELECT DISTINCT ON (podcast_id) *
			FROM podcast_progress
			WHERE user_id = $1 AND position_seconds > 0 AND completed = FALSE
			ORDER BY podcast_id, updated_at DESC
		) current_episodes
		ORDER BY updated_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load continue listening podcasts: %v", err)
	}
	defer rows.Close()

	result := make([]PodcastProgress, 0)
	for rows.Next() {
		progress, err := scanPodcastProgress(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, progress)
	}
	return result, rows.Err()
}

type podcastProgressScanner interface {
	Scan(dest ...interface{}) error
}

func scanPodcastProgress(scanner podcastProgressScanner) (PodcastProgress, error) {
	var progress PodcastProgress
	var publishedAt sql.NullTime
	err := scanner.Scan(
		&progress.UserID, &progress.PodcastID, &progress.EpisodeID, &progress.PodcastTitle,
		&progress.Publisher, &progress.EpisodeTitle, &progress.Description, &progress.ImageURL,
		&progress.AudioURL, &progress.WebsiteURL, &publishedAt, &progress.DurationSeconds,
		&progress.PositionSeconds, &progress.Completed, &progress.UpdatedAt,
	)
	if err != nil {
		return PodcastProgress{}, fmt.Errorf("failed to scan podcast progress: %v", err)
	}
	if publishedAt.Valid {
		progress.PublishedAt = &publishedAt.Time
	}
	return progress, nil
}
