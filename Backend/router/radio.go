package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"music-server/database"

	"github.com/gorilla/mux"
)

const defaultRadioBrowserURL = "https://all.api.radio-browser.info"

var radioStationIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

type radioBrowserStation struct {
	StationUUID string `json:"stationuuid"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	URLResolved string `json:"url_resolved"`
	Homepage    string `json:"homepage"`
	Favicon     string `json:"favicon"`
	Tags        string `json:"tags"`
	Country     string `json:"country"`
	CountryCode string `json:"countrycode"`
	Language    string `json:"language"`
	Codec       string `json:"codec"`
	Bitrate     int    `json:"bitrate"`
	Votes       int    `json:"votes"`
	ClickCount  int    `json:"clickcount"`
	LastCheckOK int    `json:"lastcheckok"`
}

type radioCacheEntry struct {
	Stations []database.RadioStation
	Expires  time.Time
}

var radioDirectoryCache sync.Map

func radioBrowserBaseURL() string {
	if configured := strings.TrimRight(strings.TrimSpace(os.Getenv("WAVENODE_RADIO_BROWSER_URL")), "/"); configured != "" {
		return configured
	}
	return defaultRadioBrowserURL
}

func secureOptionalURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ""
	}
	return parsed.String()
}

func radioBrowserRequest(ctx context.Context, endpoint string, params url.Values) ([]database.RadioStation, error) {
	target := radioBrowserBaseURL() + endpoint
	if encoded := params.Encode(); encoded != "" {
		target += "?" + encoded
	}
	if cached, ok := radioDirectoryCache.Load(target); ok {
		entry := cached.(radioCacheEntry)
		if time.Now().Before(entry.Expires) {
			return append([]database.RadioStation(nil), entry.Stations...), nil
		}
		radioDirectoryCache.Delete(target)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "WaveNode/"+WaveNodeVersion+" (+https://github.com/Dropzy/WaveNode)")
	response, err := (&http.Client{Timeout: 12 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("radio directory request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("radio directory returned %s", response.Status)
	}
	var raw []radioBrowserStation
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(&raw); err != nil {
		return nil, fmt.Errorf("radio directory response could not be read: %w", err)
	}
	stations := make([]database.RadioStation, 0, len(raw))
	for _, item := range raw {
		streamURL := strings.TrimSpace(item.URLResolved)
		if streamURL == "" {
			streamURL = strings.TrimSpace(item.URL)
		}
		parsed, parseErr := url.Parse(streamURL)
		if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" || item.LastCheckOK == 0 {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" || !radioStationIDPattern.MatchString(item.StationUUID) {
			continue
		}
		stations = append(stations, database.RadioStation{
			ID: item.StationUUID, Name: name, StreamURL: streamURL,
			HomepageURL: secureOptionalURL(item.Homepage), FaviconURL: secureOptionalURL(item.Favicon),
			Tags: strings.TrimSpace(item.Tags), Country: strings.TrimSpace(item.Country),
			CountryCode: strings.ToUpper(strings.TrimSpace(item.CountryCode)),
			Language:    strings.TrimSpace(item.Language), Codec: strings.ToUpper(strings.TrimSpace(item.Codec)),
			Bitrate: max(0, item.Bitrate), Votes: max(0, item.Votes), ClickCount: max(0, item.ClickCount),
		})
	}
	radioDirectoryCache.Store(target, radioCacheEntry{Stations: stations, Expires: time.Now().Add(10 * time.Minute)})
	return append([]database.RadioStation(nil), stations...), nil
}

func radioSearchParams(req *http.Request, limitDefault int) url.Values {
	params := url.Values{
		"hidebroken": {"true"}, "is_https": {"true"},
		"limit":  {strconv.Itoa(clampIntQuery(req, "limit", limitDefault, 1, 100))},
		"offset": {strconv.Itoa(clampIntQuery(req, "offset", 0, 0, 10000))},
	}
	for _, field := range []string{"name", "tag", "countrycode"} {
		if value := strings.TrimSpace(req.URL.Query().Get(field)); value != "" {
			params.Set(field, value)
		}
	}
	order := strings.ToLower(strings.TrimSpace(req.URL.Query().Get("order")))
	allowedOrders := map[string]bool{"name": true, "votes": true, "clickcount": true, "clicktrend": true, "bitrate": true, "random": true}
	if !allowedOrders[order] {
		order = "clickcount"
	}
	params.Set("order", order)
	if order != "name" && order != "random" {
		params.Set("reverse", "true")
	}
	return params
}

func markRadioFavorites(stations []database.RadioStation, favourites []database.RadioStation) {
	ids := make(map[string]bool, len(favourites))
	for _, station := range favourites {
		ids[station.ID] = true
	}
	for index := range stations {
		stations[index].Favourite = ids[stations[index].ID]
	}
}

func (r *Router) getRadioHome(w http.ResponseWriter, req *http.Request) {
	favourites, err := r.db.GetRadioFavorites(requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type result struct {
		name     string
		stations []database.RadioStation
		err      error
	}
	results := make(chan result, 3)
	queries := []struct{ name, order string }{{"popular", "clickcount"}, {"trending", "clicktrend"}}
	for _, query := range queries {
		go func(name, order string) {
			params := url.Values{"hidebroken": {"true"}, "is_https": {"true"}, "limit": {"30"}, "order": {order}, "reverse": {"true"}}
			stations, requestErr := radioBrowserRequest(req.Context(), "/json/stations/search", params)
			results <- result{name: name, stations: stations, err: requestErr}
		}(query.name, query.order)
	}
	expected := 2
	countryCode := strings.ToUpper(strings.TrimSpace(req.URL.Query().Get("country")))
	if len(countryCode) == 2 {
		expected++
		go func() {
			params := url.Values{"hidebroken": {"true"}, "is_https": {"true"}, "limit": {"30"}, "order": {"clickcount"}, "reverse": {"true"}, "countrycode": {countryCode}}
			stations, requestErr := radioBrowserRequest(req.Context(), "/json/stations/search", params)
			results <- result{name: "local", stations: stations, err: requestErr}
		}()
	}

	data := map[string]interface{}{"favourites": favourites, "popular": []database.RadioStation{}, "trending": []database.RadioStation{}, "local": []database.RadioStation{}}
	var directoryErr error
	for range expected {
		item := <-results
		if item.err != nil {
			directoryErr = item.err
			continue
		}
		markRadioFavorites(item.stations, favourites)
		data[item.name] = item.stations
	}
	if directoryErr != nil {
		data["directory_error"] = directoryErr.Error()
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": data})
}

func (r *Router) searchRadioStations(w http.ResponseWriter, req *http.Request) {
	query := strings.TrimSpace(req.URL.Query().Get("q"))
	if len(query) > 120 {
		writeJSONError(w, http.StatusBadRequest, "Radio search is too long")
		return
	}
	params := radioSearchParams(req, 50)
	if query != "" {
		params.Set("name", query)
	}
	stations, err := radioBrowserRequest(req.Context(), "/json/stations/search", params)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	favourites, err := r.db.GetRadioFavorites(requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	markRadioFavorites(stations, favourites)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": stations})
}

func radioStationByID(ctx context.Context, stationID string) (database.RadioStation, error) {
	if !radioStationIDPattern.MatchString(stationID) {
		return database.RadioStation{}, fmt.Errorf("invalid radio station ID")
	}
	stations, err := radioBrowserRequest(ctx, "/json/stations/byuuid/"+url.PathEscape(stationID), url.Values{"hidebroken": {"true"}})
	if err != nil {
		return database.RadioStation{}, err
	}
	if len(stations) == 0 {
		return database.RadioStation{}, fmt.Errorf("radio station was not found or does not provide a secure stream")
	}
	return stations[0], nil
}

func (r *Router) listRadioFavorites(w http.ResponseWriter, req *http.Request) {
	stations, err := r.db.GetRadioFavorites(requestUserID(req))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": stations})
}

func (r *Router) saveRadioFavorite(w http.ResponseWriter, req *http.Request) {
	station, err := radioStationByID(req.Context(), mux.Vars(req)["id"])
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	station, err = r.db.SaveRadioFavorite(requestUserID(req), station)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"success": true, "data": station})
}

func (r *Router) deleteRadioFavorite(w http.ResponseWriter, req *http.Request) {
	if err := r.db.DeleteRadioFavorite(requestUserID(req), mux.Vars(req)["id"]); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func validatePublicRadioStream(streamURL string) error {
	parsed, err := url.Parse(streamURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return fmt.Errorf("station stream URL is not secure")
	}
	addresses, err := net.LookupIP(parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return fmt.Errorf("station host could not be resolved")
	}
	for _, address := range addresses {
		if address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalMulticast() || address.IsLinkLocalUnicast() || address.IsUnspecified() {
			return fmt.Errorf("station stream host is not public")
		}
	}
	return nil
}

func (r *Router) getRadioMetadata(w http.ResponseWriter, req *http.Request) {
	station, err := radioStationByID(req.Context(), mux.Vars(req)["id"])
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := validatePublicRadioStream(station.StreamURL); err != nil {
		writeJSONError(w, http.StatusForbidden, err.Error())
		return
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(redirect *http.Request, _ []*http.Request) error {
			return validatePublicRadioStream(redirect.URL.String())
		},
	}
	title, metadataErr := fetchICYStreamTitleWithClient(station.StreamURL, client)
	data := map[string]interface{}{"station_title": station.Name, "stream_title": title}
	if metadataErr != nil {
		data["error"] = metadataErr.Error()
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": data})
}

func (r *Router) clickRadioStation(w http.ResponseWriter, req *http.Request) {
	stationID := mux.Vars(req)["id"]
	if !radioStationIDPattern.MatchString(stationID) {
		writeJSONError(w, http.StatusBadRequest, "Invalid radio station ID")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		target := radioBrowserBaseURL() + "/json/url/" + url.PathEscape(stationID)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return
		}
		request.Header.Set("User-Agent", "WaveNode/"+WaveNodeVersion+" (+https://github.com/Dropzy/WaveNode)")
		response, err := (&http.Client{Timeout: 8 * time.Second}).Do(request)
		if err == nil {
			response.Body.Close()
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]interface{}{"success": true})
}
