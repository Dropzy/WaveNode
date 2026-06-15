package utils

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// NormalizeArtworkURL converts stored artwork values into a client-facing URL.
// Legacy placeholder paths are returned as empty so clients can render a fallback.
func NormalizeArtworkURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(strings.ToLower(value), "default-track.png") {
		return ""
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}

	value = strings.ReplaceAll(value, "\\", "/")
	if strings.HasPrefix(value, "/api/artwork/") {
		return value
	}
	if strings.HasPrefix(value, "/artwork/") {
		return "/api" + value
	}
	if strings.HasPrefix(value, "artwork/") {
		return "/api/" + value
	}
	if strings.HasPrefix(value, "/") {
		return value
	}

	filename := filepath.Base(value)
	return fmt.Sprintf("/api/artwork/%s", url.PathEscape(filename))
}
