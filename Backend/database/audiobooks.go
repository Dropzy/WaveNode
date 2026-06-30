package database

import (
	"fmt"
	"time"
)

type AudiobookProgress struct {
	UserID          string    `json:"-"`
	BookID          string    `json:"book_id"`
	ChapterID       string    `json:"chapter_id"`
	BookTitle       string    `json:"book_title"`
	Author          string    `json:"author"`
	ChapterTitle    string    `json:"chapter_title"`
	ChapterNumber   int       `json:"chapter_number"`
	Description     string    `json:"description"`
	ImageURL        string    `json:"image_url"`
	AudioURL        string    `json:"audio_url"`
	WebsiteURL      string    `json:"website_url"`
	DurationSeconds int       `json:"duration_seconds"`
	PositionSeconds int       `json:"position_seconds"`
	Completed       bool      `json:"completed"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (db *DB) SaveAudiobookProgress(item AudiobookProgress) (AudiobookProgress, error) {
	err := db.conn.QueryRow(`INSERT INTO audiobook_progress (
		user_id, book_id, chapter_id, book_title, author, chapter_title, chapter_number,
		description, image_url, audio_url, website_url, duration_seconds, position_seconds, completed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (user_id, book_id, chapter_id) DO UPDATE SET
		book_title=EXCLUDED.book_title, author=EXCLUDED.author, chapter_title=EXCLUDED.chapter_title,
		chapter_number=EXCLUDED.chapter_number, description=EXCLUDED.description,
		image_url=EXCLUDED.image_url, audio_url=EXCLUDED.audio_url, website_url=EXCLUDED.website_url,
		duration_seconds=EXCLUDED.duration_seconds, position_seconds=EXCLUDED.position_seconds,
		completed=EXCLUDED.completed, updated_at=CURRENT_TIMESTAMP
		RETURNING updated_at`, item.UserID, item.BookID, item.ChapterID, item.BookTitle, item.Author,
		item.ChapterTitle, item.ChapterNumber, item.Description, item.ImageURL, item.AudioURL,
		item.WebsiteURL, item.DurationSeconds, item.PositionSeconds, item.Completed).Scan(&item.UpdatedAt)
	if err != nil {
		return AudiobookProgress{}, fmt.Errorf("failed to save audiobook progress: %v", err)
	}
	return item, nil
}

func (db *DB) GetAudiobookProgress(userID, bookID string) (map[string]AudiobookProgress, error) {
	rows, err := db.conn.Query(`SELECT book_id, chapter_id, book_title, author, chapter_title,
		chapter_number, description, image_url, audio_url, website_url, duration_seconds,
		position_seconds, completed, updated_at FROM audiobook_progress
		WHERE user_id=$1 AND book_id=$2`, userID, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to load audiobook progress: %v", err)
	}
	defer rows.Close()
	result := make(map[string]AudiobookProgress)
	for rows.Next() {
		var item AudiobookProgress
		item.UserID = userID
		if err := rows.Scan(&item.BookID, &item.ChapterID, &item.BookTitle, &item.Author,
			&item.ChapterTitle, &item.ChapterNumber, &item.Description, &item.ImageURL,
			&item.AudioURL, &item.WebsiteURL, &item.DurationSeconds, &item.PositionSeconds,
			&item.Completed, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan audiobook progress: %v", err)
		}
		result[item.ChapterID] = item
	}
	return result, rows.Err()
}

func (db *DB) GetContinueListeningAudiobooks(userID string, limit int) ([]AudiobookProgress, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	rows, err := db.conn.Query(`SELECT book_id, chapter_id, book_title, author, chapter_title,
		chapter_number, description, image_url, audio_url, website_url, duration_seconds,
		position_seconds, completed, updated_at FROM (
			SELECT DISTINCT ON (book_id) book_id, chapter_id, book_title, author, chapter_title,
			chapter_number, description, image_url, audio_url, website_url, duration_seconds,
			position_seconds, completed, updated_at FROM audiobook_progress
			WHERE user_id=$1 AND position_seconds > 0 AND completed=FALSE
			ORDER BY book_id, updated_at DESC
		) latest ORDER BY updated_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load audiobook listening progress: %v", err)
	}
	defer rows.Close()
	result := make([]AudiobookProgress, 0, limit)
	for rows.Next() {
		var item AudiobookProgress
		item.UserID = userID
		if err := rows.Scan(&item.BookID, &item.ChapterID, &item.BookTitle, &item.Author,
			&item.ChapterTitle, &item.ChapterNumber, &item.Description, &item.ImageURL,
			&item.AudioURL, &item.WebsiteURL, &item.DurationSeconds, &item.PositionSeconds,
			&item.Completed, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan audiobook listening progress: %v", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
