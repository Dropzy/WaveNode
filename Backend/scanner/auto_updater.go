package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"music-server/database"
)

const (
	AutoScanEnabledSettingKey  = "auto_scan_enabled"
	AutoScanIntervalSettingKey = "auto_scan_interval_minutes"
)

type AutoUpdateSettings struct {
	Enabled         bool       `json:"enabled"`
	IntervalMinutes int        `json:"interval_minutes"`
	LastCheckedAt   *time.Time `json:"last_checked_at,omitempty"`
	LastScanAt      *time.Time `json:"last_scan_at,omitempty"`
	LastReason      string     `json:"last_reason,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
}

type AutoUpdater struct {
	db      *database.DB
	scanner *Scanner
	mu      sync.RWMutex
	status  AutoUpdateSettings
	hash    string
}

func NewAutoUpdater(db *database.DB, scanner *Scanner) *AutoUpdater {
	return &AutoUpdater{db: db, scanner: scanner}
}

func (u *AutoUpdater) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		u.check(false)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.check(true)
			}
		}
	}()
}

func (u *AutoUpdater) Settings() (AutoUpdateSettings, error) {
	enabledValue, err := u.db.GetSetting(AutoScanEnabledSettingKey)
	if err != nil {
		return AutoUpdateSettings{}, err
	}
	intervalValue, err := u.db.GetSetting(AutoScanIntervalSettingKey)
	if err != nil {
		return AutoUpdateSettings{}, err
	}

	interval := 60
	if parsed, parseErr := strconv.Atoi(intervalValue); parseErr == nil && parsed >= 1 {
		interval = parsed
	}

	u.mu.RLock()
	status := u.status
	u.mu.RUnlock()
	status.Enabled = enabledValue == "true"
	status.IntervalMinutes = interval
	return status, nil
}

func (u *AutoUpdater) UpdateSettings(enabled bool, intervalMinutes int) (AutoUpdateSettings, error) {
	if intervalMinutes < 1 || intervalMinutes > 10080 {
		return AutoUpdateSettings{}, fmt.Errorf("scan interval must be between 1 minute and 7 days")
	}
	if err := u.db.SetSetting(AutoScanEnabledSettingKey, strconv.FormatBool(enabled)); err != nil {
		return AutoUpdateSettings{}, err
	}
	if err := u.db.SetSetting(AutoScanIntervalSettingKey, strconv.Itoa(intervalMinutes)); err != nil {
		return AutoUpdateSettings{}, err
	}
	return u.Settings()
}

func (u *AutoUpdater) check(allowScan bool) {
	settings, err := u.Settings()
	if err != nil {
		u.setError(err)
		return
	}
	now := time.Now()
	u.mu.Lock()
	u.status.LastCheckedAt = &now
	u.mu.Unlock()
	if !settings.Enabled {
		return
	}

	fingerprint, err := u.libraryFingerprint()
	if err != nil {
		u.setError(err)
		return
	}

	u.mu.Lock()
	previousHash := u.hash
	u.hash = fingerprint
	lastScanAt := u.status.LastScanAt
	u.status.LastError = ""
	u.mu.Unlock()

	if !allowScan || u.scanner.IsScanning() {
		return
	}

	reason := ""
	if previousHash != "" && previousHash != fingerprint {
		reason = "File changes detected"
	} else if lastScanAt == nil || now.Sub(*lastScanAt) >= time.Duration(settings.IntervalMinutes)*time.Minute {
		reason = "Scheduled scan"
	}
	if reason == "" {
		return
	}

	if _, err := u.scanner.StartScan(); err != nil {
		u.setError(err)
		return
	}
	u.mu.Lock()
	u.status.LastScanAt = &now
	u.status.LastReason = reason
	u.status.LastError = ""
	u.mu.Unlock()
	log.Printf("Automatic library update started: %s", reason)
}

func (u *AutoUpdater) libraryFingerprint() (string, error) {
	sources, err := u.db.GetMusicSources()
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, source := range sources {
		info, statErr := os.Stat(source.Path)
		if statErr != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(source.Path, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || !u.scanner.isMusicFile(strings.ToLower(filepath.Ext(path))) {
				return nil
			}
			if fileInfo, infoErr := entry.Info(); infoErr == nil {
				_, _ = fmt.Fprintf(hash, "%s|%d|%d\n", path, fileInfo.Size(), fileInfo.ModTime().UnixNano())
			}
			return nil
		})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (u *AutoUpdater) setError(err error) {
	now := time.Now()
	u.mu.Lock()
	u.status.LastCheckedAt = &now
	u.status.LastError = err.Error()
	u.mu.Unlock()
	log.Printf("Automatic library update check failed: %v", err)
}
