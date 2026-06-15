package utils

import "testing"

func TestNormalizeArtworkURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "legacy placeholder", input: "/api/static/images/default-track.png", expected: ""},
		{name: "api path", input: "/api/artwork/cover.jpg", expected: "/api/artwork/cover.jpg"},
		{name: "legacy root path", input: "/artwork/cover.jpg", expected: "/api/artwork/cover.jpg"},
		{name: "relative path", input: "artwork/cover.jpg", expected: "/api/artwork/cover.jpg"},
		{name: "filename", input: "cover image.jpg", expected: "/api/artwork/cover%20image.jpg"},
		{name: "remote", input: "https://example.com/cover.jpg", expected: "https://example.com/cover.jpg"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := NormalizeArtworkURL(test.input); actual != test.expected {
				t.Fatalf("NormalizeArtworkURL(%q) = %q, expected %q", test.input, actual, test.expected)
			}
		})
	}
}
