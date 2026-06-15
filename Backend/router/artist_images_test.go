package router

import (
	"os"
	"path/filepath"
	"testing"

	"music-server/database"
)

func TestIsLocalArtistImage(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "/artwork/artist.jpg", want: true},
		{value: "artwork/artist.jpg", want: true},
		{value: "https://example.com/artist.jpg", want: false},
		{value: "http://example.com/artist.jpg", want: false},
		{value: "", want: false},
	}

	for _, test := range tests {
		if got := isLocalArtistImage(test.value); got != test.want {
			t.Fatalf("isLocalArtistImage(%q) = %v, want %v", test.value, got, test.want)
		}
	}
}

func TestFindArtistFolderImage(t *testing.T) {
	artistDir := t.TempDir()
	albumDir := filepath.Join(artistDir, "Album")
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		t.Fatal(err)
	}

	want := []byte("artist image")
	if err := os.WriteFile(filepath.Join(artistDir, "Example Artist.png"), want, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(albumDir, "cover.jpg"), []byte("album cover"), 0644); err != nil {
		t.Fatal(err)
	}

	data, extension, found := findArtistFolderImage("Example Artist", []database.Music{{
		FilePath: filepath.Join(albumDir, "track.mp3"),
	}})
	if !found {
		t.Fatal("expected artist image to be found")
	}
	if string(data) != string(want) {
		t.Fatalf("found wrong image: %q", data)
	}
	if extension != ".png" {
		t.Fatalf("extension = %q, want .png", extension)
	}
}

func TestAppendScanErrorDeduplicatesMessages(t *testing.T) {
	errors := appendScanError(nil, "artist: failed")
	errors = appendScanError(errors, "artist: failed")
	errors = appendScanError(errors, "other artist: failed")

	if len(errors) != 2 {
		t.Fatalf("len(errors) = %d, want 2", len(errors))
	}
}
