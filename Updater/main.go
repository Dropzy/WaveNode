package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type status struct {
	CurrentVersion    string    `json:"current_version"`
	LatestVersion     string    `json:"latest_version"`
	UpdateAvailable   bool      `json:"update_available"`
	Enabled           bool      `json:"enabled"`
	CommandConfigured bool      `json:"command_configured"`
	State             string    `json:"state"`
	Message           string    `json:"message"`
	StartedAt         time.Time `json:"started_at,omitempty"`
	FinishedAt        time.Time `json:"finished_at,omitempty"`
	LogTail           []string  `json:"log_tail"`
}

type server struct {
	mu     sync.Mutex
	status status
}

var WaveNodeVersion = "dev"

func main() {
	srv := &server{
		status: status{
			CurrentVersion:    strings.TrimPrefix(env("WAVENODE_VERSION", WaveNodeVersion), "v"),
			Enabled:           true,
			CommandConfigured: true,
			State:             "idle",
			Message:           "Updater is ready.",
			LogTail:           []string{},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/status", srv.withAuth(srv.handleStatus))
	mux.HandleFunc("/update", srv.withAuth(srv.handleUpdate))

	addr := ":" + env("UPDATER_PORT", "8090")
	log.Printf("WaveNode updater listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.snapshot())
}

func (s *server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		TargetVersion string `json:"target_version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)

	s.mu.Lock()
	if s.status.State == "running" {
		current := s.status
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, current)
		return
	}
	s.status.State = "running"
	s.status.Message = "Pulling and restarting WaveNode containers..."
	s.status.LatestVersion = payload.TargetVersion
	s.status.StartedAt = time.Now()
	s.status.FinishedAt = time.Time{}
	s.status.LogTail = []string{}
	current := s.status
	s.mu.Unlock()

	go s.runUpdate()
	writeJSON(w, http.StatusAccepted, current)
}

func (s *server) runUpdate() {
	timeoutSeconds, err := strconv.Atoi(env("UPDATER_TIMEOUT_SECONDS", "900"))
	if err != nil || timeoutSeconds <= 0 {
		timeoutSeconds = 900
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	composeArgs := updaterComposeArgs()
	services := splitCSV(env("UPDATER_SERVICES", "backend,frontend"))

	var output bytes.Buffer
	steps := [][]string{
		append(append([]string{}, composeArgs...), append([]string{"pull"}, services...)...),
		append(append([]string{}, composeArgs...), append([]string{"up", "-d"}, services...)...),
		{"image", "prune", "-f"},
	}

	for _, args := range steps {
		output.WriteString("$ docker " + strings.Join(args, " ") + "\n")
		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Stdout = &output
		cmd.Stderr = &output
		if err := cmd.Run(); err != nil {
			s.finish("failed", fmt.Sprintf("Update failed: %v", err), output.String())
			return
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		s.finish("failed", "Update timed out.", output.String())
		return
	}
	s.finish("completed", "WaveNode containers updated.", output.String())
}

func updaterComposeArgs() []string {
	args := []string{"compose", "--project-name", env("UPDATER_PROJECT_NAME", "wavenode")}
	for _, file := range splitCSV(env("UPDATER_COMPOSE_FILES", "/compose/docker-compose.yml")) {
		args = append(args, "-f", file)
	}
	return args
}

func (s *server) finish(state, message, output string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.State = state
	s.status.Message = message
	s.status.FinishedAt = time.Now()
	s.status.LogTail = tailLines(output, 100)
}

func (s *server) snapshot() status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	token := strings.TrimSpace(os.Getenv("UPDATER_TOKEN"))
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func tailLines(value string, limit int) []string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filtered = append(filtered, line)
		}
	}
	if len(filtered) <= limit {
		return filtered
	}
	return filtered[len(filtered)-limit:]
}
