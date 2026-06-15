package musicbrainz

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter for API requests
type RateLimiter struct {
	tokens       int
	maxTokens    int
	refillRate   time.Duration
	lastRefill   time.Time
	mutex        sync.Mutex
	requestDelay time.Duration
}

// NewRateLimiter creates a new rate limiter
// MusicBrainz recommends 1 request per second, but we'll be more conservative
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		tokens:       1,
		maxTokens:    1,                       // Start with 1 token
		refillRate:   1100 * time.Millisecond, // 1.1 seconds between requests (conservative)
		lastRefill:   time.Now(),
		requestDelay: 500 * time.Millisecond, // Additional delay between requests
	}
}

// WaitForToken waits until a token is available
func (rl *RateLimiter) WaitForToken() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Refill tokens if needed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	if elapsed >= rl.refillRate {
		if rl.tokens < rl.maxTokens {
			rl.tokens++
			rl.lastRefill = now
		}
	}

	// If no token available, wait
	if rl.tokens == 0 {
		waitTime := rl.refillRate - elapsed
		if waitTime > 0 {
			log.Printf("Rate limit reached, waiting %v", waitTime)
			time.Sleep(waitTime)
			rl.tokens = 1
			rl.lastRefill = time.Now()
		}
	}

	// Consume token
	rl.tokens--

	// Add additional delay between requests to be extra conservative
	if rl.requestDelay > 0 {
		time.Sleep(rl.requestDelay)
	}
}

// MusicBrainzClient wraps MusicBrainz API functionality
type MusicBrainzClient struct {
	httpClient  *http.Client
	baseURL     string
	rateLimiter *RateLimiter
	userAgent   string
}

// ArtistInfo holds simplified artist information from MusicBrainz
type ArtistInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	ImageURL string   `json:"image_url"`
	Type     string   `json:"type"`
	Country  string   `json:"country"`
	Lifespan Lifespan `json:"lifespan"`
	Tags     []string `json:"tags"`
}

// Lifespan represents artist lifespan
type Lifespan struct {
	Begin string `json:"begin"`
	End   string `json:"end"`
}

// ReleaseInfo holds release/album information from MusicBrainz
type ReleaseInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ArtistID    string `json:"artist_id"`
	ArtistName  string `json:"artist_name"`
	Date        string `json:"date"`
	CoverArtURL string `json:"cover_art_url"`
}

// CoverArtInfo holds cover art information in different sizes
type CoverArtInfo struct {
	SmallURL  string `json:"small_url"`
	MediumURL string `json:"medium_url"`
	LargeURL  string `json:"large_url"`
}

// MusicBrainzSearchResponse represents the response from MusicBrainz search API
type MusicBrainzSearchResponse struct {
	Created  string    `json:"created"`
	Count    int       `json:"count"`
	Offset   int       `json:"offset"`
	Artists  []Artist  `json:"artists"`
	Releases []Release `json:"releases"`
}

// Artist represents a MusicBrainz artist
type Artist struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	SortName string   `json:"sort-name"`
	Type     string   `json:"type"`
	Country  string   `json:"country"`
	Lifespan Lifespan `json:"life-span"`
	Tags     []Tag    `json:"tags"`
}

// Release represents a MusicBrainz release
type Release struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	ArtistCredit []ArtistCredit `json:"artist-credit"`
	Date         string         `json:"date"`
}

// ArtistCredit represents artist credit information
type ArtistCredit struct {
	Artist struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
	Name string `json:"name"`
}

// Tag represents a tag from MusicBrainz
type Tag struct {
	Count int    `json:"count"`
	Name  string `json:"name"`
}

// CoverArtResponse represents response from Cover Art Archive
type CoverArtResponse struct {
	Images  []CoverArtImage `json:"images"`
	Release string          `json:"release"`
}

// CoverArtImage represents a cover art image
type CoverArtImage struct {
	ID         interface{}       `json:"id"` // Can be string or number
	Image      string            `json:"image"`
	Thumbnails map[string]string `json:"thumbnails"`
	Approved   bool              `json:"approved"`
	Comment    string            `json:"comment"`
	Types      []string          `json:"types"`
	Front      bool              `json:"front"`
	Back       bool              `json:"back"`
}

// NewMusicBrainzClient creates a new MusicBrainz client
func NewMusicBrainzClient() *MusicBrainzClient {
	return &MusicBrainzClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     "https://musicbrainz.org/ws/2",
		rateLimiter: NewRateLimiter(),
	}
}

// SearchArtists searches for artists by name and returns multiple results
func (mb *MusicBrainzClient) SearchArtists(artistName string, limit int) ([]*ArtistInfo, error) {
	if mb.httpClient == nil {
		return nil, fmt.Errorf("MusicBrainz client not initialized")
	}

	// Wait for rate limiter to allow this request
	mb.rateLimiter.WaitForToken()

	// Build search query
	query := url.QueryEscape(fmt.Sprintf(`artist:"%s"`, artistName))
	searchURL := fmt.Sprintf("%s/artist/?query=%s&limit=%d&fmt=json", mb.baseURL, query, limit)

	// Make request
	resp, err := mb.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search for artists: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz API error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var searchResponse MusicBrainzSearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse MusicBrainz response: %v", err)
	}

	var artists []*ArtistInfo
	for _, artist := range searchResponse.Artists {
		// Extract tags
		var tags []string
		for _, tag := range artist.Tags {
			tags = append(tags, tag.Name)
		}

		artists = append(artists, &ArtistInfo{
			ID:       artist.ID,
			Name:     artist.Name,
			Type:     artist.Type,
			Country:  artist.Country,
			Lifespan: artist.Lifespan,
			Tags:     tags,
		})
	}

	return artists, nil
}

// GetArtistInfo gets detailed artist information using MusicBrainz ID
func (mb *MusicBrainzClient) GetArtistInfo(artistID string) (*ArtistInfo, error) {
	if mb.httpClient == nil {
		return nil, fmt.Errorf("MusicBrainz client not initialized")
	}

	// Wait for rate limiter to allow this request
	mb.rateLimiter.WaitForToken()

	// Get artist details with URL relationships
	lookupURL := fmt.Sprintf("%s/artist/%s?inc=url-rels&fmt=json", mb.baseURL, artistID)

	resp, err := mb.httpClient.Get(lookupURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get artist details: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz API error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var artist Artist
	if err := json.Unmarshal(body, &artist); err != nil {
		return nil, fmt.Errorf("failed to parse artist response: %v", err)
	}

	// Extract tags
	var tags []string
	for _, tag := range artist.Tags {
		tags = append(tags, tag.Name)
	}

	return &ArtistInfo{
		ID:       artist.ID,
		Name:     artist.Name,
		Type:     artist.Type,
		Country:  artist.Country,
		Lifespan: artist.Lifespan,
		Tags:     tags,
	}, nil
}

// GetBestMatchForArtist finds the best matching artist for a given name
func (mb *MusicBrainzClient) GetBestMatchForArtist(artistName string) (*ArtistInfo, error) {
	artists, err := mb.SearchArtists(artistName, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to search for artist: %v", err)
	}

	if len(artists) == 0 {
		return nil, fmt.Errorf("artist not found: %s", artistName)
	}

	// Find the best match based on name similarity
	normalizedSearchName := strings.ToLower(strings.TrimSpace(artistName))
	var bestMatch *ArtistInfo
	bestScore := 0.0

	for _, artist := range artists {
		normalizedArtistName := strings.ToLower(strings.TrimSpace(artist.Name))

		// Exact match gets the highest score
		if normalizedArtistName == normalizedSearchName {
			bestMatch = artist
			break
		}

		// Calculate similarity score
		score := 0.0
		if strings.HasPrefix(normalizedArtistName, normalizedSearchName) {
			score = 0.8
		} else if strings.HasPrefix(normalizedSearchName, normalizedArtistName) {
			score = 0.7
		} else if strings.Contains(normalizedArtistName, normalizedSearchName) || strings.Contains(normalizedSearchName, normalizedArtistName) {
			score = 0.5
		}

		// Bonus for having tags (indicates more complete data)
		if len(artist.Tags) > 0 {
			score += 0.1
		}

		if score > bestScore {
			bestScore = score
			bestMatch = artist
		}
	}

	// If no good match found, use first result
	if bestMatch == nil {
		bestMatch = artists[0]
	}

	return bestMatch, nil
}

// SearchReleases searches for releases by name and artist
func (mb *MusicBrainzClient) SearchReleases(releaseName, artistName string, limit int) ([]*ReleaseInfo, error) {
	if mb.httpClient == nil {
		return nil, fmt.Errorf("MusicBrainz client not initialized")
	}

	// Wait for rate limiter to allow this request
	mb.rateLimiter.WaitForToken()

	// Build search query
	query := url.QueryEscape(fmt.Sprintf(`release:"%s" AND artist:"%s"`, releaseName, artistName))
	searchURL := fmt.Sprintf("%s/release/?query=%s&limit=%d&fmt=json", mb.baseURL, query, limit)

	resp, err := mb.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search for releases: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz API error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var searchResponse MusicBrainzSearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse MusicBrainz response: %v", err)
	}

	var releases []*ReleaseInfo
	for _, release := range searchResponse.Releases {
		artistName := ""
		if len(release.ArtistCredit) > 0 {
			artistName = release.ArtistCredit[0].Name
		}

		releases = append(releases, &ReleaseInfo{
			ID:         release.ID,
			Title:      release.Title,
			ArtistID:   "",
			ArtistName: artistName,
			Date:       release.Date,
		})
	}

	return releases, nil
}

// GetReleaseCoverArt gets cover art for a release using Cover Art Archive
func (mb *MusicBrainzClient) GetReleaseCoverArt(releaseID string) (*CoverArtInfo, error) {
	if mb.httpClient == nil {
		return nil, fmt.Errorf("MusicBrainz client not initialized")
	}

	// Wait for rate limiter to allow this request
	mb.rateLimiter.WaitForToken()

	// Cover Art Archive API URL
	coverArtURL := fmt.Sprintf("https://coverartarchive.org/release/%s", releaseID)

	resp, err := mb.httpClient.Get(coverArtURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get cover art: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("no cover art found for release: %s", releaseID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read cover art response: %v", err)
	}

	var coverArtResponse CoverArtResponse
	if err := json.Unmarshal(body, &coverArtResponse); err != nil {
		return nil, fmt.Errorf("failed to parse cover art response: %v", err)
	}

	// Find front cover art
	var frontCover *CoverArtImage
	for _, image := range coverArtResponse.Images {
		if image.Front {
			frontCover = &image
			break
		}
	}

	// If no front cover found, use the first image
	if frontCover == nil && len(coverArtResponse.Images) > 0 {
		frontCover = &coverArtResponse.Images[0]
	}

	if frontCover == nil {
		return nil, fmt.Errorf("no front cover art found")
	}

	coverArtInfo := &CoverArtInfo{
		LargeURL: frontCover.Image,
	}

	// Use thumbnails if available
	if frontCover.Thumbnails != nil {
		if small, exists := frontCover.Thumbnails["small"]; exists {
			coverArtInfo.SmallURL = small
		}
		if large, exists := frontCover.Thumbnails["large"]; exists {
			coverArtInfo.LargeURL = large
			coverArtInfo.MediumURL = large // Use large as medium if no medium available
		}
		if medium, exists := frontCover.Thumbnails["250"]; exists {
			coverArtInfo.MediumURL = medium
		}
		if small, exists := frontCover.Thumbnails["500"]; exists {
			coverArtInfo.MediumURL = small
		}
	}

	return coverArtInfo, nil
}

// GetAlbumCoverArt searches for an album and returns its cover art URL
func (mb *MusicBrainzClient) GetAlbumCoverArt(albumName, artistName string) (string, error) {
	releases, err := mb.SearchReleases(albumName, artistName, 10)
	if err != nil {
		return "", fmt.Errorf("failed to search for album: %v", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("album not found: %s by %s", albumName, artistName)
	}

	// Find the best matching release
	normalizedAlbumName := strings.ToLower(strings.TrimSpace(albumName))
	normalizedArtistName := strings.ToLower(strings.TrimSpace(artistName))
	var bestMatch *ReleaseInfo
	bestScore := 0.0

	for _, release := range releases {
		score := 0.0

		// Check album name match
		normalizedSearchAlbum := strings.ToLower(strings.TrimSpace(release.Title))
		if normalizedSearchAlbum == normalizedAlbumName {
			score += 0.5
		} else if strings.Contains(normalizedSearchAlbum, normalizedAlbumName) || strings.Contains(normalizedAlbumName, normalizedSearchAlbum) {
			score += 0.3
		}

		// Check artist name match
		normalizedReleaseArtist := strings.ToLower(strings.TrimSpace(release.ArtistName))
		if normalizedReleaseArtist == normalizedArtistName {
			score += 0.5
		} else if strings.Contains(normalizedReleaseArtist, normalizedArtistName) || strings.Contains(normalizedArtistName, normalizedReleaseArtist) {
			score += 0.3
		}

		if score > bestScore {
			bestScore = score
			bestMatch = release
		}
	}

	// If no good match found, use first result
	if bestMatch == nil {
		bestMatch = releases[0]
	}

	// Get cover art for the best match
	coverArtInfo, err := mb.GetReleaseCoverArt(bestMatch.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get cover art for album: %v", err)
	}

	return coverArtInfo.LargeURL, nil
}

// GetAlbumCoverArtSizes searches for an album and returns its cover art URLs in different sizes
func (mb *MusicBrainzClient) GetAlbumCoverArtSizes(albumName, artistName string) (*CoverArtInfo, error) {
	// Try multiple search strategies
	searchStrategies := []func(string, string) ([]*ReleaseInfo, error){
		func(album, artist string) ([]*ReleaseInfo, error) {
			return mb.SearchReleases(album, artist, 10)
		},
		// Try without "Ep" suffix
		func(album, artist string) ([]*ReleaseInfo, error) {
			cleanAlbum := strings.Replace(strings.ToLower(album), " ep", "", -1)
			cleanAlbum = strings.Replace(cleanAlbum, "vip", "", -1)
			cleanAlbum = strings.TrimSpace(cleanAlbum)
			return mb.SearchReleases(cleanAlbum, artist, 10)
		},
		// Try album-only search
		func(album, artist string) ([]*ReleaseInfo, error) {
			query := url.QueryEscape(fmt.Sprintf(`release:"%s"`, album))
			searchURL := fmt.Sprintf("%s/release/?query=%s&limit=10&fmt=json", mb.baseURL, query)

			mb.rateLimiter.WaitForToken()
			resp, err := mb.httpClient.Get(searchURL)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("MusicBrainz API error: %s", resp.Status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			var searchResponse MusicBrainzSearchResponse
			if err := json.Unmarshal(body, &searchResponse); err != nil {
				return nil, err
			}

			var releases []*ReleaseInfo
			for _, release := range searchResponse.Releases {
				artistName := ""
				if len(release.ArtistCredit) > 0 {
					artistName = release.ArtistCredit[0].Name
				}
				releases = append(releases, &ReleaseInfo{
					ID:         release.ID,
					Title:      release.Title,
					ArtistID:   "",
					ArtistName: artistName,
					Date:       release.Date,
				})
			}
			return releases, nil
		},
	}

	var releases []*ReleaseInfo
	var lastError error

	// Try each search strategy
	for _, strategy := range searchStrategies {
		releases, lastError = strategy(albumName, artistName)
		if lastError == nil && len(releases) > 0 {
			break
		}
	}

	if lastError != nil {
		return nil, fmt.Errorf("all search strategies failed: %v", lastError)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("album not found: %s by %s", albumName, artistName)
	}

	// Find the best matching release with improved scoring
	normalizedAlbumName := strings.ToLower(strings.TrimSpace(albumName))
	normalizedArtistName := strings.ToLower(strings.TrimSpace(artistName))

	// Clean album name for comparison (remove ep, vip, etc.)
	cleanAlbumName := strings.ToLower(strings.TrimSpace(albumName))
	cleanAlbumName = strings.Replace(cleanAlbumName, " ep", "", -1)
	cleanAlbumName = strings.Replace(cleanAlbumName, "vip", "", -1)
	cleanAlbumName = strings.Replace(cleanAlbumName, "lp", "", -1)
	cleanAlbumName = strings.TrimSpace(cleanAlbumName)

	var bestMatch *ReleaseInfo
	bestScore := 0.0

	for _, release := range releases {
		score := 0.0

		// Check album name match
		normalizedSearchAlbum := strings.ToLower(strings.TrimSpace(release.Title))
		cleanSearchAlbum := strings.Replace(normalizedSearchAlbum, " ep", "", -1)
		cleanSearchAlbum = strings.Replace(cleanSearchAlbum, "vip", "", -1)
		cleanSearchAlbum = strings.Replace(cleanSearchAlbum, "lp", "", -1)
		cleanSearchAlbum = strings.TrimSpace(cleanSearchAlbum)

		if normalizedSearchAlbum == normalizedAlbumName || cleanSearchAlbum == cleanAlbumName {
			score += 1.0 // Exact match gets highest score
		} else if strings.Contains(normalizedSearchAlbum, normalizedAlbumName) || strings.Contains(normalizedAlbumName, normalizedSearchAlbum) {
			score += 0.7
		} else if strings.Contains(cleanSearchAlbum, cleanAlbumName) || strings.Contains(cleanAlbumName, cleanSearchAlbum) {
			score += 0.6
		} else if strings.Contains(normalizedSearchAlbum, cleanAlbumName) || strings.Contains(cleanAlbumName, normalizedSearchAlbum) {
			score += 0.4
		}

		// Check artist name match
		normalizedReleaseArtist := strings.ToLower(strings.TrimSpace(release.ArtistName))

		// Handle "feat" variations
		artistWithoutFeat := strings.Split(normalizedArtistName, " feat")[0]
		artistWithoutFeat = strings.Split(artistWithoutFeat, " feat.")[0]
		artistWithoutFeat = strings.TrimSpace(artistWithoutFeat)

		releaseArtistWithoutFeat := strings.Split(normalizedReleaseArtist, " feat")[0]
		releaseArtistWithoutFeat = strings.Split(releaseArtistWithoutFeat, " feat.")[0]
		releaseArtistWithoutFeat = strings.TrimSpace(releaseArtistWithoutFeat)

		if normalizedReleaseArtist == normalizedArtistName {
			score += 1.0
		} else if releaseArtistWithoutFeat == artistWithoutFeat {
			score += 0.8
		} else if strings.Contains(normalizedReleaseArtist, normalizedArtistName) || strings.Contains(normalizedArtistName, normalizedReleaseArtist) {
			score += 0.6
		} else if strings.Contains(releaseArtistWithoutFeat, artistWithoutFeat) || strings.Contains(artistWithoutFeat, releaseArtistWithoutFeat) {
			score += 0.4
		}

		// Bonus for recent releases or complete data
		if release.Date != "" {
			score += 0.1
		}

		if score > bestScore {
			bestScore = score
			bestMatch = release
		}
	}

	// Only proceed if we have a reasonable match (score > 0.5)
	if bestScore < 0.5 && len(releases) > 1 {
		return nil, fmt.Errorf("no good match found for album: %s by %s (best score: %.2f)", albumName, artistName, bestScore)
	}

	// If no good match found, use first result
	if bestMatch == nil {
		bestMatch = releases[0]
	}

	// Get cover art for the best match
	coverArtInfo, err := mb.GetReleaseCoverArt(bestMatch.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cover art for album: %v", err)
	}

	return coverArtInfo, nil
}

// TestConnection tests the MusicBrainz connection
func (mb *MusicBrainzClient) TestConnection() error {
	if mb.httpClient == nil {
		return fmt.Errorf("MusicBrainz client not initialized")
	}

	// Wait for rate limiter to allow this request
	mb.rateLimiter.WaitForToken()

	// Try to get a simple search to test connection
	query := url.QueryEscape("test")
	searchURL := fmt.Sprintf("%s/artist/?query=%s&limit=1&fmt=json", mb.baseURL, query)

	resp, err := mb.httpClient.Get(searchURL)
	if err != nil {
		return fmt.Errorf("MusicBrainz connection test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MusicBrainz connection test failed: %s", resp.Status)
	}

	log.Println("MusicBrainz connection test successful")
	return nil
}

// SetUserAgent sets a custom user agent for MusicBrainz API requests
// MusicBrainz requires a proper User-Agent header for all requests
func (mb *MusicBrainzClient) SetUserAgent(appName, version, contact string) {
	userAgent := fmt.Sprintf("%s/%s ( %s )", appName, version, contact)
	mb.httpClient.Transport = &userAgentTransport{
		userAgent: userAgent,
		base:      http.DefaultTransport,
	}
}

// userAgentTransport adds User-Agent header to requests
type userAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(req)
}
