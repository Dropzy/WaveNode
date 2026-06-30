package router

import (
	"net/http/httptest"
	"testing"
)

func TestParseSSDPHeaders(t *testing.T) {
	headers := parseSSDPHeaders("HTTP/1.1 200 OK\r\nLOCATION: http://192.168.1.20/device.xml\r\nST: renderer\r\n\r\n")
	if headers["location"] != "http://192.168.1.20/device.xml" || headers["st"] != "renderer" {
		t.Fatalf("unexpected SSDP headers: %#v", headers)
	}
}

func TestRequestBaseURLUsesForwardedHeaders(t *testing.T) {
	request := httptest.NewRequest("GET", "http://backend:8080/api/outputs/cast-url", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Forwarded-Host", "music.example.test")
	if actual := requestBaseURL(request); actual != "https://music.example.test" {
		t.Fatalf("requestBaseURL returned %q", actual)
	}
}
