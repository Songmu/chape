package chapel

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/goccy/go-yaml"
)

func (c *Chapel) Dump(output io.Writer) error {
	metadata, err := c.getMetadata()
	if err != nil {
		return err
	}

	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	// Add YAML Language Server schema comment
	schemaComment := "# yaml-language-server: $schema=https://raw.githubusercontent.com/Songmu/chapel/refs/heads/main/schema.yaml\n"
	if _, err = output.Write([]byte(schemaComment)); err != nil {
		return err
	}
	_, err = output.Write(yamlData)
	return err
}

// getMetadata extracts metadata from the MP3 file
func (c *Chapel) getMetadata() (*Metadata, error) {
	// Open the MP3 file
	file, err := os.Open(c.audio)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	id3tag, err := id3v2.Open(c.audio, id3v2.Options{Parse: true})
	if err != nil {
		return nil, err
	}
	defer id3tag.Close()

	var metadata = &Metadata{}

	// Read all text frames using the centralized mapping
	readTextFrames(id3tag, metadata)

	// Try to get date from TDRC (ID3v2.4) or fall back to Year
	if dateFramer := id3tag.GetLastFrame("TDRC"); dateFramer != nil {
		if tf, ok := dateFramer.(id3v2.TextFrame); ok && tf.Text != "" {
			// Parse TDRC format
			var ts Timestamp
			if err := ts.UnmarshalYAML([]byte(tf.Text)); err == nil {
				metadata.Date = &ts
			}
		}
	} else if id3tag.Year() != "" {
		// Fall back to Year for ID3v2.3 compatibility
		var ts Timestamp
		if err := ts.UnmarshalYAML([]byte(id3tag.Year())); err == nil {
			metadata.Date = &ts
		}
	}

	// Comment frames
	commentFrames := id3tag.GetFrames(id3tag.CommonID("Comments"))
	if len(commentFrames) > 0 {
		if cf, ok := commentFrames[0].(id3v2.CommentFrame); ok {
			metadata.Comment = cf.Text
		}
	}

	// Lyrics frames
	lyricsFrames := id3tag.GetFrames("USLT") // Unsynchronised lyrics/text transcription
	if len(lyricsFrames) > 0 {
		if ulf, ok := lyricsFrames[0].(id3v2.UnsynchronisedLyricsFrame); ok {
			metadata.Lyrics = ulf.Lyrics
		}
	}

	// Priority: Chapel struct artwork > CHAPEL_SOURCE from MP3
	if c.artwork != "" {
		metadata.Artwork = c.artwork
	} else {
		pictureFrames := id3tag.GetFrames(id3tag.CommonID("Attached picture"))
		if len(pictureFrames) > 0 {
			if pf, ok := pictureFrames[0].(id3v2.PictureFrame); ok {
				if len(pf.Picture) > 0 {
					// Check for chapel source in TXXX frames first
					chapelSource := ""
					txxxFrames := id3tag.GetFrames("TXXX")
					for _, frame := range txxxFrames {
						if udtf, ok := frame.(id3v2.UserDefinedTextFrame); ok {
							// UserDefinedTextFrame has Description and Value fields
							if udtf.Description == "CHAPEL_SOURCE" {
								chapelSource = udtf.Value
								break
							}
						}
					}
					// Always prefer CHAPEL_SOURCE if available, regardless of file existence
					if chapelSource != "" {
						metadata.Artwork = chapelSource
					} else {
						metadata.Artwork = fmt.Sprintf("data:%s;base64,%s",
							pf.MimeType,
							base64.StdEncoding.EncodeToString(pf.Picture))
					}
				}
			}
		}
	}

	// Chapter frames
	chapterFrames := id3tag.GetFrames("CHAP")
	for _, frame := range chapterFrames {
		if cf, ok := frame.(id3v2.ChapterFrame); ok {
			chapter := &Chapter{
				Title: cf.Title.Text,
				Start: cf.StartTime,
			}
			metadata.Chapters = append(metadata.Chapters, chapter)
		}
	}
	slices.SortFunc(metadata.Chapters, func(a, b *Chapter) int {
		return cmp.Compare(a.Start, b.Start)
	})

	// Override artwork with Chapel struct setting if specified
	if c.artwork != "" {
		metadata.Artwork = c.artwork
	}

	// Apply artwork processing (file creation, etc.)
	if err := c.processArtwork(metadata); err != nil {
		return nil, fmt.Errorf("failed to process artwork: %w", err)
	}

	return metadata, nil
}

// processArtwork handles artwork processing logic shared between Dump and Apply
func (c *Chapel) processArtwork(metadata *Metadata) error {
	aw := metadata.Artwork
	if aw != "" {
		if !strings.HasPrefix(aw, "http://") && !strings.HasPrefix(aw, "https://") &&
			!strings.HasPrefix(aw, "data:") {
			// Local file path - check if file exists
			if _, err := os.Stat(aw); os.IsNotExist(err) {
				// File doesn't exist, try to extract from embedded artwork
				// Need to get embedded artwork data from MP3
				embeddedDataURI, err := c.getEmbeddedArtwork()
				if err != nil {
					return fmt.Errorf("failed to get embedded artwork: %w", err)
				}
				if embeddedDataURI != "" {
					// Extract from embedded data URI
					// XXX: How do we handle file extension mismatch?
					if err := c.extractArtworkToFile(embeddedDataURI, aw); err != nil {
						return fmt.Errorf("failed to extract artwork: %w", err)
					}
				}
			} else if err != nil {
				return fmt.Errorf("failed to check artwork file: %w", err)
			}
		}
	}
	return nil
}

// getEmbeddedArtwork extracts embedded artwork from MP3 as data URI
func (c *Chapel) getEmbeddedArtwork() (string, error) {
	id3tag, err := id3v2.Open(c.audio, id3v2.Options{Parse: true})
	if err != nil {
		return "", err
	}
	defer id3tag.Close()

	// Picture frames
	pictureFrames := id3tag.GetFrames(id3tag.CommonID("Attached picture"))
	if len(pictureFrames) > 0 {
		if pf, ok := pictureFrames[0].(id3v2.PictureFrame); ok {
			if len(pf.Picture) > 0 {
				return fmt.Sprintf("data:%s;base64,%s",
					pf.MimeType,
					base64.StdEncoding.EncodeToString(pf.Picture)), nil
			}
		}
	}
	return "", nil
}

// extractArtworkToFile extracts artwork from data URI and saves to file
func (c *Chapel) extractArtworkToFile(dataURI, outputPath string) error {
	// Parse data URI
	pictureData, mimeType, err := parseDataURI(dataURI)
	if err != nil {
		return err
	}

	// Determine file extension from MIME type if outputPath doesn't have one
	if filepath.Ext(outputPath) == "" {
		ext := getExtFromMimeType(mimeType)
		if ext != "" {
			outputPath = outputPath + ext
		}
	}

	// Write to file
	return os.WriteFile(outputPath, pictureData, 0644)
}

// getExtFromMimeType returns file extension for a MIME type
func getExtFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/bmp":
		return ".bmp"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
