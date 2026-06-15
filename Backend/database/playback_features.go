package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

func DefaultPlaybackProfile(userID string) PlaybackProfile {
	return PlaybackProfile{UserID: userID, ReplayGainMode: "track", TranscodeFormat: "mp3", TranscodeBitrate: 192}
}

func (db *DB) GetPlaybackProfile(userID string) (PlaybackProfile, error) {
	profile := DefaultPlaybackProfile(userID)
	err := db.conn.QueryRow(`
		SELECT replaygain_mode, replaygain_preamp_db, transcode_enabled,
		       transcode_format, transcode_bitrate, updated_at
		FROM playback_profiles WHERE user_id = $1
	`, userID).Scan(&profile.ReplayGainMode, &profile.ReplayGainPreampDB, &profile.TranscodeEnabled,
		&profile.TranscodeFormat, &profile.TranscodeBitrate, &profile.UpdatedAt)
	if err == sql.ErrNoRows {
		return profile, nil
	}
	return profile, err
}

func (db *DB) SavePlaybackProfile(profile PlaybackProfile) error {
	if profile.ReplayGainMode != "off" && profile.ReplayGainMode != "track" && profile.ReplayGainMode != "album" {
		return fmt.Errorf("replaygain mode must be off, track, or album")
	}
	profile.TranscodeFormat = strings.ToLower(strings.TrimSpace(profile.TranscodeFormat))
	if profile.TranscodeFormat != "mp3" && profile.TranscodeFormat != "opus" && profile.TranscodeFormat != "aac" {
		return fmt.Errorf("transcode format must be mp3, opus, or aac")
	}
	if profile.TranscodeBitrate < 48 || profile.TranscodeBitrate > 320 {
		return fmt.Errorf("transcode bitrate must be between 48 and 320 kbps")
	}
	_, err := db.conn.Exec(`
		INSERT INTO playback_profiles (
			user_id, replaygain_mode, replaygain_preamp_db, transcode_enabled,
			transcode_format, transcode_bitrate, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			replaygain_mode = EXCLUDED.replaygain_mode,
			replaygain_preamp_db = EXCLUDED.replaygain_preamp_db,
			transcode_enabled = EXCLUDED.transcode_enabled,
			transcode_format = EXCLUDED.transcode_format,
			transcode_bitrate = EXCLUDED.transcode_bitrate,
			updated_at = CURRENT_TIMESTAMP
	`, profile.UserID, profile.ReplayGainMode, profile.ReplayGainPreampDB,
		profile.TranscodeEnabled, profile.TranscodeFormat, profile.TranscodeBitrate)
	return err
}

func (db *DB) UpsertTrackAudioProperties(properties TrackAudioProperties) error {
	if properties.DiscNumber < 1 {
		properties.DiscNumber = 1
	}
	if properties.DiscTotal < properties.DiscNumber {
		properties.DiscTotal = properties.DiscNumber
	}
	_, err := db.conn.Exec(`
		INSERT INTO track_audio_properties (
			track_id, disc_number, disc_total, replaygain_track_db,
			replaygain_album_db, replaygain_track_peak, replaygain_album_peak
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (track_id) DO UPDATE SET
			disc_number = EXCLUDED.disc_number, disc_total = EXCLUDED.disc_total,
			replaygain_track_db = EXCLUDED.replaygain_track_db,
			replaygain_album_db = EXCLUDED.replaygain_album_db,
			replaygain_track_peak = EXCLUDED.replaygain_track_peak,
			replaygain_album_peak = EXCLUDED.replaygain_album_peak,
			updated_at = CURRENT_TIMESTAMP
	`, properties.TrackID, properties.DiscNumber, properties.DiscTotal,
		properties.ReplayGainTrackDB, properties.ReplayGainAlbumDB,
		properties.ReplayGainTrackPeak, properties.ReplayGainAlbumPeak)
	return err
}

func (db *DB) GetTrackAudioProperties(trackID string) (TrackAudioProperties, error) {
	properties := TrackAudioProperties{TrackID: trackID, DiscNumber: 1, DiscTotal: 1}
	err := db.conn.QueryRow(`
		SELECT disc_number, disc_total, replaygain_track_db, replaygain_album_db,
		       replaygain_track_peak, replaygain_album_peak
		FROM track_audio_properties WHERE track_id = $1
	`, trackID).Scan(&properties.DiscNumber, &properties.DiscTotal, &properties.ReplayGainTrackDB,
		&properties.ReplayGainAlbumDB, &properties.ReplayGainTrackPeak, &properties.ReplayGainAlbumPeak)
	if err == sql.ErrNoRows {
		return properties, nil
	}
	return properties, err
}

func ApplyTrackAudioProperties(track *Music, properties TrackAudioProperties) {
	track.DiscNumber = properties.DiscNumber
	track.DiscTotal = properties.DiscTotal
	track.ReplayGainTrackDB = properties.ReplayGainTrackDB
	track.ReplayGainAlbumDB = properties.ReplayGainAlbumDB
	track.ReplayGainTrackPeak = properties.ReplayGainTrackPeak
	track.ReplayGainAlbumPeak = properties.ReplayGainAlbumPeak
}

func (db *DB) EnrichTrackAudioProperties(tracks []Music) error {
	if len(tracks) == 0 {
		return nil
	}
	ids := make([]string, len(tracks))
	index := make(map[string]int, len(tracks))
	for position := range tracks {
		ids[position] = tracks[position].ID
		index[tracks[position].ID] = position
	}
	rows, err := db.conn.Query(`
		SELECT track_id, disc_number, disc_total, replaygain_track_db,
		       replaygain_album_db, replaygain_track_peak, replaygain_album_peak
		FROM track_audio_properties WHERE track_id = ANY($1)
	`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var properties TrackAudioProperties
		if err := rows.Scan(&properties.TrackID, &properties.DiscNumber, &properties.DiscTotal,
			&properties.ReplayGainTrackDB, &properties.ReplayGainAlbumDB,
			&properties.ReplayGainTrackPeak, &properties.ReplayGainAlbumPeak); err != nil {
			return err
		}
		if position, exists := index[properties.TrackID]; exists {
			ApplyTrackAudioProperties(&tracks[position], properties)
		}
	}
	for position := range tracks {
		if tracks[position].DiscNumber == 0 {
			tracks[position].DiscNumber = 1
			tracks[position].DiscTotal = 1
		}
	}
	return rows.Err()
}

func (db *DB) AddListeningHistory(userID, trackID, source, device string) error {
	_, err := db.conn.Exec(`
		INSERT INTO listening_history (id, user_id, track_id, played_at, source, device)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4, $5)
	`, fmt.Sprintf("history_%d", time.Now().UnixNano()), userID, trackID, source, device)
	return err
}

func (db *DB) GetListeningHistory(userID, search string, limit, offset int) ([]ListeningHistoryEntry, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	search = "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
	rows, err := db.conn.Query(`
		SELECT h.id, h.user_id, h.track_id, h.played_at, h.source, h.device,
		       m.title, m.artist, COALESCE(m.album, ''), COALESCE(m.duration, 0),
		       COALESCE(m.image_url, ''), COALESCE(m.cover_art_url, '')
		FROM listening_history h JOIN music m ON m.id = h.track_id
		WHERE h.user_id = $1 AND (
			$2 = '%%' OR LOWER(m.title) LIKE $2 OR LOWER(m.artist) LIKE $2
			OR LOWER(COALESCE(m.album, '')) LIKE $2
		)
		ORDER BY h.played_at DESC LIMIT $3 OFFSET $4
	`, userID, search, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]ListeningHistoryEntry, 0)
	for rows.Next() {
		var entry ListeningHistoryEntry
		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.TrackID, &entry.PlayedAt,
			&entry.Source, &entry.Device, &entry.Track.Title, &entry.Track.Artist,
			&entry.Track.Album, &entry.Track.Duration, &entry.Track.ImageURL,
			&entry.Track.CoverArtURL); err != nil {
			return nil, err
		}
		entry.Track.ID = entry.TrackID
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (db *DB) ClearListeningHistory(userID string) error {
	_, err := db.conn.Exec("DELETE FROM listening_history WHERE user_id = $1", userID)
	return err
}

func (db *DB) CreateSession(session UserSession) error {
	_, err := db.conn.Exec(`
		INSERT INTO user_sessions (
			id, user_id, device_name, user_agent, ip_address, last_seen_at, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $6)
	`, session.ID, session.UserID, session.DeviceName, session.UserAgent, session.IPAddress, session.ExpiresAt)
	return err
}

func (db *DB) IsSessionActive(id, userID string) bool {
	var active bool
	err := db.conn.QueryRow(`
		SELECT revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP
		FROM user_sessions WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&active)
	if err != nil || !active {
		return false
	}
	_, _ = db.conn.Exec(`
		UPDATE user_sessions SET last_seen_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND last_seen_at < CURRENT_TIMESTAMP - INTERVAL '5 minutes'
	`, id)
	return true
}

func (db *DB) GetUserSessions(userID string) ([]UserSession, error) {
	rows, err := db.conn.Query(`
		SELECT id, user_id, device_name, user_agent, ip_address, last_seen_at,
		       created_at, expires_at, revoked_at
		FROM user_sessions WHERE user_id = $1 AND expires_at > CURRENT_TIMESTAMP
		ORDER BY last_seen_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := make([]UserSession, 0)
	for rows.Next() {
		var session UserSession
		if err := rows.Scan(&session.ID, &session.UserID, &session.DeviceName, &session.UserAgent,
			&session.IPAddress, &session.LastSeenAt, &session.CreatedAt,
			&session.ExpiresAt, &session.RevokedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (db *DB) RevokeSession(userID, sessionID string) error {
	result, err := db.conn.Exec(`
		UPDATE user_sessions SET revoked_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND id = $2 AND revoked_at IS NULL
	`, userID, sessionID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("session not found")
	}
	return nil
}

func (db *DB) RevokeOtherSessions(userID, currentSessionID string) error {
	_, err := db.conn.Exec(`
		UPDATE user_sessions SET revoked_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL
	`, userID, currentSessionID)
	return err
}
