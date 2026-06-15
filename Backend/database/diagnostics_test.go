package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticTrackChecks(t *testing.T) {
	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "track.mp3")
	if err := os.WriteFile(audioPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("write test audio file: %v", err)
	}

	track := diagnosticTrack{
		ID:          "track-1",
		Title:       "Track",
		Artist:      "Artist",
		FilePath:    audioPath,
		Format:      "mp3",
		Duration:    180,
		HasMetadata: true,
		ImageURL:    "/artwork/track.jpg",
	}
	if issue := missingFileIssue(track.FilePath); issue != "" {
		t.Fatalf("expected existing file to pass, got %q", issue)
	}
	if issue := metadataIssue(track); issue != "" {
		t.Fatalf("expected complete metadata to pass, got %q", issue)
	}
	if issue := unsupportedFormatIssue(track); issue != "" {
		t.Fatalf("expected MP3 to be supported, got %q", issue)
	}
	if !hasArtwork(track) {
		t.Fatal("expected image URL to count as artwork")
	}
}

func TestDiagnosticTrackProblems(t *testing.T) {
	track := diagnosticTrack{
		ID:       "track-2",
		FilePath: filepath.Join(t.TempDir(), "missing.xyz"),
		Format:   "xyz",
	}
	if issue := missingFileIssue(track.FilePath); !strings.Contains(issue, "missing") {
		t.Fatalf("expected missing file issue, got %q", issue)
	}
	if issue := metadataIssue(track); !strings.Contains(issue, "title") || !strings.Contains(issue, "artist") {
		t.Fatalf("expected title and artist metadata issues, got %q", issue)
	}
	if issue := unsupportedFormatIssue(track); !strings.Contains(issue, ".xyz") {
		t.Fatalf("expected unsupported format issue, got %q", issue)
	}
	if hasArtwork(track) {
		t.Fatal("expected empty artwork fields to be reported")
	}
}

func TestInspectSourceIncludesStorageCapacity(t *testing.T) {
	source := inspectSource(t.TempDir())
	if !source.Accessible {
		t.Fatalf("expected temporary directory to be accessible: %s", source.Error)
	}
	if source.TotalBytes == 0 {
		t.Fatal("expected source capacity to be reported")
	}
	if source.FreeBytes > source.TotalBytes {
		t.Fatalf("free bytes %d exceed total bytes %d", source.FreeBytes, source.TotalBytes)
	}
	if source.SpaceStatus == "unavailable" || source.SpaceStatus == "unknown" {
		t.Fatalf("expected a disk space status, got %q: %s", source.SpaceStatus, source.Error)
	}
}
