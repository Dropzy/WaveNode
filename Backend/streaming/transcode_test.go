package streaming

import (
	"testing"

	"music-server/database"
)

func TestReplayGainDB(t *testing.T) {
	properties := database.TrackAudioProperties{ReplayGainTrackDB: -6, ReplayGainAlbumDB: -3}
	trackProfile := database.PlaybackProfile{ReplayGainMode: "track", ReplayGainPreampDB: 1}
	if gain := ReplayGainDB(trackProfile, properties); gain != -5 {
		t.Fatalf("track gain = %v", gain)
	}
	albumProfile := database.PlaybackProfile{ReplayGainMode: "album", ReplayGainPreampDB: -1}
	if gain := ReplayGainDB(albumProfile, properties); gain != -4 {
		t.Fatalf("album gain = %v", gain)
	}
	if gain := ReplayGainDB(database.PlaybackProfile{ReplayGainMode: "off", ReplayGainPreampDB: 5}, properties); gain != 0 {
		t.Fatalf("off gain = %v", gain)
	}
}
