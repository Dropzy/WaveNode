package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeMusicSourcePath(t *testing.T) {
	sourceDir := t.TempDir()

	actual, err := normalizeMusicSourcePath("  " + sourceDir + "  ")
	if err != nil {
		t.Fatalf("expected valid source path: %v", err)
	}
	if actual != filepath.Clean(sourceDir) {
		t.Fatalf("expected %q, got %q", filepath.Clean(sourceDir), actual)
	}
}

func TestNormalizeMusicSourcePathRejectsInvalidPaths(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "track.mp3")
	if err := os.WriteFile(filePath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		message string
	}{
		{name: "empty", path: "", message: "required"},
		{name: "relative", path: "music", message: "absolute"},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), message: "does not exist"},
		{name: "file", path: filePath, message: "directory"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeMusicSourcePath(test.path)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected error containing %q, got %v", test.message, err)
			}
		})
	}
}
