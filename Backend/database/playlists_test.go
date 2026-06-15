package database

import (
	"encoding/json"
	"testing"
)

func TestMarshalTrackIDsUsesEmptyArrayForNilTracks(t *testing.T) {
	value, err := marshalTrackIDs(nil)
	if err != nil {
		t.Fatalf("marshalTrackIDs returned an error: %v", err)
	}

	if string(value) != "[]" {
		t.Fatalf("marshalTrackIDs(nil) = %q, want []", value)
	}
}

func TestMarshalTrackIDsPreservesTracks(t *testing.T) {
	value, err := marshalTrackIDs([]string{"track-1", "track-2"})
	if err != nil {
		t.Fatalf("marshalTrackIDs returned an error: %v", err)
	}

	var tracks []string
	if err := json.Unmarshal(value, &tracks); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if len(tracks) != 2 || tracks[0] != "track-1" || tracks[1] != "track-2" {
		t.Fatalf("unexpected tracks: %#v", tracks)
	}
}

func TestAppendUniqueTrackID(t *testing.T) {
	tracks := appendUniqueTrackID([]string{"track-1"}, "track-1")
	if len(tracks) != 1 {
		t.Fatalf("duplicate track was added: %#v", tracks)
	}

	tracks = appendUniqueTrackID(tracks, "track-2")
	if len(tracks) != 2 || tracks[1] != "track-2" {
		t.Fatalf("new track was not appended: %#v", tracks)
	}
}

func TestRemoveTrackID(t *testing.T) {
	tracks := removeTrackID([]string{"track-1", "track-2", "track-1"}, "track-1")
	if len(tracks) != 1 || tracks[0] != "track-2" {
		t.Fatalf("track was not removed: %#v", tracks)
	}
}
