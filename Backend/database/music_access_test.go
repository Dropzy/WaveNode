package database

import (
	"path/filepath"
	"testing"
)

func TestPathWithinMusicRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "allowed")
	inside := filepath.Join(root, "Artist", "track.flac")
	outside := filepath.Join(filepath.Dir(root), "allowed-backup", "track.flac")

	if !pathWithinMusicRoot(inside, root) {
		t.Fatal("expected child track to be inside the allowed music root")
	}
	if !pathWithinMusicRoot(root, root) {
		t.Fatal("expected the music root itself to match")
	}
	if pathWithinMusicRoot(outside, root) {
		t.Fatal("expected similarly prefixed sibling folder to be denied")
	}
}
