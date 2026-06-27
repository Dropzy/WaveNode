package database

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// runMigrations runs database migrations to ensure schema is up to date
func (db *DB) runMigrations() error {
	musicSourcesMigration := `
		CREATE TABLE IF NOT EXISTS music_sources (
			id VARCHAR(255) PRIMARY KEY,
			path TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS app_settings (
			key VARCHAR(255) PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := db.conn.Exec(musicSourcesMigration); err != nil {
		return fmt.Errorf("failed to create music source settings: %v", err)
	}

	playlistOwnershipMigration := `
		ALTER TABLE playlists ADD COLUMN IF NOT EXISTS user_id VARCHAR(255);
		UPDATE playlists
		SET user_id = (
			SELECT id FROM users
			ORDER BY CASE WHEN role = 'admin' THEN 0 ELSE 1 END, created_at
			LIMIT 1
		)
		WHERE user_id IS NULL;
		ALTER TABLE playlists ALTER COLUMN user_id SET NOT NULL;
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'playlists_user_id_fkey'
			) THEN
				ALTER TABLE playlists
					ADD CONSTRAINT playlists_user_id_fkey
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
			END IF;
		END $$;
		CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON playlists(user_id);
	`
	if _, err := db.conn.Exec(playlistOwnershipMigration); err != nil {
		return fmt.Errorf("failed to migrate playlist ownership: %v", err)
	}

	smartPlaylistMigration := `
		ALTER TABLE playlists
			ADD COLUMN IF NOT EXISTS playlist_type VARCHAR(20) NOT NULL DEFAULT 'manual';
		ALTER TABLE playlists
			ADD COLUMN IF NOT EXISTS smart_rules JSONB;
		UPDATE playlists SET playlist_type = 'manual'
		WHERE playlist_type IS NULL OR playlist_type = '';
		CREATE INDEX IF NOT EXISTS idx_playlists_type ON playlists(playlist_type);
	`
	if _, err := db.conn.Exec(smartPlaylistMigration); err != nil {
		return fmt.Errorf("failed to migrate smart playlists: %v", err)
	}

	recentlyPlayedMigration := `
		DELETE FROM recently_played older
		USING recently_played newer
		WHERE older.user_id = newer.user_id
		  AND older.track_id = newer.track_id
		  AND (
			older.played_at < newer.played_at
			OR (older.played_at = newer.played_at AND older.id < newer.id)
		  );
		CREATE UNIQUE INDEX IF NOT EXISTS idx_recently_played_user_track
			ON recently_played(user_id, track_id);
	`
	if _, err := db.conn.Exec(recentlyPlayedMigration); err != nil {
		return fmt.Errorf("failed to migrate recently played history: %v", err)
	}

	subsonicStateMigration := `
		CREATE TABLE IF NOT EXISTS media_ratings (
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			media_id VARCHAR(255) NOT NULL,
			media_type VARCHAR(20) NOT NULL DEFAULT 'song',
			rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, media_id)
		);
		CREATE INDEX IF NOT EXISTS idx_media_ratings_user_id ON media_ratings(user_id);

		CREATE TABLE IF NOT EXISTS media_stars (
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			media_id VARCHAR(255) NOT NULL,
			media_type VARCHAR(20) NOT NULL CHECK (media_type IN ('artist', 'album')),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, media_id)
		);
		CREATE INDEX IF NOT EXISTS idx_media_stars_user_id ON media_stars(user_id);

		CREATE TABLE IF NOT EXISTS media_bookmarks (
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			track_id VARCHAR(255) NOT NULL REFERENCES music(id) ON DELETE CASCADE,
			position_ms BIGINT NOT NULL DEFAULT 0 CHECK (position_ms >= 0),
			comment TEXT NOT NULL DEFAULT '',
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, track_id)
		);
		CREATE INDEX IF NOT EXISTS idx_media_bookmarks_user_id ON media_bookmarks(user_id);

		CREATE TABLE IF NOT EXISTS user_play_queues (
			user_id VARCHAR(255) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			track_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
			current_track_id VARCHAR(255),
			position_ms BIGINT NOT NULL DEFAULT 0 CHECK (position_ms >= 0),
			changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	if _, err := db.conn.Exec(subsonicStateMigration); err != nil {
		return fmt.Errorf("failed to create Subsonic user state: %v", err)
	}

	playbackFeaturesMigration := `
		CREATE TABLE IF NOT EXISTS track_audio_properties (
			track_id VARCHAR(255) PRIMARY KEY REFERENCES music(id) ON DELETE CASCADE,
			disc_number INTEGER NOT NULL DEFAULT 1,
			disc_total INTEGER NOT NULL DEFAULT 1,
			replaygain_track_db DOUBLE PRECISION NOT NULL DEFAULT 0,
			replaygain_album_db DOUBLE PRECISION NOT NULL DEFAULT 0,
			replaygain_track_peak DOUBLE PRECISION NOT NULL DEFAULT 0,
			replaygain_album_peak DOUBLE PRECISION NOT NULL DEFAULT 0,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_track_audio_properties_disc
			ON track_audio_properties(disc_number, track_id);
		CREATE TABLE IF NOT EXISTS playback_profiles (
			user_id VARCHAR(255) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			replaygain_mode VARCHAR(20) NOT NULL DEFAULT 'track',
			replaygain_preamp_db DOUBLE PRECISION NOT NULL DEFAULT 0,
			transcode_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			transcode_format VARCHAR(20) NOT NULL DEFAULT 'mp3',
			transcode_bitrate INTEGER NOT NULL DEFAULT 192,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS user_sessions (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			device_name VARCHAR(255) NOT NULL DEFAULT 'Web browser',
			user_agent TEXT NOT NULL DEFAULT '',
			ip_address VARCHAR(100) NOT NULL DEFAULT '',
			last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			revoked_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_user_active
			ON user_sessions(user_id, last_seen_at DESC) WHERE revoked_at IS NULL;
		CREATE TABLE IF NOT EXISTS listening_history (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			track_id VARCHAR(255) NOT NULL REFERENCES music(id) ON DELETE CASCADE,
			played_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			source VARCHAR(30) NOT NULL DEFAULT 'web',
			device VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_listening_history_user_played
			ON listening_history(user_id, played_at DESC);
	`
	if _, err := db.conn.Exec(playbackFeaturesMigration); err != nil {
		return fmt.Errorf("failed to migrate playback and history features: %v", err)
	}

	podcastProgressMigration := `
		CREATE TABLE IF NOT EXISTS podcast_progress (
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			podcast_id VARCHAR(255) NOT NULL,
			episode_id VARCHAR(255) NOT NULL,
			podcast_title TEXT NOT NULL DEFAULT '',
			publisher TEXT NOT NULL DEFAULT '',
			episode_title TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			image_url TEXT NOT NULL DEFAULT '',
			audio_url TEXT NOT NULL DEFAULT '',
			website_url TEXT NOT NULL DEFAULT '',
			published_at TIMESTAMP,
			duration_seconds INTEGER NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
			position_seconds INTEGER NOT NULL DEFAULT 0 CHECK (position_seconds >= 0),
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, podcast_id, episode_id)
		);
		CREATE INDEX IF NOT EXISTS idx_podcast_progress_continue
			ON podcast_progress(user_id, updated_at DESC)
			WHERE position_seconds > 0 AND completed = FALSE;
	`
	if _, err := db.conn.Exec(podcastProgressMigration); err != nil {
		return fmt.Errorf("failed to migrate podcast progress: %v", err)
	}

	pluginMigration := `
		CREATE TABLE IF NOT EXISTS plugins (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(150) NOT NULL,
			version VARCHAR(50) NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			manifest JSONB NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_plugins_enabled ON plugins(enabled);
	`
	if _, err := db.conn.Exec(pluginMigration); err != nil {
		return fmt.Errorf("failed to create plugin registry: %v", err)
	}

	// Add play_count column if it doesn't exist (for backward compatibility)
	playCountMigration := `
		ALTER TABLE music ADD COLUMN IF NOT EXISTS play_count INTEGER DEFAULT 0;
	`

	if _, err := db.conn.Exec(playCountMigration); err != nil {
		log.Printf("Warning: Failed to run play_count migration: %v", err)
		// Don't return error as this might fail if column already exists
	} else {
		log.Println("Play count migration completed successfully")
	}

	uploadOrderMigration := `
		CREATE SEQUENCE IF NOT EXISTS music_upload_order_seq;
		ALTER TABLE music ADD COLUMN IF NOT EXISTS upload_order BIGINT;

		WITH existing_max AS (
			SELECT COALESCE(MAX(upload_order), 0) AS max_order FROM music
		),
		missing_order AS (
			SELECT id,
			       ROW_NUMBER() OVER (ORDER BY created_at ASC, ctid ASC)
			       + (SELECT max_order FROM existing_max) AS new_order
			FROM music
			WHERE upload_order IS NULL
		)
		UPDATE music
		SET upload_order = missing_order.new_order
		FROM missing_order
		WHERE music.id = missing_order.id;

		SELECT setval(
			'music_upload_order_seq',
			GREATEST(COALESCE((SELECT MAX(upload_order) FROM music), 0), 1),
			COALESCE((SELECT MAX(upload_order) FROM music), 0) > 0
		);
		ALTER TABLE music
			ALTER COLUMN upload_order SET DEFAULT nextval('music_upload_order_seq');
		ALTER TABLE music ALTER COLUMN upload_order SET NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_music_upload_order ON music(upload_order DESC);
	`
	if _, err := db.conn.Exec(uploadOrderMigration); err != nil {
		return fmt.Errorf("failed to migrate music upload order: %v", err)
	}

	// Run artists table migration first (most critical for new functionality)
	artistsMigration, err := os.ReadFile("database/migrations/create_artists_table.sql")
	if err != nil {
		log.Printf("Warning: Could not read artists migration file: %v", err)
	} else {
		// Split migration into individual statements
		statements := strings.Split(string(artistsMigration), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.conn.Exec(stmt); err != nil {
				log.Printf("Warning: Failed to run artists migration statement: %v", err)
				log.Printf("Statement: %s", stmt)
			}
		}
		log.Println("Artists table migration completed successfully")
	}

	artistForeignKeyMigration := `
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_music_artist'
				  AND conrelid = 'music'::regclass
			) THEN
				ALTER TABLE music
					ADD CONSTRAINT fk_music_artist
					FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE SET NULL;
			END IF;
		END
		$$;
	`
	if _, err := db.conn.Exec(artistForeignKeyMigration); err != nil {
		return fmt.Errorf("failed to ensure artist relationship: %v", err)
	}

	artistMetadataPipelineMigration := `
		CREATE TABLE IF NOT EXISTS artist_aliases (
			id BIGSERIAL PRIMARY KEY,
			artist_id VARCHAR(255) NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
			alias TEXT NOT NULL,
			source VARCHAR(80) NOT NULL DEFAULT 'manual',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (artist_id, alias, source)
		);
		CREATE INDEX IF NOT EXISTS idx_artist_aliases_alias ON artist_aliases(alias);

		CREATE TABLE IF NOT EXISTS artist_external_ids (
			artist_id VARCHAR(255) NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
			provider VARCHAR(80) NOT NULL,
			external_id TEXT NOT NULL,
			external_url TEXT,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (artist_id, provider)
		);

		CREATE TABLE IF NOT EXISTS artist_images (
			id BIGSERIAL PRIMARY KEY,
			artist_id VARCHAR(255) NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
			source VARCHAR(80) NOT NULL,
			image_url TEXT NOT NULL,
			thumbnail_url TEXT,
			source_page_url TEXT,
			license_name TEXT,
			license_url TEXT,
			author_name TEXT,
			attribution_text TEXT,
			width INTEGER NOT NULL DEFAULT 0,
			height INTEGER NOT NULL DEFAULT 0,
			mime_type VARCHAR(100),
			confidence_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			is_primary BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (artist_id, source, image_url)
		);
		CREATE INDEX IF NOT EXISTS idx_artist_images_artist_primary ON artist_images(artist_id, is_primary);

		CREATE TABLE IF NOT EXISTS artist_metadata_refresh_jobs (
			id BIGSERIAL PRIMARY KEY,
			artist_id VARCHAR(255) REFERENCES artists(id) ON DELETE CASCADE,
			status VARCHAR(40) NOT NULL DEFAULT 'pending',
			error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			completed_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS uploaded_artist_images (
			id BIGSERIAL PRIMARY KEY,
			artist_id VARCHAR(255) NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
			image_url TEXT NOT NULL,
			original_filename TEXT,
			author_name TEXT,
			attribution_text TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS source_api_cache (
			cache_key TEXT PRIMARY KEY,
			provider VARCHAR(80) NOT NULL,
			response_body BYTEA NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_source_api_cache_expires ON source_api_cache(expires_at);
	`
	if _, err := db.conn.Exec(artistMetadataPipelineMigration); err != nil {
		return fmt.Errorf("failed to create artist metadata pipeline tables: %v", err)
	}

	// Ensure spotify_id and popularity columns exist in artists table
	spotifyColumnsMigration := `
		ALTER TABLE artists ADD COLUMN IF NOT EXISTS spotify_id VARCHAR(255);
		ALTER TABLE artists ADD COLUMN IF NOT EXISTS popularity INTEGER DEFAULT 0;
		UPDATE artists SET spotify_id = NULL WHERE BTRIM(COALESCE(spotify_id, '')) = '';
	`

	if _, err := db.conn.Exec(spotifyColumnsMigration); err != nil {
		log.Printf("Warning: Failed to add spotify_id/popularity columns to artists table: %v", err)
	} else {
		log.Println("Spotify columns migration completed successfully")
	}

	// Run cover art migration for new functionality
	coverArtMigration, err := os.ReadFile("database/migrations/add_cover_art.sql")
	if err != nil {
		log.Printf("Warning: Could not read cover art migration file: %v", err)
	} else {
		// Split migration into individual statements
		statements := strings.Split(string(coverArtMigration), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.conn.Exec(stmt); err != nil {
				log.Printf("Warning: Failed to run cover art migration statement: %v", err)
				log.Printf("Statement: %s", stmt)
			}
		}
		log.Println("Cover art migration completed successfully")
	}

	// Create indexes for new columns (only after artists table exists)
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_music_play_count ON music(play_count)",
		"CREATE INDEX IF NOT EXISTS idx_music_artist_id ON music(artist_id)",
	}

	for _, index := range indexes {
		if _, err := db.conn.Exec(index); err != nil {
			log.Printf("Warning: Failed to create migration index: %v", err)
		} else {
			log.Printf("Index created successfully: %s", index)
		}
	}

	// Create artists indexes (only after artists table exists)
	artistIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_artists_spotify_id ON artists(spotify_id)",
		"CREATE INDEX IF NOT EXISTS idx_artists_name ON artists(name)",
	}

	for _, index := range artistIndexes {
		if _, err := db.conn.Exec(index); err != nil {
			log.Printf("Warning: Failed to create artist migration index: %v", err)
		} else {
			log.Printf("Artist index created successfully: %s", index)
		}
	}

	// Run MusicBrainz columns migration
	musicbrainzMigration, err := os.ReadFile("database/migrations/add_musicbrainz_columns.sql")
	if err != nil {
		log.Printf("Warning: Could not read MusicBrainz migration file: %v", err)
	} else {
		// Split migration into individual statements
		statements := strings.Split(string(musicbrainzMigration), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.conn.Exec(stmt); err != nil {
				log.Printf("Warning: Failed to run MusicBrainz migration statement: %v", err)
				log.Printf("Statement: %s", stmt)
			}
		}
		log.Println("MusicBrainz columns migration completed successfully")
	}
	if _, err := db.conn.Exec(`UPDATE artists SET musicbrainz_id = NULL WHERE BTRIM(COALESCE(musicbrainz_id, '')) = ''`); err != nil {
		return fmt.Errorf("failed to normalize artist MusicBrainz identifiers: %v", err)
	}

	// Ensure all required artists columns exist
	ensureColumnsMigration, err := os.ReadFile("database/migrations/ensure_artists_columns.sql")
	if err != nil {
		log.Printf("Warning: Could not read ensure artists columns migration file: %v", err)
	} else {
		// Split migration into individual statements
		statements := strings.Split(string(ensureColumnsMigration), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.conn.Exec(stmt); err != nil {
				log.Printf("Warning: Failed to run ensure artists columns migration statement: %v", err)
				log.Printf("Statement: %s", stmt)
			}
		}
		log.Println("Ensure artists columns migration completed successfully")
	}

	// Fix artist_id relationships between music and artists tables
	fixArtistIDMigration, err := os.ReadFile("migrations/fix_artist_id_relationship.sql")
	if err != nil {
		log.Printf("Warning: Could not read fix artist ID relationship migration file: %v", err)
	} else {
		// Split migration into individual statements
		statements := strings.Split(string(fixArtistIDMigration), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.conn.Exec(stmt); err != nil {
				log.Printf("Warning: Failed to run fix artist ID relationship migration statement: %v", err)
				log.Printf("Statement: %s", stmt)
			}
		}
		log.Println("Fix artist ID relationship migration completed successfully")
	}

	// Create scan_status table for enrichment and scan operations
	scanStatusMigration, err := os.ReadFile("database/migrations/create_scan_status_table.sql")
	if err != nil {
		log.Printf("Warning: Could not read scan status migration file: %v", err)
	} else {
		// Split migration into individual statements
		statements := strings.Split(string(scanStatusMigration), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.conn.Exec(stmt); err != nil {
				log.Printf("Warning: Failed to run scan status migration statement: %v", err)
				log.Printf("Statement: %s", stmt)
			}
		}
		log.Println("Scan status table migration completed successfully")
	}

	// Add image_url column to music table for track-specific artwork
	imageURLMigration, err := os.ReadFile("database/migrations/add_image_url_column.sql")
	if err != nil {
		log.Printf("Warning: Could not read image URL migration file: %v", err)
	} else {
		// Split migration into individual statements
		statements := strings.Split(string(imageURLMigration), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.conn.Exec(stmt); err != nil {
				log.Printf("Warning: Failed to run image URL migration statement: %v", err)
				log.Printf("Statement: %s", stmt)
			}
		}
		log.Println("Image URL column migration completed successfully")
	}

	// Add scan tracking fields to scan_status table
	scanTrackingMigration, err := os.ReadFile("database/migrations/add_scan_tracking_fields.sql")
	if err != nil {
		log.Printf("Warning: Could not read scan tracking fields migration file: %v", err)
	} else {
		// Split migration into individual statements
		statements := strings.Split(string(scanTrackingMigration), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.conn.Exec(stmt); err != nil {
				log.Printf("Warning: Failed to run scan tracking fields migration statement: %v", err)
				log.Printf("Statement: %s", stmt)
			}
		}
		log.Println("Scan tracking fields migration completed successfully")
	}

	return nil
}

// createTables creates necessary tables if they don't exist
func (db *DB) createTables() error {
	// Create music table
	musicTable := `
	CREATE TABLE IF NOT EXISTS music (
		id VARCHAR(255) PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		artist VARCHAR(255) NOT NULL,
		album VARCHAR(255),
		genre VARCHAR(100),
		duration INTEGER NOT NULL,
		release_date TIMESTAMP,
		file_path TEXT,
		file_name VARCHAR(255),
		file_size BIGINT,
		format VARCHAR(50),
		year INTEGER,
		track_number INTEGER,
		featuring JSONB,
		has_metadata BOOLEAN DEFAULT FALSE,
		confidence INTEGER DEFAULT 0,
		source VARCHAR(50),
		parsed_from_filename BOOLEAN DEFAULT FALSE,
		artist_image_url TEXT,
		play_count INTEGER DEFAULT 0,
		image_url TEXT,
		upload_order BIGSERIAL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.conn.Exec(musicTable); err != nil {
		return fmt.Errorf("failed to create music table: %v", err)
	}

	// Create users table
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(255) PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		role VARCHAR(50) DEFAULT 'user',
		password VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.conn.Exec(usersTable); err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	// Create playlists table
	playlistsTable := `
	CREATE TABLE IF NOT EXISTS playlists (
		id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		image_url TEXT DEFAULT '',
		track_ids JSONB,
		playlist_type VARCHAR(20) NOT NULL DEFAULT 'manual',
		smart_rules JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.conn.Exec(playlistsTable); err != nil {
		return fmt.Errorf("failed to create playlists table: %v", err)
	}
	if _, err := db.conn.Exec(`ALTER TABLE playlists ADD COLUMN IF NOT EXISTS image_url TEXT DEFAULT ''`); err != nil {
		return fmt.Errorf("failed to migrate playlist images: %v", err)
	}

	// Create liked_tracks table
	likedTracksTable := `
	CREATE TABLE IF NOT EXISTS liked_tracks (
		id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		track_id VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (track_id) REFERENCES music(id) ON DELETE CASCADE,
		UNIQUE(user_id, track_id)
	)`

	if _, err := db.conn.Exec(likedTracksTable); err != nil {
		return fmt.Errorf("failed to create liked_tracks table: %v", err)
	}

	// Create recently_played table (fixed typo: was "recently_played")
	recentlyPlayedTable := `
	CREATE TABLE IF NOT EXISTS recently_played (
		id VARCHAR(255) PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		track_id VARCHAR(255) NOT NULL,
		played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (track_id) REFERENCES music(id) ON DELETE CASCADE,
		UNIQUE(user_id, track_id)
	)`

	if _, err := db.conn.Exec(recentlyPlayedTable); err != nil {
		return fmt.Errorf("failed to create recently_played table: %v", err)
	}

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_music_title ON music(title)",
		"CREATE INDEX IF NOT EXISTS idx_music_artist ON music(artist)",
		"CREATE INDEX IF NOT EXISTS idx_music_album ON music(album)",
		"CREATE INDEX IF NOT EXISTS idx_music_genre ON music(genre)",
		"CREATE INDEX IF NOT EXISTS idx_music_file_path ON music(file_path)",
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_liked_tracks_user_id ON liked_tracks(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_liked_tracks_track_id ON liked_tracks(track_id)",
		"CREATE INDEX IF NOT EXISTS idx_recently_played_user_id ON recently_played(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_recently_played_track_id ON recently_played(track_id)",
		"CREATE INDEX IF NOT EXISTS idx_recently_played_played_at ON recently_played(played_at DESC)",
	}

	for _, index := range indexes {
		if _, err := db.conn.Exec(index); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}

	return nil
}
