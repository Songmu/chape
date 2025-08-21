package chapel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml/token"
)

// Metadata represents the metadata of an MP3 file
type Metadata struct {
	Title       string     `yaml:"title,omitempty"`
	Artist      string     `yaml:"artist,omitempty"`
	Album       string     `yaml:"album,omitempty"`
	AlbumArtist string     `yaml:"albumArtist,omitempty"`
	Year        int        `yaml:"year,omitempty"`
	Track       int        `yaml:"track,omitempty"`
	TotalTracks int        `yaml:"totalTracks,omitempty"`
	Disc        int        `yaml:"disc,omitempty"`
	TotalDiscs  int        `yaml:"totalDiscs,omitempty"`
	Genre       string     `yaml:"genre,omitempty"`
	Comment     string     `yaml:"comment,omitempty"`
	Composer    string     `yaml:"composer,omitempty"`
	Publisher   string     `yaml:"publisher,omitempty"`
	BPM         int        `yaml:"bpm,omitempty"`
	Chapters    []*Chapter `yaml:"chapters,omitempty"`
	Artwork     string     `yaml:"artwork,omitempty"`
	Lyrics      string     `yaml:"lyrics,omitempty"`
}

// Chapter represents a single chapter with start time and title
type Chapter struct {
	Title string `json:"title"`
	Start uint64 `json:"start"`
}

// FormatTime formats seconds to time string
func FormatTime(seconds uint64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

// String returns the chapter as a string
func (c *Chapter) String() string {
	return fmt.Sprintf("%s %s", FormatTime(c.Start), c.Title)
}

// MarshalYAML marshals the chapter to YAML format
func (c *Chapter) MarshalYAML() ([]byte, error) {
	s := c.String()
	if token.IsNeedQuoted(s) {
		s = strconv.Quote(s)
	}
	return []byte(s), nil
}

// UnmarshalYAML unmarshals the chapter from YAML format
func (c *Chapter) UnmarshalYAML(b []byte) error {
	str := unquote(strings.TrimSpace(string(b)))
	stuff := strings.SplitN(str, " ", 2)
	if len(stuff) != 2 {
		return fmt.Errorf("invalid chapter format: %s", str)
	}
	start, err := convertStringToStart(stuff[0])
	if err != nil {
		return fmt.Errorf("invalid chapter format: %s, %w", str, err)
	}
	*c = Chapter{
		Title: stuff[1],
		Start: start,
	}
	return nil
}

// ParseChapter parses a chapter string like "0:00 Intro" (for testing compatibility)
func ParseChapter(s string) (*Chapter, error) {
	var chapter Chapter
	err := chapter.UnmarshalYAML([]byte(s))
	if err != nil {
		return nil, err
	}
	return &chapter, nil
}

// convertStringToStart converts time string to seconds (podbard compatible)
func convertStringToStart(str string) (uint64, error) {
	if l := len(strings.Split(str, ":")); l > 3 {
		return 0, fmt.Errorf("invalid time format: %s", str)
	} else if l == 2 {
		str = "0:" + str
	}

	parts := strings.Split(str, ":")
	var hours, minutes, seconds uint64
	var err error

	if len(parts) >= 3 {
		hours, err = strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return 0, err
		}
		minutes, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return 0, err
		}
		seconds, err = strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			return 0, err
		}
	} else if len(parts) == 2 {
		minutes, err = strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return 0, err
		}
		seconds, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return 0, err
		}
	}

	return hours*3600 + minutes*60 + seconds, nil
}

// unquote removes quotes from a string, handling both single and double quotes
func unquote(s string) string {
	if len(s) <= 1 {
		return s
	}
	if s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	if s[0] == '"' {
		str, err := strconv.Unquote(s)
		if err == nil {
			return str
		}
	}
	return s
}
