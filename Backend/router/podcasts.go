package router

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const (
	applePodcastSearchURL = "https://itunes.apple.com/search"
	applePodcastLookupURL = "https://itunes.apple.com/lookup"
)

var podcastHTMLTagPattern = regexp.MustCompile(`<[^>]+>`)

type podcastSearchResult struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Publisher     string `json:"publisher"`
	Description   string `json:"description"`
	ImageURL      string `json:"image_url,omitempty"`
	ThumbnailURL  string `json:"thumbnail_url,omitempty"`
	WebsiteURL    string `json:"website_url,omitempty"`
	FeedURL       string `json:"feed_url,omitempty"`
	TotalEpisodes int    `json:"total_episodes,omitempty"`
	Explicit      bool   `json:"explicit"`
}

type podcastSearchResponse struct {
	Query      string                `json:"query"`
	Total      int                   `json:"total"`
	Count      int                   `json:"count"`
	NextOffset int                   `json:"next_offset"`
	Results    []podcastSearchResult `json:"results"`
}

type podcastEpisode struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AudioURL    string `json:"audio_url"`
	WebsiteURL  string `json:"website_url,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	Duration    int    `json:"duration"`
	Explicit    bool   `json:"explicit"`
	Progress    int    `json:"progress_seconds"`
	Completed   bool   `json:"completed"`
}

type podcastEpisodesResponse struct {
	Podcast  podcastSearchResult `json:"podcast"`
	Count    int                 `json:"count"`
	Episodes []podcastEpisode    `json:"episodes"`
}

type applePodcastResult struct {
	CollectionID           int64  `json:"collectionId"`
	CollectionName         string `json:"collectionName"`
	ArtistName             string `json:"artistName"`
	CollectionViewURL      string `json:"collectionViewUrl"`
	FeedURL                string `json:"feedUrl"`
	ArtworkURL100          string `json:"artworkUrl100"`
	ArtworkURL600          string `json:"artworkUrl600"`
	TrackCount             int    `json:"trackCount"`
	CollectionExplicitness string `json:"collectionExplicitness"`
}

type applePodcastResponse struct {
	ResultCount int                  `json:"resultCount"`
	Results     []applePodcastResult `json:"results"`
}

type podcastRSS struct {
	Channel struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Link        string `xml:"link"`
		Images      []struct {
			Href string `xml:"href,attr"`
			URL  string `xml:"url"`
		} `xml:"image"`
		Items []struct {
			GUID        string `xml:"guid"`
			Title       string `xml:"title"`
			Description string `xml:"description"`
			Encoded     string `xml:"encoded"`
			Link        string `xml:"link"`
			PubDate     string `xml:"pubDate"`
			Duration    string `xml:"duration"`
			Explicit    string `xml:"explicit"`
			Image       struct {
				Href string `xml:"href,attr"`
			} `xml:"image"`
			Enclosure struct {
				URL  string `xml:"url,attr"`
				Type string `xml:"type,attr"`
			} `xml:"enclosure"`
		} `xml:"item"`
	} `xml:"channel"`
}

func (r *Router) searchPodcasts(w http.ResponseWriter, req *http.Request) {
	query := strings.TrimSpace(req.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    podcastSearchResponse{Query: query, Results: []podcastSearchResult{}},
		})
		return
	}
	if len(query) > 120 {
		writeJSONError(w, http.StatusBadRequest, "Podcast search is too long")
		return
	}

	pageSize := clampIntQuery(req, "page_size", 10, 1, 25)
	results, err := requestApplePodcastSearch(query, pageSize)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": results})
}

func (r *Router) getPodcastEpisodes(w http.ResponseWriter, req *http.Request) {
	id := strings.TrimSpace(mux.Vars(req)["id"])
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid podcast ID")
		return
	}

	podcast, err := requestApplePodcast(id)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	if podcast.FeedURL == "" {
		writeJSONError(w, http.StatusNotFound, "This podcast does not publish an RSS feed")
		return
	}

	limit := clampIntQuery(req, "limit", 50, 1, 100)
	episodes, feed, err := requestPodcastFeed(podcast.FeedURL, limit)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	if podcast.Description == "" {
		podcast.Description = feed.Description
	}
	if podcast.WebsiteURL == "" {
		podcast.WebsiteURL = feed.WebsiteURL
	}
	if podcast.ImageURL == "" {
		podcast.ImageURL = feed.ImageURL
	}
	progress, err := r.db.GetPodcastProgress(requestUserID(req), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for index := range episodes {
		if saved, ok := progress[episodes[index].ID]; ok {
			episodes[index].Progress = saved.PositionSeconds
			episodes[index].Completed = saved.Completed
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    podcastEpisodesResponse{Podcast: podcast, Count: len(episodes), Episodes: episodes},
	})
}

func requestApplePodcastSearch(query string, pageSize int) (podcastSearchResponse, error) {
	params := url.Values{}
	params.Set("term", query)
	params.Set("media", "podcast")
	params.Set("entity", "podcast")
	params.Set("limit", strconv.Itoa(pageSize))
	params.Set("explicit", "Yes")

	var parsed applePodcastResponse
	if err := requestAppleJSON(applePodcastSearchURL+"?"+params.Encode(), &parsed); err != nil {
		return podcastSearchResponse{}, err
	}

	results := make([]podcastSearchResult, 0, len(parsed.Results))
	for _, item := range parsed.Results {
		results = append(results, mapApplePodcast(item))
	}
	return podcastSearchResponse{Query: query, Total: parsed.ResultCount, Count: len(results), Results: results}, nil
}

func requestApplePodcast(id string) (podcastSearchResult, error) {
	params := url.Values{}
	params.Set("id", id)
	params.Set("entity", "podcast")

	var parsed applePodcastResponse
	if err := requestAppleJSON(applePodcastLookupURL+"?"+params.Encode(), &parsed); err != nil {
		return podcastSearchResult{}, err
	}
	if len(parsed.Results) == 0 {
		return podcastSearchResult{}, fmt.Errorf("Podcast was not found")
	}
	return mapApplePodcast(parsed.Results[0]), nil
}

func requestAppleJSON(endpoint string, destination interface{}) error {
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "WaveNode/1.0")

	resp, err := (&http.Client{Timeout: 12 * time.Second}).Do(request)
	if err != nil {
		return fmt.Errorf("Apple podcast request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Apple podcast request returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(destination); err != nil {
		return fmt.Errorf("Apple podcast response could not be read")
	}
	return nil
}

func mapApplePodcast(item applePodcastResult) podcastSearchResult {
	return podcastSearchResult{
		ID:            strconv.FormatInt(item.CollectionID, 10),
		Title:         strings.TrimSpace(item.CollectionName),
		Publisher:     strings.TrimSpace(item.ArtistName),
		ImageURL:      item.ArtworkURL600,
		ThumbnailURL:  item.ArtworkURL100,
		WebsiteURL:    item.CollectionViewURL,
		FeedURL:       item.FeedURL,
		TotalEpisodes: item.TrackCount,
		Explicit:      strings.EqualFold(item.CollectionExplicitness, "explicit"),
	}
}

type parsedPodcastFeed struct {
	Description string
	WebsiteURL  string
	ImageURL    string
}

func requestPodcastFeed(feedURL string, limit int) ([]podcastEpisode, parsedPodcastFeed, error) {
	parsedURL, err := url.Parse(feedURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, parsedPodcastFeed{}, fmt.Errorf("Podcast feed URL is invalid")
	}

	request, err := http.NewRequest(http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, parsedPodcastFeed{}, err
	}
	request.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	request.Header.Set("User-Agent", "WaveNode/1.0")

	client := http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many podcast feed redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("unsupported podcast feed redirect")
			}
			return nil
		},
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, parsedPodcastFeed{}, fmt.Errorf("Podcast feed could not be loaded: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parsedPodcastFeed{}, fmt.Errorf("Podcast feed returned status %d", resp.StatusCode)
	}

	return parsePodcastRSS(io.LimitReader(resp.Body, 8<<20), limit)
}

func parsePodcastRSS(reader io.Reader, limit int) ([]podcastEpisode, parsedPodcastFeed, error) {
	var feed podcastRSS
	if err := xml.NewDecoder(reader).Decode(&feed); err != nil {
		return nil, parsedPodcastFeed{}, fmt.Errorf("Podcast feed could not be read")
	}

	feedImage := ""
	for _, image := range feed.Channel.Images {
		feedImage = podcastFirstNonEmpty(image.Href, image.URL)
		if feedImage != "" {
			break
		}
	}
	episodes := make([]podcastEpisode, 0, min(limit, len(feed.Channel.Items)))
	for _, item := range feed.Channel.Items {
		if len(episodes) >= limit {
			break
		}
		audioURL := strings.TrimSpace(item.Enclosure.URL)
		parsedAudioURL, err := url.Parse(audioURL)
		if err != nil || (parsedAudioURL.Scheme != "http" && parsedAudioURL.Scheme != "https") {
			continue
		}
		if !podcastEnclosureIsAudio(item.Enclosure.Type, parsedAudioURL.Path) {
			continue
		}

		description := podcastPlainText(podcastFirstNonEmpty(item.Description, item.Encoded))
		identifier := podcastFirstNonEmpty(item.GUID, audioURL)
		episodes = append(episodes, podcastEpisode{
			ID:          podcastEpisodeID(identifier),
			Title:       podcastPlainText(item.Title),
			Description: description,
			AudioURL:    audioURL,
			WebsiteURL:  strings.TrimSpace(item.Link),
			ImageURL:    podcastFirstNonEmpty(item.Image.Href, feedImage),
			PublishedAt: parsePodcastDate(item.PubDate),
			Duration:    parsePodcastDuration(item.Duration),
			Explicit:    podcastExplicit(item.Explicit),
		})
	}

	return episodes, parsedPodcastFeed{
		Description: podcastPlainText(feed.Channel.Description),
		WebsiteURL:  strings.TrimSpace(feed.Channel.Link),
		ImageURL:    feedImage,
	}, nil
}

func podcastEpisodeID(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:12])
}

func parsePodcastDuration(value string) int {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) == 0 || len(parts) > 3 {
		return 0
	}
	total := 0
	for _, part := range parts {
		number, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || number < 0 {
			return 0
		}
		total = total*60 + number
	}
	return total
}

func parsePodcastDate(value string) string {
	for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, time.RFC3339} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}
	return ""
}

func podcastExplicit(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "yes" || value == "true" || value == "explicit"
}

func podcastEnclosureIsAudio(value, path string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	if strings.HasPrefix(mediaType, "video/") {
		return false
	}
	if strings.HasPrefix(mediaType, "audio/") {
		return true
	}
	switch strings.ToLower(path[strings.LastIndex(path, ".")+1:]) {
	case "mp4", "m4v", "webm", "ogv", "mov":
		return false
	default:
		return true
	}
}

func podcastPlainText(value string) string {
	value = podcastHTMLTagPattern.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(html.UnescapeString(value)), " ")
}

func podcastFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func clampIntQuery(req *http.Request, key string, fallback, minValue, maxValue int) int {
	value, err := strconv.Atoi(req.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
