package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"music-server/database"
)

const applePodcastChartsURL = "https://rss.marketingtools.apple.com/api/v2/%s/podcasts/top/25/podcasts.json"

var podcastCountryPattern = regexp.MustCompile(`^[a-z]{2}$`)

type podcastHomeResponse struct {
	ContinueListening []database.PodcastProgress     `json:"continue_listening"`
	TopPodcasts       []podcastSearchResult          `json:"top_podcasts"`
	Subscriptions     []database.PodcastSubscription `json:"subscriptions"`
}

type applePodcastChart struct {
	Feed struct {
		Results []struct {
			ArtistName            string `json:"artistName"`
			ID                    string `json:"id"`
			Name                  string `json:"name"`
			ArtworkURL100         string `json:"artworkUrl100"`
			URL                   string `json:"url"`
			ContentAdvisoryRating string `json:"contentAdvisoryRating"`
		} `json:"results"`
	} `json:"feed"`
}

type podcastChartCacheEntry struct {
	results   []podcastSearchResult
	expiresAt time.Time
}

var podcastChartCache = struct {
	sync.RWMutex
	entries map[string]podcastChartCacheEntry
}{entries: make(map[string]podcastChartCacheEntry)}

func (r *Router) getPodcastHome(w http.ResponseWriter, req *http.Request) {
	country := strings.ToLower(strings.TrimSpace(req.URL.Query().Get("country")))
	if country == "" {
		country = "us"
	}
	if !podcastCountryPattern.MatchString(country) {
		writeJSONError(w, http.StatusBadRequest, "Invalid country code")
		return
	}

	continueListening, err := r.db.GetContinueListeningPodcasts(requestUserID(req), 12)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	subscriptions, err := r.db.ListPodcastSubscriptions(requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	topPodcasts, err := requestApplePodcastChart(country)
	if err != nil {
		// Resume data remains useful when the external chart service is unavailable.
		topPodcasts = []podcastSearchResult{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": podcastHomeResponse{
			ContinueListening: continueListening,
			TopPodcasts:       topPodcasts,
			Subscriptions:     subscriptions,
		},
	})
}

func (r *Router) updatePodcastProgress(w http.ResponseWriter, req *http.Request) {
	var progress database.PodcastProgress
	decoder := json.NewDecoder(io.LimitReader(req.Body, 256<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&progress); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid podcast progress")
		return
	}
	progress.UserID = requestUserID(req)
	progress.PodcastID = strings.TrimSpace(progress.PodcastID)
	progress.EpisodeID = strings.TrimSpace(progress.EpisodeID)
	progress.PodcastTitle = strings.TrimSpace(progress.PodcastTitle)
	progress.EpisodeTitle = strings.TrimSpace(progress.EpisodeTitle)
	progress.AudioURL = strings.TrimSpace(progress.AudioURL)
	if progress.PodcastID == "" || progress.EpisodeID == "" || progress.PodcastTitle == "" || progress.EpisodeTitle == "" || progress.AudioURL == "" {
		writeJSONError(w, http.StatusBadRequest, "Podcast and episode details are required")
		return
	}
	if progress.DurationSeconds < 0 || progress.PositionSeconds < 0 {
		writeJSONError(w, http.StatusBadRequest, "Podcast progress cannot be negative")
		return
	}
	if progress.DurationSeconds > 0 {
		if progress.PositionSeconds > progress.DurationSeconds {
			progress.PositionSeconds = progress.DurationSeconds
		}
		progress.Completed = podcastPlaybackCompleted(progress.PositionSeconds, progress.DurationSeconds)
	}
	saved, err := r.db.SavePodcastProgress(progress)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": saved})
}

func podcastPlaybackCompleted(positionSeconds, durationSeconds int) bool {
	if durationSeconds <= 0 || positionSeconds <= 0 {
		return false
	}
	remaining := durationSeconds - positionSeconds
	return remaining <= 30 || float64(positionSeconds)/float64(durationSeconds) >= 0.95
}

func requestApplePodcastChart(country string) ([]podcastSearchResult, error) {
	podcastChartCache.RLock()
	cached, ok := podcastChartCache.entries[country]
	podcastChartCache.RUnlock()
	if ok && time.Now().Before(cached.expiresAt) {
		return append([]podcastSearchResult(nil), cached.results...), nil
	}

	var chart applePodcastChart
	if err := requestAppleJSON(fmt.Sprintf(applePodcastChartsURL, country), &chart); err != nil {
		if ok {
			return append([]podcastSearchResult(nil), cached.results...), nil
		}
		return nil, err
	}
	results := make([]podcastSearchResult, 0, len(chart.Feed.Results))
	for _, item := range chart.Feed.Results {
		if _, err := strconv.ParseInt(item.ID, 10, 64); err != nil {
			continue
		}
		imageURL := strings.Replace(item.ArtworkURL100, "/100x100bb.", "/600x600bb.", 1)
		results = append(results, podcastSearchResult{
			ID:           item.ID,
			Title:        strings.TrimSpace(item.Name),
			Publisher:    strings.TrimSpace(item.ArtistName),
			ImageURL:     imageURL,
			ThumbnailURL: item.ArtworkURL100,
			WebsiteURL:   item.URL,
			Explicit:     strings.EqualFold(item.ContentAdvisoryRating, "explicit") || strings.EqualFold(item.ContentAdvisoryRating, "explict"),
		})
	}
	podcastChartCache.Lock()
	podcastChartCache.entries[country] = podcastChartCacheEntry{results: results, expiresAt: time.Now().Add(time.Hour)}
	podcastChartCache.Unlock()
	return append([]podcastSearchResult(nil), results...), nil
}
