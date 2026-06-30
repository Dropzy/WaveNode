package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLibriVoxMapsProviderResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("title") != "^sherlock" || req.URL.Query().Get("extended") != "1" {
			t.Fatalf("unexpected query: %s", req.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"books":[{"id":"314","title":"Adventures of Sherlock Holmes","description":"<p>Stories &amp; mysteries.</p>","language":"English","num_sections":"12","url_librivox":"https://librivox.org/book","url_iarchive":"https://archive.org/details/sherlock_librivox","totaltimesecs":3600,"authors":[{"first_name":"Arthur Conan","last_name":"Doyle"}],"genres":[{"name":"Detective Fiction"}]}]}`))
	}))
	defer server.Close()
	previousURL := librivoxAudiobooksURL
	librivoxAudiobooksURL = server.URL
	defer func() { librivoxAudiobooksURL = previousURL }()

	books, err := requestLibriVox("sherlock", "", 20, false)
	if err != nil || len(books) != 1 {
		t.Fatalf("unexpected provider response: books=%#v error=%v", books, err)
	}
	book := mapAudiobookSummary(books[0])
	if book.Author != "Arthur Conan Doyle" || book.Description != "Stories & mysteries." || book.ChapterCount != 12 {
		t.Fatalf("unexpected mapped book: %#v", book)
	}
	if book.ImageURL != "https://archive.org/services/img/sherlock_librivox" {
		t.Fatalf("unexpected cover URL: %s", book.ImageURL)
	}
}

func TestRepairLibriVoxText(t *testing.T) {
	if value := repairLibriVoxText("HonorÃ© and donât"); value != "Honoré and don’t" {
		t.Fatalf("unexpected repaired text: %q", value)
	}
}

func TestAudiobookArchiveImageRejectsMissingIdentifier(t *testing.T) {
	if value := audiobookArchiveImage("://"); value != "" {
		t.Fatalf("expected no image, got %q", value)
	}
}
