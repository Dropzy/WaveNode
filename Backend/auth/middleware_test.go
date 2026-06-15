package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsBrowserStreamRequest(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/api/music/track-1/stream", true},
		{http.MethodGet, "/api/music/track-1", false},
		{http.MethodGet, "/api/playlists/track-1/stream", false},
		{http.MethodPost, "/api/music/track-1/stream", false},
	}
	for _, test := range tests {
		request := httptest.NewRequest(test.method, test.path, nil)
		if got := isBrowserStreamRequest(request); got != test.want {
			t.Fatalf("%s %s = %v, want %v", test.method, test.path, got, test.want)
		}
	}
}
