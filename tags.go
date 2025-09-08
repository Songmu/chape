package chape

import (
	"reflect"
	"strconv"

	"github.com/bogem/id3v2/v2"
)

// tagMapping represents the mapping between ID3v2 tags and Metadata fields
type tagMapping struct {
	tagID     string // ID3v2 tag ID (e.g., "TIT2")
	fieldName string // Metadata struct field name (e.g., "Title")
	// Optional custom converter functions (if nil, use reflection)
	toString   func(*Metadata) string  // Custom function to convert field to string
	fromString func(*Metadata, string) // Custom function to set field from string
}

// textFrameMappings defines all text frame mappings
var textFrameMappings = []tagMapping{
	{tagID: "TIT2", fieldName: "Title"},
	{tagID: "TIT3", fieldName: "Subtitle"},
	{tagID: "TPE1", fieldName: "Artist"},
	{tagID: "TALB", fieldName: "Album"},
	{tagID: "TPE2", fieldName: "AlbumArtist"},
	{tagID: "TIT1", fieldName: "Grouping"},
	{tagID: "TCON", fieldName: "Genre"},
	{tagID: "TCOM", fieldName: "Composer"},
	{tagID: "TPUB", fieldName: "Publisher"},
	{tagID: "TCOP", fieldName: "Copyright"},
	{
		tagID:     "TLAN",
		fieldName: "Language",
		toString: func(m *Metadata) string {
			return normalizeLanguageCode(m.Language)
		},
	},
	{
		tagID:     "TBPM",
		fieldName: "BPM",
		toString: func(m *Metadata) string {
			if m.BPM == 0 {
				return ""
			}
			return strconv.Itoa(m.BPM)
		},
		fromString: func(m *Metadata, v string) {
			if bpm, err := strconv.Atoi(v); err == nil {
				m.BPM = bpm
			}
		},
	},
	{
		tagID:     "TRCK",
		fieldName: "Track",
		toString: func(m *Metadata) string {
			return m.Track.String()
		},
		fromString: func(m *Metadata, v string) {
			current, total := parseNumberPair(v)
			if current > 0 {
				m.Track = &NumberInSet{Current: current, Total: total}
			}
		},
	},
	{
		tagID:     "TPOS",
		fieldName: "Disc",
		toString: func(m *Metadata) string {
			return m.Disc.String()
		},
		fromString: func(m *Metadata, v string) {
			current, total := parseNumberPair(v)
			if current > 0 {
				m.Disc = &NumberInSet{Current: current, Total: total}
			}
		},
	},
}

// getValue gets the string value from Metadata for a mapping
func (tm *tagMapping) getValue(metadata *Metadata) string {
	// Use custom converter if available
	if tm.toString != nil {
		return tm.toString(metadata)
	}
	// Use reflection for simple string fields
	return getFieldString(metadata, tm.fieldName)
}

// setValue sets the string value to Metadata for a mapping
func (tm *tagMapping) setValue(metadata *Metadata, value string) {
	// Use custom converter if available
	if tm.fromString != nil {
		tm.fromString(metadata, value)
		return
	}
	// Use reflection for simple string fields
	setFieldString(metadata, tm.fieldName, value)
}

// getFieldString gets string field value from Metadata using reflection
func getFieldString(metadata *Metadata, fieldName string) string {
	r := reflect.ValueOf(metadata).Elem()
	f := r.FieldByName(fieldName)
	if f.IsValid() && f.Kind() == reflect.String {
		return f.String()
	}
	return ""
}

// setFieldString sets string field value to Metadata using reflection
func setFieldString(metadata *Metadata, fieldName string, value string) {
	r := reflect.ValueOf(metadata).Elem()
	f := r.FieldByName(fieldName)
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString(value)
	}
}

// applyTextFrames applies text frames to ID3 tag
func applyTextFrames(id3tag *id3v2.Tag, metadata *Metadata) {
	for _, mapping := range textFrameMappings {
		// Delete existing frame
		id3tag.DeleteFrames(mapping.tagID)

		// Get value from metadata
		value := mapping.getValue(metadata)

		// Add frame if value is not empty
		if value != "" {
			id3tag.AddTextFrame(mapping.tagID, id3v2.EncodingUTF8, value)
		}
	}
}

// readTextFrames reads text frames from ID3 tag
func readTextFrames(id3tag *id3v2.Tag, metadata *Metadata) {
	for _, mapping := range textFrameMappings {
		if framer := id3tag.GetLastFrame(mapping.tagID); framer != nil {
			if tf, ok := framer.(id3v2.TextFrame); ok && tf.Text != "" {
				mapping.setValue(metadata, tf.Text)
			}
		}
	}
}
