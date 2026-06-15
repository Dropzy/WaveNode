package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoggingURIHidesWebSocketToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws?token=secret-token&client=web", nil)
	uri := sanitizedRequestURI(req)

	if uri == req.RequestURI || uri == "" {
		t.Fatalf("expected sanitized URI, got %q", uri)
	}
	if uri != "/ws?client=web&token=%5BREDACTED%5D" {
		t.Fatalf("unexpected sanitized URI: %q", uri)
	}
}

func TestLoggingURIHidesSubsonicCredentials(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/rest/ping.view?u=admin&p=secret&t=hash&s=salt&apiKey=key&v=1.16.1&c=test",
		nil,
	)
	uri := sanitizedRequestURI(req)

	for _, secret := range []string{"secret", "hash", "salt", "apiKey=key"} {
		if strings.Contains(uri, secret) {
			t.Fatalf("sanitized URI exposed %q: %s", secret, uri)
		}
	}
	if !strings.Contains(uri, "u=admin") || !strings.Contains(uri, "v=1.16.1") {
		t.Fatalf("sanitized URI removed non-sensitive client details: %s", uri)
	}
}

func TestCORSRejectsDisallowedPreflight(t *testing.T) {
	config := struct {
		AllowedOrigins []string `json:"allowed_origins"`
		AllowedMethods []string `json:"allowed_methods"`
		AllowedHeaders []string `json:"allowed_headers"`
	}{
		AllowedOrigins: []string{"https://music.example"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}

	handler := CORSMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/music", nil)
	req.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("disallowed origin must not receive an allow-origin header")
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeadersMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing content type protection header")
	}
	if !strings.Contains(recorder.Header().Get("Content-Security-Policy"), "media-src 'self' blob: https:") {
		t.Fatal("content security policy must allow approved HTTPS media")
	}
}

func TestLoginRateLimitCountsFailuresAndResetsAfterSuccess(t *testing.T) {
	status := http.StatusUnauthorized
	handler := LoginRateLimit(2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))

	request := func() int {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		handler.ServeHTTP(recorder, req)
		return recorder.Code
	}

	if request() != http.StatusUnauthorized || request() != http.StatusUnauthorized {
		t.Fatal("failed logins should reach the login handler until the limit is reached")
	}
	if request() != http.StatusTooManyRequests {
		t.Fatal("expected failed login attempts to be rate limited")
	}

	status = http.StatusOK
	secondClient := func() int {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "127.0.0.2:1234"
		handler.ServeHTTP(recorder, req)
		return recorder.Code
	}
	if secondClient() != http.StatusOK {
		t.Fatal("successful login should be allowed")
	}
	status = http.StatusUnauthorized
	if secondClient() != http.StatusUnauthorized || secondClient() != http.StatusUnauthorized {
		t.Fatal("a successful login should reset the client's failed attempt count")
	}
}
