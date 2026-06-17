package artistmeta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWikimediaCommonsProviderImageCandidateForFileFiltersLicense(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Query().Get("titles"), "File:Example.jpg") {
			t.Fatalf("unexpected title query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"query": {
				"pages": {
					"1": {
						"title": "File:Example.jpg",
						"imageinfo": [{
							"url": "https://upload.wikimedia.org/example.jpg",
							"thumburl": "https://upload.wikimedia.org/thumb/example.jpg",
							"descriptionurl": "https://commons.wikimedia.org/wiki/File:Example.jpg",
							"mime": "image/jpeg",
							"width": 1200,
							"height": 800,
							"extmetadata": {
								"LicenseShortName": {"value": "CC BY-SA 4.0"},
								"LicenseUrl": {"value": "https://creativecommons.org/licenses/by-sa/4.0/"},
								"Artist": {"value": "Example Author"}
							}
						}]
					}
				}
			}
		}`))
	}))
	defer server.Close()

	provider := NewWikimediaCommonsProvider(nil)
	provider.APIURL = server.URL

	candidate, err := provider.ImageCandidateForFile(context.Background(), "Example.jpg", 0.91)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Source != "wikimedia_commons" {
		t.Fatalf("source = %q", candidate.Source)
	}
	if candidate.LicenseName != "CC BY-SA 4.0" {
		t.Fatalf("license = %q", candidate.LicenseName)
	}
	if candidate.AttributionText == "" || !strings.Contains(candidate.AttributionText, "Example Author") {
		t.Fatalf("missing attribution: %q", candidate.AttributionText)
	}
}

func TestWikimediaCommonsProviderRejectsNonReusableLicense(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"query": {"pages": {"1": {"title": "File:Example.jpg", "imageinfo": [{
				"url": "https://upload.wikimedia.org/example.jpg",
				"extmetadata": {"LicenseShortName": {"value": "All rights reserved"}}
			}]}}}
		}`))
	}))
	defer server.Close()

	provider := NewWikimediaCommonsProvider(nil)
	provider.APIURL = server.URL

	if _, err := provider.ImageCandidateForFile(context.Background(), "Example.jpg", 0.91); err == nil {
		t.Fatal("expected non-reusable license to be rejected")
	}
}
