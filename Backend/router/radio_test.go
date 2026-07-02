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
