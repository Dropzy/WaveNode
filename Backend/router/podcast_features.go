package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"music-server/database"
)

func (r *Router) listPodcastSubscriptions(w http.ResponseWriter, req *http.Request) {
	items, err := r.db.ListPodcastSubscriptions(requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": items})
}

func (r *Router) savePodcastSubscription(w http.ResponseWriter, req *http.Request) {
	var item database.PodcastSubscription
	decoder := json.NewDecoder(io.LimitReader(req.Body, 256<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid podcast subscription")
		return
	}
	item.UserID = requestUserID(req)
	item.PodcastID = strings.TrimSpace(item.PodcastID)
	item.Title = strings.TrimSpace(item.Title)
	item.PlaybackSpeed = clampPodcastSpeed(item.PlaybackSpeed)
	if item.PodcastID == "" || item.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "Podcast ID and title are required")
		return
	}
	for _, rawURL := range []string{item.ImageURL, item.ThumbnailURL, item.WebsiteURL, item.FeedURL} {
		if rawURL != "" && !validHTTPURL(rawURL) {
			writeJSONError(w, http.StatusBadRequest, "Podcast URLs must use HTTP or HTTPS")
			return
		}
	}
	saved, err := r.db.SavePodcastSubscription(item)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": saved})
}

func (r *Router) deletePodcastSubscription(w http.ResponseWriter, req *http.Request) {
	podcastID := strings.TrimSpace(mux.Vars(req)["id"])
	if podcastID == "" {
		writeJSONError(w, http.StatusBadRequest, "Podcast ID is required")
		return
	}
	if err := r.db.DeletePodcastSubscription(requestUserID(req), podcastID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *Router) getPodcastPreferences(w http.ResponseWriter, req *http.Request) {
	preferences, err := r.db.GetPodcastPreferences(requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": preferences})
}

func (r *Router) updatePodcastPreferences(w http.ResponseWriter, req *http.Request) {
	var preferences database.PodcastPreferences
	decoder := json.NewDecoder(io.LimitReader(req.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&preferences); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid podcast preferences")
		return
	}
	preferences.UserID = requestUserID(req)
	preferences.DefaultPlaybackSpeed = clampPodcastSpeed(preferences.DefaultPlaybackSpeed)
	if preferences.SkipBackSeconds < 5 || preferences.SkipBackSeconds > 120 ||
		preferences.SkipForwardSeconds < 5 || preferences.SkipForwardSeconds > 300 {
		writeJSONError(w, http.StatusBadRequest, "Podcast skip intervals are out of range")
		return
	}
	saved, err := r.db.SavePodcastPreferences(preferences)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": saved})
}

func (r *Router) getPodcastQueue(w http.ResponseWriter, req *http.Request) {
	queue, err := r.db.GetPodcastQueue(requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": queue})
}

func (r *Router) updatePodcastQueue(w http.ResponseWriter, req *http.Request) {
	var queue database.PodcastQueue
	decoder := json.NewDecoder(io.LimitReader(req.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&queue); err != nil || !json.Valid(queue.Items) {
		writeJSONError(w, http.StatusBadRequest, "Invalid podcast queue")
		return
	}
	var items []json.RawMessage
	if err := json.Unmarshal(queue.Items, &items); err != nil || len(items) > 200 {
		writeJSONError(w, http.StatusBadRequest, "Podcast queue must contain at most 200 episodes")
		return
	}
	if queue.CurrentIndex < 0 || (len(items) > 0 && queue.CurrentIndex >= len(items)) || queue.PositionSeconds < 0 {
		writeJSONError(w, http.StatusBadRequest, "Podcast queue position is invalid")
		return
	}
	queue.UserID = requestUserID(req)
	saved, err := r.db.SavePodcastQueue(queue)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": saved})
}

type podcastChapter struct {
	StartTime float64 `json:"startTime"`
	Title     string  `json:"title"`
	ImageURL  string  `json:"img,omitempty"`
	URL       string  `json:"url,omitempty"`
}

type podcastChapterDocument struct {
	Version  string           `json:"version,omitempty"`
	Chapters []podcastChapter `json:"chapters"`
}

func (r *Router) getPodcastChapters(w http.ResponseWriter, req *http.Request) {
	chaptersURL := strings.TrimSpace(req.URL.Query().Get("url"))
	if !validHTTPURL(chaptersURL) {
		writeJSONError(w, http.StatusBadRequest, "Invalid podcast chapters URL")
		return
	}
	request, err := http.NewRequestWithContext(req.Context(), http.MethodGet, chaptersURL, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid podcast chapters URL")
		return
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "WaveNode/1.0")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "Podcast chapters could not be loaded")
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		writeJSONError(w, http.StatusBadGateway, "Podcast chapters returned an error")
		return
	}
	var document podcastChapterDocument
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&document); err != nil || len(document.Chapters) > 500 {
		writeJSONError(w, http.StatusBadGateway, "Podcast chapters are invalid")
		return
	}
	filtered := make([]podcastChapter, 0, len(document.Chapters))
	for _, chapter := range document.Chapters {
		chapter.Title = strings.TrimSpace(chapter.Title)
		if chapter.StartTime < 0 || chapter.Title == "" {
			continue
		}
		filtered = append(filtered, chapter)
	}
	document.Chapters = filtered
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": document})
}

func validHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func clampPodcastSpeed(value float64) float64 {
	if value < 0.5 || value > 3 {
		return 1
	}
	return value
}
