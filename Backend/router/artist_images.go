package router

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"music-server/database"
	"music-server/metadata"
	"music-server/utils"
	"music-server/websocket"

	"github.com/gorilla/mux"
)

const maxArtistImageSize = 10 << 20

var supportedArtistImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func (r *Router) uploadArtistImage(w http.ResponseWriter, req *http.Request) {
	artist, err := r.db.GetArtistByHash(mux.Vars(req)["id"])
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	req.Body = http.MaxBytesReader(w, req.Body, maxArtistImageSize+(1<<20))
	if err := req.ParseMultipartForm(maxArtistImageSize); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Image must be 10 MB or smaller")
		return
	}

	file, _, err := req.FormFile("image")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Select an image to upload")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxArtistImageSize+1))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Could not read the selected image")
		return
	}
	if len(data) == 0 || len(data) > maxArtistImageSize {
		writeJSONError(w, http.StatusBadRequest, "Image must be between 1 byte and 10 MB")
		return
	}

	extension, ok := supportedArtistImageTypes[http.DetectContentType(data)]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "Use a JPEG, PNG, WebP, or GIF image")
		return
	}

	imageURL, err := saveArtistImage(data, artist, "manual", extension)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := r.db.UpdateArtistImage(artist.ID, imageURL); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	artist.ImageURL = imageURL
	artist.ImageSmallURL = imageURL
	artist.ImageMediumURL = imageURL
	artist.ImageLargeURL = imageURL
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": artist})
}

func (r *Router) deleteArtistImage(w http.ResponseWriter, req *http.Request) {
	artist, err := r.db.GetArtistByHash(mux.Vars(req)["id"])
	if err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := r.db.UpdateArtistImage(artist.ID, ""); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (r *Router) discoverArtistImages(w http.ResponseWriter, req *http.Request) {
	activeScan, err := r.scanStore.GetCurrentScan()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if activeScan != nil {
		writeJSONError(w, http.StatusConflict, "Wait for the current maintenance job to finish")
		return
	}

	scan, err := r.scanStore.CreateScan("artist-image-discovery")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	websocket.BroadcastScanUpdate(scan)
	go r.runArtistImageDiscovery(scan.ID)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"success": true,
		"message": "Local artist image discovery started",
		"data":    scan,
	})
}

func (r *Router) runArtistImageDiscovery(scanID string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.finishArtistImageDiscovery(scanID, "failed", 0, 0, 0, 0, []string{
				fmt.Sprintf("Artist image discovery stopped unexpectedly: %v", recovered),
			})
		}
	}()

	artists, err := r.db.GetAllArtists()
	if err != nil {
		r.finishArtistImageDiscovery(scanID, "failed", 0, 0, 0, 0, []string{err.Error()})
		return
	}

	total := len(artists)
	folderImages := 0
	embeddedImages := 0
	unchanged := 0
	errors := make([]string, 0)
	parser := metadata.NewMetadataParser()

	for index := range artists {
		artist := &artists[index]
		if isLocalArtistImage(artist.ImageURL) && utils.ArtworkExists(artist.ImageURL) {
			unchanged++
			r.updateArtistImageDiscovery(scanID, artist.Name, index+1, total, folderImages, embeddedImages, unchanged, errors)
			continue
		}

		tracks, trackErr := r.db.GetArtistTracks(artist.Name)
		if trackErr != nil {
			errors = appendScanError(errors, fmt.Sprintf("%s: %v", artist.Name, trackErr))
			unchanged++
			r.updateArtistImageDiscovery(scanID, artist.Name, index+1, total, folderImages, embeddedImages, unchanged, errors)
			continue
		}
		if len(tracks) == 0 {
			unchanged++
			r.updateArtistImageDiscovery(scanID, artist.Name, index+1, total, folderImages, embeddedImages, unchanged, errors)
			continue
		}

		if data, extension, found := findArtistFolderImage(artist.Name, tracks); found {
			imageURL, saveErr := saveArtistImage(data, artist, "folder", extension)
			if saveErr != nil {
				errors = appendScanError(errors, fmt.Sprintf("%s: %v", artist.Name, saveErr))
			} else if updateErr := r.db.UpdateArtistImage(artist.ID, imageURL); updateErr != nil {
				errors = appendScanError(errors, fmt.Sprintf("%s: %v", artist.Name, updateErr))
			} else {
				folderImages++
				r.updateArtistImageDiscovery(scanID, artist.Name, index+1, total, folderImages, embeddedImages, unchanged, errors)
				continue
			}
		}

		foundEmbedded := false
		var embeddedError error
		for _, track := range tracks {
			data, format, extractErr := parser.ExtractArtwork(track.FilePath)
			if extractErr != nil || len(data) == 0 {
				continue
			}

			imageURL, saveErr := saveArtistImage(data, artist, "embedded", "."+strings.TrimPrefix(format, "."))
			if saveErr != nil {
				embeddedError = saveErr
				break
			}
			if updateErr := r.db.UpdateArtistImage(artist.ID, imageURL); updateErr != nil {
				embeddedError = updateErr
			} else {
				embeddedImages++
				foundEmbedded = true
			}
			break
		}
		if embeddedError != nil {
			errors = appendScanError(errors, fmt.Sprintf("%s: %v", artist.Name, embeddedError))
		}
		if !foundEmbedded {
			unchanged++
		}
		r.updateArtistImageDiscovery(scanID, artist.Name, index+1, total, folderImages, embeddedImages, unchanged, errors)
	}

	status := "completed"
	if len(errors) > 0 {
		status = "completed_with_errors"
	}
	r.finishArtistImageDiscovery(scanID, status, total, folderImages, embeddedImages, unchanged, errors)
}

func (r *Router) updateArtistImageDiscovery(scanID, artistName string, processed, total, folderImages, embeddedImages, unchanged int, errors []string) {
	if err := r.scanStore.UpdateScanResult(scanID, "running", artistName, processed, total, folderImages, embeddedImages, unchanged, errors, false); err != nil {
		return
	}
	if scan, err := r.scanStore.GetScan(scanID); err == nil {
		websocket.BroadcastScanUpdate(scan)
	}
}

func (r *Router) finishArtistImageDiscovery(scanID, status string, processed, folderImages, embeddedImages, unchanged int, errors []string) {
	total := processed
	if scan, err := r.scanStore.GetScan(scanID); err == nil && scan.TotalFiles > total {
		total = scan.TotalFiles
	}
	if err := r.scanStore.UpdateScanResult(scanID, status, "", processed, total, folderImages, embeddedImages, unchanged, errors, true); err != nil {
		return
	}
	if scan, err := r.scanStore.GetScan(scanID); err == nil {
		websocket.BroadcastScanUpdate(scan)
	}
}

func appendScanError(errors []string, message string) []string {
	const maxScanErrors = 100
	for _, existing := range errors {
		if existing == message {
			return errors
		}
	}
	if len(errors) >= maxScanErrors {
		return errors
	}
	return append(errors, message)
}

func isLocalArtistImage(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value != "" && !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://")
}

func saveArtistImage(data []byte, artist *database.Artist, source, extension string) (string, error) {
	extension = strings.ToLower(extension)
	if extension == ".jpeg" {
		extension = ".jpg"
	}
	if _, ok := supportedArtistImageTypes[http.DetectContentType(data)]; !ok {
		return "", fmt.Errorf("unsupported artist image type")
	}

	artworkDir := utils.ArtworkDirectory()
	if err := os.MkdirAll(artworkDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create artwork directory: %w", err)
	}

	artistHash := database.GenerateArtistHash(artist.ID)
	contentHash := database.GenerateArtistHash(source + fmt.Sprintf("%x", data))
	filename := fmt.Sprintf("artist-%s-%s-%s%s", source, artistHash[:16], contentHash[:12], extension)
	path := filepath.Join(artworkDir, filename)
	imageURL := "/artwork/" + filename

	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return imageURL, nil
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return imageURL, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to save artist image: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return "", fmt.Errorf("failed to save artist image: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("failed to finish artist image: %w", err)
	}

	return imageURL, nil
}

func findArtistFolderImage(artistName string, tracks []database.Music) ([]byte, string, bool) {
	directories := make(map[string]struct{})
	for _, track := range tracks {
		directory := filepath.Dir(track.FilePath)
		directories[directory] = struct{}{}
		directories[filepath.Dir(directory)] = struct{}{}
	}

	orderedDirectories := make([]string, 0, len(directories))
	for directory := range directories {
		orderedDirectories = append(orderedDirectories, directory)
	}
	sort.Strings(orderedDirectories)

	normalizedArtist := normalizeImageName(artistName)
	for _, directory := range orderedDirectories {
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			extension := strings.ToLower(filepath.Ext(entry.Name()))
			if extension != ".jpg" && extension != ".jpeg" && extension != ".png" && extension != ".webp" && extension != ".gif" {
				continue
			}

			base := normalizeImageName(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
			if base != "artist" && base != "band" && base != "portrait" && base != normalizedArtist {
				continue
			}

			data, readErr := os.ReadFile(filepath.Join(directory, entry.Name()))
			if readErr == nil && len(data) > 0 {
				return data, extension, true
			}
		}
	}

	return nil, "", false
}

func normalizeImageName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "")
	return replacer.Replace(value)
}
