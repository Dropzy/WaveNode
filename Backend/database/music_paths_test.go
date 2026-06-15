package database

import (
	"path/filepath"
	"testing"
)

func TestPathWithinSources(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "music")

	if !pathWithinSources(filepath.Join(root, "Artist", "track.flac"), []string{root}) {
		t.Fatal("expected track below source root to match")
	}
	if pathWithinSources(filepath.Join(string(filepath.Separator), "other", "track.flac"), []string{root}) {
		t.Fatal("expected track outside source root not to match")
	}
	if pathWithinSources(root+"-backup"+string(filepath.Separator)+"track.flac", []string{root}) {
		t.Fatal("expected similarly prefixed path not to match")
	}
}
