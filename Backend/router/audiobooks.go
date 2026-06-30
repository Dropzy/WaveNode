package router

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"music-server/database"

	"github.com/gorilla/mux"
)

var librivoxAudiobooksURL = "https://librivox.org/api/feed/audiobooks/"

var audiobookHTMLPattern = regexp.MustCompile(`<[^>]+>`)

type audiobookAuthor struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type librivoxSection struct {
	ID            string `json:"id"`
	SectionNumber string `json:"section_number"`
	Title         string `json:"title"`
	ListenURL     string `json:"listen_url"`
	Playtime      string `json:"playtime"`
	Readers       []struct {
		DisplayName string `json:"display_name"`
	} `json:"readers"`
}

type librivoxBook struct {
	ID            string            `json:"id"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Language      string            `json:"language"`
	CopyrightYear string            `json:"copyright_year"`
	NumSections   string            `json:"num_sections"`
	URLLibriVox   string            `json:"url_librivox"`
	URLArchive    string            `json:"url_iarchive"`
	TotalTimeSecs int               `json:"totaltimesecs"`
	Authors       []audiobookAuthor `json:"authors"`
	Sections      []librivoxSection `json:"sections"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

type librivoxResponse struct {
	Books []librivoxBook `json:"books"`
}

type audiobookSummary struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Description   string   `json:"description"`
	ImageURL      string   `json:"image_url"`
	WebsiteURL    string   `json:"website_url"`
	Language      string   `json:"language"`
	CopyrightYear string   `json:"copyright_year"`
	ChapterCount  int      `json:"chapter_count"`
	Duration      int      `json:"duration_seconds"`
	Genres        []string `json:"genres"`
}

type audiobookChapter struct {
	ID        string   `json:"id"`
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	AudioURL  string   `json:"audio_url"`
	Duration  int      `json:"duration_seconds"`
	Readers   []string `json:"readers"`
	Progress  int      `json:"progress_seconds"`
	Completed bool     `json:"completed"`
}

type audiobookDetail struct {
	Book     audiobookSummary   `json:"book"`
	Chapters []audiobookChapter `json:"chapters"`
}

type audiobookHome struct {
	ContinueListening []database.AudiobookProgress `json:"continue_listening"`
	Featured          []audiobookSummary           `json:"featured"`
}

type audiobookCacheEntry struct {
	Books     []librivoxBook
	ExpiresAt time.Time
}

var audiobookCache = struct {
	sync.RWMutex
	entries map[string]audiobookCacheEntry
}{entries: make(map[string]audiobookCacheEntry)}

func (r *Router) searchAudiobooks(w http.ResponseWriter, req *http.Request) {
	query := strings.TrimSpace(req.URL.Query().Get("q"))
	if len(query) > 120 {
		writeJSONError(w, http.StatusBadRequest, "Audiobook search is too long")
		return
	}
	limit := clampIntQuery(req, "limit", 20, 1, 40)
	books, err := requestLibriVox(query, "", limit, false)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	if query != "" {
		byAuthor, authorErr := requestLibriVoxFilter("author", query, "", limit, false)
		if authorErr == nil {
			seen := make(map[string]bool, len(books))
			for _, book := range books {
				seen[book.ID] = true
			}
			for _, book := range byAuthor {
				if !seen[book.ID] {
					books = append(books, book)
					seen[book.ID] = true
				}
			}
			if len(books) > limit {
				books = books[:limit]
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": mapAudiobookSummaries(books)})
}

func (r *Router) getAudiobookHome(w http.ResponseWriter, req *http.Request) {
	continued, err := r.db.GetContinueListeningAudiobooks(requestUserID(req), 12)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	books, err := requestLibriVox("", "", 24, false)
	if err != nil {
		books = []librivoxBook{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": audiobookHome{
		ContinueListening: continued,
		Featured:          mapAudiobookSummaries(books),
	}})
}

func (r *Router) getAudiobook(w http.ResponseWriter, req *http.Request) {
	id := strings.TrimSpace(mux.Vars(req)["id"])
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid audiobook ID")
		return
	}
	books, err := requestLibriVox("", id, 1, true)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	if len(books) == 0 {
		writeJSONError(w, http.StatusNotFound, "Audiobook was not found")
		return
	}
	book := books[0]
	progress, err := r.db.GetAudiobookProgress(requestUserID(req), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	chapters := make([]audiobookChapter, 0, len(book.Sections))
	for index, section := range book.Sections {
		number, _ := strconv.Atoi(section.SectionNumber)
		if number <= 0 {
			number = index + 1
		}
		duration, _ := strconv.Atoi(section.Playtime)
		readers := make([]string, 0, len(section.Readers))
		for _, reader := range section.Readers {
			if name := repairLibriVoxText(strings.TrimSpace(reader.DisplayName)); name != "" {
				readers = append(readers, name)
			}
		}
		chapter := audiobookChapter{ID: section.ID, Number: number, Title: repairLibriVoxText(strings.TrimSpace(section.Title)),
			AudioURL: section.ListenURL, Duration: duration, Readers: readers}
		if saved, ok := progress[section.ID]; ok {
			chapter.Progress = saved.PositionSeconds
			chapter.Completed = saved.Completed
		}
		chapters = append(chapters, chapter)
	}
	sort.SliceStable(chapters, func(i, j int) bool { return chapters[i].Number < chapters[j].Number })
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": audiobookDetail{
		Book: mapAudiobookSummary(book), Chapters: chapters,
	}})
}

func (r *Router) updateAudiobookProgress(w http.ResponseWriter, req *http.Request) {
	var item database.AudiobookProgress
	decoder := json.NewDecoder(io.LimitReader(req.Body, 256<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid audiobook progress")
		return
	}
	item.UserID = requestUserID(req)
	item.BookID = strings.TrimSpace(item.BookID)
	item.ChapterID = strings.TrimSpace(item.ChapterID)
	item.BookTitle = strings.TrimSpace(item.BookTitle)
	item.ChapterTitle = strings.TrimSpace(item.ChapterTitle)
	item.AudioURL = strings.TrimSpace(item.AudioURL)
	if item.BookID == "" || item.ChapterID == "" || item.BookTitle == "" || item.ChapterTitle == "" || item.AudioURL == "" {
		writeJSONError(w, http.StatusBadRequest, "Audiobook and chapter details are required")
		return
	}
	if item.PositionSeconds < 0 || item.DurationSeconds < 0 || item.ChapterNumber < 0 {
		writeJSONError(w, http.StatusBadRequest, "Audiobook progress cannot be negative")
		return
	}
	if item.DurationSeconds > 0 {
		item.PositionSeconds = min(item.PositionSeconds, item.DurationSeconds)
		item.Completed = podcastPlaybackCompleted(item.PositionSeconds, item.DurationSeconds)
	}
	saved, err := r.db.SaveAudiobookProgress(item)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": saved})
}

func requestLibriVox(query, id string, limit int, extended bool) ([]librivoxBook, error) {
	if query != "" {
		query = "^" + strings.TrimPrefix(query, "^")
	}
	return requestLibriVoxFilter("title", query, id, limit, extended)
}

func requestLibriVoxFilter(field, query, id string, limit int, extended bool) ([]librivoxBook, error) {
	params := url.Values{"format": {"json"}, "limit": {strconv.Itoa(limit)}}
	if query != "" {
		params.Set(field, query)
	}
	if id != "" {
		params.Set("id", id)
	}
	params.Set("extended", "1")
	fields := "{id,title,description,language,copyright_year,num_sections,url_librivox,url_iarchive,totaltimesecs,authors,genres}"
	if extended {
		fields = "{id,title,description,language,copyright_year,num_sections,url_librivox,url_iarchive,totaltimesecs,authors,genres,sections}"
	}
	params.Set("fields", fields)
	endpoint := librivoxAudiobooksURL + "?" + params.Encode()
	audiobookCache.RLock()
	cached, ok := audiobookCache.entries[endpoint]
	audiobookCache.RUnlock()
	if ok && time.Now().Before(cached.ExpiresAt) {
		return append([]librivoxBook(nil), cached.Books...), nil
	}
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "WaveNode/"+WaveNodeVersion+" (https://github.com/Dropzy/WaveNode)")
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("LibriVox could not be reached: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return []librivoxBook{}, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("LibriVox returned status %d", response.StatusCode)
	}
	var payload librivoxResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 32<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("LibriVox response could not be read")
	}
	audiobookCache.Lock()
	audiobookCache.entries[endpoint] = audiobookCacheEntry{Books: payload.Books, ExpiresAt: time.Now().Add(6 * time.Hour)}
	audiobookCache.Unlock()
	return append([]librivoxBook(nil), payload.Books...), nil
}

func mapAudiobookSummaries(books []librivoxBook) []audiobookSummary {
	result := make([]audiobookSummary, 0, len(books))
	for _, book := range books {
		result = append(result, mapAudiobookSummary(book))
	}
	return result
}

func mapAudiobookSummary(book librivoxBook) audiobookSummary {
	chapterCount, _ := strconv.Atoi(book.NumSections)
	genres := make([]string, 0, len(book.Genres))
	for _, genre := range book.Genres {
		if name := repairLibriVoxText(strings.TrimSpace(genre.Name)); name != "" {
			genres = append(genres, name)
		}
	}
	return audiobookSummary{ID: book.ID, Title: repairLibriVoxText(strings.TrimSpace(book.Title)), Author: audiobookAuthorName(book.Authors),
		Description: audiobookPlainText(book.Description), ImageURL: audiobookArchiveImage(book.URLArchive),
		WebsiteURL: book.URLLibriVox, Language: repairLibriVoxText(book.Language), CopyrightYear: book.CopyrightYear,
		ChapterCount: chapterCount, Duration: book.TotalTimeSecs, Genres: genres}
}

func audiobookAuthorName(authors []audiobookAuthor) string {
	names := make([]string, 0, len(authors))
	for _, author := range authors {
		name := repairLibriVoxText(strings.TrimSpace(strings.TrimSpace(author.FirstName) + " " + strings.TrimSpace(author.LastName)))
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "Unknown author"
	}
	return strings.Join(names, ", ")
}

func audiobookArchiveImage(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	identifier := path.Base(strings.TrimRight(parsed.Path, "/"))
	if identifier == "" || identifier == "." || identifier == "/" {
		return ""
	}
	return "https://archive.org/services/img/" + url.PathEscape(identifier)
}

func audiobookPlainText(value string) string {
	return repairLibriVoxText(strings.Join(strings.Fields(html.UnescapeString(audiobookHTMLPattern.ReplaceAllString(value, " "))), " "))
}

func repairLibriVoxText(value string) string {
	if !strings.ContainsAny(value, "ÃÂâ") {
		return value
	}
	bytes := make([]byte, 0, len(value))
	for _, character := range value {
		if character > 255 {
			return value
		}
		bytes = append(bytes, byte(character))
	}
	if !utf8.Valid(bytes) {
		return value
	}
	return string(bytes)
}
