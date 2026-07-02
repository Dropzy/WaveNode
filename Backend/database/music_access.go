package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// SetUserMusicAccess replaces a user's source-folder permissions atomically.
// Administrators are always unrestricted regardless of the stored values.
func (db *DB) SetUserMusicAccess(userID string, restricted bool, sourceIDs []string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin library permission update: %v", err)
	}
	defer tx.Rollback()

	var role string
	if err := tx.QueryRow("SELECT role FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&role); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to load user: %v", err)
	}
	if role == "admin" {
		restricted = false
		sourceIDs = nil
	}

	if _, err := tx.Exec("UPDATE users SET library_restricted = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1", userID, restricted); err != nil {
		return fmt.Errorf("failed to update library restriction: %v", err)
	}
	if _, err := tx.Exec("DELETE FROM user_music_sources WHERE user_id = $1", userID); err != nil {
		return fmt.Errorf("failed to replace library permissions: %v", err)
	}

	seen := make(map[string]struct{}, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		sourceID = strings.TrimSpace(sourceID)
		if sourceID == "" {
			continue
		}
		if _, exists := seen[sourceID]; exists {
			continue
		}
		seen[sourceID] = struct{}{}
		result, err := tx.Exec(`
			INSERT INTO user_music_sources (user_id, source_id)
			SELECT $1, id FROM music_sources WHERE id = $2
		`, userID, sourceID)
		if err != nil {
			return fmt.Errorf("failed to assign music source: %v", err)
		}
		if count, _ := result.RowsAffected(); count == 0 {
			return fmt.Errorf("music source not found: %s", sourceID)
		}
	}
	return tx.Commit()
}

func (db *DB) GetUserMusicSourceIDs(userID string) ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT source_id FROM user_music_sources WHERE user_id = $1 ORDER BY source_id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user music sources: %v", err)
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to read user music source: %v", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (db *DB) userMusicRoots(userID string) (bool, []string, error) {
	var role string
	var restricted bool
	if err := db.conn.QueryRow("SELECT role, library_restricted FROM users WHERE id = $1", userID).Scan(&role, &restricted); err != nil {
		return false, nil, err
	}
	if role == "admin" || !restricted {
		return false, nil, nil
	}
	rows, err := db.conn.Query(`
		SELECT ms.path
		FROM music_sources ms
		JOIN user_music_sources ums ON ums.source_id = ms.id
		WHERE ums.user_id = $1
	`, userID)
	if err != nil {
		return true, nil, err
	}
	defer rows.Close()
	roots := make([]string, 0)
	for rows.Next() {
		var root string
		if err := rows.Scan(&root); err != nil {
			return true, nil, err
		}
		roots = append(roots, root)
	}
	return true, roots, rows.Err()
}

func pathWithinMusicRoot(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
		root = strings.ToLower(root)
	}
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (db *DB) FilterMusicForUser(userID string, tracks []Music) ([]Music, error) {
	restricted, roots, err := db.userMusicRoots(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load library permissions: %v", err)
	}
	if !restricted {
		return tracks, nil
	}
	filtered := make([]Music, 0, len(tracks))
	for _, track := range tracks {
		for _, root := range roots {
			if pathWithinMusicRoot(track.FilePath, root) {
				filtered = append(filtered, track)
				break
			}
		}
	}
	return filtered, nil
}

func (db *DB) UserCanAccessMusic(userID string, track *Music) (bool, error) {
	filtered, err := db.FilterMusicForUser(userID, []Music{*track})
	return len(filtered) == 1, err
}
