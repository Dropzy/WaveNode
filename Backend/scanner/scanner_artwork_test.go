package scanner

import (
	"testing"

	"music-server/database"
)

func TestApplyScannedArtworkSetsTrackCoverFields(t *testing.T) {
	song := &database.Song{}
	artworkURL := "/artwork/embedded-cover.jpg"

	applyScannedArtwork(song, artworkURL)

	if song.ImageURL != artworkURL ||
		song.CoverArtURL != artworkURL ||
		song.CoverArtSmallURL != artworkURL ||
		song.CoverArtMediumURL != artworkURL ||
		song.CoverArtLargeURL != artworkURL {
		t.Fatal("expected embedded artwork URL in every track cover field")
	}
	if song.CoverArtSource != "embedded" {
		t.Fatalf("CoverArtSource = %q, want embedded", song.CoverArtSource)
	}
}

func TestApplyScannedArtworkLeavesMissingArtworkBlank(t *testing.T) {
	song := &database.Song{}

	applyScannedArtwork(song, "")

	if song.ImageURL != "" || song.CoverArtURL != "" || song.CoverArtSource != "" {
		t.Fatal("expected tracks without embedded artwork to remain blank")
	}
}
