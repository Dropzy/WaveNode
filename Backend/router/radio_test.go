package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRadioBrowserRequestFiltersUnsafeAndBrokenStations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"stationuuid":"042d3140-227c-4fac-9387-4903b692d5f2","name":"Secure","url_resolved":"https://radio.example.test/live","codec":"mp3","bitrate":192,"lastcheckok":1},
			{"stationuuid":"d1a54d2e-623e-4970-ab11-35f7b56c5ec3","name":"Insecure","url_resolved":"http://radio.example.test/live","lastcheckok":1},
			{"stationuuid":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","name":"Broken","url_resolved":"https://radio.example.test/broken","lastcheckok":0}
		]`))
	}))
	defer server.Close()
	t.Setenv("WAVENODE_RADIO_BROWSER_URL", server.URL)

	stations, err := radioBrowserRequest(context.Background(), "/json/stations/search", url.Values{"limit": {"3"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) != 1 || stations[0].Name != "Secure" || stations[0].Codec != "MP3" || stations[0].Bitrate != 192 {
		t.Fatalf("stations = %#v", stations)
	}
}

func TestValidatePublicRadioStreamRejectsLocalAddress(t *testing.T) {
	if err := validatePublicRadioStream("https://127.0.0.1/live"); err == nil {
		t.Fatal("expected loopback radio stream to be rejected")
	}
}

func TestRadioSearchParamsNormalizesGenreTag(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/radio/stations?tag=Hip+Hop", nil)
	params := radioSearchParams(req, 50)
	if got := params.Get("tag"); got != "hip hop" {
		t.Fatalf("tag = %q, want %q", got, "hip hop")
	}
}

func TestValidateExternalHandoffTrackUsesOriginalPodcastURL(t *testing.T) {
	track, err := validateExternalHandoffTrack(playbackHandoffTrack{
		ID: "podcast:show:episode", Title: "Episode", IsExternal: true, ExternalKind: "podcast",
		StreamURL:       "file:///data/user/0/org.wavenode.player/files/episode.mp3",
		PodcastAudioURL: "https://media.example.test/episode.mp3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if track.StreamURL != "https://media.example.test/episode.mp3" || track.PodcastAudioURL != track.StreamURL {
		t.Fatalf("unexpected podcast stream: %#v", track)
	}
}

func TestValidateExternalHandoffTrackRejectsInsecureStream(t *testing.T) {
	_, err := validateExternalHandoffTrack(playbackHandoffTrack{
		ID: "radio:station", Title: "Station", IsExternal: true, ExternalKind: "radio",
		StreamURL: "http://radio.example.test/live",
	})
	if err == nil {
		t.Fatal("expected insecure external stream to be rejected")
	}
}
