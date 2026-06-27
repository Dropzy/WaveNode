package router

import (
	"strings"
	"testing"
)

func TestParsePodcastRSS(t *testing.T) {
	const document = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Example Show</title>
    <description><![CDATA[An <b>example</b> podcast.]]></description>
    <link>https://example.com/show</link>
    <itunes:image href="https://example.com/show.jpg" />
    <item>
      <guid>episode-one</guid>
      <title>Episode One</title>
      <content:encoded><![CDATA[The <strong>first</strong> episode.]]></content:encoded>
      <link>https://example.com/episodes/one</link>
      <pubDate>Fri, 27 Jun 2026 12:30:00 +0000</pubDate>
      <itunes:duration>01:02:03</itunes:duration>
      <itunes:explicit>yes</itunes:explicit>
      <enclosure url="https://cdn.example.com/one.mp3" type="audio/mpeg" />
    </item>
    <item><title>Missing audio</title></item>
    <item>
      <title>Video episode</title>
      <enclosure url="https://cdn.example.com/video.mp4" type="video/mp4" />
    </item>
  </channel>
</rss>`

	episodes, feed, err := parsePodcastRSS(strings.NewReader(document), 10)
	if err != nil {
		t.Fatalf("parsePodcastRSS returned an error: %v", err)
	}
	if len(episodes) != 1 {
		t.Fatalf("expected 1 playable episode, got %d", len(episodes))
	}
	if feed.Description != "An example podcast." || feed.ImageURL != "https://example.com/show.jpg" {
		t.Fatalf("unexpected feed metadata: %+v", feed)
	}
	episode := episodes[0]
	if episode.Title != "Episode One" || episode.Description != "The first episode." {
		t.Fatalf("unexpected episode text: %+v", episode)
	}
	if episode.Duration != 3723 || !episode.Explicit {
		t.Fatalf("unexpected episode metadata: %+v", episode)
	}
	if episode.PublishedAt != "2026-06-27T12:30:00Z" {
		t.Fatalf("unexpected published date: %s", episode.PublishedAt)
	}
}

func TestParsePodcastDuration(t *testing.T) {
	tests := map[string]int{"45": 45, "12:34": 754, "01:02:03": 3723, "invalid": 0}
	for value, expected := range tests {
		if actual := parsePodcastDuration(value); actual != expected {
			t.Errorf("parsePodcastDuration(%q) = %d, expected %d", value, actual, expected)
		}
	}
}

func TestPodcastPlaybackCompleted(t *testing.T) {
	tests := []struct {
		name     string
		position int
		duration int
		want     bool
	}{
		{name: "unfinished", position: 300, duration: 1000, want: false},
		{name: "ninety five percent", position: 950, duration: 1000, want: true},
		{name: "within final thirty seconds", position: 575, duration: 600, want: true},
		{name: "unknown duration", position: 575, duration: 0, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := podcastPlaybackCompleted(test.position, test.duration); got != test.want {
				t.Fatalf("podcastPlaybackCompleted(%d, %d) = %v, want %v", test.position, test.duration, got, test.want)
			}
		})
	}
}
