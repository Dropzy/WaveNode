package router

import (
	"encoding/json"
	"testing"
)

func TestValidatePluginManifestAcceptsRadioRow(t *testing.T) {
	raw := []byte(`{
		"schema_version": 1,
		"id": "example.radio",
		"name": "Example Radio",
		"version": "1.0.0",
		"permissions": ["network", "playback"],
		"contributes": {
			"home_rows": [{
				"id": "radio-row",
				"title": "Radio",
				"type": "radio",
				"items": [{
					"id": "station-one",
					"title": "Station One",
					"stream_url": "https://radio.example.test/live.mp3"
				}]
			}]
		}
	}`)
	manifest, err := validatePluginManifest(raw)
	if err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
	if manifest.ID != "example.radio" || len(manifest.Contributes.HomeRows) != 1 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}

func TestValidatePluginManifestAcceptsDownloadTrackAction(t *testing.T) {
	raw := []byte(`{
		"schema_version": 1,
		"id": "mobile.downloads",
		"name": "Mobile Downloads",
		"version": "1.0.0",
		"permissions": ["download"],
		"contributes": {
			"track_actions": [{
				"id": "download-to-device",
				"label": "Download to Device",
				"icon": "download",
				"action_type": "download",
				"url": "/api/music/{id}/download"
			}]
		}
	}`)
	manifest, err := validatePluginManifest(raw)
	if err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
	if manifest.ID != "mobile.downloads" || len(manifest.Contributes.TrackActions) != 1 {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}

func TestValidatePluginManifestRejectsUnsafeAndUnknownFields(t *testing.T) {
	tests := map[string]string{
		"javascript stream": `{
			"schema_version":1,"id":"bad.radio","name":"Bad","version":"1",
			"permissions":["network","playback"],
			"contributes":{"home_rows":[{"id":"bad-row","title":"Bad","type":"radio",
			"items":[{"id":"bad-item","title":"Bad","stream_url":"javascript:alert(1)"}]}]}
		}`,
		"missing permissions": `{
			"schema_version":1,"id":"bad.radio","name":"Bad","version":"1",
			"contributes":{"home_rows":[{"id":"bad-row","title":"Bad","type":"radio",
			"items":[{"id":"bad-item","title":"Bad","stream_url":"https://example.test/live"}]}]}
		}`,
		"unknown field": `{
			"schema_version":1,"id":"bad.radio","name":"Bad","version":"1","executable":"bad.js",
			"permissions":["network","playback"],
			"contributes":{"home_rows":[{"id":"bad-row","title":"Bad","type":"radio",
			"items":[{"id":"bad-item","title":"Bad","stream_url":"https://example.test/live"}]}]}
		}`,
		"download action without permission": `{
			"schema_version":1,"id":"bad.downloads","name":"Bad","version":"1",
			"contributes":{"track_actions":[{"id":"download","label":"Download","action_type":"download","url":"/api/music/{id}/download"}]}
		}`,
		"download action unsafe URL": `{
			"schema_version":1,"id":"bad.downloads","name":"Bad","version":"1","permissions":["download"],
			"contributes":{"track_actions":[{"id":"download","label":"Download","action_type":"download","url":"https://example.test/{id}"}]}
		}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := validatePluginManifest([]byte(raw)); err == nil {
				t.Fatal("expected manifest validation to fail")
			}
		})
	}
}

func TestPluginManifestRoundTrip(t *testing.T) {
	manifest := PluginManifest{
		SchemaVersion: 1,
		ID:            "roundtrip.radio",
		Name:          "Roundtrip",
		Version:       "1",
		Permissions:   []string{"network", "playback"},
		Contributes: PluginContributions{HomeRows: []PluginHomeRow{{
			ID: "stations", Title: "Stations", Type: "radio",
			Items: []PluginRowItem{{
				ID: "one", Title: "One", StreamURL: "https://example.test/live",
			}},
		}}},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validatePluginManifest(raw); err != nil {
		t.Fatalf("round-trip validation failed: %v", err)
	}
}
