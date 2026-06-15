package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ArtworkDirectory returns the writable directory used for generated artwork.
func ArtworkDirectory() string {
	if configured := strings.TrimSpace(os.Getenv("WAVENODE_ARTWORK_PATH")); configured != "" {
		return filepath.Clean(configured)
	}
	if cacheDirectory, err := os.UserCacheDir(); err == nil && cacheDirectory != "" {
		return filepath.Join(cacheDirectory, "WaveNode", "artwork")
	}
	return filepath.Join(os.TempDir(), "WaveNode", "artwork")
}

// ArtworkSearchDirectories includes current and legacy artwork locations.
func ArtworkSearchDirectories() []string {
	candidates := []string{
		ArtworkDirectory(),
		filepath.Join(os.TempDir(), "WaveNode", "artwork"),
		"artwork",
		filepath.Join("..", "artwork"),
	}
	seen := make(map[string]struct{})
	directories := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		directories = append(directories, clean)
	}
	return directories
}

// MigrateLegacyArtwork copies files from previous artwork locations into the
// current persistent directory without replacing existing files.
func MigrateLegacyArtwork() (int, error) {
	destination := ArtworkDirectory()
	if err := os.MkdirAll(destination, 0755); err != nil {
		return 0, fmt.Errorf("failed to create artwork directory: %w", err)
	}

	migrated := 0
	for _, source := range ArtworkSearchDirectories() {
		if sameArtworkPath(source, destination) {
			continue
		}

		entries, err := os.ReadDir(source)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			sourcePath := filepath.Join(source, entry.Name())
			destinationPath := filepath.Join(destination, entry.Name())
			if _, err := os.Stat(destinationPath); err == nil {
				continue
			}
			if err := copyArtworkFile(sourcePath, destinationPath); err != nil {
				return migrated, err
			}
			migrated++
		}
	}
	return migrated, nil
}

// ArtworkExists reports whether a stored artwork URL resolves to a real file.
func ArtworkExists(value string) bool {
	filename := filepath.Base(strings.TrimSpace(value))
	if filename == "" || filename == "." {
		return false
	}
	for _, directory := range ArtworkSearchDirectories() {
		info, err := os.Stat(filepath.Join(directory, filename))
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func copyArtworkFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open legacy artwork: %w", err)
	}
	defer input.Close()

	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to create migrated artwork: %w", err)
	}

	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		os.Remove(destination)
		return fmt.Errorf("failed to migrate artwork: %w", err)
	}
	if err := output.Close(); err != nil {
		os.Remove(destination)
		return fmt.Errorf("failed to finish migrated artwork: %w", err)
	}
	return nil
}

func sameArtworkPath(first, second string) bool {
	firstPath, firstErr := filepath.Abs(first)
	secondPath, secondErr := filepath.Abs(second)
	if firstErr != nil || secondErr != nil {
		return filepath.Clean(first) == filepath.Clean(second)
	}
	return strings.EqualFold(filepath.Clean(firstPath), filepath.Clean(secondPath))
}
