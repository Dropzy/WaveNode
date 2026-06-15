package utils

import (
	"log"
	"path/filepath"
	"strings"
)

// GetDefaultTrackImage returns a default image URL for tracks that don't have embedded artwork
func GetDefaultTrackImage() string {
	return "/api/static/images/default-track.png"
}

// GetDefaultAlbumImage returns a default image URL for albums that don't have cover art
func GetDefaultAlbumImage() string {
	return "/api/static/images/default-album.png"
}

// GetDefaultArtistImage returns a default image URL for artists that don't have images
func GetDefaultArtistImage() string {
	return "/api/static/images/default-artist.png"
}

// ResolveTrackImageURL determines the best image URL to use for a track
// It prioritizes: 1) Embedded artwork, 2) Album cover art, 3) Default track image
func ResolveTrackImageURL(imageURL, coverArtURL, coverArtSmallURL, coverArtMediumURL string) string {
	// First priority: Embedded artwork from metadata
	if imageURL != "" && !isDefaultImage(imageURL) {
		return imageURL
	}

	// Second priority: Album cover art (prefer medium quality, fallback to small)
	if coverArtMediumURL != "" && !isDefaultImage(coverArtMediumURL) {
		return coverArtMediumURL
	}
	if coverArtSmallURL != "" && !isDefaultImage(coverArtSmallURL) {
		return coverArtSmallURL
	}
	if coverArtURL != "" && !isDefaultImage(coverArtURL) {
		return coverArtURL
	}

	// Final fallback: Default track image
	return GetDefaultTrackImage()
}

// isDefaultImage checks if an image URL is a default/placeholder image
func isDefaultImage(imageURL string) bool {
	if imageURL == "" {
		return true
	}

	lowerURL := strings.ToLower(imageURL)
	return strings.Contains(lowerURL, "default") ||
		strings.Contains(lowerURL, "placeholder") ||
		strings.Contains(lowerURL, "no-image") ||
		strings.Contains(lowerURL, "missing")
}

// GetImageSource determines the source/type of an image URL
func GetImageSource(imageURL string) string {
	if imageURL == "" {
		return "none"
	}

	if strings.HasPrefix(imageURL, "data:image/") {
		return "embedded"
	}

	if strings.HasPrefix(imageURL, "/api/static/images/") {
		if strings.Contains(imageURL, "default") {
			return "default"
		}
		return "static"
	}

	if strings.HasPrefix(imageURL, "http") {
		return "external"
	}

	if strings.HasPrefix(imageURL, "/api/music/artwork/") {
		return "extracted"
	}

	return "unknown"
}

// ValidateImageURL checks if an image URL is valid and accessible
func ValidateImageURL(imageURL string) bool {
	if imageURL == "" {
		return false
	}

	// Check for common image file extensions in static URLs
	if strings.HasPrefix(imageURL, "/api/static/") || strings.HasPrefix(imageURL, "http") {
		ext := strings.ToLower(filepath.Ext(imageURL))
		validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
		for _, validExt := range validExts {
			if ext == validExt {
				return true
			}
		}
		return false
	}

	// Allow data URLs and API URLs
	return strings.HasPrefix(imageURL, "data:image/") ||
		strings.HasPrefix(imageURL, "/api/music/artwork/") ||
		strings.HasPrefix(imageURL, "/api/album/")
}

// LogImageInfo logs information about image resolution for debugging
func LogImageInfo(trackID, trackTitle string, originalImageURL, resolvedImageURL, source string) {
	log.Printf("Track %s (%s): Image resolved from '%s' to '%s' (source: %s)",
		trackID, trackTitle, originalImageURL, resolvedImageURL, source)
}

// GetImageQuality determines the quality level of an image URL
func GetImageQuality(imageURL string) string {
	if imageURL == "" {
		return "none"
	}

	lowerURL := strings.ToLower(imageURL)
	if strings.Contains(lowerURL, "large") || strings.Contains(lowerURL, "high") {
		return "high"
	}
	if strings.Contains(lowerURL, "medium") || strings.Contains(lowerURL, "med") {
		return "medium"
	}
	if strings.Contains(lowerURL, "small") || strings.Contains(lowerURL, "thumb") {
		return "low"
	}
	if strings.HasPrefix(imageURL, "data:image/") {
		return "original"
	}

	return "unknown"
}

// FormatImageForSize formats an image URL to request a specific size if the API supports it
func FormatImageForSize(baseURL, size string) string {
	if baseURL == "" {
		return ""
	}

	// If it's already a data URL or external URL, return as-is
	if strings.HasPrefix(baseURL, "data:image/") || strings.HasPrefix(baseURL, "http") {
		return baseURL
	}

	// For API endpoints, try to add size parameter
	if strings.HasPrefix(baseURL, "/api/") {
		if strings.Contains(baseURL, "?") {
			return baseURL + "&size=" + size
		}
		return baseURL + "?size=" + size
	}

	return baseURL
}
