package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"music-server/database"
)

func TestParseLRCSortsTimestampsAndAppliesOffset(t *testing.T) {
	lines, plain := parseLRC("[offset:+250]\n[00:02.50]Second\n[00:01.0][00:03.000]First\n[ar:Artist]")
	if len(lines) != 3 {
		t.Fatalf("expected 3 timed lines, got %#v", lines)
	}
	if lines[0].TimeMS != 1250 || lines[0].Text != "First" || lines[1].TimeMS != 2750 || lines[2].TimeMS != 3250 {
		t.Fatalf("unexpected parsed lines: %#v", lines)
	}
	if plain != "Second\nFirst" {
		t.Fatalf("unexpected plain lyrics: %q", plain)
	}
}

func TestLoadLocalLyricsPrefersLRCOverText(t *testing.T) {
	directory := t.TempDir()
	audioPath := filepath.Join(directory, "track.mp3")
	if err := os.WriteFile(filepath.Join(directory, "track.lrc"), []byte("[00:01.00]Synced"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "track.txt"), []byte("Plain"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, found := loadLocalLyrics(&database.Music{FilePath: audioPath})
	if !found || !result.Synced || result.Source != "local" || len(result.Lines) != 1 {
		t.Fatalf("unexpected local lyrics result: %#v", result)
	}
}

func TestFetchLRCLIBLyricsParsesSyncedFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("track_name") != "Test Song" || req.Header.Get("User-Agent") == "" {
			t.Fatalf("unexpected provider request: %s", req.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"instrumental":false,"plainLyrics":"","syncedLyrics":"[00:01.00]Hello\n[00:02.50]World"}`))
	}))
	defer server.Close()
	previousURL := lyricsProviderURL
	lyricsProviderURL = server.URL
	defer func() { lyricsProviderURL = previousURL }()

	request := httptest.NewRequest(http.MethodGet, "/api/music/test/lyrics", nil)
	result, found := fetchLRCLIBLyrics(request, &database.Music{Title: "Test Song", Artist: "Artist", Album: "Album", Duration: 180})
	if !found || !result.Synced || result.PlainText != "Hello\nWorld" || len(result.Lines) != 2 {
		t.Fatalf("unexpected provider lyrics: %#v", result)
	}
}
