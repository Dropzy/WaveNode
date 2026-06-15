package streaming

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"music-server/database"
)

type Options struct {
	Format   string
	Bitrate  int
	Offset   float64
	GainDB   float64
	Download bool
}

func ContentType(format string) string {
	switch format {
	case "opus":
		return "audio/ogg"
	case "aac":
		return "audio/aac"
	default:
		return "audio/mpeg"
	}
}

func SourceContentType(format, path string) string {
	suffix := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(format)), ".")
	if suffix == "" {
		suffix = strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	}
	switch suffix {
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "m4a", "mp4":
		return "audio/mp4"
	case "aac":
		return "audio/aac"
	case "ogg", "oga":
		return "audio/ogg"
	case "opus":
		return "audio/opus"
	case "wav", "wave":
		return "audio/wav"
	case "aif", "aiff":
		return "audio/aiff"
	}
	if contentType := mime.TypeByExtension("." + suffix); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

func ReplayGainDB(profile database.PlaybackProfile, properties database.TrackAudioProperties) float64 {
	gain := 0.0
	switch profile.ReplayGainMode {
	case "track":
		gain = properties.ReplayGainTrackDB
	case "album":
		gain = properties.ReplayGainAlbumDB
		if gain == 0 {
			gain = properties.ReplayGainTrackDB
		}
	}
	if profile.ReplayGainMode != "off" {
		gain += profile.ReplayGainPreampDB
	}
	return gain
}

func Serve(w http.ResponseWriter, req *http.Request, track database.Music, options Options) error {
	if options.Format == "" {
		options.Format = "mp3"
	}
	if options.Bitrate < 48 || options.Bitrate > 320 {
		options.Bitrate = 192
	}
	args := []string{"-hide_banner", "-loglevel", "error"}
	if options.Offset > 0 {
		args = append(args, "-ss", strconv.FormatFloat(options.Offset, 'f', 3, 64))
	}
	args = append(args, "-i", track.FilePath, "-vn")
	if options.GainDB != 0 {
		args = append(args, "-af", fmt.Sprintf("volume=%gdB", options.GainDB))
	}
	switch options.Format {
	case "opus":
		args = append(args, "-c:a", "libopus", "-b:a", fmt.Sprintf("%dk", options.Bitrate), "-f", "ogg", "pipe:1")
	case "aac":
		args = append(args, "-c:a", "aac", "-b:a", fmt.Sprintf("%dk", options.Bitrate), "-f", "adts", "pipe:1")
	default:
		options.Format = "mp3"
		args = append(args, "-c:a", "libmp3lame", "-b:a", fmt.Sprintf("%dk", options.Bitrate), "-f", "mp3", "pipe:1")
	}
	command := exec.CommandContext(req.Context(), "ffmpeg", args...)
	command.Stderr = nil
	output, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}
	w.Header().Set("Content-Type", ContentType(options.Format))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-WaveNode-Transcoded", options.Format)
	if options.Download {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, track.Title, options.Format))
	}
	if _, err := io.Copy(w, output); err != nil {
		_ = command.Process.Kill()
		return err
	}
	return command.Wait()
}
