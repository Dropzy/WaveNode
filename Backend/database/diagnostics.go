package database

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const diagnosticDetailLimit = 100

var supportedAudioExtensions = map[string]struct{}{
	".mp3":  {},
	".flac": {},
	".wav":  {},
	".m4a":  {},
	".aac":  {},
	".ogg":  {},
	".wma":  {},
	".opus": {},
}

type SourceDiagnostic struct {
	Path        string  `json:"path"`
	Accessible  bool    `json:"accessible"`
	Error       string  `json:"error,omitempty"`
	TotalBytes  uint64  `json:"total_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
	SpaceStatus string  `json:"space_status"`
}

type TrackDiagnostic struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	FilePath string `json:"file_path"`
	Format   string `json:"format"`
	Issue    string `json:"issue"`
}

type DuplicateDiagnostic struct {
	Title    string   `json:"title"`
	Artist   string   `json:"artist"`
	Count    int      `json:"count"`
	TrackIDs []string `json:"track_ids"`
	Paths    []string `json:"paths"`
}

type LibraryDiagnostics struct {
	IndexedTracks            int                   `json:"indexed_tracks"`
	HealthyTracks            int                   `json:"healthy_tracks"`
	HealthScore              int                   `json:"health_score"`
	IssueCount               int                   `json:"issue_count"`
	MissingFiles             int                   `json:"missing_files"`
	DuplicateGroups          int                   `json:"duplicate_groups"`
	InvalidMetadata          int                   `json:"invalid_metadata"`
	UnsupportedFormats       int                   `json:"unsupported_formats"`
	MissingArtwork           int                   `json:"missing_artwork"`
	Sources                  []SourceDiagnostic    `json:"sources"`
	UnavailableSource        int                   `json:"unavailable_sources"`
	LowSpaceSources          int                   `json:"low_space_sources"`
	MissingFileDetails       []TrackDiagnostic     `json:"missing_file_details"`
	InvalidMetadataDetails   []TrackDiagnostic     `json:"invalid_metadata_details"`
	UnsupportedFormatDetails []TrackDiagnostic     `json:"unsupported_format_details"`
	MissingArtworkDetails    []TrackDiagnostic     `json:"missing_artwork_details"`
	DuplicateDetails         []DuplicateDiagnostic `json:"duplicate_details"`
	DetailsTruncated         bool                  `json:"details_truncated"`
	GeneratedAt              time.Time             `json:"generated_at"`
}

type diagnosticTrack struct {
	ID            string
	Title         string
	Artist        string
	Album         string
	FilePath      string
	Format        string
	Duration      int
	HasMetadata   bool
	ImageURL      string
	CoverArtURL   string
	CoverArtSmall string
	CoverArtMed   string
	CoverArtLarge string
}

func (db *DB) GetLibraryDiagnostics() (*LibraryDiagnostics, error) {
	diagnostics := &LibraryDiagnostics{
		Sources:                  make([]SourceDiagnostic, 0),
		MissingFileDetails:       make([]TrackDiagnostic, 0),
		InvalidMetadataDetails:   make([]TrackDiagnostic, 0),
		UnsupportedFormatDetails: make([]TrackDiagnostic, 0),
		MissingArtworkDetails:    make([]TrackDiagnostic, 0),
		DuplicateDetails:         make([]DuplicateDiagnostic, 0),
		GeneratedAt:              time.Now().UTC(),
	}

	rows, err := db.conn.Query(`
		SELECT id, COALESCE(title, ''), COALESCE(artist, ''), COALESCE(album, ''),
		       COALESCE(file_path, ''), COALESCE(format, ''), COALESCE(duration, 0),
		       COALESCE(has_metadata, false), COALESCE(image_url, ''),
		       COALESCE(cover_art_url, ''), COALESCE(cover_art_small_url, ''),
		       COALESCE(cover_art_medium_url, ''), COALESCE(cover_art_large_url, '')
		FROM music
		ORDER BY upload_order DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	affectedTracks := make(map[string]struct{})
	duplicateTracks := make(map[string][]diagnosticTrack)
	tracks := make([]diagnosticTrack, 0, diagnostics.IndexedTracks)
	for rows.Next() {
		var track diagnosticTrack
		if err := rows.Scan(
			&track.ID, &track.Title, &track.Artist, &track.Album, &track.FilePath,
			&track.Format, &track.Duration, &track.HasMetadata, &track.ImageURL,
			&track.CoverArtURL, &track.CoverArtSmall, &track.CoverArtMed, &track.CoverArtLarge,
		); err != nil {
			return nil, err
		}

		diagnostics.IndexedTracks++
		tracks = append(tracks, track)
		key := strings.ToLower(strings.TrimSpace(track.Title)) + "\x00" + strings.ToLower(strings.TrimSpace(track.Artist))
		duplicateTracks[key] = append(duplicateTracks[key], track)

		if issue := metadataIssue(track); issue != "" {
			diagnostics.InvalidMetadata++
			diagnostics.IssueCount++
			affectedTracks[track.ID] = struct{}{}
			appendTrackDetail(&diagnostics.InvalidMetadataDetails, track, issue, diagnostics)
		}
		if issue := unsupportedFormatIssue(track); issue != "" {
			diagnostics.UnsupportedFormats++
			diagnostics.IssueCount++
			affectedTracks[track.ID] = struct{}{}
			appendTrackDetail(&diagnostics.UnsupportedFormatDetails, track, issue, diagnostics)
		}
		if !hasArtwork(track) {
			diagnostics.MissingArtwork++
			diagnostics.IssueCount++
			affectedTracks[track.ID] = struct{}{}
			appendTrackDetail(&diagnostics.MissingArtworkDetails, track, "No embedded or enriched artwork", diagnostics)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for index, issue := range missingFileIssues(tracks) {
		if issue == "" {
			continue
		}
		track := tracks[index]
		diagnostics.MissingFiles++
		diagnostics.IssueCount++
		affectedTracks[track.ID] = struct{}{}
		appendTrackDetail(&diagnostics.MissingFileDetails, track, issue, diagnostics)
	}

	duplicateKeys := make([]string, 0)
	for key, tracks := range duplicateTracks {
		if len(tracks) > 1 {
			duplicateKeys = append(duplicateKeys, key)
		}
	}
	sort.Strings(duplicateKeys)
	for _, key := range duplicateKeys {
		tracks := duplicateTracks[key]
		diagnostics.DuplicateGroups++
		diagnostics.IssueCount++
		for _, track := range tracks {
			affectedTracks[track.ID] = struct{}{}
		}
		if len(diagnostics.DuplicateDetails) >= diagnosticDetailLimit {
			diagnostics.DetailsTruncated = true
			continue
		}
		detail := DuplicateDiagnostic{
			Title:    tracks[0].Title,
			Artist:   tracks[0].Artist,
			Count:    len(tracks),
			TrackIDs: make([]string, 0, len(tracks)),
			Paths:    make([]string, 0, len(tracks)),
		}
		for _, track := range tracks {
			detail.TrackIDs = append(detail.TrackIDs, track.ID)
			detail.Paths = append(detail.Paths, track.FilePath)
		}
		diagnostics.DuplicateDetails = append(diagnostics.DuplicateDetails, detail)
	}

	sources, err := db.GetMusicSources()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		item := inspectSource(source.Path)
		if !item.Accessible {
			diagnostics.UnavailableSource++
			diagnostics.IssueCount++
		} else if item.SpaceStatus == "warning" || item.SpaceStatus == "critical" {
			diagnostics.LowSpaceSources++
			diagnostics.IssueCount++
		}
		diagnostics.Sources = append(diagnostics.Sources, item)
	}

	diagnostics.HealthyTracks = diagnostics.IndexedTracks - len(affectedTracks)
	if diagnostics.HealthyTracks < 0 {
		diagnostics.HealthyTracks = 0
	}
	if diagnostics.IndexedTracks == 0 {
		diagnostics.HealthScore = 100
	} else {
		diagnostics.HealthScore = int(math.Round(float64(diagnostics.HealthyTracks) * 100 / float64(diagnostics.IndexedTracks)))
	}
	if diagnostics.UnavailableSource > 0 && diagnostics.HealthScore > 75 {
		diagnostics.HealthScore = 75
	} else if diagnostics.LowSpaceSources > 0 && diagnostics.HealthScore > 90 {
		diagnostics.HealthScore = 90
	}

	return diagnostics, nil
}

func appendTrackDetail(details *[]TrackDiagnostic, track diagnosticTrack, issue string, diagnostics *LibraryDiagnostics) {
	if len(*details) >= diagnosticDetailLimit {
		diagnostics.DetailsTruncated = true
		return
	}
	*details = append(*details, TrackDiagnostic{
		ID:       track.ID,
		Title:    track.Title,
		Artist:   track.Artist,
		Album:    track.Album,
		FilePath: track.FilePath,
		Format:   track.Format,
		Issue:    issue,
	})
}

func missingFileIssue(path string) string {
	if strings.TrimSpace(path) == "" {
		return "No file path is stored"
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "Audio file is missing"
		}
		return "Audio file cannot be accessed"
	}
	if !info.Mode().IsRegular() {
		return "Path is not an audio file"
	}
	return ""
}

func missingFileIssues(tracks []diagnosticTrack) []string {
	issues := make([]string, len(tracks))
	if len(tracks) == 0 {
		return issues
	}

	workerCount := runtime.GOMAXPROCS(0) * 2
	if workerCount < 4 {
		workerCount = 4
	}
	if workerCount > 16 {
		workerCount = 16
	}
	if workerCount > len(tracks) {
		workerCount = len(tracks)
	}

	jobs := make(chan int)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go func() {
			defer workers.Done()
			for index := range jobs {
				issues[index] = missingFileIssue(tracks[index].FilePath)
			}
		}()
	}
	for index := range tracks {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	return issues
}

func metadataIssue(track diagnosticTrack) string {
	var problems []string
	if strings.TrimSpace(track.Title) == "" {
		problems = append(problems, "title")
	}
	if strings.TrimSpace(track.Artist) == "" {
		problems = append(problems, "artist")
	}
	if track.Duration <= 0 {
		problems = append(problems, "duration")
	}
	if len(problems) == 0 && track.HasMetadata {
		return ""
	}
	if len(problems) == 0 {
		return "Metadata was inferred from the filename"
	}
	return "Missing or invalid " + strings.Join(problems, ", ")
}

func unsupportedFormatIssue(track diagnosticTrack) string {
	extension := strings.ToLower(filepath.Ext(track.FilePath))
	if extension == "" && track.Format != "" {
		extension = "." + strings.TrimPrefix(strings.ToLower(strings.TrimSpace(track.Format)), ".")
	}
	if _, supported := supportedAudioExtensions[extension]; supported {
		return ""
	}
	if extension == "" {
		return "Audio format is unknown"
	}
	return fmt.Sprintf("%s is not a supported audio format", extension)
}

func hasArtwork(track diagnosticTrack) bool {
	return strings.TrimSpace(track.ImageURL) != "" ||
		strings.TrimSpace(track.CoverArtURL) != "" ||
		strings.TrimSpace(track.CoverArtSmall) != "" ||
		strings.TrimSpace(track.CoverArtMed) != "" ||
		strings.TrimSpace(track.CoverArtLarge) != ""
}

func inspectSource(path string) SourceDiagnostic {
	item := SourceDiagnostic{Path: path, SpaceStatus: "unavailable"}
	info, err := os.Stat(path)
	if err != nil {
		item.Error = err.Error()
		return item
	}
	if !info.IsDir() {
		item.Error = "path is not a directory"
		return item
	}
	item.Accessible = true

	total, free, err := diskSpace(path)
	if err != nil {
		item.SpaceStatus = "unknown"
		item.Error = err.Error()
		return item
	}
	item.TotalBytes = total
	item.FreeBytes = free
	item.UsedBytes = total - free
	if total > 0 {
		item.UsedPercent = math.Round((float64(item.UsedBytes)/float64(total))*1000) / 10
	}
	freePercent := 100 - item.UsedPercent
	switch {
	case free < 1<<30 || freePercent < 5:
		item.SpaceStatus = "critical"
	case free < 5<<30 || freePercent < 10:
		item.SpaceStatus = "warning"
	default:
		item.SpaceStatus = "healthy"
	}
	return item
}
