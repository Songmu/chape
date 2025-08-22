package chapel

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

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
			current, total := parseNumberPair(tf.Text)
			if current > 0 {
				metadata.Track = &NumberInSet{Current: current, Total: total}
			}
		}
	}

	if discFramer := id3tag.GetLastFrame("TPOS"); discFramer != nil {
		if tf, ok := discFramer.(id3v2.TextFrame); ok {
			current, total := parseNumberPair(tf.Text)
			if current > 0 {
				metadata.Disc = &NumberInSet{Current: current, Total: total}
			}
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
