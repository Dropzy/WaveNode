package database

import "fmt"

func (db *DB) GetRadioFavorites(userID string) ([]RadioStation, error) {
	rows, err := db.conn.Query(`
		SELECT station_id, name, stream_url, homepage_url, favicon_url, tags,
			country, country_code, language, codec, bitrate, votes, click_count, created_at
		FROM radio_favorites
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load radio favourites: %v", err)
	}
	defer rows.Close()

	stations := make([]RadioStation, 0)
	for rows.Next() {
		var station RadioStation
		if err := rows.Scan(
			&station.ID, &station.Name, &station.StreamURL, &station.HomepageURL,
			&station.FaviconURL, &station.Tags, &station.Country, &station.CountryCode,
			&station.Language, &station.Codec, &station.Bitrate, &station.Votes,
			&station.ClickCount, &station.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to read radio favourite: %v", err)
		}
		station.Favourite = true
		stations = append(stations, station)
	}
	return stations, rows.Err()
}

func (db *DB) SaveRadioFavorite(userID string, station RadioStation) (RadioStation, error) {
	err := db.conn.QueryRow(`
		INSERT INTO radio_favorites (
			user_id, station_id, name, stream_url, homepage_url, favicon_url, tags,
			country, country_code, language, codec, bitrate, votes, click_count, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, station_id) DO UPDATE SET
			name = EXCLUDED.name, stream_url = EXCLUDED.stream_url,
			homepage_url = EXCLUDED.homepage_url, favicon_url = EXCLUDED.favicon_url,
			tags = EXCLUDED.tags, country = EXCLUDED.country,
			country_code = EXCLUDED.country_code, language = EXCLUDED.language,
			codec = EXCLUDED.codec, bitrate = EXCLUDED.bitrate,
			votes = EXCLUDED.votes, click_count = EXCLUDED.click_count
		RETURNING created_at
	`, userID, station.ID, station.Name, station.StreamURL, station.HomepageURL,
		station.FaviconURL, station.Tags, station.Country, station.CountryCode,
		station.Language, station.Codec, station.Bitrate, station.Votes, station.ClickCount,
	).Scan(&station.CreatedAt)
	if err != nil {
		return RadioStation{}, fmt.Errorf("failed to save radio favourite: %v", err)
	}
	station.Favourite = true
	return station, nil
}

func (db *DB) DeleteRadioFavorite(userID, stationID string) error {
	if _, err := db.conn.Exec(`DELETE FROM radio_favorites WHERE user_id = $1 AND station_id = $2`, userID, stationID); err != nil {
		return fmt.Errorf("failed to remove radio favourite: %v", err)
	}
	return nil
}
