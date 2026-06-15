package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArtworkDirectoryUsesConfiguredPath(t *testing.T) {
	configured := filepath.Join(t.TempDir(), "covers")
	t.Setenv("WAVENODE_ARTWORK_PATH", configured)

	if got := ArtworkDirectory(); got != filepath.Clean(configured) {
		t.Fatalf("ArtworkDirectory() = %q, want %q", got, filepath.Clean(configured))
	}
}

func TestArtworkSearchDirectoriesIncludesLegacyPath(t *testing.T) {
	t.Setenv("WAVENODE_ARTWORK_PATH", filepath.Join(t.TempDir(), "covers"))
	for _, directory := range ArtworkSearchDirectories() {
		if directory == "artwork" {
			return
		}
	}
	t.Fatal("legacy artwork directory was not included")
}

func TestArtworkSearchDirectoriesIncludesTemporaryLegacyPath(t *testing.T) {
	t.Setenv("WAVENODE_ARTWORK_PATH", filepath.Join(t.TempDir(), "covers"))
	expected := filepath.Join(os.TempDir(), "WaveNode", "artwork")
	for _, directory := range ArtworkSearchDirectories() {
		if directory == expected {
			return
		}
	}
	t.Fatal("temporary legacy artwork directory was not included")
}

func TestArtworkExistsFindsLegacyFile(t *testing.T) {
	t.Setenv("WAVENODE_ARTWORK_PATH", filepath.Join(t.TempDir(), "configured"))

	legacy := filepath.Join(os.TempDir(), "WaveNode", "artwork")
	if err := os.MkdirAll(legacy, 0755); err != nil {
		t.Fatal(err)
	}
	filename := fmt.Sprintf("artist-test-%d.jpg", time.Now().UnixNano())
	path := filepath.Join(legacy, filename)
	if err := os.WriteFile(path, []byte("image"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(path) })

	if !ArtworkExists("/artwork/" + filename) {
		t.Fatal("expected legacy artwork file to be found")
	}
}
