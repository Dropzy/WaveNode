package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const legacyMusicPathImportKey = "legacy_music_path_imported"

func normalizeMusicSourcePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("music source path is required")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("music source path must be absolute")
	}

	cleanPath := filepath.Clean(path)
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("music source does not exist")
		}
		return "", fmt.Errorf("music source is not accessible: %v", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("music source must be a directory")
	}

	return cleanPath, nil
}

func (db *DB) GetMusicSources() ([]MusicSource, error) {
	rows, err := db.conn.Query(`
		SELECT id, path, created_at
		FROM music_sources
		ORDER BY created_at, path
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query music sources: %v", err)
	}
	defer rows.Close()

	sources := make([]MusicSource, 0)
	for rows.Next() {
		var source MusicSource
		if err := rows.Scan(&source.ID, &source.Path, &source.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to read music source: %v", err)
		}
		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate music sources: %v", err)
	}
	return sources, nil
}

func (db *DB) AddMusicSource(path string) (*MusicSource, error) {
	cleanPath, err := normalizeMusicSourcePath(path)
	if err != nil {
		return nil, err
	}

	source := &MusicSource{
		ID:        fmt.Sprintf("source_%d", time.Now().UnixNano()),
		Path:      cleanPath,
		CreatedAt: time.Now(),
	}

	err = db.conn.QueryRow(`
		INSERT INTO music_sources (id, path, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (path) DO NOTHING
		RETURNING id, path, created_at
	`, source.ID, source.Path, source.CreatedAt).Scan(&source.ID, &source.Path, &source.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("music source already exists")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to add music source: %v", err)
	}

	return source, nil
}

func (db *DB) DeleteMusicSource(id string) error {
	result, err := db.conn.Exec("DELETE FROM music_sources WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to remove music source: %v", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to confirm music source removal: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("music source not found")
	}
	return nil
}

// ImportLegacyMusicSource imports MUSIC_PATH once, allowing it to be removed
// permanently from the UI even if the old environment variable remains set.
func (db *DB) ImportLegacyMusicSource(path string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin legacy source import: %v", err)
	}
	defer tx.Rollback()

	var imported string
	err = tx.QueryRow("SELECT value FROM app_settings WHERE key = $1", legacyMusicPathImportKey).Scan(&imported)
	if err == nil {
		return tx.Commit()
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check legacy source import: %v", err)
	}

	path = strings.TrimSpace(path)
	if path != "" {
		cleanPath, normalizeErr := normalizeMusicSourcePath(path)
		if normalizeErr != nil {
			return fmt.Errorf("legacy music source is invalid: %v", normalizeErr)
		}
		if _, err := tx.Exec(`
			INSERT INTO music_sources (id, path, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (path) DO NOTHING
		`, fmt.Sprintf("source_%d", time.Now().UnixNano()), cleanPath, time.Now()); err != nil {
			return fmt.Errorf("failed to import legacy music source: %v", err)
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ($1, 'true', $2)
	`, legacyMusicPathImportKey, time.Now()); err != nil {
		return fmt.Errorf("failed to record legacy source import: %v", err)
	}

	return tx.Commit()
}
