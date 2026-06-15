package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// FindSimilarAlbums searches for albums with similar names using fuzzy matching
func (db *DB) FindSimilarAlbums(searchTerm string, limit int) ([]Album, error) {
	// Split search term into words for better matching
	words := strings.Fields(strings.ToLower(searchTerm))
	if len(words) == 0 {
		return []Album{}, nil
	}

	// Build a query that looks for albums containing any of the search words
	var whereConditions []string
	var args []interface{}

	for i, word := range words {
		// Skip very short words
		if len(word) < 2 {
			continue
		}
		whereConditions = append(whereConditions, fmt.Sprintf("LOWER(album) LIKE $%d", i+1))
		args = append(args, "%"+word+"%")
	}

	if len(whereConditions) == 0 {
		return []Album{}, nil
	}

	whereClause := strings.Join(whereConditions, " OR ")

	limitClause := ""
	if limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT %d", limit)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT album, 
		       CASE 
		         WHEN COUNT(DISTINCT artist) > 1 THEN 'Various Artists'
		         WHEN MIN(artist) IS NULL OR MIN(artist) = '' THEN 'Unknown Artist'
		         ELSE MIN(artist)
		       END as artist,
		       COUNT(*) as track_count,
		       MIN(year) as year
		FROM music 
		WHERE album IS NOT NULL AND album != '' AND (%s)
		GROUP BY album
		ORDER BY 
			CASE 
				WHEN LOWER(album) = LOWER($1) THEN 1
				WHEN LOWER(album) LIKE LOWER($1) || '%%' THEN 2
				WHEN LOWER(album) LIKE '%%' || LOWER($1) THEN 3
				ELSE 4
			END,
			album
		%s
	`, whereClause, limitClause)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar albums: %v", err)
	}
	defer rows.Close()

	var albums []Album
	for rows.Next() {
		var album Album
		var year sql.NullInt32

		err := rows.Scan(&album.Name, &album.Artist, &album.TrackCount, &year)
		if err != nil {
			return nil, fmt.Errorf("failed to scan album row: %v", err)
		}

		// Handle nullable year field
		if year.Valid {
			album.Year = int(year.Int32)
		}

		albums = append(albums, album)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating album rows: %v", err)
	}

	return albums, nil
}

// GetAlbumTracksWithFallback tries exact match first, then falls back to similar albums
func (db *DB) GetAlbumTracksWithFallback(albumName, artistName string) ([]Music, []Album, error) {
	// Try exact match first
	tracks, err := db.GetAlbumTracks(albumName, artistName)
	if err == nil && len(tracks) > 0 {
		return tracks, nil, nil
	}

	// If no exact match, find similar albums
	similarAlbums, err := db.FindSimilarAlbums(albumName, 10)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find similar albums: %v", err)
	}

	return nil, similarAlbums, nil
}
