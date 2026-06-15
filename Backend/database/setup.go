package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const ArtworkPathSettingKey = "artwork_path"

var ErrSetupAlreadyComplete = errors.New("initial setup has already been completed")

type InitialSetupInput struct {
	Username    string
	Email       string
	Password    string
	MusicPaths  []string
	ArtworkPath string
}

func (db *DB) IsSetupRequired() (bool, error) {
	var count int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return false, fmt.Errorf("failed to check setup status: %v", err)
	}
	return count == 0, nil
}

func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM app_settings WHERE key = $1", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to load setting %q: %v", key, err)
	}
	return value, nil
}

func (db *DB) SetSetting(key, value string) error {
	_, err := db.conn.Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`, key, value)
	if err != nil {
		return fmt.Errorf("failed to save setting %q: %v", key, err)
	}
	return nil
}

func (db *DB) CompleteInitialSetup(input InitialSetupInput) (*User, error) {
	cleanPaths := make([]string, 0, len(input.MusicPaths))
	seen := make(map[string]struct{})
	for _, sourcePath := range input.MusicPaths {
		cleanPath, err := normalizeMusicSourcePath(sourcePath)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[cleanPath]; exists {
			continue
		}
		seen[cleanPath] = struct{}{}
		cleanPaths = append(cleanPaths, cleanPath)
	}
	if len(cleanPaths) == 0 {
		return nil, fmt.Errorf("at least one music folder is required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to secure administrator password: %v", err)
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin initial setup: %v", err)
	}
	defer tx.Rollback()

	// Serializes setup attempts so two open browser tabs cannot create two first users.
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", int64(8751423901)); err != nil {
		return nil, fmt.Errorf("failed to lock initial setup: %v", err)
	}

	var userCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return nil, fmt.Errorf("failed to confirm setup status: %v", err)
	}
	if userCount > 0 {
		return nil, ErrSetupAlreadyComplete
	}

	now := time.Now()
	user := &User{
		ID:        fmt.Sprintf("user_%d", now.UnixNano()),
		Username:  strings.TrimSpace(input.Username),
		Email:     strings.TrimSpace(input.Email),
		Role:      "admin",
		Password:  string(hashedPassword),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := tx.Exec(`
		INSERT INTO users (id, username, email, role, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.Username, user.Email, user.Role, user.Password, user.CreatedAt, user.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to create administrator: %v", err)
	}

	for index, sourcePath := range cleanPaths {
		if _, err := tx.Exec(`
			INSERT INTO music_sources (id, path, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (path) DO NOTHING
		`, fmt.Sprintf("source_%d_%d", now.UnixNano(), index), sourcePath, now); err != nil {
			return nil, fmt.Errorf("failed to save music folder: %v", err)
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`, ArtworkPathSettingKey, input.ArtworkPath, now); err != nil {
		return nil, fmt.Errorf("failed to save artwork folder: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to complete initial setup: %v", err)
	}
	return user, nil
}
