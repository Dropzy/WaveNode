package database

import (
	"encoding/json"
	"time"
)

// Artist represents an artist with comprehensive MusicBrainz integration
type Artist struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	MusicBrainzID  string   `json:"musicbrainz_id"`
	MusicBrainzURL string   `json:"musicbrainz_url"`
	ImageURL       string   `json:"image_url"`
	ImageSmallURL  string   `json:"image_small_url"`
	ImageMediumURL string   `json:"image_medium_url"`
	ImageLargeURL  string   `json:"image_large_url"`
	Country        string   `json:"country"`
	Tags           []string `json:"tags"`
	Biography      string   `json:"biography"`
	// Keep Spotify fields for backward compatibility during migration
	SpotifyID      string            `json:"spotify_id,omitempty"`
	SpotifyURL     string            `json:"spotify_url,omitempty"`
	Followers      int               `json:"followers,omitempty"`
	Popularity     int               `json:"popularity,omitempty"`
	Genres         []string          `json:"genres,omitempty"`
	ExternalURLs   map[string]string `json:"external_urls,omitempty"`
	URI            string            `json:"uri,omitempty"`
	HREF           string            `json:"href,omitempty"`
	Type           string            `json:"type"`
	APIData        string            `json:"api_data"`
	LastEnrichedAt *time.Time        `json:"last_enriched_at"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// Music represents a music track
type Music struct {
	ID                     string     `json:"id"`
	Title                  string     `json:"title"`
	Artist                 string     `json:"artist"`
	ArtistID               string     `json:"artist_id"`
	ArtistImageURL         string     `json:"artist_image_url"`
	Album                  string     `json:"album"`
	Genre                  string     `json:"genre"`
	Duration               int        `json:"duration"`
	ReleaseDate            *time.Time `json:"release_date"`
	FilePath               string     `json:"file_path"`
	FileName               string     `json:"file_name"`
	FileSize               int64      `json:"file_size"`
	Format                 string     `json:"format"`
	Year                   int        `json:"year"`
	TrackNumber            int        `json:"track_number"`
	DiscNumber             int        `json:"disc_number"`
	DiscTotal              int        `json:"disc_total"`
	ReplayGainTrackDB      float64    `json:"replaygain_track_db"`
	ReplayGainAlbumDB      float64    `json:"replaygain_album_db"`
	ReplayGainTrackPeak    float64    `json:"replaygain_track_peak"`
	ReplayGainAlbumPeak    float64    `json:"replaygain_album_peak"`
	Featuring              []string   `json:"featuring,omitempty"`
	HasMetadata            bool       `json:"has_metadata"`
	Confidence             int        `json:"confidence"` // 0-100, higher is better
	Source                 string     `json:"source"`     // "embedded", "filename", "mixed"
	ParsedFromFilename     bool       `json:"parsed_from_filename"`
	PlayCount              int        `json:"play_count"`
	ImageURL               string     `json:"image_url"` // Track-specific artwork from metadata
	CoverArtURL            string     `json:"cover_art_url"`
	CoverArtSmallURL       string     `json:"cover_art_small_url"`
	CoverArtMediumURL      string     `json:"cover_art_medium_url"`
	CoverArtLargeURL       string     `json:"cover_art_large_url"`
	CoverArtSource         string     `json:"cover_art_source"`
	LastCoverArtEnrichedAt *time.Time `json:"last_cover_art_enriched_at"`
	UploadOrder            int64      `json:"upload_order"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// Album represents an album in system
type Album struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Artist            string `json:"artist"`
	TrackCount        int    `json:"track_count"`
	Year              int    `json:"year"`
	CoverArtURL       string `json:"cover_art_url"`
	CoverArtSmallURL  string `json:"cover_art_small_url"`
	CoverArtMediumURL string `json:"cover_art_medium_url"`
	CoverArtLargeURL  string `json:"cover_art_large_url"`
	CoverArtSource    string `json:"cover_art_source"`
}

// MusicSource is a server-visible directory scanned for audio files.
type MusicSource struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

// Plugin stores an administrator-installed declarative extension manifest.
type Plugin struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	Enabled   bool            `json:"enabled"`
	Manifest  json.RawMessage `json:"manifest"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Playlist represents a collection of music tracks
type Playlist struct {
	ID          string              `json:"id"`
	UserID      string              `json:"user_id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Type        string              `json:"type"`
	SmartRules  *SmartPlaylistRules `json:"smart_rules,omitempty"`
	TrackIDs    []string            `json:"track_ids"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type SmartPlaylistRules struct {
	Match         string                   `json:"match"`
	Conditions    []SmartPlaylistCondition `json:"conditions"`
	Groups        []SmartPlaylistGroup     `json:"groups,omitempty"`
	SortBy        string                   `json:"sort_by"`
	SortDirection string                   `json:"sort_direction"`
	Limit         int                      `json:"limit"`
}

type SmartPlaylistGroup struct {
	Match      string                   `json:"match"`
	Conditions []SmartPlaylistCondition `json:"conditions,omitempty"`
	Groups     []SmartPlaylistGroup     `json:"groups,omitempty"`
}

type SmartPlaylistCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// User represents a user account
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"` // "admin" or "user"
	Password  string    `json:"-"`    // Don't include password in JSON responses
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LikedTrack represents a user's liked track
type LikedTrack struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TrackID   string    `json:"track_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RecentlyPlayedTrack represents a user's recently played track
type RecentlyPlayedTrack struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	TrackID  string    `json:"track_id"`
	PlayedAt time.Time `json:"played_at"`
}

type ListeningHistoryEntry struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	TrackID  string    `json:"track_id"`
	PlayedAt time.Time `json:"played_at"`
	Source   string    `json:"source"`
	Device   string    `json:"device"`
	Track    Music     `json:"track"`
}

type UserSession struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	DeviceName string     `json:"device_name"`
	UserAgent  string     `json:"user_agent"`
	IPAddress  string     `json:"ip_address"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

type PlaybackProfile struct {
	UserID             string    `json:"user_id"`
	ReplayGainMode     string    `json:"replaygain_mode"`
	ReplayGainPreampDB float64   `json:"replaygain_preamp_db"`
	TranscodeEnabled   bool      `json:"transcode_enabled"`
	TranscodeFormat    string    `json:"transcode_format"`
	TranscodeBitrate   int       `json:"transcode_bitrate"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type TrackAudioProperties struct {
	TrackID             string  `json:"track_id"`
	DiscNumber          int     `json:"disc_number"`
	DiscTotal           int     `json:"disc_total"`
	ReplayGainTrackDB   float64 `json:"replaygain_track_db"`
	ReplayGainAlbumDB   float64 `json:"replaygain_album_db"`
	ReplayGainTrackPeak float64 `json:"replaygain_track_peak"`
	ReplayGainAlbumPeak float64 `json:"replaygain_album_peak"`
}

type MediaRating struct {
	MediaID   string    `json:"media_id"`
	MediaType string    `json:"media_type"`
	Rating    int       `json:"rating"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MediaBookmark struct {
	TrackID    string    `json:"track_id"`
	PositionMS int64     `json:"position_ms"`
	Comment    string    `json:"comment"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PlayQueue struct {
	TrackIDs       []string  `json:"track_ids"`
	CurrentTrackID string    `json:"current_track_id"`
	PositionMS     int64     `json:"position_ms"`
	ChangedAt      time.Time `json:"changed_at"`
}

// ScanStatus represents the status of a media scan
type ScanStatus struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`   // "library" or "enrichment"
	Status        string     `json:"status"` // "running", "stopping", "stopped", "completed", "failed"
	Progress      int        `json:"progress"`
	TotalFiles    int        `json:"total_files"`
	Processed     int        `json:"processed"`
	CurrentFile   string     `json:"current_file"`
	Errors        []string   `json:"errors"`
	SongsAdded    int        `json:"songs_added"`
	SongsUpdated  int        `json:"songs_updated"`  // New field for updated tracks
	TracksSkipped int        `json:"tracks_skipped"` // New field for skipped tracks (duplicates)
	Duplicates    int        `json:"duplicates"`     // Keep for backward compatibility
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}
