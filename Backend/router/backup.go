package router

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"music-server/database"
	"music-server/utils"
)

const maxBackupUploadSize = 1024 << 20
const maxBackupArtworkFileSize = 50 << 20
const maxBackupArtworkTotalSize = 900 << 20

func (r *Router) downloadBackup(w http.ResponseWriter, req *http.Request) {
	snapshot, err := r.db.CreateBackupSnapshot()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("wavenode-backup-%s.zip", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-store")

	archive := zip.NewWriter(w)
	manifest, err := archive.Create("backup.json")
	if err != nil {
		return
	}
	if err := json.NewEncoder(manifest).Encode(snapshot); err != nil {
		return
	}

	artworkDirectory := utils.ArtworkDirectory()
	entries, _ := os.ReadDir(artworkDirectory)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		source, openErr := os.Open(filepath.Join(artworkDirectory, entry.Name()))
		if openErr != nil {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			source.Close()
			continue
		}
		header, headerErr := zip.FileInfoHeader(info)
		if headerErr != nil {
			source.Close()
			continue
		}
		header.Name = filepath.ToSlash(filepath.Join("artwork", entry.Name()))
		header.Method = zip.Store
		destination, createErr := archive.CreateHeader(header)
		if createErr == nil {
			_, _ = io.Copy(destination, source)
		}
		source.Close()
	}
	_ = archive.Close()
}

func (r *Router) restoreBackup(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, maxBackupUploadSize)
	if err := req.ParseMultipartForm(maxBackupUploadSize); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Backup archive is too large or invalid")
		return
	}
	uploaded, _, err := req.FormFile("backup")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "A backup archive is required")
		return
	}
	defer uploaded.Close()

	temporary, err := os.CreateTemp("", "wavenode-restore-*.zip")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Could not prepare restore")
		return
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := io.Copy(temporary, uploaded); err != nil {
		temporary.Close()
		writeJSONError(w, http.StatusBadRequest, "Could not read backup archive")
		return
	}
	if err := temporary.Close(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Could not finish reading backup")
		return
	}

	archive, err := zip.OpenReader(temporaryPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Backup archive is not a valid zip file")
		return
	}
	defer archive.Close()

	var snapshot database.BackupSnapshot
	artworkFiles := make(map[string][]byte)
	var artworkBytes int64
	for _, file := range archive.File {
		cleanName := filepath.ToSlash(filepath.Clean(file.Name))
		if cleanName == "backup.json" {
			reader, openErr := file.Open()
			if openErr != nil {
				writeJSONError(w, http.StatusBadRequest, "Backup manifest cannot be read")
				return
			}
			decodeErr := json.NewDecoder(reader).Decode(&snapshot)
			reader.Close()
			if decodeErr != nil {
				writeJSONError(w, http.StatusBadRequest, "Backup manifest is invalid")
				return
			}
			continue
		}
		if !strings.HasPrefix(cleanName, "artwork/") || strings.Contains(cleanName, "..") {
			continue
		}
		if file.UncompressedSize64 > maxBackupArtworkFileSize || artworkBytes+int64(file.UncompressedSize64) > maxBackupArtworkTotalSize {
			writeJSONError(w, http.StatusBadRequest, "Backup artwork exceeds the supported size limit")
			return
		}
		reader, openErr := file.Open()
		if openErr != nil {
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, maxBackupArtworkFileSize+1))
		reader.Close()
		if readErr != nil || len(data) > maxBackupArtworkFileSize {
			writeJSONError(w, http.StatusBadRequest, "Backup artwork cannot be read")
			return
		}
		artworkBytes += int64(len(data))
		artworkFiles[filepath.Base(cleanName)] = data
	}
	if snapshot.FormatVersion == 0 {
		writeJSONError(w, http.StatusBadRequest, "Backup manifest is missing")
		return
	}

	if err := r.db.RestoreBackupSnapshot(&snapshot); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	artworkDirectory := utils.ArtworkDirectory()
	if err := os.MkdirAll(artworkDirectory, 0755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Database restored, but the artwork directory is unavailable")
		return
	}
	entries, err := os.ReadDir(artworkDirectory)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Database restored, but stored artwork could not be replaced")
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			if err := os.Remove(filepath.Join(artworkDirectory, entry.Name())); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Database restored, but existing artwork could not be removed")
				return
			}
		}
	}
	for name, data := range artworkFiles {
		if err := os.WriteFile(filepath.Join(artworkDirectory, filepath.Base(name)), data, 0644); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Database restored, but artwork could not be written")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Backup restored. Sign in again to refresh your session.",
	})
}
