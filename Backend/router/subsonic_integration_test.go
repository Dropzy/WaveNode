//go:build integration

package router

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"music-server/auth"
	"music-server/database"
	"music-server/handlers"
	"music-server/scanner"
	"music-server/websocket"

	_ "github.com/lib/pq"
)

const (
	integrationUsername = "subsonic-admin"
	integrationPassword = "integration-password"
)

type subsonicIntegration struct {
	server     *httptest.Server
	db         *database.DB
	adminDB    *sql.DB
	database   string
	adminURL   string
	musicDir   string
	artworkDir string
	trackOne   database.Music
	trackTwo   database.Music
	artist     database.Artist
	albumID    string
}

func TestSubsonicIntegration(t *testing.T) {
	env := newSubsonicIntegration(t)

	t.Run("authentication", env.testAuthentication)
	t.Run("media", env.testMedia)
	t.Run("playlists", env.testPlaylists)
	t.Run("ratings and stars", env.testRatingsAndStars)
	t.Run("bookmarks and queue", env.testBookmarksAndQueue)
	t.Run("user administration", env.testUserAdministration)
	t.Run("endpoint coverage", env.testEndpointCoverage)
}

func newSubsonicIntegration(t *testing.T) *subsonicIntegration {
	t.Helper()
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	backendRoot := filepath.Clean(filepath.Join(originalWorkingDirectory, ".."))
	if err := os.Chdir(backendRoot); err != nil {
		t.Fatalf("switch to backend root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWorkingDirectory)
	})

	adminURL := os.Getenv("SUBSONIC_TEST_DATABASE_URL")
	if adminURL == "" {
		t.Fatal("SUBSONIC_TEST_DATABASE_URL is required; run scripts/test-subsonic.ps1")
	}
	adminDB, err := sql.Open("postgres", adminURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := adminDB.Ping(); err != nil {
		t.Fatalf("connect to integration PostgreSQL: %v", err)
	}
	databaseName := "wavenode_subsonic_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if _, err := adminDB.Exec(`CREATE DATABASE ` + quoteIdentifier(databaseName)); err != nil {
		t.Fatalf("create temporary database: %v", err)
	}
	databaseURL := databaseURLWithName(t, adminURL, databaseName)
	db, err := database.NewDB(database.Config{ConnectionString: databaseURL})
	if err != nil {
		dropIntegrationDatabase(adminDB, databaseName)
		t.Fatalf("initialize temporary database: %v", err)
	}

	root := t.TempDir()
	musicDir := filepath.Join(root, "music")
	artworkDir := filepath.Join(root, "artwork")
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artworkDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WAVENODE_ARTWORK_PATH", artworkDir)

	artist := database.Artist{
		ID:             database.GenerateArtistHash("Integration Artist"),
		Name:           "Integration Artist",
		Biography:      "Synthetic integration-test artist",
		MusicBrainzID:  "integration-mbid",
		ImageURL:       "artist.jpg",
		ImageSmallURL:  "artist.jpg",
		ImageMediumURL: "artist.jpg",
		ImageLargeURL:  "artist.jpg",
	}
	if err := db.AddArtist(&artist); err != nil {
		t.Fatalf("seed artist: %v", err)
	}

	audioOne := append([]byte("ID3"), bytes.Repeat([]byte{0x11, 0x22, 0x33, 0x44}, 2048)...)
	audioTwo := append([]byte("ID3"), bytes.Repeat([]byte{0x55, 0x66, 0x77, 0x7f}, 1024)...)
	pathOne := filepath.Join(musicDir, "one.mp3")
	pathTwo := filepath.Join(musicDir, "two.mp3")
	if err := os.WriteFile(pathOne, audioOne, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathTwo, audioTwo, 0644); err != nil {
		t.Fatal(err)
	}
	cover := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0xff, 0xd9}
	if err := os.WriteFile(filepath.Join(artworkDir, "cover.jpg"), cover, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artworkDir, "artist.jpg"), cover, 0644); err != nil {
		t.Fatal(err)
	}

	trackOne := database.Music{
		ID: "song-one", Title: "First Song", Artist: artist.Name, ArtistID: artist.ID,
		Album: "Integration Album", Genre: "Electronic", Duration: 180,
		FilePath: pathOne, FileName: "one.mp3", FileSize: int64(len(audioOne)),
		Format: "mp3", Year: 2026, TrackNumber: 1, HasMetadata: true,
		CoverArtURL: "cover.jpg", CoverArtSmallURL: "cover.jpg",
		CoverArtMediumURL: "cover.jpg", CoverArtLargeURL: "cover.jpg",
	}
	trackTwo := database.Music{
		ID: "song-two", Title: "Second Song", Artist: artist.Name, ArtistID: artist.ID,
		Album: "Integration Album", Genre: "Electronic", Duration: 210,
		FilePath: pathTwo, FileName: "two.mp3", FileSize: int64(len(audioTwo)),
		Format: "mp3", Year: 2026, TrackNumber: 2, HasMetadata: true,
		CoverArtURL: "cover.jpg",
	}
	if err := db.AddMusic(&trackOne); err != nil {
		t.Fatalf("seed first song: %v", err)
	}
	if err := db.AddMusic(&trackTwo); err != nil {
		t.Fatalf("seed second song: %v", err)
	}
	if _, err := db.AddMusicSource(musicDir); err != nil {
		t.Fatalf("seed music source: %v", err)
	}
	user := &database.User{ID: "integration-admin", Username: integrationUsername, Email: "subsonic@example.test", Role: "admin"}
	if err := db.CreateUser(user, integrationPassword); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	pluginManifest := json.RawMessage(`{
		"schema_version":1,
		"id":"integration.radio",
		"name":"Integration Radio",
		"version":"1.0.0",
		"permissions":["network","playback"],
		"contributes":{"home_rows":[{
			"id":"stations",
			"title":"Integration stations",
			"type":"radio",
			"items":[{
				"id":"station-one",
				"title":"Station One",
				"stream_url":"https://radio.example.test/live.mp3",
				"homepage_url":"https://radio.example.test"
			}]
		}]}
	}`)
	if err := db.UpsertPlugin(database.Plugin{
		ID: "integration.radio", Name: "Integration Radio", Version: "1.0.0",
		Enabled: true, Manifest: pluginManifest,
	}); err != nil {
		t.Fatalf("seed plugin: %v", err)
	}

	authHandler := auth.NewAuthHandler(db, []byte("integration-jwt-secret"))
	wsManager := websocket.NewWebSocketManager(authHandler)
	musicHandler := handlers.NewMusicHandler(db)
	playlistHandler := handlers.NewPlaylistHandler(db)
	appRouter := NewRouter(authHandler, musicHandler, playlistHandler, wsManager, db, struct {
		AllowedOrigins []string `json:"allowed_origins"`
		AllowedMethods []string `json:"allowed_methods"`
		AllowedHeaders []string `json:"allowed_headers"`
	}{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST"}, AllowedHeaders: []string{"Content-Type"}})
	testScanner := scanner.NewScanner(db, musicDir)
	handlers.InitScanner(testScanner)
	server := httptest.NewServer(appRouter.SetupRoutes())

	env := &subsonicIntegration{
		server: server, db: db, adminDB: adminDB, database: databaseName, adminURL: adminURL,
		musicDir: musicDir, artworkDir: artworkDir, trackOne: trackOne, trackTwo: trackTwo,
		artist: artist, albumID: albumIDForTrack(trackOne),
	}
	t.Cleanup(func() {
		server.Close()
		handlers.InitScanner(nil)
		_ = db.Close()
		dropIntegrationDatabase(adminDB, databaseName)
		_ = adminDB.Close()
	})
	return env
}

func (env *subsonicIntegration) testAuthentication(t *testing.T) {
	t.Run("clear password JSON", func(t *testing.T) {
		response := env.call(t, "ping", nil, authClear, "json")
		assertSubsonicOK(t, response)
	})
	t.Run("hex password XML", func(t *testing.T) {
		response := env.rawCall(t, "ping", nil, authHex, "xml", nil)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", response.StatusCode)
		}
		var document struct {
			Status string `xml:"status,attr"`
		}
		if err := xml.Unmarshal(response.Body, &document); err != nil {
			t.Fatalf("decode XML: %v", err)
		}
		if document.Status != "ok" {
			t.Fatalf("XML status = %q", document.Status)
		}
	})
	t.Run("wrong password", func(t *testing.T) {
		values := url.Values{"u": {integrationUsername}, "p": {"wrong"}, "v": {"1.16.1"}, "c": {"integration"}, "f": {"json"}}
		response := env.rawRequest(t, "ping", values, nil)
		assertSubsonicErrorCode(t, decodeSubsonicJSON(t, response.Body), 40)
	})
	t.Run("missing parameters", func(t *testing.T) {
		response := env.rawRequest(t, "ping", url.Values{"f": {"json"}}, nil)
		assertSubsonicErrorCode(t, decodeSubsonicJSON(t, response.Body), 10)
	})
	t.Run("unsupported version", func(t *testing.T) {
		values := env.authValues(authClear, "json")
		values.Set("v", "1.17.0")
		response := env.rawRequest(t, "ping", values, nil)
		assertSubsonicErrorCode(t, decodeSubsonicJSON(t, response.Body), 30)
	})
	t.Run("token authentication rejection", func(t *testing.T) {
		values := url.Values{"u": {integrationUsername}, "t": {"hash"}, "s": {"salt"}, "v": {"1.16.1"}, "c": {"integration"}, "f": {"json"}}
		response := env.rawRequest(t, "ping", values, nil)
		assertSubsonicErrorCode(t, decodeSubsonicJSON(t, response.Body), 41)
	})
}

func (env *subsonicIntegration) testEndpointCoverage(t *testing.T) {
	readEndpoints := []struct {
		method string
		values url.Values
	}{
		{"ping", nil}, {"getLicense", nil}, {"getOpenSubsonicExtensions", nil},
		{"getMusicFolders", nil}, {"getGenres", nil}, {"getArtists", nil}, {"getIndexes", nil},
		{"getArtist", url.Values{"id": {env.artist.ID}}},
		{"getAlbum", url.Values{"id": {env.albumID}}},
		{"getSong", url.Values{"id": {env.trackOne.ID}}},
		{"getMusicDirectory", url.Values{"id": {env.artist.ID}}},
		{"getMusicDirectory", url.Values{"id": {env.albumID}}},
		{"getAlbumList", url.Values{"type": {"alphabeticalByName"}}},
		{"getAlbumList2", url.Values{"type": {"recent"}}},
		{"getRandomSongs", url.Values{"size": {"2"}}},
		{"getSongsByGenre", url.Values{"genre": {"Electronic"}}},
		{"getNowPlaying", nil}, {"getStarred", nil}, {"getStarred2", nil},
		{"search", url.Values{"query": {"First"}}},
		{"search2", url.Values{"query": {"First"}}},
		{"search3", url.Values{"query": {"First"}}},
		{"getTopSongs", url.Values{"artist": {env.artist.Name}}},
		{"getSimilarSongs", url.Values{"id": {env.trackOne.ID}}},
		{"getSimilarSongs2", url.Values{"id": {env.trackOne.ID}}},
		{"getArtistInfo", url.Values{"id": {env.artist.ID}}},
		{"getArtistInfo2", url.Values{"id": {env.artist.ID}}},
		{"getAlbumInfo", url.Values{"id": {env.albumID}}},
		{"getAlbumInfo2", url.Values{"id": {env.albumID}}},
		{"getVideos", nil},
		{"getLyrics", url.Values{"artist": {env.artist.Name}, "title": {env.trackOne.Title}}},
		{"getLyricsBySongId", url.Values{"id": {env.trackOne.ID}}},
		{"getBookmarks", nil}, {"getPlayQueue", nil},
		{"getUsers", nil}, {"getUser", url.Values{"username": {integrationUsername}}},
		{"getShares", nil}, {"getPodcasts", nil}, {"getNewestPodcasts", nil},
		{"getInternetRadioStations", nil}, {"getChatMessages", nil},
		{"getScanStatus", nil},
	}
	for _, endpoint := range readEndpoints {
		name := endpoint.method
		if endpoint.values.Get("id") != "" {
			name += "/" + endpoint.values.Get("id")
		}
		t.Run(name, func(t *testing.T) {
			assertSubsonicOK(t, env.call(t, endpoint.method, endpoint.values, authClear, "json"))
		})
	}

	radio := env.call(t, "getInternetRadioStations", nil, authClear, "json")
	stations := asSlice(radio["internetRadioStations"].(map[string]interface{})["internetRadioStation"])
	if len(stations) != 1 {
		t.Fatalf("internet radio station count = %d, want 1", len(stations))
	}
	station := stations[0].(map[string]interface{})
	if station["id"] != "integration.radio:station-one" || station["streamUrl"] != "https://radio.example.test/live.mp3" {
		t.Fatalf("unexpected internet radio station: %#v", station)
	}

	unsupported := []struct {
		method string
		code   int
	}{
		{"tokenInfo", 41}, {"getVideoInfo", 70}, {"getAvatar", 70},
		{"hls", 0}, {"getCaptions", 0}, {"jukeboxControl", 50},
		{"createShare", 50}, {"updateShare", 50}, {"deleteShare", 50},
		{"refreshPodcasts", 50}, {"createPodcastChannel", 50}, {"deletePodcastChannel", 50},
		{"deletePodcastEpisode", 50}, {"downloadPodcastEpisode", 50},
		{"createInternetRadioStation", 50}, {"updateInternetRadioStation", 50},
		{"deleteInternetRadioStation", 50}, {"addChatMessage", 50},
	}
	for _, endpoint := range unsupported {
		t.Run(endpoint.method, func(t *testing.T) {
			assertSubsonicErrorCode(t, env.call(t, endpoint.method, nil, authClear, "json"), endpoint.code)
		})
	}

	t.Run("startScan", func(t *testing.T) {
		assertSubsonicOK(t, env.call(t, "startScan", nil, authClear, "json"))
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			response := env.call(t, "getScanStatus", nil, authClear, "json")
			assertSubsonicOK(t, response)
			scan := response["scanStatus"].(map[string]interface{})
			if scanning, _ := scan["scanning"].(bool); !scanning {
				return
			}
			time.Sleep(25 * time.Millisecond)
		}
		t.Fatal("integration scan did not finish")
	})
}

func (env *subsonicIntegration) testPlaylists(t *testing.T) {
	created := env.call(t, "createPlaylist", url.Values{
		"name": {"Integration Playlist"}, "songId": {env.trackOne.ID},
	}, authClear, "json")
	assertSubsonicOK(t, created)
	playlist := created["playlist"].(map[string]interface{})
	playlistID := fmt.Sprint(playlist["id"])
	if playlist["songCount"].(float64) != 1 {
		t.Fatalf("created songCount = %v", playlist["songCount"])
	}

	env.assertPlaylistTrackIDs(t, playlistID, []string{env.trackOne.ID})
	assertSubsonicOK(t, env.call(t, "updatePlaylist", url.Values{
		"playlistId": {playlistID}, "name": {"Renamed Playlist"}, "comment": {"updated"},
		"songIdToAdd": {env.trackTwo.ID},
	}, authClear, "json"))
	env.assertPlaylistTrackIDs(t, playlistID, []string{env.trackOne.ID, env.trackTwo.ID})

	assertSubsonicOK(t, env.call(t, "updatePlaylist", url.Values{
		"playlistId": {playlistID}, "songIndexToRemove": {"0"},
	}, authClear, "json"))
	env.assertPlaylistTrackIDs(t, playlistID, []string{env.trackTwo.ID})

	list := env.call(t, "getPlaylists", nil, authClear, "json")
	assertSubsonicOK(t, list)
	root := list["playlists"].(map[string]interface{})
	if root["lastModified"].(float64) <= 0 {
		t.Fatal("getPlaylists did not include a revision")
	}
	assertSubsonicOK(t, env.call(t, "deletePlaylist", url.Values{"id": {playlistID}}, authClear, "json"))
	assertSubsonicErrorCode(t, env.call(t, "getPlaylist", url.Values{"id": {playlistID}}, authClear, "json"), 70)

	smart := &database.Playlist{
		UserID: "integration-admin",
		Name:   "Smart Integration Playlist",
		Type:   database.PlaylistTypeSmart,
		SmartRules: &database.SmartPlaylistRules{
			Match: "all", SortBy: "title", SortDirection: "asc", Limit: 50,
			Conditions: []database.SmartPlaylistCondition{
				{Field: "artist", Operator: "contains", Value: env.trackOne.Artist},
			},
		},
	}
	if err := env.db.AddPlaylist(smart); err != nil {
		t.Fatalf("create smart playlist: %v", err)
	}
	smartResponse := env.call(t, "getPlaylist", url.Values{"id": {smart.ID}}, authClear, "json")
	assertSubsonicOK(t, smartResponse)
	if smartResponse["playlist"].(map[string]interface{})["songCount"].(float64) < 1 {
		t.Fatal("smart playlist was not exposed as a playable Subsonic snapshot")
	}
	assertSubsonicErrorCode(t, env.call(t, "updatePlaylist", url.Values{
		"playlistId": {smart.ID}, "songIdToAdd": {env.trackTwo.ID},
	}, authClear, "json"), 50)
	assertSubsonicErrorCode(t, env.call(t, "deletePlaylist", url.Values{"id": {smart.ID}}, authClear, "json"), 50)
}

func (env *subsonicIntegration) testRatingsAndStars(t *testing.T) {
	for _, rating := range []string{"1", "5", "0"} {
		assertSubsonicOK(t, env.call(t, "setRating", url.Values{"id": {env.trackOne.ID}, "rating": {rating}}, authClear, "json"))
		song := env.call(t, "getSong", url.Values{"id": {env.trackOne.ID}}, authClear, "json")["song"].(map[string]interface{})
		if rating == "0" {
			if _, exists := song["userRating"]; exists {
				t.Fatal("rating 0 did not clear userRating")
			}
			if _, exists := song["averageRating"]; exists {
				t.Fatal("rating 0 did not clear averageRating")
			}
		} else if fmt.Sprint(song["userRating"]) != rating {
			t.Fatalf("userRating = %v, want %s", song["userRating"], rating)
		} else if fmt.Sprint(song["averageRating"]) != rating {
			t.Fatalf("averageRating = %v, want %s", song["averageRating"], rating)
		}
	}
	assertSubsonicErrorCode(t, env.call(t, "setRating", url.Values{"id": {env.trackOne.ID}, "rating": {"6"}}, authClear, "json"), 10)

	assertSubsonicOK(t, env.call(t, "star", url.Values{
		"id": {env.trackOne.ID}, "albumId": {env.albumID}, "artistId": {env.artist.ID},
	}, authClear, "json"))
	starred := env.call(t, "getStarred2", nil, authClear, "json")["starred2"].(map[string]interface{})
	if len(asSlice(starred["song"])) != 1 || len(asSlice(starred["album"])) != 1 || len(asSlice(starred["artist"])) != 1 {
		t.Fatalf("unexpected starred response: %#v", starred)
	}
	assertSubsonicOK(t, env.call(t, "unstar", url.Values{
		"id": {env.trackOne.ID}, "albumId": {env.albumID}, "artistId": {env.artist.ID},
	}, authClear, "json"))

	assertSubsonicOK(t, env.call(t, "scrobble", url.Values{"id": {env.trackOne.ID}}, authClear, "json"))
	track, err := env.db.GetMusic(env.trackOne.ID)
	if err != nil || track.PlayCount < 1 {
		t.Fatalf("scrobble did not increment play count: %#v, %v", track, err)
	}
}

func (env *subsonicIntegration) testBookmarksAndQueue(t *testing.T) {
	assertSubsonicOK(t, env.call(t, "createBookmark", url.Values{
		"id": {env.trackOne.ID}, "position": {"12345"}, "comment": {"resume"},
	}, authClear, "json"))
	bookmarks := env.call(t, "getBookmarks", nil, authClear, "json")["bookmarks"].(map[string]interface{})
	bookmark := firstMap(t, bookmarks["bookmark"])
	if bookmark["position"].(float64) != 12345 || bookmark["comment"] != "resume" {
		t.Fatalf("unexpected bookmark: %#v", bookmark)
	}
	assertSubsonicOK(t, env.call(t, "deleteBookmark", url.Values{"id": {env.trackOne.ID}}, authClear, "json"))
	if len(asSlice(env.call(t, "getBookmarks", nil, authClear, "json")["bookmarks"].(map[string]interface{})["bookmark"])) != 0 {
		t.Fatal("bookmark was not deleted")
	}

	assertSubsonicOK(t, env.call(t, "savePlayQueue", url.Values{
		"id": {env.trackOne.ID, env.trackTwo.ID}, "current": {env.trackTwo.ID}, "position": {"6789"},
	}, authClear, "json"))
	queue := env.call(t, "getPlayQueue", nil, authClear, "json")["playQueue"].(map[string]interface{})
	if queue["current"] != env.trackTwo.ID || queue["position"].(float64) != 6789 {
		t.Fatalf("unexpected play queue: %#v", queue)
	}
	entries := asSlice(queue["entry"])
	if len(entries) != 2 {
		t.Fatalf("queue entries = %d", len(entries))
	}
}

func (env *subsonicIntegration) testMedia(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		response := env.rawCall(t, "stream", url.Values{"id": {env.trackOne.ID}}, authClear, "json", nil)
		if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "audio/mpeg" {
			t.Fatalf("stream status/type = %d/%s", response.StatusCode, response.Header.Get("Content-Type"))
		}
		if int64(len(response.Body)) != env.trackOne.FileSize || !bytes.HasPrefix(response.Body, []byte("ID3")) {
			t.Fatalf("unexpected stream body: %d bytes", len(response.Body))
		}
	})
	t.Run("range seeking", func(t *testing.T) {
		response := env.rawCall(t, "stream", url.Values{"id": {env.trackOne.ID}}, authClear, "json", http.Header{"Range": {"bytes=100-1099"}})
		if response.StatusCode != http.StatusPartialContent {
			t.Fatalf("range status = %d", response.StatusCode)
		}
		if len(response.Body) != 1000 || response.Header.Get("Content-Range") != fmt.Sprintf("bytes 100-1099/%d", env.trackOne.FileSize) {
			t.Fatalf("unexpected range response: %d, %s", len(response.Body), response.Header.Get("Content-Range"))
		}
	})
	t.Run("download", func(t *testing.T) {
		response := env.rawCall(t, "download", url.Values{"id": {env.trackTwo.ID}}, authClear, "json", nil)
		if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Content-Disposition"), "attachment") {
			t.Fatalf("download headers = %#v", response.Header)
		}
	})
	t.Run("cover art", func(t *testing.T) {
		response := env.rawCall(t, "getCoverArt", url.Values{"id": {env.trackOne.ID}}, authClear, "json", nil)
		if response.StatusCode != http.StatusOK || !bytes.HasPrefix(response.Body, []byte{0xff, 0xd8}) {
			t.Fatalf("unexpected cover art: status=%d type=%s body=%q bytes=%x", response.StatusCode, response.Header.Get("Content-Type"), response.Body, response.Body)
		}
	})
	t.Run("missing stream", func(t *testing.T) {
		response := env.rawCall(t, "stream", url.Values{"id": {"missing"}}, authClear, "json", nil)
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("missing stream status = %d", response.StatusCode)
		}
		assertSubsonicErrorCode(t, decodeSubsonicJSON(t, response.Body), 70)
	})
}

func (env *subsonicIntegration) testUserAdministration(t *testing.T) {
	assertSubsonicOK(t, env.call(t, "createUser", url.Values{
		"username": {"integration-user"}, "password": {"user-password"},
		"email": {"user@example.test"}, "adminRole": {"false"},
	}, authClear, "json"))
	userResponse := env.call(t, "getUser", url.Values{"username": {"integration-user"}}, authClear, "json")
	assertSubsonicOK(t, userResponse)
	assertSubsonicOK(t, env.call(t, "updateUser", url.Values{
		"username": {"integration-user"}, "adminRole": {"true"},
	}, authClear, "json"))
	assertSubsonicOK(t, env.call(t, "changePassword", url.Values{
		"username": {"integration-user"},
		"password": {"enc:" + hex.EncodeToString([]byte("changed-password"))},
	}, authClear, "json"))
	if _, err := env.db.ValidatePassword("integration-user", "changed-password"); err != nil {
		t.Fatalf("password was not changed: %v", err)
	}
	assertSubsonicOK(t, env.call(t, "deleteUser", url.Values{"username": {"integration-user"}}, authClear, "json"))
	if _, err := env.db.GetUserByUsername("integration-user"); err == nil {
		t.Fatal("user was not deleted")
	}

	assertSubsonicErrorCode(t, env.call(t, "updateUser", url.Values{
		"username": {integrationUsername}, "adminRole": {"false"},
	}, authClear, "json"), 50)
	assertSubsonicErrorCode(t, env.call(t, "deleteUser", url.Values{
		"username": {integrationUsername},
	}, authClear, "json"), 50)

	secondary := &database.User{
		ID: "secondary-admin", Username: "secondary-admin",
		Email: "secondary-admin@example.test", Role: "admin",
	}
	if err := env.db.CreateUser(secondary, "secondary-password"); err != nil {
		t.Fatalf("create secondary administrator: %v", err)
	}
	if err := env.db.UpdateUserRole(secondary.ID, "user"); err != nil {
		t.Fatalf("demote administrator when another remains: %v", err)
	}
	if err := env.db.UpdateUserRole(secondary.ID, "admin"); err != nil {
		t.Fatalf("restore secondary administrator: %v", err)
	}
	if err := env.db.DeleteUser(secondary.ID); err != nil {
		t.Fatalf("delete administrator when another remains: %v", err)
	}
}

type authMode int

const (
	authClear authMode = iota
	authHex
)

type rawSubsonicResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (env *subsonicIntegration) call(t *testing.T, method string, values url.Values, mode authMode, format string) map[string]interface{} {
	t.Helper()
	response := env.rawCall(t, method, values, mode, format, nil)
	if response.StatusCode >= 500 {
		t.Fatalf("%s returned HTTP %d: %s", method, response.StatusCode, response.Body)
	}
	return decodeSubsonicJSON(t, response.Body)
}

func (env *subsonicIntegration) rawCall(t *testing.T, method string, values url.Values, mode authMode, format string, headers http.Header) rawSubsonicResponse {
	t.Helper()
	query := env.authValues(mode, format)
	for key, entries := range values {
		for _, value := range entries {
			query.Add(key, value)
		}
	}
	return env.rawRequestWithHeaders(t, method, query, headers)
}

func (env *subsonicIntegration) rawRequest(t *testing.T, method string, values url.Values, headers http.Header) rawSubsonicResponse {
	t.Helper()
	return env.rawRequestWithHeaders(t, method, values, headers)
}

func (env *subsonicIntegration) rawRequestWithHeaders(t *testing.T, method string, values url.Values, headers http.Header) rawSubsonicResponse {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, env.server.URL+"/rest/"+method+".view?"+values.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for key, entries := range headers {
		for _, value := range entries {
			request.Header.Add(key, value)
		}
	}
	response, err := env.server.Client().Do(request)
	if err != nil {
		t.Fatalf("%s request failed: %v", method, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return rawSubsonicResponse{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: body}
}

func (env *subsonicIntegration) authValues(mode authMode, format string) url.Values {
	password := integrationPassword
	if mode == authHex {
		password = "enc:" + hex.EncodeToString([]byte(password))
	}
	return url.Values{
		"u": {integrationUsername}, "p": {password}, "v": {"1.16.1"},
		"c": {"wavenode-integration"}, "f": {format},
	}
}

func (env *subsonicIntegration) assertPlaylistTrackIDs(t *testing.T, playlistID string, expected []string) {
	t.Helper()
	response := env.call(t, "getPlaylist", url.Values{"id": {playlistID}}, authClear, "json")
	assertSubsonicOK(t, response)
	entries := asSlice(response["playlist"].(map[string]interface{})["entry"])
	actual := make([]string, 0, len(entries))
	for _, entry := range entries {
		actual = append(actual, entry.(map[string]interface{})["id"].(string))
	}
	if strings.Join(actual, ",") != strings.Join(expected, ",") {
		t.Fatalf("playlist tracks = %v, want %v", actual, expected)
	}
}

func decodeSubsonicJSON(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var document map[string]map[string]interface{}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("decode Subsonic JSON: %v\n%s", err, body)
	}
	response := document["subsonic-response"]
	if response == nil {
		t.Fatalf("missing subsonic-response: %s", body)
	}
	return response
}

func assertSubsonicOK(t *testing.T, response map[string]interface{}) {
	t.Helper()
	if response["status"] != "ok" {
		t.Fatalf("Subsonic response failed: %#v", response)
	}
}

func assertSubsonicErrorCode(t *testing.T, response map[string]interface{}, expected int) {
	t.Helper()
	if response["status"] != "failed" {
		t.Fatalf("expected failed response: %#v", response)
	}
	errorValue, ok := response["error"].(map[string]interface{})
	if !ok || int(errorValue["code"].(float64)) != expected {
		t.Fatalf("error = %#v, want code %d", response["error"], expected)
	}
}

func firstMap(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()
	items := asSlice(value)
	if len(items) == 0 {
		t.Fatal("expected at least one item")
	}
	return items[0].(map[string]interface{})
}

func asSlice(value interface{}) []interface{} {
	if value == nil {
		return []interface{}{}
	}
	if items, ok := value.([]interface{}); ok {
		return items
	}
	return []interface{}{value}
}

func databaseURLWithName(t *testing.T, rawURL, databaseName string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + databaseName
	return parsed.String()
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func dropIntegrationDatabase(adminDB *sql.DB, databaseName string) {
	_, _ = adminDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, databaseName)
	_, _ = adminDB.Exec(`DROP DATABASE IF EXISTS ` + quoteIdentifier(databaseName))
}
