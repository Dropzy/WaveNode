package router

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"music-server/database"

	"github.com/gorilla/mux"
)

const (
	scrobbleSettingsKeyPrefix = "scrobble.settings."
	lastFMAppSettingsKey      = "scrobble.lastfm_app"
	listenBrainzSubmitURL     = "https://api.listenbrainz.org/1/submit-listens"
	lastFMAPIURL              = "https://ws.audioscrobbler.com/2.0/"
)

type scrobbleSettings struct {
	ListenBrainzEnabled bool   `json:"listenbrainz_enabled"`
	ListenBrainzToken   string `json:"listenbrainz_token,omitempty"`
	LastFMEnabled       bool   `json:"lastfm_enabled"`
	LastFMSessionKey    string `json:"lastfm_session_key,omitempty"`
	LastFMUsername      string `json:"lastfm_username,omitempty"`
	LastFMPendingToken  string `json:"lastfm_pending_token,omitempty"`
	LastFMPendingAPIKey string `json:"lastfm_pending_api_key,omitempty"`
}

type scrobbleSettingsResponse struct {
	ListenBrainzEnabled    bool   `json:"listenbrainz_enabled"`
	HasListenBrainzToken   bool   `json:"has_listenbrainz_token"`
	LastFMEnabled          bool   `json:"lastfm_enabled"`
	LastFMServerConfigured bool   `json:"lastfm_server_configured"`
	LastFMUsername         string `json:"lastfm_username,omitempty"`
	HasLastFMSessionKey    bool   `json:"has_lastfm_session_key"`
	HasLastFMPendingToken  bool   `json:"has_lastfm_pending_token"`
}

type scrobblePayload struct {
	ListenedAt int64 `json:"listened_at,omitempty"`
}

type lastFMAppSettings struct {
	APIKey       string `json:"api_key"`
	SharedSecret string `json:"shared_secret,omitempty"`
}

type lastFMAppSettingsResponse struct {
	APIKey          string `json:"api_key"`
	HasSharedSecret bool   `json:"has_shared_secret"`
	Configured      bool   `json:"configured"`
}

type lastFMStartResponse struct {
	AuthURL string `json:"auth_url"`
}

type lastFMCompleteRequest struct {
	Token string `json:"token"`
}

func (r *Router) getScrobbleSettings(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}
	settings, err := r.loadScrobbleSettings(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	appSettings, err := r.loadLastFMAppSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": scrobbleSettingsView(settings, appSettings)})
}

func (r *Router) updateScrobbleSettings(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}

	existing, err := r.loadScrobbleSettings(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var incoming scrobbleSettings
	if err := json.NewDecoder(req.Body).Decode(&incoming); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid scrobbling settings")
		return
	}

	next := existing
	next.ListenBrainzEnabled = incoming.ListenBrainzEnabled
	next.LastFMEnabled = incoming.LastFMEnabled

	incoming.ListenBrainzToken = strings.TrimSpace(incoming.ListenBrainzToken)
	if incoming.ListenBrainzToken != "" {
		next.ListenBrainzToken = incoming.ListenBrainzToken
	}
	if next.LastFMEnabled && next.LastFMSessionKey == "" {
		writeJSONError(w, http.StatusBadRequest, "Connect Last.fm before enabling Last.fm scrobbling")
		return
	}
	if next.ListenBrainzEnabled && next.ListenBrainzToken == "" {
		writeJSONError(w, http.StatusBadRequest, "ListenBrainz scrobbling needs a user token")
		return
	}

	if err := r.saveScrobbleSettings(userID, next); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	appSettings, err := r.loadLastFMAppSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": scrobbleSettingsView(next, appSettings)})
}

func (r *Router) startLastFMAuth(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}
	existing, err := r.loadScrobbleSettings(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	appSettings, err := r.loadLastFMAppSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !appSettings.configured() {
		writeJSONError(w, http.StatusBadRequest, "Last.fm is not configured by an administrator yet")
		return
	}

	token := strings.TrimSpace(existing.LastFMPendingToken)
	if token == "" || existing.LastFMPendingAPIKey != appSettings.APIKey || existing.LastFMSessionKey != "" {
		var err error
		token, err = requestLastFMToken(appSettings.APIKey, appSettings.SharedSecret)
		if err != nil {
			writeJSONError(w, http.StatusBadGateway, err.Error())
			return
		}

		existing.LastFMPendingToken = token
		existing.LastFMPendingAPIKey = appSettings.APIKey
		if existing.LastFMSessionKey != "" {
			existing.LastFMSessionKey = ""
			existing.LastFMUsername = ""
			existing.LastFMEnabled = false
		}
		if err := r.saveScrobbleSettings(userID, existing); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	authURL := "https://www.last.fm/api/auth/?" + url.Values{
		"api_key": {appSettings.APIKey},
		"token":   {token},
		"cb":      {lastFMCallbackURL(req)},
	}.Encode()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    lastFMStartResponse{AuthURL: authURL},
	})
}

func (r *Router) completeLastFMAuth(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}
	settings, err := r.loadScrobbleSettings(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	appSettings, err := r.loadLastFMAppSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !appSettings.configured() || settings.LastFMPendingToken == "" {
		writeJSONError(w, http.StatusBadRequest, "Start the Last.fm connection first")
		return
	}
	if settings.LastFMPendingAPIKey != "" && settings.LastFMPendingAPIKey != appSettings.APIKey {
		settings.LastFMPendingToken = ""
		settings.LastFMPendingAPIKey = ""
		_ = r.saveScrobbleSettings(userID, settings)
		writeJSONError(w, http.StatusBadRequest, "Last.fm settings changed. Open Last.fm again to approve the new connection.")
		return
	}
	token := strings.TrimSpace(req.URL.Query().Get("token"))
	if token == "" && req.Body != nil {
		var payload lastFMCompleteRequest
		_ = json.NewDecoder(req.Body).Decode(&payload)
		token = strings.TrimSpace(payload.Token)
	}
	if token == "" {
		token = settings.LastFMPendingToken
	}
	if token != settings.LastFMPendingToken {
		writeJSONError(w, http.StatusBadRequest, "Last.fm returned an unexpected auth token")
		return
	}

	sessionKey, username, err := requestLastFMSession(appSettings.APIKey, appSettings.SharedSecret, token)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	settings.LastFMSessionKey = sessionKey
	settings.LastFMUsername = username
	settings.LastFMPendingToken = ""
	settings.LastFMPendingAPIKey = ""
	settings.LastFMEnabled = true
	if err := r.saveScrobbleSettings(userID, settings); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": scrobbleSettingsView(settings, appSettings)})
}

func (r *Router) disconnectLastFM(w http.ResponseWriter, req *http.Request) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}
	settings, err := r.loadScrobbleSettings(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	settings.LastFMEnabled = false
	settings.LastFMSessionKey = ""
	settings.LastFMUsername = ""
	settings.LastFMPendingToken = ""
	settings.LastFMPendingAPIKey = ""
	if err := r.saveScrobbleSettings(userID, settings); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	appSettings, err := r.loadLastFMAppSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": scrobbleSettingsView(settings, appSettings)})
}

func (r *Router) scrobbleNowPlaying(w http.ResponseWriter, req *http.Request) {
	r.handleScrobbleEvent(w, req, "now_playing")
}

func (r *Router) scrobbleListened(w http.ResponseWriter, req *http.Request) {
	r.handleScrobbleEvent(w, req, "listened")
}

func (r *Router) handleScrobbleEvent(w http.ResponseWriter, req *http.Request, event string) {
	userID := requestUserID(req)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Failed to get user information")
		return
	}
	trackID := strings.TrimSpace(mux.Vars(req)["id"])
	if trackID == "" {
		writeJSONError(w, http.StatusBadRequest, "Track ID is required")
		return
	}
	track, err := r.db.GetMusic(trackID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Track not found")
		return
	}
	allowed, accessErr := r.requestCanAccessMusic(req, track)
	if accessErr != nil || !allowed {
		writeJSONError(w, http.StatusNotFound, "Track not found")
		return
	}
	payload := scrobblePayload{}
	_ = json.NewDecoder(req.Body).Decode(&payload)
	listenedAt := time.Now()
	if payload.ListenedAt > 0 {
		listenedAt = time.Unix(payload.ListenedAt, 0)
	}

	go r.submitScrobble(userID, *track, event, listenedAt)
	writeJSON(w, http.StatusAccepted, map[string]interface{}{"success": true})
}

func (r *Router) submitScrobble(userID string, track database.Music, event string, listenedAt time.Time) {
	settings, err := r.loadScrobbleSettings(userID)
	if err != nil {
		log.Printf("Scrobble skipped: failed to load settings: %v", err)
		return
	}
	if settings.ListenBrainzEnabled && settings.ListenBrainzToken != "" {
		if err := submitListenBrainzScrobble(settings.ListenBrainzToken, track, event, listenedAt); err != nil {
			log.Printf("ListenBrainz scrobble failed for %s: %v", track.ID, err)
		}
	}
	if settings.LastFMEnabled && settings.LastFMSessionKey != "" {
		appSettings, err := r.loadLastFMAppSettings()
		if err != nil {
			log.Printf("Last.fm scrobble skipped: failed to load app settings: %v", err)
			return
		}
		if !appSettings.configured() {
			log.Printf("Last.fm scrobble skipped: Last.fm is not configured")
			return
		}
		if err := submitLastFMScrobble(appSettings, settings.LastFMSessionKey, track, event, listenedAt); err != nil {
			log.Printf("Last.fm scrobble failed for %s: %v", track.ID, err)
		}
	}
}

func (r *Router) loadScrobbleSettings(userID string) (scrobbleSettings, error) {
	value, err := r.db.GetSetting(scrobbleSettingsKeyPrefix + userID)
	if err != nil || strings.TrimSpace(value) == "" {
		return scrobbleSettings{}, err
	}
	var settings scrobbleSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return scrobbleSettings{}, fmt.Errorf("failed to read scrobbling settings: %v", err)
	}
	return settings, nil
}

func (r *Router) saveScrobbleSettings(userID string, settings scrobbleSettings) error {
	encoded, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("could not save scrobbling settings: %v", err)
	}
	return r.db.SetSetting(scrobbleSettingsKeyPrefix+userID, string(encoded))
}

func scrobbleSettingsView(settings scrobbleSettings, appSettings lastFMAppSettings) scrobbleSettingsResponse {
	return scrobbleSettingsResponse{
		ListenBrainzEnabled:    settings.ListenBrainzEnabled,
		HasListenBrainzToken:   settings.ListenBrainzToken != "",
		LastFMEnabled:          settings.LastFMEnabled,
		LastFMServerConfigured: appSettings.configured(),
		LastFMUsername:         settings.LastFMUsername,
		HasLastFMSessionKey:    settings.LastFMSessionKey != "",
		HasLastFMPendingToken:  settings.LastFMPendingToken != "" && settings.LastFMSessionKey == "",
	}
}

func (r *Router) getAdminLastFMIntegration(w http.ResponseWriter, req *http.Request) {
	settings, err := r.loadLastFMAppSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": lastFMAppSettingsView(settings)})
}

func (r *Router) updateAdminLastFMIntegration(w http.ResponseWriter, req *http.Request) {
	existing, err := r.loadLastFMAppSettings()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var incoming lastFMAppSettings
	if err := json.NewDecoder(req.Body).Decode(&incoming); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid Last.fm integration settings")
		return
	}
	incoming.APIKey = strings.TrimSpace(incoming.APIKey)
	incoming.SharedSecret = strings.TrimSpace(incoming.SharedSecret)
	if incoming.SharedSecret == "" {
		incoming.SharedSecret = existing.SharedSecret
	}
	if incoming.APIKey == "" || incoming.SharedSecret == "" {
		writeJSONError(w, http.StatusBadRequest, "Last.fm API key and shared secret are required")
		return
	}
	if err := r.saveLastFMAppSettings(incoming); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": lastFMAppSettingsView(incoming)})
}

func (r *Router) loadLastFMAppSettings() (lastFMAppSettings, error) {
	value, err := r.db.GetSetting(lastFMAppSettingsKey)
	if err != nil || strings.TrimSpace(value) == "" {
		return lastFMAppSettings{}, err
	}
	var settings lastFMAppSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return lastFMAppSettings{}, fmt.Errorf("failed to read Last.fm integration settings: %v", err)
	}
	settings.APIKey = strings.TrimSpace(settings.APIKey)
	settings.SharedSecret = strings.TrimSpace(settings.SharedSecret)
	return settings, nil
}

func (r *Router) saveLastFMAppSettings(settings lastFMAppSettings) error {
	encoded, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("could not save Last.fm integration settings: %v", err)
	}
	return r.db.SetSetting(lastFMAppSettingsKey, string(encoded))
}

func lastFMAppSettingsView(settings lastFMAppSettings) lastFMAppSettingsResponse {
	return lastFMAppSettingsResponse{
		APIKey:          settings.APIKey,
		HasSharedSecret: settings.SharedSecret != "",
		Configured:      settings.configured(),
	}
}

func (settings lastFMAppSettings) configured() bool {
	return strings.TrimSpace(settings.APIKey) != "" && strings.TrimSpace(settings.SharedSecret) != ""
}

func requestLastFMToken(apiKey, sharedSecret string) (string, error) {
	params := map[string]string{
		"api_key": apiKey,
		"method":  "auth.getToken",
	}
	params["api_sig"] = lastFMSignature(params, sharedSecret)
	params["format"] = "json"

	body, err := postLastFM(params)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Token   string      `json:"token"`
		Error   interface{} `json:"error"`
		Message string      `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("Last.fm token response could not be read")
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("Last.fm rejected the token request: %v %s", parsed.Error, parsed.Message)
	}
	if parsed.Token == "" {
		return "", fmt.Errorf("Last.fm did not return an auth token")
	}
	return parsed.Token, nil
}

func requestLastFMSession(apiKey, sharedSecret, token string) (string, string, error) {
	params := map[string]string{
		"api_key": apiKey,
		"method":  "auth.getSession",
		"token":   token,
	}
	params["api_sig"] = lastFMSignature(params, sharedSecret)
	params["format"] = "json"

	body, err := postLastFM(params)
	if err != nil {
		return "", "", err
	}
	var parsed struct {
		Session struct {
			Name string `json:"name"`
			Key  string `json:"key"`
		} `json:"session"`
		Error   interface{} `json:"error"`
		Message string      `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", fmt.Errorf("Last.fm session response could not be read")
	}
	if parsed.Error != nil {
		return "", "", fmt.Errorf("Last.fm rejected the session request: %v %s", parsed.Error, parsed.Message)
	}
	if parsed.Session.Key == "" {
		return "", "", fmt.Errorf("Last.fm did not return a session key")
	}
	return parsed.Session.Key, parsed.Session.Name, nil
}

func postLastFM(params map[string]string) ([]byte, error) {
	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	resp, err := http.PostForm(lastFMAPIURL, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if message := lastFMErrorMessage(body); message != "" {
			return nil, fmt.Errorf("%s", message)
		}
		return nil, fmt.Errorf("Last.fm returned %s", resp.Status)
	}
	return body, nil
}

func lastFMErrorMessage(body []byte) string {
	var parsed struct {
		Error   interface{} `json:"error"`
		Message string      `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	if parsed.Error == nil && parsed.Message == "" {
		return ""
	}
	if parsed.Message != "" {
		return fmt.Sprintf("Last.fm rejected the request: %s", parsed.Message)
	}
	return fmt.Sprintf("Last.fm rejected the request: %v", parsed.Error)
}

func submitListenBrainzScrobble(token string, track database.Music, event string, listenedAt time.Time) error {
	listenType := "single"
	payload := map[string]interface{}{
		"listen_type": listenType,
		"payload": []map[string]interface{}{{
			"listened_at": listenedAt.Unix(),
			"track_metadata": map[string]interface{}{
				"artist_name":  track.Artist,
				"track_name":   track.Title,
				"release_name": track.Album,
				"additional_info": map[string]interface{}{
					"duration_ms":       track.Duration * 1000,
					"media_player":      "WaveNode",
					"submission_client": "WaveNode",
					"origin_url":        "wavenode://track/" + track.ID,
				},
			},
		}},
	}
	if event == "now_playing" {
		payload["listen_type"] = "playing_now"
		delete(payload["payload"].([]map[string]interface{})[0], "listened_at")
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, listenBrainzSubmitURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WaveNode/0.1")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ListenBrainz returned %s", resp.Status)
	}
	return nil
}

func submitLastFMScrobble(appSettings lastFMAppSettings, sessionKey string, track database.Music, event string, listenedAt time.Time) error {
	params := map[string]string{
		"api_key":   appSettings.APIKey,
		"artist":    track.Artist,
		"method":    "track.scrobble",
		"sk":        sessionKey,
		"timestamp": strconv.FormatInt(listenedAt.Unix(), 10),
		"track":     track.Title,
	}
	if track.Album != "" {
		params["album"] = track.Album
	}
	if event == "now_playing" {
		params["method"] = "track.updateNowPlaying"
		delete(params, "timestamp")
	}
	if track.Duration > 0 {
		params["duration"] = strconv.Itoa(track.Duration)
	}
	params["api_sig"] = lastFMSignature(params, appSettings.SharedSecret)
	params["format"] = "json"

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	resp, err := http.PostForm(lastFMAPIURL, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Last.fm returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if err != nil {
		return err
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		if code, ok := parsed["error"]; ok {
			return fmt.Errorf("Last.fm rejected the scrobble: %v", code)
		}
	}
	return nil
}

func lastFMSignature(params map[string]string, sharedSecret string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key != "format" && key != "callback" && key != "api_sig" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString(params[key])
	}
	builder.WriteString(sharedSecret)
	sum := md5.Sum([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func lastFMCallbackURL(req *http.Request) string {
	origin := strings.TrimSpace(req.Header.Get("Origin"))
	if origin != "" {
		return strings.TrimRight(origin, "/") + "/lastfm/callback"
	}

	proto := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = "http"
		if req.TLS != nil {
			proto = "https"
		}
	}
	host := strings.TrimSpace(req.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = req.Host
	}
	return proto + "://" + host + "/lastfm/callback"
}
