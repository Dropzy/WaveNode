package router

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"music-server/utils"
)

var WaveNodeVersion = "0.1.7"

var serverStartedAt = time.Now()

func (r *Router) getSystemStatus(w http.ResponseWriter, req *http.Request) {
	databaseStats := r.db.Stats()
	artworkBytes, artworkFiles := directoryUsage(utils.ArtworkDirectory())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"version":           WaveNodeVersion,
			"uptime_seconds":    int64(time.Since(serverStartedAt).Seconds()),
			"go_version":        runtime.Version(),
			"goroutines":        runtime.NumGoroutine(),
			"active_streams":    r.musicHandler.ActiveStreams(),
			"database_open":     databaseStats.OpenConnections,
			"database_in_use":   databaseStats.InUse,
			"database_idle":     databaseStats.Idle,
			"artwork_bytes":     artworkBytes,
			"artwork_files":     artworkFiles,
			"automatic_updates": r.autoUpdater != nil,
		},
	})
}

func (r *Router) getLibraryDiagnostics(w http.ResponseWriter, req *http.Request) {
	diagnostics, err := r.db.GetLibraryDiagnostics()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": diagnostics})
}

func directoryUsage(root string) (int64, int) {
	var bytes int64
	var files int
	_ = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			bytes += info.Size()
			files++
		}
		return nil
	})
	return bytes, files
}
