package chapel

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/goccy/go-yaml"
)

func (c *Chapel) Dump(output io.Writer) error {
	// Open the MP3 file
	file, err := os.Open(c.audio)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	id3tag, err := id3v2.Open(c.audio, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer id3tag.Close()

	var metadata = &Metadata{}
	// Read basic metadata from ID3v2
	metadata.Title = id3tag.Title()
	metadata.Artist = id3tag.Artist()
	metadata.Album = id3tag.Album()
	metadata.Genre = id3tag.Genre()

	// Try to get date from TDRC (ID3v2.4) or fall back to Year
	if dateFramer := id3tag.GetLastFrame("TDRC"); dateFramer != nil {
		if tf, ok := dateFramer.(id3v2.TextFrame); ok {
			metadata.Date = tf.Text
		}
	} else if id3tag.Year() != "" {
		// Fall back to Year for ID3v2.3 compatibility
		metadata.Date = id3tag.Year()
	}

	// Get additional metadata from specific frames
	if albumArtistFramer := id3tag.GetLastFrame("TPE2"); albumArtistFramer != nil {
		if tf, ok := albumArtistFramer.(id3v2.TextFrame); ok {
			metadata.AlbumArtist = tf.Text
		}
	}

	if composerFramer := id3tag.GetLastFrame("TCOM"); composerFramer != nil {
		if tf, ok := composerFramer.(id3v2.TextFrame); ok {
			metadata.Composer = tf.Text
		}
	}

	if publisherFramer := id3tag.GetLastFrame("TPUB"); publisherFramer != nil {
		if tf, ok := publisherFramer.(id3v2.TextFrame); ok {
			metadata.Publisher = tf.Text
		}
	}

	if bpmFramer := id3tag.GetLastFrame("TBPM"); bpmFramer != nil {
		if tf, ok := bpmFramer.(id3v2.TextFrame); ok {
			if bpm, err := strconv.Atoi(tf.Text); err == nil {
				metadata.BPM = bpm
			}
		}
	}

	if trackFramer := id3tag.GetLastFrame("TRCK"); trackFramer != nil {
		if tf, ok := trackFramer.(id3v2.TextFrame); ok {
			parseTrack(tf.Text, metadata)
		}
	}

	if discFramer := id3tag.GetLastFrame("TPOS"); discFramer != nil {
		if tf, ok := discFramer.(id3v2.TextFrame); ok {
			parseDisc(tf.Text, metadata)
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
	lyricsFrames := id3tag.GetFrames(id3tag.CommonID("Lyrics"))
	if len(lyricsFrames) > 0 {
		if ulf, ok := lyricsFrames[0].(id3v2.UnsynchronisedLyricsFrame); ok {
			metadata.Lyrics = ulf.Lyrics
		}
	}

	// Picture frames
	pictureFrames := id3tag.GetFrames(id3tag.CommonID("Attached picture"))
	if len(pictureFrames) > 0 {
		if pf, ok := pictureFrames[0].(id3v2.PictureFrame); ok {
			if len(pf.Picture) > 0 {
				metadata.Artwork = fmt.Sprintf("data:%s;base64,%s",
					pf.MimeType,
					base64.StdEncoding.EncodeToString(pf.Picture))
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

	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}
	_, err = output.Write(yamlData)
	return err
}

// parseNumberPair parses strings like "1" or "1/10" and returns current and total values
func parseNumberPair(s string) (current, total int) {
	parts := strings.Split(s, "/")
	if len(parts) > 0 && parts[0] != "" {
		if c, err := strconv.Atoi(parts[0]); err == nil {
			current = c
		}
	}
	if len(parts) > 1 && parts[1] != "" {
		if t, err := strconv.Atoi(parts[1]); err == nil {
			total = t
		}
	}
	return current, total
}

// parseTrack parses track string like "1" or "1/10"
func parseTrack(s string, metadata *Metadata) {
	current, total := parseNumberPair(s)
	if current > 0 {
		metadata.Track = current
	}
	if total > 0 {
		metadata.TotalTracks = total
	}
}

// parseDisc parses disc string like "1" or "1/2"
func parseDisc(s string, metadata *Metadata) {
	current, total := parseNumberPair(s)
	if current > 0 {
		metadata.Disc = current
	}
	if total > 0 {
		metadata.TotalDiscs = total
	}
}
