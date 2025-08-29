package chapel

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml/token"
)

// Metadata represents the metadata of an MP3 file
type Metadata struct {
	Title       string       `yaml:"title"`                 // TIT2 tag (Title/songname/content description)
	Subtitle    string       `yaml:"subtitle,omitempty"`    // TIT3 tag (Subtitle/Description refinement)
	Artist      string       `yaml:"artist"`                // TPE1 tag (Lead performer(s)/Soloist(s))
	Album       string       `yaml:"album"`                 // TALB tag (Album/Movie/Show title)
	AlbumArtist string       `yaml:"albumArtist,omitempty"` // TPE2 tag (Band/orchestra/accompaniment)
	Grouping    string       `yaml:"grouping,omitempty"`    // TIT1 tag (Content group description)
	Date        *Timestamp   `yaml:"date,omitempty"`        // TDRC tag for ID3v2.4 (Recording time)
	Track       *NumberInSet `yaml:"track,omitempty"`       // TRCK tag (Track number/Position in set)
	Disc        *NumberInSet `yaml:"disc,omitempty"`        // TPOS tag (Part of a set)
	Genre       string       `yaml:"genre,omitempty"`       // TCON tag (Content type/Genre)
	Comment     string       `yaml:"comment,omitempty"`     // COMM tag (Comments)
	Composer    string       `yaml:"composer,omitempty"`    // TCOM tag (Composer)
	Publisher   string       `yaml:"publisher,omitempty"`   // TPUB tag (Publisher)
	BPM         int          `yaml:"bpm,omitempty"`         // TBPM tag (BPM - Beats per minute)
	Chapters    []*Chapter   `yaml:"chapters,omitempty"`    // CHAP tag (Chapter frames)
	Artwork     string       `yaml:"artwork,omitempty"`     // APIC tag (Attached picture)
	Lyrics      string       `yaml:"lyrics,omitempty"`      // USLT tag (Unsynchronised lyric/text transcription)
}

// NumberInSet represents a current/total number pair in ID3v2 format (e.g., "3/10", "1/2")
type NumberInSet struct {
	Current int
	Total   int
}

// Timestamp wraps time.Time for ID3v2 timestamp format as defined in ID3v2.4.0-structure.
// The timestamp fields are based on a subset of ISO 8601 and can have varying levels of precision.
// All time stamps are UTC. Valid formats: yyyy, yyyy-MM, yyyy-MM-dd, yyyy-MM-ddTHH, yyyy-MM-ddTHH:mm, yyyy-MM-ddTHH:mm:ss
type Timestamp struct {
	time.Time
	Precision Precision
}

// Precision represents the precision level of the timestamp
type Precision int

const (
	PrecisionYear Precision = iota
	PrecisionMonth
	PrecisionDay
	PrecisionHour
	PrecisionMinute
	PrecisionSecond
)

// Chapter represents a single chapter with start time and title
type Chapter struct {
	Title string        `json:"title"`
	Start time.Duration `json:"start"`
}

// String returns the chapter as a string in WebVTT format
func (c *Chapter) String() string {
	// Format duration to WebVTT time string
	ms := c.Start.Milliseconds()
	hours := ms / 3600000
	minutes := (ms % 3600000) / 60000
	seconds := (ms % 60000) / 1000
	millis := ms % 1000

	var timeStr string
	// Format without milliseconds if they are zero
	if millis == 0 {
		if hours > 0 {
			timeStr = fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
		} else {
			timeStr = fmt.Sprintf("%d:%02d", minutes, seconds)
		}
	} else {
		// Format with milliseconds
		if hours > 0 {
			timeStr = fmt.Sprintf("%d:%02d:%02d.%03d", hours, minutes, seconds, millis)
		} else {
			timeStr = fmt.Sprintf("%d:%02d.%03d", minutes, seconds, millis)
		}
	}

	return fmt.Sprintf("%s %s", timeStr, c.Title)
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

	// Parse WebVTT time format
	timeStr := stuff[0]
	colonParts := strings.Split(timeStr, ":")
	if len(colonParts) < 2 || len(colonParts) > 3 {
		return fmt.Errorf("invalid time format: %s", timeStr)
	}

	var hours, minutes int
	var secondsStr string

	if len(colonParts) == 3 {
		// Format: H:MM:SS.mmm
		h, err := strconv.Atoi(colonParts[0])
		if err != nil {
			return fmt.Errorf("invalid hours: %s", colonParts[0])
		}
		hours = h

		m, err := strconv.Atoi(colonParts[1])
		if err != nil {
			return fmt.Errorf("invalid minutes: %s", colonParts[1])
		}
		minutes = m
		secondsStr = colonParts[2]
	} else {
		// Format: M:SS.mmm or MM:SS.mmm
		m, err := strconv.Atoi(colonParts[0])
		if err != nil {
			return fmt.Errorf("invalid minutes: %s", colonParts[0])
		}
		minutes = m
		secondsStr = colonParts[1]
	}

	// Parse seconds and milliseconds
	var seconds int
	var millis int

	if strings.Contains(secondsStr, ".") {
		parts := strings.Split(secondsStr, ".")
		s, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid seconds: %s", parts[0])
		}
		seconds = s

		if len(parts[1]) > 0 {
			// Pad or trim to 3 digits for milliseconds
			msStr := parts[1]
			if len(msStr) > 3 {
				msStr = msStr[:3]
			} else {
				msStr = msStr + strings.Repeat("0", 3-len(msStr))
			}
			ms, err := strconv.Atoi(msStr)
			if err != nil {
				return fmt.Errorf("invalid milliseconds: %s", parts[1])
			}
			millis = ms
		}
	} else {
		s, err := strconv.Atoi(secondsStr)
		if err != nil {
			return fmt.Errorf("invalid seconds: %s", secondsStr)
		}
		seconds = s
	}

	totalMs := int64(hours)*3600000 + int64(minutes)*60000 + int64(seconds)*1000 + int64(millis)

	*c = Chapter{
		Title: stuff[1],
		Start: time.Duration(totalMs) * time.Millisecond,
	}
	return nil
}

// String returns number in set in ID3v2 format
func (n *NumberInSet) String() string {
	if n.Total > 0 {
		return fmt.Sprintf("%d/%d", n.Current, n.Total)
	}
	return fmt.Sprintf("%d", n.Current)
}

// MarshalYAML marshals number in set to YAML format
func (n *NumberInSet) MarshalYAML() ([]byte, error) {
	return []byte(n.String()), nil
}

// UnmarshalYAML unmarshals number in set from YAML format
func (n *NumberInSet) UnmarshalYAML(b []byte) error {
	str := unquote(strings.TrimSpace(string(b)))
	current, total := parseNumberPair(str)
	*n = NumberInSet{Current: current, Total: total}
	return nil
}

// String returns timestamp in ID3v2 format
func (t *Timestamp) String() string {
	if t.Time.IsZero() {
		return ""
	}

	switch t.Precision {
	case PrecisionYear:
		return t.Time.Format("2006")
	case PrecisionMonth:
		return t.Time.Format("2006-01")
	case PrecisionDay:
		return t.Time.Format("2006-01-02")
	case PrecisionHour:
		return t.Time.UTC().Format("2006-01-02T15")
	case PrecisionMinute:
		return t.Time.UTC().Format("2006-01-02T15:04")
	case PrecisionSecond:
		return t.Time.UTC().Format("2006-01-02T15:04:05")
	default:
		return t.Time.Format("2006")
	}
}

// MarshalYAML marshals timestamp to YAML format
func (t *Timestamp) MarshalYAML() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalYAML unmarshals timestamp from YAML format
func (t *Timestamp) UnmarshalYAML(b []byte) error {
	str := unquote(strings.TrimSpace(string(b)))
	if str == "" {
		*t = Timestamp{}
		return nil
	}

	// Try parsing different precision levels
	formats := []struct {
		layout    string
		precision Precision
	}{
		{"2006-01-02T15:04:05", PrecisionSecond},
		{"2006-01-02T15:04", PrecisionMinute},
		{"2006-01-02T15", PrecisionHour},
		{"2006-01-02", PrecisionDay},
		{"2006-01", PrecisionMonth},
		{"2006", PrecisionYear},
	}

	for _, format := range formats {
		if parsedTime, err := time.ParseInLocation(format.layout, str, time.UTC); err == nil {
			*t = Timestamp{Time: parsedTime, Precision: format.precision}
			return nil
		}
	}

	return fmt.Errorf("invalid timestamp format: %s", str)
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
