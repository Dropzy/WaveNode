package router

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"music-server/database"
	"music-server/handlers"
	"music-server/utils"
)

type setupRequest struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	MusicPaths  []string `json:"music_paths"`
	ArtworkPath string   `json:"artwork_path"`
}

func (r *Router) getSetupStatus(w http.ResponseWriter, req *http.Request) {
	required, err := r.db.IsSetupRequired()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	artworkPath, err := r.db.GetSetting(database.ArtworkPathSettingKey)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if artworkPath == "" {
		artworkPath = utils.ArtworkDirectory()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"required":             required,
			"token_required":       required && r.setupToken != "",
			"default_artwork_path": artworkPath,
			"registration_enabled": r.authHandler.RegistrationEnabled(),
		},
	})
}

func (r *Router) browseSetupDirectories(w http.ResponseWriter, req *http.Request) {
	required, err := r.db.IsSetupRequired()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !required {
		writeJSONError(w, http.StatusConflict, "Initial setup has already been completed")
		return
	}
	if !r.setupAuthorized(req) {
		writeJSONError(w, http.StatusUnauthorized, "The setup access code is incorrect")
		return
	}
	r.browseMusicDirectories(w, req)
}

func (r *Router) completeSetup(w http.ResponseWriter, req *http.Request) {
	required, err := r.db.IsSetupRequired()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !required {
		writeJSONError(w, http.StatusConflict, "Initial setup has already been completed")
		return
	}
	if !r.setupAuthorized(req) {
		writeJSONError(w, http.StatusUnauthorized, "The setup access code is incorrect")
		return
	}

	var payload setupRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "The setup details could not be read")
		return
	}

	payload.Username = strings.TrimSpace(payload.Username)
	payload.Email = strings.TrimSpace(payload.Email)
	if len(payload.Username) < 3 {
		writeJSONError(w, http.StatusBadRequest, "Administrator username must be at least 3 characters")
		return
	}
	if _, err := mail.ParseAddress(payload.Email); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Enter a valid administrator email address")
		return
	}
	if len(payload.Password) < 8 {
		writeJSONError(w, http.StatusBadRequest, "Administrator password must be at least 8 characters")
		return
	}

	artworkPath, err := prepareWritableDirectory(payload.ArtworkPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := r.db.CompleteInitialSetup(database.InitialSetupInput{
		Username:    payload.Username,
		Email:       payload.Email,
		Password:    payload.Password,
		MusicPaths:  payload.MusicPaths,
		ArtworkPath: artworkPath,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, database.ErrSetupAlreadyComplete) {
			status = http.StatusConflict
		}
		writeJSONError(w, status, err.Error())
		return
	}

	if err := os.Setenv("WAVENODE_ARTWORK_PATH", artworkPath); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Setup completed, but artwork storage could not be activated")
		return
	}

	token, err := r.authHandler.GenerateJWT(user.ID, req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Setup completed, but sign-in could not be started")
		return
	}

	var scan interface{}
	var scanWarning string
	if handlers.ScannerInstance == nil {
		scanWarning = "The library scanner is not available. Start a scan from administration."
	} else {
		startedScan, scanErr := handlers.ScannerInstance.StartScan()
		if scanErr != nil {
			scanWarning = fmt.Sprintf("Setup completed, but the first scan could not start: %v", scanErr)
		} else {
			scan = startedScan
		}
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "WaveNode setup completed",
		"data": map[string]interface{}{
			"token":        token,
			"user":         user,
			"scan":         scan,
			"scan_warning": scanWarning,
		},
	})
}

func (r *Router) setupAuthorized(req *http.Request) bool {
	if r.setupToken == "" {
		return true
	}
	provided := strings.TrimSpace(req.Header.Get("X-WaveNode-Setup-Token"))
	return subtle.ConstantTimeCompare([]byte(provided), []byte(r.setupToken)) == 1
}

func prepareWritableDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("an artwork storage folder is required")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("artwork storage folder must be an absolute path")
	}

	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(cleanPath, 0755); err != nil {
		return "", fmt.Errorf("artwork storage folder cannot be created: %v", err)
	}
	info, err := os.Stat(cleanPath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("artwork storage folder is not accessible")
	}

	testFile, err := os.CreateTemp(cleanPath, ".wavenode-write-test-*")
	if err != nil {
		return "", fmt.Errorf("artwork storage folder is not writable: %v", err)
	}
	testName := testFile.Name()
	if closeErr := testFile.Close(); closeErr != nil {
		os.Remove(testName)
		return "", fmt.Errorf("artwork storage folder could not be verified: %v", closeErr)
	}
	if err := os.Remove(testName); err != nil {
		return "", fmt.Errorf("artwork storage folder could not be verified: %v", err)
	}
	return cleanPath, nil
}
