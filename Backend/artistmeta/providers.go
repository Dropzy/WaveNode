package artistmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultUserAgent = "WaveNode/0.1 (self-hosted open-source music server; https://github.com/)"

type Provider interface {
	Name() string
}

type httpJSONClient struct {
	client    *http.Client
	cache     APIResponseCache
	userAgent string
	ttl       time.Duration
	limiter   *providerLimiter
}

type providerLimiter struct {
	mu       sync.Mutex
	lastCall time.Time
	interval time.Duration
}

func newHTTPJSONClient(cache APIResponseCache, interval time.Duration) httpJSONClient {
	return httpJSONClient{
		client:    &http.Client{Timeout: 20 * time.Second},
		cache:     cache,
		userAgent: defaultUserAgent,
		ttl:       7 * 24 * time.Hour,
		limiter:   &providerLimiter{interval: interval},
	}
}

func (c httpJSONClient) getJSON(ctx context.Context, provider, rawURL string, target interface{}) error {
	key := provider + ":" + rawURL
	if c.cache != nil {
		if payload, ok, err := c.cache.GetSourceAPICache(key); err == nil && ok {
			return json.Unmarshal(payload, target)
		}
	}

	if c.limiter != nil {
		c.limiter.wait()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned %s", provider, resp.Status)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return err
	}
	if c.cache != nil {
		_ = c.cache.SetSourceAPICache(key, provider, payload, time.Now().Add(c.ttl))
	}
	return nil
}

func (l *providerLimiter) wait() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interval <= 0 {
		return
	}
	now := time.Now()
	next := l.lastCall.Add(l.interval)
	if now.Before(next) {
		time.Sleep(next.Sub(now))
	}
	l.lastCall = time.Now()
}

type MusicBrainzProvider struct {
	http    httpJSONClient
	BaseURL string
}

func NewMusicBrainzProvider(cache APIResponseCache) *MusicBrainzProvider {
	return &MusicBrainzProvider{
		http:    newHTTPJSONClient(cache, 1100*time.Millisecond),
		BaseURL: "https://musicbrainz.org/ws/2",
	}
}

func (p *MusicBrainzProvider) Name() string { return "musicbrainz" }

type mbSearchResponse struct {
	Artists []mbArtist `json:"artists"`
}

type mbArtist struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	SortName       string       `json:"sort-name"`
	Type           string       `json:"type"`
	Country        string       `json:"country"`
	Disambiguation string       `json:"disambiguation"`
	Score          int          `json:"score"`
	Tags           []mbTag      `json:"tags"`
	Relations      []mbRelation `json:"relations"`
}

type mbTag struct {
	Name string `json:"name"`
}

type mbRelation struct {
	Type string `json:"type"`
	URL  struct {
		Resource string `json:"resource"`
	} `json:"url"`
}

func (p *MusicBrainzProvider) BestArtistMatch(ctx context.Context, name string) (*ArtistMatch, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("artist name is required")
	}
	searchURL := fmt.Sprintf("%s/artist?query=%s&limit=10&fmt=json", p.BaseURL, url.QueryEscape(fmt.Sprintf(`artist:"%s"`, name)))
	var response mbSearchResponse
	if err := p.http.getJSON(ctx, p.Name(), searchURL, &response); err != nil {
		return nil, err
	}
	if len(response.Artists) == 0 {
		return nil, fmt.Errorf("artist not found")
	}

	sort.SliceStable(response.Artists, func(i, j int) bool {
		return scoreMusicBrainzArtist(name, response.Artists[i]) > scoreMusicBrainzArtist(name, response.Artists[j])
	})
	best := response.Artists[0]
	match := mbArtistToMatch(name, best)

	lookupURL := fmt.Sprintf("%s/artist/%s?inc=url-rels+tags&fmt=json", p.BaseURL, url.PathEscape(best.ID))
	var detailed mbArtist
	if err := p.http.getJSON(ctx, p.Name(), lookupURL, &detailed); err == nil {
		detailed.Score = best.Score
		match = mbArtistToMatch(name, detailed)
	}
	return &match, nil
}

func mbArtistToMatch(query string, artist mbArtist) ArtistMatch {
	tags := make([]string, 0, len(artist.Tags))
	for _, tag := range artist.Tags {
		if tag.Name != "" {
			tags = append(tags, tag.Name)
		}
	}
	match := ArtistMatch{
		MBID:            artist.ID,
		Name:            artist.Name,
		SortName:        artist.SortName,
		Type:            artist.Type,
		Country:         artist.Country,
		Disambiguation:  artist.Disambiguation,
		Tags:            tags,
		ConfidenceScore: scoreMusicBrainzArtist(query, artist),
	}
	for _, rel := range artist.Relations {
		resource := strings.TrimSpace(rel.URL.Resource)
		if resource == "" {
			continue
		}
		if strings.Contains(resource, "wikidata.org/wiki/") {
			match.WikidataID = path.Base(resource)
		}
		if strings.Contains(resource, "commons.wikimedia.org/wiki/Category:") {
			match.CommonsCategory = strings.TrimPrefix(path.Base(resource), "Category:")
		}
	}
	return match
}

func scoreMusicBrainzArtist(query string, artist mbArtist) float64 {
	query = normalizeName(query)
	name := normalizeName(artist.Name)
	score := float64(artist.Score) / 100
	if name == query {
		score += 0.25
	} else if strings.Contains(name, query) || strings.Contains(query, name) {
		score += 0.08
	}
	if artist.Disambiguation != "" {
		score += 0.02
	}
	return math.Min(score, 1)
}

func normalizeName(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

type WikidataProvider struct {
	http    httpJSONClient
	BaseURL string
}

func NewWikidataProvider(cache APIResponseCache) *WikidataProvider {
	return &WikidataProvider{
		http:    newHTTPJSONClient(cache, 250*time.Millisecond),
		BaseURL: "https://www.wikidata.org/wiki/Special:EntityData",
	}
}

func (p *WikidataProvider) Name() string { return "wikidata" }

type wikidataEntityResponse struct {
	Entities map[string]wikidataEntity `json:"entities"`
}

type wikidataEntity struct {
	Claims map[string][]wikidataClaim `json:"claims"`
}

type wikidataClaim struct {
	Mainsnak struct {
		Datavalue struct {
			Value interface{} `json:"value"`
		} `json:"datavalue"`
	} `json:"mainsnak"`
}

func (p *WikidataProvider) CommonsFileForEntity(ctx context.Context, entityID string) (string, error) {
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return "", fmt.Errorf("wikidata entity is required")
	}
	var response wikidataEntityResponse
	if err := p.http.getJSON(ctx, p.Name(), fmt.Sprintf("%s/%s.json", p.BaseURL, url.PathEscape(entityID)), &response); err != nil {
		return "", err
	}
	entity, ok := response.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("wikidata entity not found")
	}
	claims := entity.Claims["P18"]
	if len(claims) == 0 {
		return "", fmt.Errorf("wikidata image claim not found")
	}
	file, ok := claims[0].Mainsnak.Datavalue.Value.(string)
	if !ok || strings.TrimSpace(file) == "" {
		return "", fmt.Errorf("wikidata image claim is invalid")
	}
	return strings.TrimSpace(file), nil
}

type WikimediaCommonsProvider struct {
	http   httpJSONClient
	APIURL string
}

func NewWikimediaCommonsProvider(cache APIResponseCache) *WikimediaCommonsProvider {
	return &WikimediaCommonsProvider{
		http:   newHTTPJSONClient(cache, 250*time.Millisecond),
		APIURL: "https://commons.wikimedia.org/w/api.php",
	}
}

func (p *WikimediaCommonsProvider) Name() string { return "wikimedia_commons" }

type commonsImageInfoResponse struct {
	Query struct {
		Pages map[string]struct {
			Title     string `json:"title"`
			ImageInfo []struct {
				URL         string `json:"url"`
				Description string `json:"descriptionurl"`
				Mime        string `json:"mime"`
				Width       int    `json:"width"`
				Height      int    `json:"height"`
				ExtMetadata map[string]struct {
					Value string `json:"value"`
				} `json:"extmetadata"`
				ThumbURL string `json:"thumburl"`
			} `json:"imageinfo"`
		} `json:"pages"`
	} `json:"query"`
}

func (p *WikimediaCommonsProvider) ImageCandidateForFile(ctx context.Context, fileName string, confidence float64) (*ImageCandidate, error) {
	fileName = strings.TrimSpace(strings.TrimPrefix(fileName, "File:"))
	if fileName == "" {
		return nil, fmt.Errorf("commons file is required")
	}
	params := url.Values{}
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("prop", "imageinfo")
	params.Set("titles", "File:"+fileName)
	params.Set("iiprop", "url|mime|size|extmetadata")
	params.Set("iiurlwidth", "512")

	var response commonsImageInfoResponse
	if err := p.http.getJSON(ctx, p.Name(), p.APIURL+"?"+params.Encode(), &response); err != nil {
		return nil, err
	}
	for _, page := range response.Query.Pages {
		if len(page.ImageInfo) == 0 {
			continue
		}
		info := page.ImageInfo[0]
		license := cleanHTMLish(info.ExtMetadata["LicenseShortName"].Value)
		if license == "" {
			license = cleanHTMLish(info.ExtMetadata["License"].Value)
		}
		if !IsReusableLicense(license) {
			return nil, fmt.Errorf("commons image license is not reusable: %s", license)
		}
		author := cleanHTMLish(info.ExtMetadata["Artist"].Value)
		sourcePage := info.Description
		if sourcePage == "" {
			sourcePage = "https://commons.wikimedia.org/wiki/" + strings.ReplaceAll(url.PathEscape(page.Title), "%3A", ":")
		}
		candidate := &ImageCandidate{
			Source:          p.Name(),
			ImageURL:        info.URL,
			ThumbnailURL:    info.ThumbURL,
			SourcePageURL:   sourcePage,
			LicenseName:     license,
			LicenseURL:      cleanHTMLish(info.ExtMetadata["LicenseUrl"].Value),
			AuthorName:      author,
			Width:           info.Width,
			Height:          info.Height,
			MimeType:        info.Mime,
			ConfidenceScore: confidence,
		}
		candidate.AttributionText = BuildAttribution(candidate.AuthorName, candidate.LicenseName, candidate.SourcePageURL)
		return candidate, nil
	}
	return nil, fmt.Errorf("commons image not found")
}

func cleanHTMLish(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "&quot;", `"`)
	value = strings.ReplaceAll(value, "&amp;", "&")
	for {
		start := strings.Index(value, "<")
		end := strings.Index(value, ">")
		if start < 0 || end < start {
			break
		}
		value = strings.TrimSpace(value[:start] + " " + value[end+1:])
	}
	return strings.Join(strings.Fields(value), " ")
}

type FanartTvProvider struct {
	APIKey string
}

func (p *FanartTvProvider) Name() string { return "fanarttv" }

func (p *FanartTvProvider) Enabled() bool {
	return strings.TrimSpace(p.APIKey) != ""
}

type UploadedImageProvider struct{}

func (p UploadedImageProvider) Name() string { return "uploaded" }
