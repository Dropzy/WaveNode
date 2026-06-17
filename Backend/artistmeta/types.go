package artistmeta

import "time"

type ArtistMatch struct {
	MBID            string
	Name            string
	SortName        string
	Type            string
	Country         string
	Disambiguation  string
	Tags            []string
	WikidataID      string
	CommonsCategory string
	ConfidenceScore float64
}

type ImageCandidate struct {
	ArtistID        string    `json:"artist_id"`
	Source          string    `json:"source"`
	ImageURL        string    `json:"image_url"`
	ThumbnailURL    string    `json:"thumbnail_url"`
	SourcePageURL   string    `json:"source_page_url"`
	LicenseName     string    `json:"license_name"`
	LicenseURL      string    `json:"license_url"`
	AuthorName      string    `json:"author_name"`
	AttributionText string    `json:"attribution_text"`
	Width           int       `json:"width"`
	Height          int       `json:"height"`
	MimeType        string    `json:"mime_type"`
	ConfidenceScore float64   `json:"confidence_score"`
	IsPrimary       bool      `json:"is_primary"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type LookupResult struct {
	Artist     ArtistMatch      `json:"artist"`
	Image      *ImageCandidate  `json:"image,omitempty"`
	Candidates []ImageCandidate `json:"candidates"`
	Refreshed  bool             `json:"refreshed"`
}

type APIResponseCache interface {
	GetSourceAPICache(cacheKey string) ([]byte, bool, error)
	SetSourceAPICache(cacheKey string, provider string, payload []byte, expiresAt time.Time) error
}
