package chapel

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
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

func (c *Chapel) Apply(input io.Reader, yes bool) error {
	var newMetadata Metadata
	if err := yaml.NewDecoder(input).Decode(&newMetadata); err != nil {
		return fmt.Errorf("failed to decode YAML: %w", err)
	}

	// Get current metadata from MP3 file
	currentMetadata, err := c.getMetadata()
	if err != nil {
		return fmt.Errorf("failed to read current metadata: %w", err)
	}

	// Normalize both metadata by marshaling them to YAML
	currentYAMLData, err := yaml.Marshal(currentMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal current metadata: %w", err)
	}

	normalizedNewYAMLData, err := yaml.Marshal(&newMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal new metadata: %w", err)
	}

	currentYAML := string(currentYAMLData)
	newYAML := string(normalizedNewYAMLData)

	if currentYAML == newYAML {
		log.Println("No changes to apply.")
		return nil
	}
	if !yes {
		// Compare and show diff if different
		diff := generateDiff(currentYAML, newYAML)
		log.Printf("The following changes will be applied:\n%s\n", diff)
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
			log.Println("Changes not applied.")
			return nil
		}
	}
	// Apply changes to MP3 file
	err = c.writeMetadata(&newMetadata)
	if err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	log.Println("Metadata updated successfully.")
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

	// Set version and encoding
	id3tag.SetVersion(4)
	id3tag.SetDefaultEncoding(id3v2.EncodingUTF8)

	// Set basic metadata with UTF-8 encoding for multibyte support
	// Delete and re-add frames to avoid duplicates
	id3tag.DeleteFrames("TIT2")
	if metadata.Title != "" {
		id3tag.AddTextFrame("TIT2", id3v2.EncodingUTF8, metadata.Title)
	}

	id3tag.DeleteFrames("TIT3")
	if metadata.Subtitle != "" {
		id3tag.AddTextFrame("TIT3", id3v2.EncodingUTF8, metadata.Subtitle)
	}

	id3tag.DeleteFrames("TPE1")
	if metadata.Artist != "" {
		id3tag.AddTextFrame("TPE1", id3v2.EncodingUTF8, metadata.Artist)
	}

	id3tag.DeleteFrames("TALB")
	if metadata.Album != "" {
		id3tag.AddTextFrame("TALB", id3v2.EncodingUTF8, metadata.Album)
	}

	id3tag.DeleteFrames("TIT1")
	if metadata.Grouping != "" {
		id3tag.AddTextFrame("TIT1", id3v2.EncodingUTF8, metadata.Grouping)
	}

	id3tag.DeleteFrames("TCON")
	if metadata.Genre != "" {
		id3tag.AddTextFrame("TCON", id3v2.EncodingUTF8, metadata.Genre)
	}

	// Set date using TDRC tag (ID3v2.4) and Year for compatibility
	id3tag.DeleteFrames("TDRC")
	id3tag.DeleteFrames("TYER") // Also delete legacy year frame
	if metadata.Date != nil && !metadata.Date.Time.IsZero() {
		// Set Year for ID3v2.3 compatibility. It should be performed before add TDRC
		yearStr := metadata.Date.Time.Format("2006")
		id3tag.SetYear(yearStr)

		dateStr := metadata.Date.String()
		id3tag.AddTextFrame("TDRC", id3v2.EncodingUTF8, dateStr)
	}

	// Set additional text frames
	id3tag.DeleteFrames("TPE2")
	if metadata.AlbumArtist != "" {
		id3tag.AddTextFrame("TPE2", id3v2.EncodingUTF8, metadata.AlbumArtist)
	}

	id3tag.DeleteFrames("TCOM")
	if metadata.Composer != "" {
		id3tag.AddTextFrame("TCOM", id3v2.EncodingUTF8, metadata.Composer)
	}

	id3tag.DeleteFrames("TPUB")
	if metadata.Publisher != "" {
		id3tag.AddTextFrame("TPUB", id3v2.EncodingUTF8, metadata.Publisher)
	}

	id3tag.DeleteFrames("TCOP")
	if metadata.Copyright != "" {
		id3tag.AddTextFrame("TCOP", id3v2.EncodingUTF8, metadata.Copyright)
	}

	id3tag.DeleteFrames("TLAN")
	if metadata.Language != "" {
		id3tag.AddTextFrame("TLAN", id3v2.EncodingUTF8, metadata.Language)
	}

	id3tag.DeleteFrames("TBPM")
	if metadata.BPM != 0 {
		id3tag.AddTextFrame("TBPM", id3v2.EncodingUTF8, fmt.Sprintf("%d", metadata.BPM))
	}

	// Set track information
	id3tag.DeleteFrames("TRCK")
	if metadata.Track != nil && metadata.Track.Current > 0 {
		id3tag.AddTextFrame("TRCK", id3v2.EncodingUTF8, metadata.Track.String())
	}

	// Set disc information
	id3tag.DeleteFrames("TPOS")
	if metadata.Disc != nil && metadata.Disc.Current > 0 {
		id3tag.AddTextFrame("TPOS", id3v2.EncodingUTF8, metadata.Disc.String())
	}

	// Set comment
	id3tag.DeleteFrames(id3tag.CommonID("Comments"))
	if metadata.Comment != "" {
		id3tag.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "",
			Text:        metadata.Comment,
		})
	}

	// Set lyrics
	// First, delete existing lyrics frames
	id3tag.DeleteFrames("USLT") // Unsynchronised lyrics/text transcription
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

			// Store artwork source in TXXX frame
			// Skip data URIs as they don't need source tracking
			if !strings.HasPrefix(metadata.Artwork, "data:") {
				txxxFrames := id3tag.GetFrames("TXXX")
				var preservedFrames []id3v2.UserDefinedTextFrame
				// Collect all non-CHAPEL_SOURCE TXXX frames
				for _, frame := range txxxFrames {
					if udtf, ok := frame.(id3v2.UserDefinedTextFrame); ok {
						if udtf.Description != "CHAPEL_SOURCE" {
							preservedFrames = append(preservedFrames, udtf)
						}
					}
				}
				// Clear all TXXX frames and re-add preserved ones
				id3tag.DeleteFrames("TXXX")
				for _, frame := range preservedFrames {
					id3tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
						Encoding:    frame.Encoding,
						Description: frame.Description,
						Value:       frame.Value,
					})
				}
				// Add new CHAPEL_SOURCE frame
				id3tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
					Encoding:    id3v2.EncodingUTF8,
					Description: "CHAPEL_SOURCE",
					Value:       metadata.Artwork,
				})
			}
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

// parseArtwork parses artwork string (data URI, HTTP/HTTPS URL, or file path) and returns picture data and MIME type
func parseArtwork(artwork string) ([]byte, string, error) {
	if strings.HasPrefix(artwork, "data:") {
		// Parse data URI
		return parseDataURI(artwork)
	} else if strings.HasPrefix(artwork, "http://") || strings.HasPrefix(artwork, "https://") {
		// Download from HTTP/HTTPS URL
		return parseHTTPURL(artwork)
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

var userAgent = "chapel/" + Version + " (+https://github.com/Songmu/chapel)"

// parseHTTPURL downloads artwork from HTTP/HTTPS URL and returns picture data and MIME type
func parseHTTPURL(url string) ([]byte, string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with User-Agent header
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)

	// Download the image
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image from %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image from %s: HTTP %d", url, resp.StatusCode)
	}

	// Read the response body
	pictureData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data from %s: %w", url, err)
	}

	// Determine MIME type from Content-Type header
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		// Fallback: try to determine from URL extension
		mimeType = getMimeTypeFromExt(filepath.Ext(url))
		if mimeType == "" {
			return nil, "", fmt.Errorf("unable to determine MIME type for %s", url)
		}
	}

	return pictureData, mimeType, nil
}
