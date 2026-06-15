# Music Metadata Parser

A comprehensive metadata extraction system for the Music Server V2 that handles both embedded metadata and intelligent filename parsing.

## Features

### 🎵 Smart Filename Parsing
- **Standard Format**: `Artist - Title.mp3`
- **Featuring Artists**: `Golden - Huntrix ft saja boys.mp3`
- **Bracket Removal**: Automatically removes `[Official Video]`, `(Remix)`, etc.
- **Title Case Conversion**: Proper capitalization of artist and title names

### 🏷️ Embedded Metadata Support
- **MP3**: ID3v1, ID3v2.2, ID3v2.3, ID3v2.4 tags
- **FLAC**: Vorbis comments
- **OGG**: Vorbis comments
- **M4A/AAC**: iTunes-style metadata

### 🔍 Intelligent Fallback
1. **Primary**: Extract embedded metadata from audio files
2. **Secondary**: Parse filename if embedded metadata is incomplete
3. **Confidence Scoring**: Rate metadata quality (0-100%)
4. **Source Tracking**: Know where data came from

## Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "music-server/metadata"
)

func main() {
    parser := metadata.NewMetadataParser()
    
    // Extract metadata from a file
    trackInfo, err := parser.ExtractMetadata("/path/to/song.mp3")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Artist: %s\n", trackInfo.Artist)
    fmt.Printf("Title: %s\n", trackInfo.Title)
    fmt.Printf("Album: %s\n", trackInfo.Album)
    fmt.Printf("Genre: %s\n", trackInfo.Genre)
    fmt.Printf("Year: %d\n", trackInfo.Year)
    fmt.Printf("Duration: %d seconds\n", trackInfo.Duration)
    fmt.Printf("Confidence: %d%%\n", trackInfo.Confidence)
    fmt.Printf("Source: %s\n", trackInfo.Source)
}
```

### Check Audio Files

```go
parser := metadata.NewMetadataParser()

if parser.IsAudioFile("song.mp3") {
    fmt.Println("This is an audio file")
}
```

## Filename Patterns Supported

### Standard Format
```
Artist - Title.mp3
Queen - Bohemian Rhapsody.mp3
```

### With Featuring Artists
```
Artist - Title ft Featuring Artist.mp3
Artist - Title feat. Featuring Artist.mp3
Artist - Title featuring Featuring Artist.mp3

Examples:
Golden - Huntrix ft saja boys.mp3
Song - Artist feat. Guest & Guest2.mp3
Track - Main Artist featuring Guest One, Guest Two and Guest Three.mp3
```

### With Brackets (Automatically Removed)
```
Artist - Title [Official Video].mp3
Artist - Title (Remix Version).mp3
Artist - Title [Live] (Studio Version).mp3
```

## Metadata Structure

```go
type TrackInfo struct {
    // Core metadata
    Title      string `json:"title"`
    Artist     string `json:"artist"`
    Album      string `json:"album"`
    Genre      string `json:"genre"`
    Year       int    `json:"year"`
    Duration   int    `json:"duration"`  // in seconds
    
    // Additional info
    Featuring  []string `json:"featuring,omitempty"`
    FilePath   string    `json:"file_path"`
    Confidence int       `json:"confidence"` // 0-100
    Source     string    `json:"source"`     // "embedded", "filename", "fallback"
}
```

## Confidence Scoring

### Embedded Metadata
- **90-100%**: Complete metadata with artist, title, album
- **70-89%**: Partial metadata (artist + title)
- **50-69%**: Basic metadata (title only)

### Filename Parsing
- **60-80%**: Well-formatted filename with clear artist/title separation
- **40-59%**: Basic filename parsing
- **20-39%**: Poorly formatted filename

## Integration with Scanner

The metadata parser is integrated into the music library scanner:

```go
// In scanner.go
scanner := NewScanner(db, scanStore, mediaDir)
err := scanner.ScanLibrary(scanID)
```

### Library Scan Process
1. **Discover Files**: Recursively find all audio files
2. **Extract Metadata**: Use parser for each file
3. **Database Integration**: Store with confidence scoring
4. **Progress Updates**: Real-time WebSocket updates
5. **Error Handling**: Continue processing individual file failures

### Enrichment Scan
1. **Re-process Existing**: Re-extract metadata with improved parsing
2. **Upgrade Quality**: Replace lower confidence with higher confidence data
3. **Fill Gaps**: Add missing album, genre, year information
4. **Preserve Manual**: Don't overwrite manually curated metadata

## Configuration

### Environment Variables
```bash
# Directory to scan for music files
MEDIA_SCAN_DIR="/path/to/your/music/library"

# Server configuration
SERVER_PORT="8080"
SERVER_HOST="localhost"
```

### Supported File Extensions
- `.mp3` - MPEG Audio Layer 3
- `.flac` - Free Lossless Audio Codec
- `.wav` - Waveform Audio File Format
- `.ogg` - Ogg Vorbis
- `.m4a` - MPEG-4 Audio
- `.aac` - Advanced Audio Coding

## Error Handling

The parser gracefully handles various error conditions:

### File Access Errors
- Missing files are logged and skipped
- Permission errors are reported but don't stop scanning
- Corrupted files are identified and bypassed

### Metadata Errors
- Invalid ID3 tags are ignored
- Encoding issues are handled gracefully
- Malformed filenames are parsed as much as possible

### Fallback Behavior
```go
trackInfo, err := parser.ExtractMetadata("file.mp3")
if err != nil {
    // Check if we got partial data from filename
    if trackInfo != nil && trackInfo.Confidence > 0 {
        // Use filename-parsed data
        log.Printf("Using filename parsing: %s - %s", trackInfo.Artist, trackInfo.Title)
    } else {
        // Complete failure
        log.Printf("Failed to extract metadata: %v", err)
    }
}
```

## Performance Considerations

### Large Libraries
- **Concurrent Processing**: Files are processed sequentially to avoid I/O overload
- **Memory Efficient**: Metadata is processed file-by-file
- **Progress Tracking**: Real-time updates for large scans

### Optimization Tips
1. **SSD Storage**: Faster file access for metadata extraction
2. **Organized Files**: Well-named files improve parsing accuracy
3. **Regular Scans**: Keep metadata up-to-date

## Testing

Run the test suite:

```bash
cd Backend
go test ./metadata -v
```

### Test Coverage
- Audio file detection
- Filename parsing patterns
- Featuring artist extraction
- Bracket removal
- Title case conversion
- Integration tests

## Examples

### Real-world Filename Parsing

```go
testCases := []string{
    "Diamonds - Vibe Chemistry.mp3",
    // Result: Artist="Vibe Chemistry", Title="Diamonds"
    
    "Golden - Huntrix ft saja boys.mp3", 
    // Result: Artist="Huntrix", Title="Golden", Featuring=["saja boys"]
    
    "Queen - Bohemian Rhapsody [Official Video].mp3",
    // Result: Artist="Queen", Title="Bohemian Rhapsody"
    
    "Artist - Track (Remix) [Live].mp3",
    // Result: Artist="Artist", Title="Track"
}
```

### Metadata Quality Improvement

```go
// Initial scan - filename only
trackInfo := &TrackInfo{
    Artist:     "artist",
    Title:      "title", 
    Confidence: 60,
    Source:     "filename",
}

// After enrichment - embedded metadata found
enrichedInfo := &TrackInfo{
    Artist:     "Artist Name",
    Title:      "Track Title",
    Album:      "Album Name",
    Genre:      "Rock",
    Year:       2023,
    Duration:   240,
    Confidence: 95,
    Source:     "embedded",
}
```

## Troubleshooting

### Common Issues

1. **Low Confidence Scores**
   - Check filename format: `Artist - Title.mp3`
   - Ensure files have embedded metadata
   - Verify file extensions are supported

2. **Missing Artists/Titles**
   - Files may not have embedded metadata
   - Filename format might be non-standard
   - Check for encoding issues in file names

3. **Scanning Performance**
   - Large libraries take time to process
   - Network storage can be slower
   - Consider incremental scanning

### Debug Logging

Enable verbose logging:

```go
parser := metadata.NewMetadataParser()
trackInfo, err := parser.ExtractMetadata(file)

if err != nil {
    log.Printf("Metadata extraction failed for %s: %v", file, err)
} else {
    log.Printf("Extracted: %s - %s (%d%% confidence from %s)", 
        trackInfo.Artist, trackInfo.Title, trackInfo.Confidence, trackInfo.Source)
}
```

## Future Enhancements

### Planned Features
- **Acoustic Fingerprinting**: Identify tracks by audio signature
- **Online Database Lookup**: Fetch metadata from MusicBrainz, Discogs
- **Album Artwork**: Extract and store cover images
- **Lyrics Integration**: Extract embedded lyrics
- **Batch Processing**: Parallel processing for large libraries

### Extensibility
The parser is designed to be extensible:

```go
// Add custom file format support
func (p *MetadataParser) isCustomFormat(filename string) bool {
    return strings.HasSuffix(strings.ToLower(filename), ".custom")
}

// Add custom parsing logic
func (p *MetadataParser) parseCustomFormat(filename string) *TrackInfo {
    // Custom parsing implementation
}
```

## Contributing

When adding new features:

1. **Add Tests**: Cover new functionality with unit tests
2. **Update Documentation**: Keep this README current
3. **Error Handling**: Gracefully handle edge cases
4. **Performance**: Consider impact on large libraries
5. **Backward Compatibility**: Don't break existing functionality

## License

This metadata parser is part of the Music Server V2 project. See the main project license for details.
