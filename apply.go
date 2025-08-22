package chapel

import (
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Songmu/prompter"
	"github.com/bogem/id3v2/v2"
	"github.com/goccy/go-yaml"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/tcolgate/mp3"
)

func (c *Chapel) Apply(input io.Reader) error {
	// Read YAML from input
	yamlData, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Parse YAML to metadata
	var newMetadata Metadata
	err = yaml.Unmarshal(yamlData, &newMetadata)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Get current metadata from MP3 file
	var currentOutput strings.Builder
	err = c.Dump(&currentOutput)
	if err != nil {
		return fmt.Errorf("failed to read current metadata: %w", err)
	}

	currentYAML := currentOutput.String()
	newYAML := string(yamlData)

	if currentYAML == newYAML {
		fmt.Println("No changes to apply.")
		return nil
	}
	// Compare and show diff if different
	diff := generateDiff(currentYAML, newYAML)
	fmt.Printf("The following changes will be applied:\n%s\n", diff)
	// Check if input is os.Stdin (when called from pipe/redirect)
	// Type assertion to check if input is *os.File and if it's stdin
	if file, ok := input.(*os.File); ok && file == os.Stdin {
		// Input is from stdin (e.g., chapel apply < file.yaml)
		// Need to reopen terminal for user interaction

		// Use /dev/tty on Unix-like systems, CON on Windows
		consoleDevice := "/dev/tty"
		if runtime.GOOS == "windows" {
			consoleDevice = "CON"
		}

		tty, err := os.OpenFile(consoleDevice, os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", consoleDevice, err)
		}
		defer tty.Close()

		// Temporarily replace stdin with tty
		oldStdin := os.Stdin
		os.Stdin = tty
		defer func() { os.Stdin = oldStdin }()
	}
	if !prompter.YN("Apply these changes?", true) {
		fmt.Println("Changes not applied.")
		return nil
	}
	// Apply changes to MP3 file
	err = c.writeMetadata(&newMetadata)
	if err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	fmt.Println("Metadata updated successfully.")
	return nil
}

// generateDiff creates a human-readable diff between old and new YAML
func generateDiff(oldYAML, newYAML string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldYAML, newYAML, false)
	return dmp.DiffPrettyText(diffs)
}

// writeMetadata writes metadata to the MP3 file
func (c *Chapel) writeMetadata(metadata *Metadata) error {
	// Get audio duration for chapter end times
	audioDuration, err := c.getAudioDuration()
	if err != nil {
		return fmt.Errorf("failed to get audio duration: %w", err)
	}

	// Open the MP3 file for writing
	id3tag, err := id3v2.Open(c.audio, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer id3tag.Close()

	// Clear existing frames to avoid duplicates
	id3tag.DeleteAllFrames()

	// Set basic metadata with UTF-8 encoding for multibyte support
	if metadata.Title != "" {
		id3tag.AddTextFrame("TIT2", id3v2.EncodingUTF8, metadata.Title)
	}
	if metadata.Artist != "" {
		id3tag.AddTextFrame("TPE1", id3v2.EncodingUTF8, metadata.Artist)
	}
	if metadata.Album != "" {
		id3tag.AddTextFrame("TALB", id3v2.EncodingUTF8, metadata.Album)
	}
	if metadata.Genre != "" {
		id3tag.AddTextFrame("TCON", id3v2.EncodingUTF8, metadata.Genre)
	}

	// Set date using TDRC tag (ID3v2.4) and Year for compatibility
	if metadata.Date != nil && !metadata.Date.Time.IsZero() {
		dateStr := metadata.Date.String()
		id3tag.AddTextFrame("TDRC", id3v2.EncodingUTF8, dateStr)

		// Also set Year for ID3v2.3 compatibility
		yearStr := metadata.Date.Time.Format("2006")
		id3tag.SetYear(yearStr)
	}

	// Set additional text frames
	if metadata.AlbumArtist != "" {
		id3tag.AddTextFrame("TPE2", id3v2.EncodingUTF8, metadata.AlbumArtist)
	}
	if metadata.Composer != "" {
		id3tag.AddTextFrame("TCOM", id3v2.EncodingUTF8, metadata.Composer)
	}
	if metadata.Publisher != "" {
		id3tag.AddTextFrame("TPUB", id3v2.EncodingUTF8, metadata.Publisher)
	}
	if metadata.BPM != 0 {
		id3tag.AddTextFrame("TBPM", id3v2.EncodingUTF8, fmt.Sprintf("%d", metadata.BPM))
	}

	// Set track information
	if metadata.Track != nil && metadata.Track.Current > 0 {
		id3tag.AddTextFrame("TRCK", id3v2.EncodingUTF8, metadata.Track.String())
	}

	// Set disc information
	if metadata.Disc != nil && metadata.Disc.Current > 0 {
		id3tag.AddTextFrame("TPOS", id3v2.EncodingUTF8, metadata.Disc.String())
	}

	// Set comment
	if metadata.Comment != "" {
		id3tag.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "",
			Text:        metadata.Comment,
		})
	}

	// Set lyrics
	if metadata.Lyrics != "" {
		id3tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
			Encoding: id3v2.EncodingUTF8,
			Language: "eng",
			Lyrics:   metadata.Lyrics,
		})
	}

	// Set artwork
	if metadata.Artwork != "" {
		pictureData, mimeType, err := parseArtwork(metadata.Artwork)
		if err != nil {
			return fmt.Errorf("failed to parse artwork: %w", err)
		}

		if len(pictureData) > 0 {
			// Delete existing picture frames
			id3tag.DeleteFrames("APIC")

			pictureFrame := id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    mimeType,
				PictureType: id3v2.PTFrontCover,
				Description: "",
				Picture:     pictureData,
			}
			id3tag.AddAttachedPicture(pictureFrame)
		}
	}

	// Set chapters
	// First, delete existing chapter frames
	id3tag.DeleteFrames("CHAP")

	for i, chapter := range metadata.Chapters {
		// Create proper chapter frame
		startTime := chapter.Start
		var endTime time.Duration

		// Set end time to next chapter's start time or audio duration for last chapter
		if i+1 < len(metadata.Chapters) {
			endTime = metadata.Chapters[i+1].Start
		} else {
			endTime = audioDuration // Use actual audio duration for last chapter
		}

		chapterFrame := id3v2.ChapterFrame{
			ElementID: fmt.Sprintf("chp%d", i),
			StartTime: startTime,
			EndTime:   endTime,
			// If these bytes are all set to 0xFF then the value should be ignored and
			// the start/end time value should be utilized.
			// cf. https://id3.org/id3v2-chapters-1.0
			StartOffset: math.MaxUint32,
			EndOffset:   math.MaxUint32,
			Title: &id3v2.TextFrame{
				Encoding: id3v2.EncodingUTF8,
				Text:     chapter.Title,
			},
			Description: &id3v2.TextFrame{
				Encoding: id3v2.EncodingUTF8,
				Text:     "",
			},
		}

		id3tag.AddChapterFrame(chapterFrame)
	}

	// Save changes
	err = id3tag.Save()
	if err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// getAudioDuration calculates the actual duration of the MP3 file
func (c *Chapel) getAudioDuration() (time.Duration, error) {
	file, err := os.Open(c.audio)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return readMP3Duration(file)
}

// readMP3Duration calculates the duration of MP3 file by decoding frames
func readMP3Duration(r io.ReadSeeker) (time.Duration, error) {
	var (
		t       time.Duration
		f       mp3.Frame
		skipped int
		d       = mp3.NewDecoder(r)
	)

	for {
		if err := d.Decode(&f, &skipped); err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		t = t + f.Duration()
	}

	return t, nil
}

// parseArtwork parses artwork string (data URI or file path) and returns picture data and MIME type
func parseArtwork(artwork string) ([]byte, string, error) {
	if strings.HasPrefix(artwork, "data:") {
		// Parse data URI
		return parseDataURI(artwork)
	} else {
		// Treat as file path
		return parseFilePath(artwork)
	}
}

// parseDataURI parses data URI and returns picture data and MIME type
func parseDataURI(dataURI string) ([]byte, string, error) {
	// Format: data:image/jpeg;base64,<base64data>
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid data URI format")
	}

	header := parts[0]
	data := parts[1]

	// Extract MIME type from header
	if !strings.HasPrefix(header, "data:") {
		return nil, "", fmt.Errorf("invalid data URI header")
	}

	headerParts := strings.Split(header[5:], ";") // Remove "data:" prefix
	if len(headerParts) < 2 || headerParts[1] != "base64" {
		return nil, "", fmt.Errorf("only base64 data URIs are supported")
	}

	mimeType := headerParts[0]

	// Decode base64 data
	pictureData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 data: %w", err)
	}

	return pictureData, mimeType, nil
}

// parseFilePath parses file path and returns picture data and MIME type
func parseFilePath(filePath string) ([]byte, string, error) {
	// Read file
	pictureData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Determine MIME type from file extension
	mimeType := getMimeTypeFromExt(filepath.Ext(filePath))
	if mimeType == "" {
		return nil, "", fmt.Errorf("unsupported image format: %s", filepath.Ext(filePath))
	}

	return pictureData, mimeType, nil
}

// getMimeTypeFromExt returns MIME type based on file extension
func getMimeTypeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}
