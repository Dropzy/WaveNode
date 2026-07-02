package database

import (
	"encoding/json"
	"fmt"
	"time"
)

const BackupFormatVersion = 6

var backupTables = []string{
	"users",
	"artists",
	"music",
	"playlists",
	"liked_tracks",
	"recently_played",
	"music_sources",
	"app_settings",
	"scan_status",
	"plugins",
	"track_audio_properties",
	"playback_profiles",
	"user_sessions",
	"listening_history",
	"radio_favorites",
}

type BackupSnapshot struct {
	FormatVersion int                        `json:"format_version"`
	CreatedAt     time.Time                  `json:"created_at"`
	Tables        map[string]json.RawMessage `json:"tables"`
}

func (db *DB) CreateBackupSnapshot() (*BackupSnapshot, error) {
	snapshot := &BackupSnapshot{
		FormatVersion: BackupFormatVersion,
		CreatedAt:     time.Now().UTC(),
		Tables:        make(map[string]json.RawMessage, len(backupTables)),
	}
	for _, table := range backupTables {
		var data []byte
		query := fmt.Sprintf(`
			SELECT COALESCE(json_agg(row_to_json(records)), '[]'::json)::text
			FROM (SELECT * FROM %s) records
		`, table)
		if err := db.conn.QueryRow(query).Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to export %s: %v", table, err)
		}
		snapshot.Tables[table] = json.RawMessage(data)
	}
	return snapshot, nil
}

func (db *DB) RestoreBackupSnapshot(snapshot *BackupSnapshot) error {
	if snapshot == nil || snapshot.FormatVersion < 3 || snapshot.FormatVersion > BackupFormatVersion {
		return fmt.Errorf("unsupported backup format")
	}
	requiredTableCount := len(backupTables)
	if snapshot.FormatVersion == 3 {
		requiredTableCount = 9
	} else if snapshot.FormatVersion == 4 {
		requiredTableCount = 10
	} else if snapshot.FormatVersion == 5 {
		requiredTableCount = len(backupTables) - 1
	}
	requiredTables := backupTables[:requiredTableCount]
	for _, table := range requiredTables {
		if _, exists := snapshot.Tables[table]; !exists {
			return fmt.Errorf("backup is missing table %s", table)
		}
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin restore: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		TRUNCATE TABLE radio_favorites, listening_history, user_sessions, playback_profiles,
			track_audio_properties, liked_tracks, recently_played, playlists,
			music, artists, music_sources, app_settings, scan_status, plugins, users CASCADE
	`); err != nil {
		return fmt.Errorf("failed to clear existing data: %v", err)
	}

	for _, table := range backupTables {
		data, exists := snapshot.Tables[table]
		if !exists {
			continue
		}
		if string(data) == "[]" {
			continue
		}
		query := fmt.Sprintf(`
			INSERT INTO %s
			SELECT * FROM json_populate_recordset(NULL::%s, $1::json)
		`, table, table)
		if _, err := tx.Exec(query, data); err != nil {
			return fmt.Errorf("failed to restore %s: %v", table, err)
		}
	}

	if _, err := tx.Exec(`
		SELECT setval(
			'music_upload_order_seq',
			GREATEST(COALESCE((SELECT MAX(upload_order) FROM music), 0), 1),
			COALESCE((SELECT MAX(upload_order) FROM music), 0) > 0
		)
	`); err != nil {
		return fmt.Errorf("failed to restore music upload sequence: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit restore: %v", err)
	}
	return nil
}
