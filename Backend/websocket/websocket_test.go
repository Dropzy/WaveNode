package websocket

import (
	"net/http/httptest"
	"testing"
)

func TestWebSocketOriginPolicy(t *testing.T) {
	check := websocketOriginAllowed([]string{"https://music.example"})

	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{name: "same origin", host: "music.example", origin: "https://music.example", want: true},
		{name: "configured origin", host: "internal:8080", origin: "https://music.example", want: true},
		{name: "native client", host: "music.example", origin: "", want: true},
		{name: "foreign browser origin", host: "music.example", origin: "https://evil.example", want: false},
		{name: "lookalike host", host: "music.example", origin: "https://music.example.evil.test", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "http://"+test.host+"/ws", nil)
			request.Host = test.host
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if got := check(request); got != test.want {
				t.Fatalf("origin policy returned %v, want %v", got, test.want)
			}
		})
	}
}
