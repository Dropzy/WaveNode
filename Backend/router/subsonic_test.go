package router

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"music-server/database"
)

func TestDecodeSubsonicPassword(t *testing.T) {
	clear, err := decodeSubsonicPassword("secret")
	if err != nil || clear != "secret" {
		t.Fatalf("clear password = %q, %v", clear, err)
	}
	encoded, err := decodeSubsonicPassword("enc:736563726574")
	if err != nil || encoded != "secret" {
		t.Fatalf("encoded password = %q, %v", encoded, err)
	}
}

func TestSupportedSubsonicVersion(t *testing.T) {
	for _, version := range []string{"1.8.0", "1.13.0", "1.16.1"} {
		if !supportedSubsonicVersion(version) {
			t.Fatalf("expected %s to be supported", version)
		}
	}
	for _, version := range []string{"2.0.0", "1.17.0", "invalid"} {
		if supportedSubsonicVersion(version) {
			t.Fatalf("expected %s to be rejected", version)
		}
	}
}

func TestSubsonicCredentialCacheKeyDoesNotExposePassword(t *testing.T) {
	key := subsonicCredentialCacheKey("admin", "secret-password")
	if strings.Contains(key, "admin") || strings.Contains(key, "secret-password") {
		t.Fatalf("credential cache key exposed credentials: %s", key)
	}
	if key != subsonicCredentialCacheKey("admin", "secret-password") {
		t.Fatal("credential cache key must be deterministic")
	}
}

func TestWriteSubsonicJSONResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeSubsonicResponse(recorder, "json", map[string]interface{}{
		"license": map[string]interface{}{"valid": true},
	})
	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	response := payload["subsonic-response"]
	if response["status"] != "ok" || response["version"] != subsonicAPIVersion {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestWriteSubsonicXMLResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeSubsonicResponse(recorder, "xml", map[string]interface{}{
		"musicFolders": map[string]interface{}{
			"musicFolder": []interface{}{map[string]interface{}{"id": 1, "name": "Music"}},
		},
	})
	body := recorder.Body.Bytes()
	if !bytes.Contains(body, []byte(`<musicFolder id="1" name="Music"></musicFolder>`)) {
		t.Fatalf("missing music folder element: %s", body)
	}
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	for {
		if _, err := decoder.Token(); err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("invalid XML response: %v", err)
		}
	}
}

func TestSubsonicSongWithoutAlbumHasNoParent(t *testing.T) {
	song := subsonicSongMap(database.Music{ID: "track-1", Title: "Loose Track"})
	if _, exists := song["parent"]; exists {
		t.Fatal("track without album metadata must not advertise an album parent")
	}
}

func TestApplySubsonicRatings(t *testing.T) {
	response := map[string]interface{}{
		"album": map[string]interface{}{
			"id": "album-1",
			"song": []interface{}{
				map[string]interface{}{"id": "song-1"},
				map[string]interface{}{"id": "song-2"},
			},
		},
	}
	applySubsonicRatings(
		response,
		map[string]int{"album-1": 4, "song-2": 5},
		map[string]float64{"album-1": 3.5, "song-1": 2.5, "song-2": 4.25},
	)
	album := response["album"].(map[string]interface{})
	if album["userRating"] != 4 {
		t.Fatalf("album rating = %#v", album["userRating"])
	}
	if album["averageRating"] != 3.5 {
		t.Fatalf("album average rating = %#v", album["averageRating"])
	}
	songs := album["song"].([]interface{})
	if _, exists := songs[0].(map[string]interface{})["userRating"]; exists {
		t.Fatal("unrated song should not include userRating")
	}
	if songs[0].(map[string]interface{})["averageRating"] != 2.5 {
		t.Fatalf("song average rating = %#v", songs[0].(map[string]interface{})["averageRating"])
	}
	if songs[1].(map[string]interface{})["userRating"] != 5 {
		t.Fatalf("song rating = %#v", songs[1].(map[string]interface{})["userRating"])
	}
}

func TestSubsonicResponseIncludesMedia(t *testing.T) {
	if !subsonicResponseIncludesMedia("getAlbum") {
		t.Fatal("getAlbum should include ratings")
	}
	if subsonicResponseIncludesMedia("ping") {
		t.Fatal("ping must not load media ratings")
	}
}

func TestSubsonicPlaylistPreservesSubsecondChangedTime(t *testing.T) {
	changed := time.Date(2026, time.June, 14, 12, 30, 45, 123456789, time.UTC)
	playlist := subsonicPlaylistMap(database.Playlist{
		ID:        "playlist-1",
		Name:      "Test",
		CreatedAt: changed,
		UpdatedAt: changed,
	}, "user", nil)
	if playlist["changed"] != "2026-06-14T12:30:45.123456789Z" {
		t.Fatalf("changed timestamp lost precision: %v", playlist["changed"])
	}
}

func TestPlaylistRevisionUsesLatestUpdate(t *testing.T) {
	playlists := []database.Playlist{
		{UpdatedAt: time.UnixMilli(1000)},
		{UpdatedAt: time.UnixMilli(2500)},
	}
	if revision := playlistRevision(playlists); revision != 2500 {
		t.Fatalf("playlist revision = %d", revision)
	}
}

func TestSubsonicAudioContentType(t *testing.T) {
	tests := map[string]string{
		"mp3":  "audio/mpeg",
		"flac": "audio/flac",
		"m4a":  "audio/mp4",
		"ogg":  "audio/ogg",
		"opus": "audio/opus",
		"wav":  "audio/wav",
	}
	for format, expected := range tests {
		if actual := subsonicAudioContentType(format, ""); actual != expected {
			t.Fatalf("%s content type = %s, want %s", format, actual, expected)
		}
	}
}
