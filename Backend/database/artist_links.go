package database

import (
	"fmt"
	"strings"
)

// RepairMissingArtistLinks creates missing artist records and links tracks that
// were imported before scans began assigning artist IDs.
func (db *DB) RepairMissingArtistLinks() (int64, error) {
	rows, err := db.conn.Query(`
		SELECT DISTINCT artist
		FROM music
		WHERE (artist_id IS NULL OR artist_id = '')
		  AND artist IS NOT NULL
		  AND BTRIM(artist) != ''
		ORDER BY artist
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to find unlinked track artists: %v", err)
	}

	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to read unlinked track artist: %v", err)
		}
		names = append(names, strings.TrimSpace(name))
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("failed to close unlinked artist query: %v", err)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to iterate unlinked track artists: %v", err)
	}

	var repaired int64
	for _, name := range names {
		primaryArtistName := PrimaryArtistName(name)
		if primaryArtistName == "" {
			continue
		}
		artist, err := db.ArtistOrStoreArtist(primaryArtistName)
		if err != nil {
			return repaired, err
		}
		result, err := db.conn.Exec(`
			UPDATE music
			SET artist_id = $1, updated_at = CURRENT_TIMESTAMP
			WHERE (artist_id IS NULL OR artist_id = '') AND artist = $2
		`, artist.ID, name)
		if err != nil {
			return repaired, fmt.Errorf("failed to link tracks for artist %q: %v", name, err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return repaired, fmt.Errorf("failed to count repaired tracks for artist %q: %v", name, err)
		}
		repaired += count
	}
	return repaired, nil
}

func nullableString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
