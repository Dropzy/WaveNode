package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"music-server/database"

	"github.com/gorilla/mux"
)

const (
	maxPluginManifestBytes = 256 * 1024
	maxPluginHomeRows      = 10
	maxPluginRowItems      = 100
	maxPluginTrackActions  = 20
)

var pluginIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,98}[a-z0-9]$`)

type PluginManifest struct {
	SchemaVersion int                 `json:"schema_version"`
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Version       string              `json:"version"`
	Description   string              `json:"description,omitempty"`
	Homepage      string              `json:"homepage,omitempty"`
	Permissions   []string            `json:"permissions,omitempty"`
	Contributes   PluginContributions `json:"contributes"`
}

type PluginContributions struct {
	HomeRows     []PluginHomeRow     `json:"home_rows,omitempty"`
	TrackActions []PluginTrackAction `json:"track_actions,omitempty"`
}

type PluginTrackAction struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Icon       string `json:"icon,omitempty"`
	ActionType string `json:"action_type"`
	URL        string `json:"url"`
}

type PluginHomeRow struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Subtitle string          `json:"subtitle,omitempty"`
	Type     string          `json:"type"`
	Items    []PluginRowItem `json:"items"`
}

type PluginRowItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	StreamURL   string `json:"stream_url"`
	HomepageURL string `json:"homepage_url,omitempty"`
}

type runtimeHomeRow struct {
	PluginID string          `json:"plugin_id"`
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Subtitle string          `json:"subtitle,omitempty"`
	Type     string          `json:"type"`
	Items    []PluginRowItem `json:"items"`
}

type runtimeTrackAction struct {
	PluginID   string `json:"plugin_id"`
	ID         string `json:"id"`
	Label      string `json:"label"`
	Icon       string `json:"icon,omitempty"`
	ActionType string `json:"action_type"`
	URL        string `json:"url"`
}

func validatePluginManifest(raw []byte) (*PluginManifest, error) {
	if len(raw) == 0 || len(raw) > maxPluginManifestBytes {
		return nil, fmt.Errorf("plugin manifest must be between 1 byte and 256 KB")
	}

	var manifest PluginManifest
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid plugin manifest: %v", err)
	}
	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported plugin schema version")
	}
	manifest.ID = strings.ToLower(strings.TrimSpace(manifest.ID))
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Version = strings.TrimSpace(manifest.Version)
	if !pluginIDPattern.MatchString(manifest.ID) {
		return nil, fmt.Errorf("plugin ID must use 3-100 lowercase letters, numbers, dots, or hyphens")
	}
	if manifest.Name == "" || len(manifest.Name) > 150 {
		return nil, fmt.Errorf("plugin name must be between 1 and 150 characters")
	}
	if manifest.Version == "" || len(manifest.Version) > 50 {
		return nil, fmt.Errorf("plugin version must be between 1 and 50 characters")
	}
	if len(manifest.Description) > 500 {
		return nil, fmt.Errorf("plugin description must be 500 characters or fewer")
	}
	if manifest.Homepage != "" {
		if len(manifest.Homepage) > 2048 {
			return nil, fmt.Errorf("plugin homepage is too long")
		}
		if err := validatePluginURL(manifest.Homepage); err != nil {
			return nil, fmt.Errorf("invalid plugin homepage: %v", err)
		}
	}
	allowedPermissions := map[string]bool{"network": true, "playback": true, "download": true}
	permissions := make(map[string]bool)
	for _, permission := range manifest.Permissions {
		permission = strings.ToLower(strings.TrimSpace(permission))
		if !allowedPermissions[permission] {
			return nil, fmt.Errorf("unsupported plugin permission %q", permission)
		}
		permissions[permission] = true
	}
	if len(manifest.Contributes.HomeRows) > maxPluginHomeRows {
		return nil, fmt.Errorf("plugins may contribute at most %d home rows", maxPluginHomeRows)
	}
	if len(manifest.Contributes.TrackActions) > maxPluginTrackActions {
		return nil, fmt.Errorf("plugins may contribute at most %d track actions", maxPluginTrackActions)
	}

	rowIDs := make(map[string]struct{})
	for rowIndex := range manifest.Contributes.HomeRows {
		row := &manifest.Contributes.HomeRows[rowIndex]
		row.ID = strings.TrimSpace(row.ID)
		row.Title = strings.TrimSpace(row.Title)
		row.Type = strings.ToLower(strings.TrimSpace(row.Type))
		if !pluginIDPattern.MatchString(row.ID) {
			return nil, fmt.Errorf("home row %d has an invalid ID", rowIndex+1)
		}
		if _, exists := rowIDs[row.ID]; exists {
			return nil, fmt.Errorf("home row ID %q is duplicated", row.ID)
		}
		rowIDs[row.ID] = struct{}{}
		if row.Title == "" || len(row.Title) > 100 {
			return nil, fmt.Errorf("home row %q must have a title of 1-100 characters", row.ID)
		}
		if row.Type != "radio" {
			return nil, fmt.Errorf("home row %q uses unsupported type %q", row.ID, row.Type)
		}
		if !permissions["network"] || !permissions["playback"] {
			return nil, fmt.Errorf("radio plugins require network and playback permissions")
		}
		if len(row.Items) == 0 || len(row.Items) > maxPluginRowItems {
			return nil, fmt.Errorf("home row %q must contain 1-%d items", row.ID, maxPluginRowItems)
		}
		itemIDs := make(map[string]struct{})
		for itemIndex := range row.Items {
			item := &row.Items[itemIndex]
			item.ID = strings.TrimSpace(item.ID)
			item.Title = strings.TrimSpace(item.Title)
			item.StreamURL = strings.TrimSpace(item.StreamURL)
			if !pluginIDPattern.MatchString(item.ID) {
				return nil, fmt.Errorf("item %d in row %q has an invalid ID", itemIndex+1, row.ID)
			}
			if _, exists := itemIDs[item.ID]; exists {
				return nil, fmt.Errorf("item ID %q is duplicated in row %q", item.ID, row.ID)
			}
			itemIDs[item.ID] = struct{}{}
			if item.Title == "" || len(item.Title) > 150 {
				return nil, fmt.Errorf("item %q must have a title of 1-150 characters", item.ID)
			}
			if err := validatePluginURL(item.StreamURL); err != nil {
				return nil, fmt.Errorf("item %q has an invalid stream URL: %v", item.ID, err)
			}
			for label, value := range map[string]string{
				"image URL": item.ImageURL, "homepage URL": item.HomepageURL,
			} {
				if value != "" {
					if err := validatePluginURL(value); err != nil {
						return nil, fmt.Errorf("item %q has an invalid %s: %v", item.ID, label, err)
					}
				}
			}
		}
	}
	actionIDs := make(map[string]struct{})
	for actionIndex := range manifest.Contributes.TrackActions {
		action := &manifest.Contributes.TrackActions[actionIndex]
		action.ID = strings.TrimSpace(action.ID)
		action.Label = strings.TrimSpace(action.Label)
		action.Icon = strings.ToLower(strings.TrimSpace(action.Icon))
		action.ActionType = strings.ToLower(strings.TrimSpace(action.ActionType))
		action.URL = strings.TrimSpace(action.URL)
		if !pluginIDPattern.MatchString(action.ID) {
			return nil, fmt.Errorf("track action %d has an invalid ID", actionIndex+1)
		}
		if _, exists := actionIDs[action.ID]; exists {
			return nil, fmt.Errorf("track action ID %q is duplicated", action.ID)
		}
		actionIDs[action.ID] = struct{}{}
		if action.Label == "" || len(action.Label) > 80 {
			return nil, fmt.Errorf("track action %q must have a label of 1-80 characters", action.ID)
		}
		if action.ActionType != "download" {
			return nil, fmt.Errorf("track action %q uses unsupported action type %q", action.ID, action.ActionType)
		}
		if !permissions["download"] {
			return nil, fmt.Errorf("download track actions require download permission")
		}
		if action.URL != "/api/music/{id}/download" {
			return nil, fmt.Errorf("track action %q must use the approved download URL template", action.ID)
		}
	}
	if len(manifest.Contributes.HomeRows) == 0 && len(manifest.Contributes.TrackActions) == 0 {
		return nil, fmt.Errorf("plugin does not contribute any supported features")
	}
	return &manifest, nil
}

func validatePluginURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("only absolute HTTP and HTTPS URLs are supported")
	}
	return nil
}

func (r *Router) getAdminPlugins(w http.ResponseWriter, _ *http.Request) {
	plugins, err := r.db.GetPlugins(false)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": plugins})
}

func (r *Router) installPlugin(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, maxPluginManifestBytes)
	raw, err := readJSONBody(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	manifest, err := validatePluginManifest(raw)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	normalized, err := json.Marshal(manifest)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Could not normalize plugin manifest")
		return
	}
	plugin := database.Plugin{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version,
		Enabled: true, Manifest: normalized,
	}
	if err := r.db.UpsertPlugin(plugin); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"success": true, "data": plugin})
}

func readJSONBody(req *http.Request) ([]byte, error) {
	var raw json.RawMessage
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("plugin manifest could not be read")
	}
	return raw, nil
}

func (r *Router) setPluginEnabled(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload.Enabled == nil {
		writeJSONError(w, http.StatusBadRequest, "enabled must be true or false")
		return
	}
	if err := r.db.SetPluginEnabled(mux.Vars(req)["id"], *payload.Enabled); err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (r *Router) deletePlugin(w http.ResponseWriter, req *http.Request) {
	if err := r.db.DeletePlugin(mux.Vars(req)["id"]); err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (r *Router) getPluginHomeRows(w http.ResponseWriter, _ *http.Request) {
	rows, err := r.pluginHomeRows()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": rows})
}

func (r *Router) getPluginRadioMetadata(w http.ResponseWriter, req *http.Request) {
	streamURL := strings.TrimSpace(req.URL.Query().Get("stream_url"))
	if streamURL == "" {
		writeJSONError(w, http.StatusBadRequest, "stream_url is required")
		return
	}

	item, ok, err := r.findPluginRadioItem(streamURL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeJSONError(w, http.StatusForbidden, "stream_url is not provided by an enabled radio plugin")
		return
	}

	title, metadataErr := fetchICYStreamTitle(streamURL)
	if metadataErr != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"station_title": item.Title,
				"stream_title":  "",
				"error":         metadataErr.Error(),
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"station_title": item.Title,
			"stream_title":  title,
		},
	})
}

func (r *Router) getPluginTrackActions(w http.ResponseWriter, _ *http.Request) {
	actions, err := r.pluginTrackActions()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": actions})
}

func (r *Router) findPluginRadioItem(streamURL string) (PluginRowItem, bool, error) {
	rows, err := r.pluginHomeRows()
	if err != nil {
		return PluginRowItem{}, false, err
	}
	for _, row := range rows {
		if row.Type != "radio" {
			continue
		}
		for _, item := range row.Items {
			if item.StreamURL == streamURL {
				return item, true, nil
			}
		}
	}
	return PluginRowItem{}, false, nil
}

func fetchICYStreamTitle(streamURL string) (string, error) {
	return fetchICYStreamTitleWithClient(streamURL, &http.Client{Timeout: 10 * time.Second})
}

func fetchICYStreamTitleWithClient(streamURL string, client *http.Client) (string, error) {
	request, err := http.NewRequest(http.MethodGet, streamURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Icy-MetaData", "1")
	request.Header.Set("User-Agent", "WaveNode/0.1 radio metadata")

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("station returned %s", response.Status)
	}

	metaInterval, err := strconv.Atoi(response.Header.Get("icy-metaint"))
	if err != nil || metaInterval <= 0 || metaInterval > 1<<20 {
		return "", fmt.Errorf("station does not expose ICY metadata")
	}

	reader := bufio.NewReader(response.Body)
	for attempts := 0; attempts < 8; attempts++ {
		if _, err := io.CopyN(io.Discard, reader, int64(metaInterval)); err != nil {
			return "", err
		}

		lengthByte, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		metadataLength := int(lengthByte) * 16
		if metadataLength == 0 {
			continue
		}

		block := make([]byte, metadataLength)
		if _, err := io.ReadFull(reader, block); err != nil {
			return "", err
		}
		if title := parseICYStreamTitle(string(block)); title != "" {
			return title, nil
		}
	}

	return "", fmt.Errorf("no current stream title found")
}

func parseICYStreamTitle(metadata string) string {
	const marker = "StreamTitle='"
	start := strings.Index(metadata, marker)
	if start < 0 {
		return ""
	}
	start += len(marker)
	end := strings.Index(metadata[start:], "';")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(metadata[start : start+end])
}

func (r *Router) pluginHomeRows() ([]runtimeHomeRow, error) {
	plugins, err := r.db.GetPlugins(true)
	if err != nil {
		return nil, err
	}
	rows := make([]runtimeHomeRow, 0)
	for _, plugin := range plugins {
		manifest, err := validatePluginManifest(plugin.Manifest)
		if err != nil {
			continue
		}
		for _, row := range manifest.Contributes.HomeRows {
			rows = append(rows, runtimeHomeRow{
				PluginID: plugin.ID, ID: row.ID, Title: row.Title,
				Subtitle: row.Subtitle, Type: row.Type, Items: row.Items,
			})
		}
	}
	return rows, nil
}

func (r *Router) pluginTrackActions() ([]runtimeTrackAction, error) {
	plugins, err := r.db.GetPlugins(true)
	if err != nil {
		return nil, err
	}
	actions := make([]runtimeTrackAction, 0)
	for _, plugin := range plugins {
		manifest, err := validatePluginManifest(plugin.Manifest)
		if err != nil {
			continue
		}
		for _, action := range manifest.Contributes.TrackActions {
			actions = append(actions, runtimeTrackAction{
				PluginID: plugin.ID, ID: action.ID, Label: action.Label,
				Icon: action.Icon, ActionType: action.ActionType, URL: action.URL,
			})
		}
	}
	return actions, nil
}

func (r *Router) subsonicInternetRadioStations(user *database.User) (map[string]interface{}, *subsonicError) {
	rows, err := r.pluginHomeRows()
	if err != nil {
		return nil, internalSubsonicError(err)
	}
	stations := make([]interface{}, 0)
	for _, row := range rows {
		if row.Type != "radio" {
			continue
		}
		for _, item := range row.Items {
			station := map[string]interface{}{
				"id": row.PluginID + ":" + item.ID, "name": item.Title, "streamUrl": item.StreamURL,
			}
			if item.HomepageURL != "" {
				station["homepageUrl"] = item.HomepageURL
			}
			stations = append(stations, station)
		}
	}
	favourites, favoriteErr := r.db.GetRadioFavorites(user.ID)
	if favoriteErr != nil {
		return nil, internalSubsonicError(favoriteErr)
	}
	for _, station := range favourites {
		item := map[string]interface{}{
			"id": "radio:" + station.ID, "name": station.Name, "streamUrl": station.StreamURL,
		}
		if station.HomepageURL != "" {
			item["homepageUrl"] = station.HomepageURL
		}
		stations = append(stations, item)
	}
	return map[string]interface{}{
		"internetRadioStations": map[string]interface{}{"internetRadioStation": stations},
	}, nil
}
