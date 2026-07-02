package router

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type castToken struct {
	UserID, TrackID string
	ExpiresAt       time.Time
}

type outputDevice struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Protocol    string `json:"protocol"`
	Location    string `json:"-"`
	ControlURL  string `json:"-"`
	ServiceType string `json:"-"`
}

func (r *Router) createCastURL(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		TrackID string `json:"track_id"`
	}
	decoder := json.NewDecoder(io.LimitReader(req.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&payload) != nil || strings.TrimSpace(payload.TrackID) == "" {
		writeJSONError(w, http.StatusBadRequest, "Track ID is required")
		return
	}
	payload.TrackID = strings.TrimSpace(payload.TrackID)
	track, err := r.db.GetMusic(payload.TrackID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Track was not found")
		return
	}
	allowed, accessErr := r.requestCanAccessMusic(req, track)
	if accessErr != nil || !allowed {
		writeJSONError(w, http.StatusNotFound, "Track was not found")
		return
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Could not create cast URL")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	expiresAt := time.Now().Add(4 * time.Hour)
	r.castTokens.Store(token, castToken{UserID: requestUserID(req), TrackID: payload.TrackID, ExpiresAt: expiresAt})
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{
		"url":        fmt.Sprintf("%s/api/cast/%s/music/%s?transcode=true&format=mp3&bitrate=320", requestBaseURL(req), token, url.PathEscape(payload.TrackID)),
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
	}})
}

func (r *Router) streamCastMusic(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	value, ok := r.castTokens.Load(vars["token"])
	if !ok {
		http.Error(w, "Cast link is invalid", http.StatusUnauthorized)
		return
	}
	grant := value.(castToken)
	if time.Now().After(grant.ExpiresAt) || grant.TrackID != vars["id"] {
		r.castTokens.Delete(vars["token"])
		http.Error(w, "Cast link has expired", http.StatusUnauthorized)
		return
	}
	r.musicHandler.StreamMusic(w, req.WithContext(context.WithValue(req.Context(), "user_id", grant.UserID)))
}

func requestBaseURL(req *http.Request) string {
	scheme := strings.TrimSpace(strings.Split(req.Header.Get("X-Forwarded-Proto"), ",")[0])
	if scheme != "http" && scheme != "https" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(strings.Split(req.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = req.Host
	}
	return scheme + "://" + host
}

func (r *Router) discoverOutputDevices(w http.ResponseWriter, req *http.Request) {
	devices, err := discoverDLNARenderers(req.Context())
	if err != nil && len(devices) == 0 {
		writeJSONError(w, http.StatusBadGateway, "Could not discover media renderers")
		return
	}
	for _, device := range devices {
		r.outputDevices.Store(device.ID, device)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": devices})
}

func (r *Router) playOnDLNADevice(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		DeviceID string `json:"device_id"`
		MediaURL string `json:"media_url"`
		Title    string `json:"title"`
	}
	decoder := json.NewDecoder(io.LimitReader(req.Body, 128<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&payload) != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid DLNA playback request")
		return
	}
	value, ok := r.outputDevices.Load(strings.TrimSpace(payload.DeviceID))
	if !ok || !validHTTPURL(payload.MediaURL) {
		writeJSONError(w, http.StatusBadRequest, "Media renderer or URL is invalid")
		return
	}
	device := value.(outputDevice)
	if err := dlnaSetAndPlay(req.Context(), device, payload.MediaURL, payload.Title); err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": device})
}

func discoverDLNARenderers(ctx context.Context) ([]outputDevice, error) {
	connection, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	message := strings.Join([]string{"M-SEARCH * HTTP/1.1", "HOST: 239.255.255.250:1900", `MAN: "ssdp:discover"`, "MX: 2", "ST: urn:schemas-upnp-org:device:MediaRenderer:1", "", ""}, "\r\n")
	if _, err = connection.WriteToUDP([]byte(message), &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 1900}); err != nil {
		return nil, err
	}
	_ = connection.SetReadDeadline(time.Now().Add(2200 * time.Millisecond))
	locations := map[string]struct{}{}
	buffer := make([]byte, 64<<10)
	for {
		n, _, readErr := connection.ReadFromUDP(buffer)
		if readErr != nil {
			break
		}
		if location := parseSSDPHeaders(string(buffer[:n]))["location"]; validHTTPURL(location) {
			locations[location] = struct{}{}
		}
	}
	devices := make([]outputDevice, 0, len(locations))
	for location := range locations {
		if device, fetchErr := fetchDLNADevice(ctx, location); fetchErr == nil {
			devices = append(devices, device)
		}
	}
	return devices, nil
}

func parseSSDPHeaders(message string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(message, "\r\n") {
		if key, value, ok := strings.Cut(line, ":"); ok {
			result[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
		}
	}
	return result
}

type upnpDeviceDescription struct {
	Device struct {
		FriendlyName string `xml:"friendlyName"`
		Services     []struct {
			ServiceType string `xml:"serviceType"`
			ControlURL  string `xml:"controlURL"`
		} `xml:"serviceList>service"`
	} `xml:"device"`
}

func fetchDLNADevice(ctx context.Context, location string) (outputDevice, error) {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	response, err := (&http.Client{Timeout: 4 * time.Second}).Do(request)
	if err != nil {
		return outputDevice{}, err
	}
	defer response.Body.Close()
	var description upnpDeviceDescription
	if response.StatusCode/100 != 2 || xml.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&description) != nil {
		return outputDevice{}, fmt.Errorf("invalid renderer description")
	}
	base, _ := url.Parse(location)
	for _, service := range description.Device.Services {
		if strings.Contains(service.ServiceType, ":AVTransport:") {
			control, parseErr := base.Parse(service.ControlURL)
			if parseErr != nil {
				continue
			}
			digest := sha256.Sum256([]byte(location + service.ControlURL))
			return outputDevice{ID: hex.EncodeToString(digest[:8]), Name: strings.TrimSpace(description.Device.FriendlyName), Protocol: "dlna", Location: location, ControlURL: control.String(), ServiceType: service.ServiceType}, nil
		}
	}
	return outputDevice{}, fmt.Errorf("renderer has no AVTransport service")
}

func dlnaSetAndPlay(ctx context.Context, device outputDevice, mediaURL, title string) error {
	metadata := fmt.Sprintf(`<DIDL-Lite xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/"><item id="0" parentID="0" restricted="1"><dc:title>%s</dc:title><upnp:class>object.item.audioItem.musicTrack</upnp:class><res protocolInfo="http-get:*:audio/mpeg:*">%s</res></item></DIDL-Lite>`, html.EscapeString(title), html.EscapeString(mediaURL))
	setBody := fmt.Sprintf(`<u:SetAVTransportURI xmlns:u="%s"><InstanceID>0</InstanceID><CurrentURI>%s</CurrentURI><CurrentURIMetaData>%s</CurrentURIMetaData></u:SetAVTransportURI>`, device.ServiceType, html.EscapeString(mediaURL), html.EscapeString(metadata))
	if err := sendDLNASOAP(ctx, device, "SetAVTransportURI", setBody); err != nil {
		return err
	}
	return sendDLNASOAP(ctx, device, "Play", fmt.Sprintf(`<u:Play xmlns:u="%s"><InstanceID>0</InstanceID><Speed>1</Speed></u:Play>`, device.ServiceType))
}

func sendDLNASOAP(ctx context.Context, device outputDevice, action, body string) error {
	envelope := `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body>` + body + `</s:Body></s:Envelope>`
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, device.ControlURL, bytes.NewBufferString(envelope))
	request.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	request.Header.Set("SOAPACTION", fmt.Sprintf(`"%s#%s"`, device.ServiceType, action))
	response, err := (&http.Client{Timeout: 8 * time.Second}).Do(request)
	if err != nil {
		return fmt.Errorf("renderer did not respond: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("renderer rejected %s (%d)", action, response.StatusCode)
	}
	return nil
}
