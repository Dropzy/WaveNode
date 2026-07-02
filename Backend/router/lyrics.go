package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"music-server/database"

	"github.com/gorilla/mux"
)

var lyricsProviderURL = "https://lrclib.net/api/get"

var (
	lrcTimestampPattern = regexp.MustCompile(`\[(\d{1,3}):(\d{2})(?:[.:](\d{1,3}))?\]`)
	lrcWordTimePattern  = regexp.MustCompile(`<\d{1,3}:\d{2}(?:[.:]\d{1,3})?>`)
	lrcOffsetPattern    = regexp.MustCompile(`(?i)^\[offset:([+-]?\d+)\]$`)
)

type lyricLine struct {
	TimeMS int64  `json:"time_ms"`
	Text   string `json:"text"`
}

type lyricsResult struct {
	TrackID      string      `json:"track_id"`
	Available    bool        `json:"available"`
	Synced       bool        `json:"synced"`
	Instrumental bool        `json:"instrumental"`
	PlainText    string      `json:"plain_text"`
	Lines        []lyricLine `json:"lines"`
	Source       string      `json:"source"`
}

type cachedLyrics struct {
	Result    lyricsResult
	ExpiresAt time.Time
}

type lrclibResponse struct {
	Instrumental bool   `json:"instrumental"`
	PlainLyrics  string `json:"plainLyrics"`
	SyncedLyrics string `json:"syncedLyrics"`
}

func (r *Router) getLyrics(w http.ResponseWriter, req *http.Request) {
	trackID := mux.Vars(req)["id"]
	track, err := r.db.GetMusic(trackID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Track was not found")
		return
	}
	allowed, accessErr := r.requestCanAccessMusic(req, track)
	if accessErr != nil || !allowed {
		writeJSONError(w, http.StatusNotFound, "Track was not found")
		return
	}
	result := r.resolveLyrics(req, track)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": result})
}

func (r *Router) resolveLyrics(req *http.Request, track *database.Music) lyricsResult {
	if cached, ok := r.lyricsCache.Load(track.ID); ok {
		entry := cached.(cachedLyrics)
		if time.Now().Before(entry.ExpiresAt) {
			return entry.Result
		}
		r.lyricsCache.Delete(track.ID)
	}

	result := lyricsResult{TrackID: track.ID, Lines: []lyricLine{}}
	if local, found := loadLocalLyrics(track); found {
		result = local
	} else if remote, found := fetchLRCLIBLyrics(req, track); found {
		result = remote
	}
	result.TrackID = track.ID
	ttl := 15 * time.Minute
	if result.Available || result.Instrumental {
		ttl = 24 * time.Hour
	}
	r.lyricsCache.Store(track.ID, cachedLyrics{Result: result, ExpiresAt: time.Now().Add(ttl)})
	return result
}

func (r *Router) subsonicLyrics(req *http.Request) map[string]interface{} {
	artist := strings.TrimSpace(req.FormValue("artist"))
	title := strings.TrimSpace(req.FormValue("title"))
	value := ""
	if tracks, err := r.db.SearchMusic(strings.TrimSpace(artist + " " + title)); err == nil {
		tracks, _ = r.filterMusicForRequest(req, tracks)
		for index := range tracks {
			if strings.EqualFold(strings.TrimSpace(tracks[index].Artist), artist) && strings.EqualFold(strings.TrimSpace(tracks[index].Title), title) {
				value = r.resolveLyrics(req, &tracks[index]).PlainText
				break
			}
		}
	}
	return map[string]interface{}{"lyrics": map[string]interface{}{"artist": artist, "title": title, "value": value}}
}

func (r *Router) subsonicStructuredLyrics(req *http.Request) map[string]interface{} {
	structured := []interface{}{}
	track, err := r.db.GetMusic(req.FormValue("id"))
	if err == nil {
		allowed, _ := r.requestCanAccessMusic(req, track)
		if !allowed {
			return map[string]interface{}{"lyricsList": map[string]interface{}{"structuredLyrics": structured}}
		}
		lyrics := r.resolveLyrics(req, track)
		if lyrics.Available {
			lines := make([]map[string]interface{}, 0)
			if lyrics.Synced {
				for _, line := range lyrics.Lines {
					lines = append(lines, map[string]interface{}{"start": line.TimeMS, "value": line.Text})
				}
			} else {
				for _, line := range strings.Split(lyrics.PlainText, "\n") {
					lines = append(lines, map[string]interface{}{"value": line})
				}
			}
			structured = append(structured, map[string]interface{}{
				"displayArtist": track.Artist,
				"displayTitle":  track.Title,
				"language":      "und",
				"synced":        lyrics.Synced,
				"line":          lines,
			})
		}
	}
	return map[string]interface{}{"lyricsList": map[string]interface{}{"structuredLyrics": structured}}
}

func loadLocalLyrics(track *database.Music) (lyricsResult, bool) {
	base := strings.TrimSuffix(track.FilePath, filepath.Ext(track.FilePath))
	for _, candidate := range []string{base + ".lrc", base + ".LRC", base + ".txt", base + ".TXT"} {
		content, err := readLyricsFile(candidate)
		if err != nil {
			continue
		}
		result := lyricsResult{Available: strings.TrimSpace(content) != "", PlainText: strings.TrimSpace(content), Lines: []lyricLine{}, Source: "local"}
		if strings.EqualFold(filepath.Ext(candidate), ".lrc") {
			result.Lines, result.PlainText = parseLRC(content)
			result.Synced = len(result.Lines) > 0
			result.Available = result.Synced || result.PlainText != ""
		}
		return result, result.Available
	}
	return lyricsResult{}, false
}

func readLyricsFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, (2<<20)+1))
	if err != nil || len(content) > 2<<20 {
		return "", fmt.Errorf("lyrics file is too large")
	}
	return strings.TrimPrefix(string(content), "\ufeff"), nil
}

func parseLRC(content string) ([]lyricLine, string) {
	offsetMS := int64(0)
	lines := make([]lyricLine, 0)
	plain := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if match := lrcOffsetPattern.FindStringSubmatch(raw); match != nil {
			offsetMS, _ = strconv.ParseInt(match[1], 10, 64)
			continue
		}
		matches := lrcTimestampPattern.FindAllStringSubmatch(raw, -1)
		text := strings.TrimSpace(lrcWordTimePattern.ReplaceAllString(lrcTimestampPattern.ReplaceAllString(raw, ""), ""))
		if len(matches) == 0 {
			if raw != "" && !strings.HasPrefix(raw, "[") {
				plain = append(plain, raw)
			}
			continue
		}
		plain = append(plain, text)
		for _, match := range matches {
			minutes, _ := strconv.ParseInt(match[1], 10, 64)
			seconds, _ := strconv.ParseInt(match[2], 10, 64)
			fraction := int64(0)
			if match[3] != "" {
				fraction, _ = strconv.ParseInt(match[3], 10, 64)
				switch len(match[3]) {
				case 1:
					fraction *= 100
				case 2:
					fraction *= 10
				}
			}
			lines = append(lines, lyricLine{TimeMS: max(0, (minutes*60+seconds)*1000+fraction+offsetMS), Text: text})
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].TimeMS < lines[j].TimeMS })
	return lines, strings.TrimSpace(strings.Join(plain, "\n"))
}

func fetchLRCLIBLyrics(req *http.Request, track *database.Music) (lyricsResult, bool) {
	query := url.Values{
		"track_name":  {track.Title},
		"artist_name": {track.Artist},
		"album_name":  {track.Album},
		"duration":    {strconv.Itoa(track.Duration)},
	}
	request, err := http.NewRequestWithContext(req.Context(), http.MethodGet, lyricsProviderURL+"?"+query.Encode(), nil)
	if err != nil {
		return lyricsResult{}, false
	}
	request.Header.Set("User-Agent", "WaveNode/"+WaveNodeVersion+" (https://github.com/Dropzy/WaveNode)")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return lyricsResult{}, false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return lyricsResult{}, false
	}
	var payload lrclibResponse
	if json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload) != nil {
		return lyricsResult{}, false
	}
	lines, derivedPlain := parseLRC(payload.SyncedLyrics)
	plain := strings.TrimSpace(payload.PlainLyrics)
	if plain == "" {
		plain = derivedPlain
	}
	return lyricsResult{
		Available:    plain != "" || len(lines) > 0,
		Synced:       len(lines) > 0,
		Instrumental: payload.Instrumental,
		PlainText:    plain,
		Lines:        lines,
		Source:       "lrclib",
	}, plain != "" || len(lines) > 0 || payload.Instrumental
}
