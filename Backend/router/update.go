package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultUpdateRepository = "Dropzy/WaveNode"

type UpdateStatus struct {
	CurrentVersion    string    `json:"current_version"`
	LatestVersion     string    `json:"latest_version"`
	UpdateAvailable   bool      `json:"update_available"`
	ReleaseURL        string    `json:"release_url"`
	ReleaseNotes      string    `json:"release_notes"`
	CheckedAt         time.Time `json:"checked_at,omitempty"`
	Enabled           bool      `json:"enabled"`
	CommandConfigured bool      `json:"command_configured"`
	UpdaterURL        string    `json:"updater_url,omitempty"`
	Repository        string    `json:"repository"`
	State             string    `json:"state"`
	Message           string    `json:"message"`
	StartedAt         time.Time `json:"started_at,omitempty"`
	FinishedAt        time.Time `json:"finished_at,omitempty"`
	LogTail           []string  `json:"log_tail"`
}

type UpdateManager struct {
	mu     sync.Mutex
	status UpdateStatus
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func NewUpdateManager(currentVersion string) *UpdateManager {
	repository := strings.TrimSpace(os.Getenv("WAVENODE_UPDATE_REPOSITORY"))
	if repository == "" {
		repository = defaultUpdateRepository
	}
	command := strings.TrimSpace(os.Getenv("WAVENODE_UPDATE_COMMAND"))
	updaterURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAVENODE_UPDATER_URL")), "/")

	return &UpdateManager{
		status: UpdateStatus{
			CurrentVersion:    currentVersion,
			Enabled:           command != "" || updaterURL != "",
			CommandConfigured: command != "" || updaterURL != "",
			UpdaterURL:        updaterURL,
			Repository:        repository,
			State:             "idle",
			Message:           "Update checks have not run yet.",
			LogTail:           []string{},
		},
	}
}

func (m *UpdateManager) Status() UpdateStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *UpdateManager) RefreshStatus(ctx context.Context) UpdateStatus {
	updaterURL := m.updaterURL()
	if updaterURL == "" {
		return m.Status()
	}

	var remote UpdateStatus
	if err := updaterRequest(ctx, http.MethodGet, updaterURL+"/status", nil, &remote); err != nil {
		m.mu.Lock()
		m.status.Enabled = true
		m.status.CommandConfigured = true
		m.status.UpdaterURL = updaterURL
		if m.status.State == "running" || m.status.State == "checking" {
			m.status.State = "failed"
		}
		m.status.Message = "Updater service is not reachable: " + err.Error()
		status := m.status
		m.mu.Unlock()
		return status
	}

	m.mu.Lock()
	m.status.Enabled = true
	m.status.CommandConfigured = true
	m.status.UpdaterURL = updaterURL
	m.status.State = remote.State
	m.status.Message = remote.Message
	m.status.StartedAt = remote.StartedAt
	m.status.FinishedAt = remote.FinishedAt
	m.status.LogTail = remote.LogTail
	if remote.CurrentVersion != "" {
		m.status.CurrentVersion = remote.CurrentVersion
	}
	if remote.LatestVersion != "" {
		m.status.LatestVersion = remote.LatestVersion
	}
	status := m.status
	m.mu.Unlock()
	return status
}

func (m *UpdateManager) Check(ctx context.Context) (UpdateStatus, error) {
	m.mu.Lock()
	if m.status.State == "running" {
		status := m.status
		m.mu.Unlock()
		return status, fmt.Errorf("an update is already running")
	}
	m.status.State = "checking"
	m.status.Message = "Checking for updates..."
	m.mu.Unlock()

	release, err := fetchLatestRelease(ctx, m.repository())

	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.CheckedAt = time.Now()
	if err != nil {
		m.status.State = "failed"
		m.status.Message = err.Error()
		return m.status, err
	}

	m.status.LatestVersion = strings.TrimSpace(release.TagName)
	m.status.ReleaseURL = release.HTMLURL
	m.status.ReleaseNotes = release.Body
	m.status.UpdateAvailable = compareVersions(release.TagName, m.status.CurrentVersion) > 0
	if m.status.UpdateAvailable {
		m.status.State = "ready"
		m.status.Message = fmt.Sprintf("WaveNode %s is available.", release.TagName)
	} else {
		m.status.State = "idle"
		m.status.Message = "WaveNode is up to date."
	}
	return m.status, nil
}

func (m *UpdateManager) Run(ctx context.Context) (UpdateStatus, error) {
	if updaterURL := m.updaterURL(); updaterURL != "" {
		return m.runUpdater(ctx, updaterURL)
	}

	command := strings.TrimSpace(os.Getenv("WAVENODE_UPDATE_COMMAND"))
	if command == "" {
		m.mu.Lock()
		m.status.Enabled = false
		m.status.CommandConfigured = false
		m.status.State = "unavailable"
		m.status.Message = "No update command is configured on this server."
		status := m.status
		m.mu.Unlock()
		return status, fmt.Errorf("no update command is configured on this server")
	}

	m.mu.Lock()
	if m.status.State == "running" || m.status.State == "checking" {
		status := m.status
		m.mu.Unlock()
		return status, fmt.Errorf("an update action is already running")
	}
	m.status.Enabled = true
	m.status.CommandConfigured = true
	m.status.State = "running"
	m.status.Message = "Update is running..."
	m.status.StartedAt = time.Now()
	m.status.FinishedAt = time.Time{}
	m.status.LogTail = []string{}
	status := m.status
	m.mu.Unlock()

	go m.runCommand(ctx, command)
	return status, nil
}

func (m *UpdateManager) runUpdater(ctx context.Context, updaterURL string) (UpdateStatus, error) {
	m.mu.Lock()
	if m.status.State == "running" || m.status.State == "checking" {
		status := m.status
		m.mu.Unlock()
		return status, fmt.Errorf("an update action is already running")
	}
	m.status.Enabled = true
	m.status.CommandConfigured = true
	m.status.UpdaterURL = updaterURL
	m.status.State = "running"
	m.status.Message = "Update is running..."
	m.status.StartedAt = time.Now()
	m.status.FinishedAt = time.Time{}
	m.status.LogTail = []string{}
	status := m.status
	m.mu.Unlock()

	payload := strings.NewReader(fmt.Sprintf(`{"target_version":%q}`, status.LatestVersion))
	var remote UpdateStatus
	if err := updaterRequest(ctx, http.MethodPost, updaterURL+"/update", payload, &remote); err != nil {
		m.mu.Lock()
		m.status.State = "failed"
		m.status.Message = "Updater service rejected the update: " + err.Error()
		status = m.status
		m.mu.Unlock()
		return status, err
	}

	m.mu.Lock()
	m.status.State = remote.State
	m.status.Message = remote.Message
	m.status.StartedAt = remote.StartedAt
	m.status.FinishedAt = remote.FinishedAt
	m.status.LogTail = remote.LogTail
	status = m.status
	m.mu.Unlock()
	return status, nil
}

func (m *UpdateManager) repository() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status.Repository
}

func (m *UpdateManager) updaterURL() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.UpdaterURL != "" {
		return m.status.UpdaterURL
	}
	return strings.TrimRight(strings.TrimSpace(os.Getenv("WAVENODE_UPDATER_URL")), "/")
}

func (m *UpdateManager) runCommand(parent context.Context, command string) {
	timeout := updateTimeout()
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	shell := "/bin/sh"
	args := []string{"-c", command}
	if _, err := os.Stat(shell); err != nil {
		shell = "sh"
	}

	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = append(os.Environ(),
		"WAVENODE_CURRENT_VERSION="+m.Status().CurrentVersion,
		"WAVENODE_TARGET_VERSION="+m.Status().LatestVersion,
	)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.FinishedAt = time.Now()
	m.status.LogTail = tailLines(output.String(), 80)
	if ctx.Err() == context.DeadlineExceeded {
		m.status.State = "failed"
		m.status.Message = fmt.Sprintf("Update timed out after %s.", timeout)
		return
	}
	if err != nil {
		m.status.State = "failed"
		m.status.Message = fmt.Sprintf("Update failed: %v", err)
		return
	}
	m.status.State = "completed"
	m.status.Message = "Update command completed. WaveNode may restart shortly."
}

func fetchLatestRelease(ctx context.Context, repository string) (*githubRelease, error) {
	repository = strings.Trim(repository, "/ ")
	if repository == "" || !strings.Contains(repository, "/") {
		return nil, fmt.Errorf("WAVENODE_UPDATE_REPOSITORY must be in owner/repo format")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repository+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "WaveNode/"+WaveNodeVersion)

	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check GitHub releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no GitHub release was found for %s", repository)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub release check failed with status %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub release response: %w", err)
	}
	if strings.TrimSpace(release.TagName) == "" {
		return nil, fmt.Errorf("latest GitHub release did not include a version tag")
	}
	return &release, nil
}

func updateTimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WAVENODE_UPDATE_TIMEOUT_SECONDS")))
	if err != nil || seconds <= 0 {
		seconds = 900
	}
	return time.Duration(seconds) * time.Second
}

var versionPartPattern = regexp.MustCompile(`\d+|[A-Za-z]+`)

func compareVersions(left, right string) int {
	leftParts := versionPartPattern.FindAllString(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(left)), "v"), -1)
	rightParts := versionPartPattern.FindAllString(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(right)), "v"), -1)
	maxParts := len(leftParts)
	if len(rightParts) > maxParts {
		maxParts = len(rightParts)
	}

	for i := 0; i < maxParts; i++ {
		leftPart := "0"
		rightPart := "0"
		if i < len(leftParts) {
			leftPart = leftParts[i]
		}
		if i < len(rightParts) {
			rightPart = rightParts[i]
		}

		leftNumber, leftNumberErr := strconv.Atoi(leftPart)
		rightNumber, rightNumberErr := strconv.Atoi(rightPart)
		if leftNumberErr == nil && rightNumberErr == nil {
			if leftNumber > rightNumber {
				return 1
			}
			if leftNumber < rightNumber {
				return -1
			}
			continue
		}

		if leftPart > rightPart {
			return 1
		}
		if leftPart < rightPart {
			return -1
		}
	}
	return 0
}

func tailLines(value string, limit int) []string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, line)
		}
	}
	if len(filtered) <= limit {
		return filtered
	}
	return filtered[len(filtered)-limit:]
}

func updaterRequest(ctx context.Context, method, target string, body io.Reader, out *UpdateStatus) error {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	token := strings.TrimSpace(os.Getenv("WAVENODE_UPDATER_TOKEN"))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = resp.Status
		}
		return errors.New(message)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return err
		}
	}
	return nil
}

func (r *Router) getUpdateStatus(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": r.updateManager.RefreshStatus(req.Context())})
}

func (r *Router) checkForUpdate(w http.ResponseWriter, req *http.Request) {
	status, err := r.updateManager.Check(req.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": false, "error": err.Error(), "data": status})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": status})
}

func (r *Router) runUpdate(w http.ResponseWriter, req *http.Request) {
	status, err := r.updateManager.Run(context.Background())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": err.Error(), "data": status})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": status})
}
