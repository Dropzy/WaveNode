package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupAuthorization(t *testing.T) {
	protected := &Router{setupToken: "correct-setup-code"}

	request := httptest.NewRequest(http.MethodPost, "/api/setup/complete", nil)
	if protected.setupAuthorized(request) {
		t.Fatal("missing setup access code must be rejected")
	}

	request.Header.Set("X-WaveNode-Setup-Token", "wrong-setup-code")
	if protected.setupAuthorized(request) {
		t.Fatal("incorrect setup access code must be rejected")
	}

	request.Header.Set("X-WaveNode-Setup-Token", "correct-setup-code")
	if !protected.setupAuthorized(request) {
		t.Fatal("correct setup access code must be accepted")
	}

	unprotected := &Router{}
	if !unprotected.setupAuthorized(httptest.NewRequest(http.MethodPost, "/api/setup/complete", nil)) {
		t.Fatal("local setup without a configured access code must remain available")
	}
}
