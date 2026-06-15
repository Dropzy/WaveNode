package metadata

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"music-server/utils"

	"github.com/dhowden/tag"
)

// TrackInfo holds extracted metadata information
type TrackInfo struct {
	Title       string   `json:"title"`
	Artist      string   `json:"artist"`
	Album       string   `json:"album"`
	Genre       string   `json:"genre"`
	Year        int      `json:"year"`
	TrackNumber int      `json:"track_number"`
	DiscNumber  int      `json:"disc_number"`
	DiscTotal   int      `json:"disc_total"`
	Duration    int      `json:"duration"` // in seconds
	Featuring   []string `json:"featuring,omitempty"`
	FilePath    string   `json:"file_path"`
	FileName    string   `json:"file_name"`
	FileSize    int64    `json:"file_size"`
	Format      string   `json:"format"`

	// Metadata quality indicators
	HasMetadata        bool   `json:"has_metadata"`
	Confidence         int    `json:"confidence"` // 0-100, higher is better
	Source             string `json:"source"`     // "embedded", "filename", "mixed"
	ParsedFromFilename bool   `json:"parsed_from_filename"`

	// Artwork information
	HasArtwork          bool    `json:"has_artwork"`
	ArtworkData         []byte  `json:"-"`              // Raw image data (not exported to JSON)
	ArtworkFormat       string  `json:"artwork_format"` // "jpg", "png", etc.
	ArtworkSize         int     `json:"artwork_size"`   // Size in bytes
	ReplayGainTrackDB   float64 `json:"replaygain_track_db"`
	ReplayGainAlbumDB   float64 `json:"replaygain_album_db"`
	ReplayGainTrackPeak float64 `json:"replaygain_track_peak"`
	ReplayGainAlbumPeak float64 `json:"replaygain_album_peak"`
}

// MetadataParser handles extraction of metadata from audio files
type MetadataParser struct {
	// Configuration
	PreferEmbeddedMetadata bool
	MinConfidenceThreshold int

	// Filename parsing patterns
	artistTitlePattern    *regexp.Regexp
	featuringPattern      *regexp.Regexp
	bracketRemovalPattern *regexp.Regexp
	trackNumberPattern    *regexp.Regexp
}

// NewMetadataParser creates a new metadata parser instance
func NewMetadataParser() *MetadataParser {
	return &MetadataParser{
		PreferEmbeddedMetadata: true,
		MinConfidenceThreshold: 50,

		// Common patterns for filename parsing (MP3 only)
		artistTitlePattern:    regexp.MustCompile(`^(.+?)\s*-\s*(.+?)(?:\s*\[.*?\])?$`),
		featuringPattern:      regexp.MustCompile(`(?i)\b(ft\.?|feat\.?|featuring)\s+(.+?)(?:\s|\)|$|$)`),
		bracketRemovalPattern: regexp.MustCompile(`\s*\[.*?\]\s*|\s*\(.*?\)\s*`),
		trackNumberPattern:    regexp.MustCompile(`^(\d{1,3})[\.\-\s]+`),
	}
}

// ExtractMetadata extracts metadata from a given file path
func (p *MetadataParser) ExtractMetadata(filePath string) (*TrackInfo, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Initialize track info
	track := &TrackInfo{
		FilePath: filePath,
		FileName: filepath.Base(filePath),
		FileSize: fileInfo.Size(),
		Format:   strings.ToLower(filepath.Ext(filePath)[1:]),
	}

	// Always try filename parsing first
	filenameMetadata := p.parseFilename(track.FileName)
	if filenameMetadata != nil {
		p.mergeFilenameMetadata(track, filenameMetadata)
		track.ParsedFromFilename = true
		track.Source = "filename"
		track.Confidence = 60 // Medium confidence for filename parsing
	}

	// Try to extract embedded metadata to enhance filename data
	embeddedMetadata, err := p.extractEmbeddedMetadata(filePath)
	if err == nil && embeddedMetadata != nil {
		p.mergeEmbeddedArtwork(track, embeddedMetadata)

		// Only use embedded metadata if it's better than filename data
		if p.shouldUseEmbeddedMetadata(track, embeddedMetadata) {
			p.mergeEmbeddedMetadata(track, embeddedMetadata)
			track.HasMetadata = true
			track.Source = "mixed"
			track.Confidence = 95 // Highest confidence for combined data
		}
	}

	// If still no metadata, use basic filename parsing
	if !track.HasMetadata && track.Title == "" && track.Artist == "" {
		basicMetadata := p.basicFilenameParse(track.FileName)
		if basicMetadata != nil {
			p.mergeFilenameMetadata(track, basicMetadata)
			track.Source = "filename"
			track.Confidence = 40 // Low confidence for basic parsing
			track.ParsedFromFilename = true
		}
	}

	// Always extract duration using ffmpeg regardless of metadata source
	if track.Duration == 0 {
		duration := p.extractDuration(filePath)
		if duration > 0 {
			track.Duration = duration
		}
	}

	// Clean up and normalize the metadata
	p.cleanupMetadata(track)

	return track, nil
}

// extractEmbeddedMetadata reads metadata from audio file tags
func (p *MetadataParser) extractEmbeddedMetadata(filePath string) (*TrackInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	track := &TrackInfo{
		Title:  metadata.Title(),
		Artist: metadata.Artist(),
		Album:  metadata.Album(),
		Genre:  metadata.Genre(),
		Year:   int(metadata.Year()),
	}

	// Extract track number if available
	trackNum, _ := metadata.Track()
	if trackNum > 0 {
		track.TrackNumber = trackNum
	}
	track.DiscNumber, track.DiscTotal = metadata.Disc()
	if track.DiscNumber < 1 {
		track.DiscNumber = 1
	}
	if track.DiscTotal < track.DiscNumber {
		track.DiscTotal = track.DiscNumber
	}
	applyReplayGainTags(track, metadata.Raw())

	// Try to extract duration using ffmpeg
	duration := p.extractDuration(filePath)
	if duration > 0 {
		track.Duration = duration
	}

	// Extract featuring artists from title if present
	if track.Title != "" {
		track.Featuring = p.extractFeaturingArtists(track.Title)
		// Remove featuring info from title for cleaner display
		track.Title = p.cleanFeaturingFromTitle(track.Title)
	}

	// Extract artwork if available
	if picture := metadata.Picture(); picture != nil {
		track.HasArtwork = true
		track.ArtworkData = picture.Data
		track.ArtworkSize = len(picture.Data)
		track.ArtworkFormat = p.getArtworkFormat(picture.MIMEType)
	}

	return track, nil
}

func applyReplayGainTags(track *TrackInfo, raw map[string]interface{}) {
	for key, rawValue := range raw {
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, " ", "_"), "-", "_"))
		value := parseReplayGainValue(fmt.Sprint(rawValue))
		switch {
		case strings.Contains(normalized, "replaygain_track_gain"):
			track.ReplayGainTrackDB = value
		case strings.Contains(normalized, "replaygain_album_gain"):
			track.ReplayGainAlbumDB = value
		case strings.Contains(normalized, "replaygain_track_peak"):
			track.ReplayGainTrackPeak = value
		case strings.Contains(normalized, "replaygain_album_peak"):
			track.ReplayGainAlbumPeak = value
		}
	}
}

func parseReplayGainValue(value string) float64 {
	match := regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`).FindString(value)
	parsed, _ := strconv.ParseFloat(match, 64)
	return parsed
}

// ExtractArtwork reads only the embedded picture from an audio file. Unlike
// ExtractMetadata, it avoids duration probing and filename parsing.
func (p *MetadataParser) ExtractArtwork(filePath string) ([]byte, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	fileMetadata, err := tag.ReadFrom(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read audio metadata: %w", err)
	}

	picture := fileMetadata.Picture()
	if picture == nil || len(picture.Data) == 0 {
		return nil, "", nil
	}

	return picture.Data, p.getArtworkFormat(picture.MIMEType), nil
}

// parseFilename attempts to extract metadata from filename patterns
func (p *MetadataParser) parseFilename(filename string) *TrackInfo {
	// Remove file extension
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Try artist - title pattern
	matches := p.artistTitlePattern.FindStringSubmatch(nameWithoutExt)
	if len(matches) >= 3 {
		artist := strings.TrimSpace(matches[1])
		title := strings.TrimSpace(matches[2])

		// Extract featuring artists
		featuring := p.extractFeaturingArtists(title)
		title = p.cleanFeaturingFromTitle(title)

		// Remove bracketed content
		title = p.bracketRemovalPattern.ReplaceAllString(title, "")
		title = strings.TrimSpace(title)

		return &TrackInfo{
			Title:     title,
			Artist:    artist,
			Featuring: featuring,
		}
	}

	return nil
}

// basicFilenameParse is a fallback for simple filename parsing
func (p *MetadataParser) basicFilenameParse(filename string) *TrackInfo {
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Try to split by common separators
	separators := []string{" - ", " – ", " — ", "-", "|"}

	for _, sep := range separators {
		if strings.Contains(nameWithoutExt, sep) {
			parts := strings.SplitN(nameWithoutExt, sep, 2)
			if len(parts) == 2 {
				artist := strings.TrimSpace(parts[0])
				title := strings.TrimSpace(parts[1])

				// Clean up title
				title = p.bracketRemovalPattern.ReplaceAllString(title, "")
				title = strings.TrimSpace(title)

				if artist != "" && title != "" {
					featuring := p.extractFeaturingArtists(title)
					title = p.cleanFeaturingFromTitle(title)

					return &TrackInfo{
						Title:     title,
						Artist:    artist,
						Featuring: featuring,
					}
				}
			}
		}
	}

	return nil
}

// extractFeaturingArtists extracts featuring artists from a string
func (p *MetadataParser) extractFeaturingArtists(text string) []string {
	matches := p.featuringPattern.FindStringSubmatch(text)
	if len(matches) >= 3 {
		featuringText := strings.TrimSpace(matches[2])
		// Split by common separators
		separators := []string{"&", "and", ",", "+"}
		artists := []string{featuringText}

		for _, sep := range separators {
			for i, artist := range artists {
				if strings.Contains(artist, sep) {
					parts := strings.Split(artist, sep)
					artists = append(artists[:i], append(parts, artists[i+1:]...)...)
					break
				}
			}
		}

		// Clean up and filter empty strings
		var cleanArtists []string
		for _, artist := range artists {
			artist = strings.TrimSpace(artist)
			if artist != "" {
				cleanArtists = append(cleanArtists, artist)
			}
		}

		return cleanArtists
	}
	return nil
}

// cleanFeaturingFromTitle removes featuring information from title
func (p *MetadataParser) cleanFeaturingFromTitle(title string) string {
	return p.featuringPattern.ReplaceAllString(title, "")
}

// mergeEmbeddedMetadata merges embedded metadata into track info
func (p *MetadataParser) mergeEmbeddedMetadata(track *TrackInfo, embedded *TrackInfo) {
	if embedded.Title != "" {
		track.Title = embedded.Title
	}
	if embedded.Artist != "" {
		track.Artist = embedded.Artist
	}
	if embedded.Album != "" {
		track.Album = embedded.Album
	}
	if embedded.Genre != "" {
		track.Genre = embedded.Genre
	}
	if embedded.Year > 0 {
		track.Year = embedded.Year
	}
	if embedded.TrackNumber > 0 {
		track.TrackNumber = embedded.TrackNumber
	}
	if embedded.DiscNumber > 0 {
		track.DiscNumber = embedded.DiscNumber
		track.DiscTotal = embedded.DiscTotal
	}
	track.ReplayGainTrackDB = embedded.ReplayGainTrackDB
	track.ReplayGainAlbumDB = embedded.ReplayGainAlbumDB
	track.ReplayGainTrackPeak = embedded.ReplayGainTrackPeak
	track.ReplayGainAlbumPeak = embedded.ReplayGainAlbumPeak
	if embedded.Duration > 0 {
		track.Duration = embedded.Duration
	}
	if len(embedded.Featuring) > 0 {
		track.Featuring = embedded.Featuring
	}
	p.mergeEmbeddedArtwork(track, embedded)
}

func (p *MetadataParser) mergeEmbeddedArtwork(track *TrackInfo, embedded *TrackInfo) {
	if embedded.HasArtwork && len(embedded.ArtworkData) > 0 {
		track.HasArtwork = true
		track.ArtworkData = embedded.ArtworkData
		track.ArtworkFormat = embedded.ArtworkFormat
		track.ArtworkSize = embedded.ArtworkSize
	}
}

// mergeFilenameMetadata merges filename-parsed metadata into track info
func (p *MetadataParser) mergeFilenameMetadata(track *TrackInfo, filename *TrackInfo) {
	if track.Title == "" && filename.Title != "" {
		track.Title = filename.Title
	}
	if track.Artist == "" && filename.Artist != "" {
		track.Artist = filename.Artist
	}
	if len(track.Featuring) == 0 && len(filename.Featuring) > 0 {
		track.Featuring = filename.Featuring
	}
}

// shouldUseEmbeddedMetadata determines if embedded metadata should override filename data
func (p *MetadataParser) shouldUseEmbeddedMetadata(track *TrackInfo, embedded *TrackInfo) bool {
	// Don't use embedded metadata if both title and artist are empty
	if embedded.Title == "" && embedded.Artist == "" {
		return false
	}

	// If filename parsing gave us good data, be more selective about overriding it
	if track.Title != "" && track.Artist != "" {
		// Only use embedded metadata if it has both artist and title AND they're substantial
		if embedded.Title != "" && embedded.Artist != "" &&
			len(embedded.Title) > 2 && len(embedded.Artist) > 2 {
			return true
		}
		// Otherwise, stick with filename data
		return false
	}

	// If we have partial filename data, only use embedded to fill missing pieces
	if track.Title != "" && embedded.Title == "" {
		return false // Keep filename title if embedded has no title
	}
	if track.Artist != "" && embedded.Artist == "" {
		return false // Keep filename artist if embedded has no artist
	}

	// Use embedded metadata if it can fill missing critical fields
	if (track.Title == "" && embedded.Title != "") || (track.Artist == "" && embedded.Artist != "") {
		return true
	}

	// Prefer embedded metadata for album, genre, year if filename data is missing
	if (track.Album == "" && embedded.Album != "") ||
		(track.Genre == "" && embedded.Genre != "") ||
		(track.Year == 0 && embedded.Year > 0) {
		return true
	}

	return false
}

// enhanceWithFilenameData enhances existing metadata with filename data
func (p *MetadataParser) enhanceWithFilenameData(track *TrackInfo, filename *TrackInfo) {
	// Only use filename data if embedded data is missing or low quality
	if track.Title == "" || len(track.Title) < 3 {
		if filename.Title != "" && len(filename.Title) > 2 {
			track.Title = filename.Title
		}
	}

	if track.Artist == "" || len(track.Artist) < 3 {
		if filename.Artist != "" && len(filename.Artist) > 2 {
			track.Artist = filename.Artist
		}
	}

	if len(track.Featuring) == 0 && len(filename.Featuring) > 0 {
		track.Featuring = filename.Featuring
	}
}

// cleanupMetadata normalizes and cleans up the extracted metadata
func (p *MetadataParser) cleanupMetadata(track *TrackInfo) {
	// Trim whitespace
	track.Title = strings.TrimSpace(track.Title)
	track.Artist = strings.TrimSpace(track.Artist)
	track.Album = strings.TrimSpace(track.Album)
	track.Genre = strings.TrimSpace(track.Genre)

	// Remove extra whitespace
	track.Title = regexp.MustCompile(`\s+`).ReplaceAllString(track.Title, " ")
	track.Artist = regexp.MustCompile(`\s+`).ReplaceAllString(track.Artist, " ")
	track.Album = regexp.MustCompile(`\s+`).ReplaceAllString(track.Album, " ")
	track.Genre = regexp.MustCompile(`\s+`).ReplaceAllString(track.Genre, " ")

	// Capitalize properly (simple title case)
	track.Title = p.titleCase(track.Title)
	track.Artist = p.titleCase(track.Artist)
	track.Album = p.titleCase(track.Album)

	// Clean up featuring artists
	for i, artist := range track.Featuring {
		track.Featuring[i] = strings.TrimSpace(p.titleCase(artist))
	}

	// Set has_metadata flag
	track.HasMetadata = track.Title != "" || track.Artist != ""
}

// titleCase converts string to title case with proper DJ normalization
func (p *MetadataParser) titleCase(s string) string {
	if s == "" {
		return s
	}

	// Handle common exceptions that shouldn't be capitalized or need special handling
	exceptions := map[string]bool{
		"ft": true, "feat": true, "featuring": true,
		"vs": true, "remix": true, "edit": true,
		"mix": true, "version": true,
	}

	words := strings.Fields(s)
	for i, word := range words {
		lower := strings.ToLower(word)

		// Special handling for DJ and FX - always capitalize as "DJ" and "FX"
		if lower == "dj" {
			words[i] = "DJ"
		} else if lower == "fx" {
			words[i] = "FX"
		} else if exceptions[lower] {
			words[i] = lower
		} else {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			}
		}
	}

	return strings.Join(words, " ")
}

// IsAudioFile checks if the file is a supported audio format (MP3 only)
func (p *MetadataParser) IsAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".mp3"
}

// GetSupportedFormats returns a list of supported audio formats (MP3 only)
func (p *MetadataParser) GetSupportedFormats() []string {
	return []string{".mp3"}
}

// extractDuration reads the container duration without decoding the audio.
func (p *MetadataParser) extractDuration(filePath string) int {
	return p.extractDurationWithFFProbe(filePath)
}

// extractDurationWithFFProbe reads duration metadata without decoding the file.
func (p *MetadataParser) extractDurationWithFFProbe(filePath string) int {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return 0
	}

	durationStr := strings.TrimSpace(out.String())
	if durationStr == "" {
		return 0
	}

	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0
	}

	return int(duration)
}

// getArtworkFormat converts MIME type to file extension
func (p *MetadataParser) getArtworkFormat(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "jpg" // Default to jpg for unknown formats
	}
}

// SaveArtwork saves extracted artwork to disk and returns the file path
func (p *MetadataParser) SaveArtwork(artworkData []byte, artist, title, format string) (string, error) {
	if len(artworkData) == 0 {
		return "", fmt.Errorf("no artwork data to save")
	}

	// Create artwork directory if it doesn't exist
	artworkDir := utils.ArtworkDirectory()
	if err := os.MkdirAll(artworkDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create artwork directory: %w", err)
	}

	// Generate unique filename using hash
	hash := sha256.Sum256(artworkData)
	hashStr := fmt.Sprintf("%x", hash)
	filename := fmt.Sprintf("%s_%s_%s", artist, title, hashStr[:8])

	// Clean filename to remove invalid characters (more comprehensive for cross-platform)
	filename = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`).ReplaceAllString(filename, "_")

	// Ensure format is clean (remove any "image/" prefix if present)
	if strings.Contains(format, "/") {
		parts := strings.Split(format, "/")
		if len(parts) > 1 {
			format = parts[1]
		}
	}

	fullPath := filepath.Join(artworkDir, filename+"."+format)

	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() && info.Size() == int64(len(artworkData)) {
		return fullPath, nil
	}

	// Save artwork to file
	if err := os.WriteFile(fullPath, artworkData, 0644); err != nil {
		return "", fmt.Errorf("failed to save artwork: %w", err)
	}

	return fullPath, nil
}

// parseDurationFromFFmpegOutput parses duration from ffmpeg error output
func (p *MetadataParser) parseDurationFromFFmpegOutput(output string) int {
	// Look for duration pattern like "Duration: 00:03:45.67"
	durationPattern := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2}\.\d{2})`)
	matches := durationPattern.FindStringSubmatch(output)

	if len(matches) >= 4 {
		hours, _ := strconv.Atoi(matches[1])
		minutes, _ := strconv.Atoi(matches[2])
		seconds, _ := strconv.ParseFloat(matches[3], 64)

		totalSeconds := hours*3600 + minutes*60 + int(seconds)
		return totalSeconds
	}

	return 0
}
