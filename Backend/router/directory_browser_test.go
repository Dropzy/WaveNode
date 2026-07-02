package router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListBrowsableDirectoryIncludesSupportedAudioFiles(t *testing.T) {
	root := t.TempDir()
	album := filepath.Join(root, "Album")
	if err := os.Mkdir(album, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "track.FLAC"), []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}

	directories, audioFiles, count, err := listBrowsableDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(directories) != 1 || directories[0].Name != "Album" {
		t.Fatalf("directories = %#v", directories)
	}
	if count != 1 || len(audioFiles) != 1 {
		t.Fatalf("audio files = %#v, count = %d", audioFiles, count)
	}
	if audioFiles[0].Name != "track.FLAC" || audioFiles[0].Format != "flac" || audioFiles[0].Size != 5 {
		t.Fatalf("audio file = %#v", audioFiles[0])
	}
}
