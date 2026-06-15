package metadata

import (
	"bytes"
	"testing"
)

func TestMergeEmbeddedMetadataPreservesArtwork(t *testing.T) {
	parser := NewMetadataParser()
	track := &TrackInfo{Title: "Track", Artist: "Artist"}
	artwork := []byte("embedded-image")
	embedded := &TrackInfo{
		HasArtwork:    true,
		ArtworkData:   artwork,
		ArtworkFormat: "png",
		ArtworkSize:   len(artwork),
	}

	parser.mergeEmbeddedMetadata(track, embedded)

	if !track.HasArtwork {
		t.Fatal("expected embedded artwork to be preserved")
	}
	if !bytes.Equal(track.ArtworkData, artwork) {
		t.Fatal("expected embedded artwork data to be copied")
	}
	if track.ArtworkFormat != "png" {
		t.Fatalf("expected png artwork format, got %q", track.ArtworkFormat)
	}
	if track.ArtworkSize != len(artwork) {
		t.Fatalf("expected artwork size %d, got %d", len(artwork), track.ArtworkSize)
	}
}

func TestMergeEmbeddedArtworkDoesNotRequireTextMetadata(t *testing.T) {
	parser := NewMetadataParser()
	track := &TrackInfo{Title: "Filename Title", Artist: "Filename Artist"}
	artwork := []byte("embedded-image")
	embedded := &TrackInfo{
		HasArtwork:    true,
		ArtworkData:   artwork,
		ArtworkFormat: "jpg",
		ArtworkSize:   len(artwork),
	}

	if parser.shouldUseEmbeddedMetadata(track, embedded) {
		t.Fatal("expected sparse embedded text metadata to be rejected")
	}

	parser.mergeEmbeddedArtwork(track, embedded)

	if !track.HasArtwork || !bytes.Equal(track.ArtworkData, artwork) {
		t.Fatal("expected artwork to be accepted independently of text metadata")
	}
}

func TestApplyReplayGainTags(t *testing.T) {
	track := &TrackInfo{}
	applyReplayGainTags(track, map[string]interface{}{
		"REPLAYGAIN_TRACK_GAIN": "-7.25 dB",
		"replaygain_album_gain": "+1.50 dB",
		"REPLAYGAIN_TRACK_PEAK": "0.9876",
	})
	if track.ReplayGainTrackDB != -7.25 || track.ReplayGainAlbumDB != 1.5 {
		t.Fatalf("unexpected gains: track=%v album=%v", track.ReplayGainTrackDB, track.ReplayGainAlbumDB)
	}
	if track.ReplayGainTrackPeak != 0.9876 {
		t.Fatalf("unexpected peak: %v", track.ReplayGainTrackPeak)
	}
}
