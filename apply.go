package chapel

import (
	"fmt"
	"io"
	"math"
	"os"
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

	// Set basic metadata
	id3tag.SetTitle(metadata.Title)
	id3tag.SetArtist(metadata.Artist)
	id3tag.SetAlbum(metadata.Album)
	id3tag.SetGenre(metadata.Genre)

	// Set date using TDRC tag (ID3v2.4) and Year for compatibility
	if metadata.Date != "" {
		id3tag.AddTextFrame("TDRC", id3v2.EncodingUTF8, metadata.Date)

		// Also set Year for ID3v2.3 compatibility
		// Extract year part from date (supports YYYY, YYYY-MM, YYYY-MM-DD formats)
		yearStr := metadata.Date
		if len(yearStr) >= 4 {
			yearStr = yearStr[:4]
			id3tag.SetYear(yearStr)
		}
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
	if metadata.Track != 0 || metadata.TotalTracks != 0 {
		trackStr := fmt.Sprintf("%d", metadata.Track)
		if metadata.TotalTracks != 0 {
			trackStr = fmt.Sprintf("%d/%d", metadata.Track, metadata.TotalTracks)
		}
		id3tag.AddTextFrame("TRCK", id3v2.EncodingUTF8, trackStr)
	}

	// Set disc information
	if metadata.Disc != 0 || metadata.TotalDiscs != 0 {
		discStr := fmt.Sprintf("%d", metadata.Disc)
		if metadata.TotalDiscs != 0 {
			discStr = fmt.Sprintf("%d/%d", metadata.Disc, metadata.TotalDiscs)
		}
		id3tag.AddTextFrame("TPOS", id3v2.EncodingUTF8, discStr)
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

	// Set chapters
	// First, delete existing chapter frames
	id3tag.DeleteFrames("CHAP")

	for i, chapter := range metadata.Chapters {
		// Create proper chapter frame
		startTime := time.Duration(chapter.Start) * time.Second
		var endTime time.Duration

		// Set end time to next chapter's start time or audio duration for last chapter
		if i+1 < len(metadata.Chapters) {
			endTime = time.Duration(metadata.Chapters[i+1].Start) * time.Second
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
