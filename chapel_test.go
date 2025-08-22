package chapel

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
)

func TestMetadataYAMLMarshal(t *testing.T) {
	date2024, _ := time.Parse("2006", "2024")
	metadata := &Metadata{
		Title:       "Test Song",
		Artist:      "Test Artist",
		Album:       "Test Album",
		AlbumArtist: "Test Album Artist",
		Date:        &Timestamp{Time: date2024, Precision: PrecisionYear},
		Track:       &NumberInSet{Current: 1, Total: 10},
		Genre:       "Podcast",
		Chapters: []*Chapter{
			{Start: 0, Title: "Introduction"},
			{Start: 90 * time.Second, Title: "Main Topic"},
			{Start: 945 * time.Second, Title: "Conclusion"},
		},
	}

	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	yamlStr := string(yamlData)

	// Check that required fields are present
	if !strings.Contains(yamlStr, "title: Test Song") {
		t.Errorf("YAML should contain title")
	}
	if !strings.Contains(yamlStr, "artist: Test Artist") {
		t.Errorf("YAML should contain artist")
	}
	if !strings.Contains(yamlStr, "date: 2024") {
		t.Errorf("YAML should contain date")
	}
	if !strings.Contains(yamlStr, "track: 1/10") {
		t.Errorf("YAML should contain track in current/total format")
	}
	if !strings.Contains(yamlStr, "- 0:00 Introduction") {
		t.Errorf("YAML should contain chapters with formatted time")
	}
}

func TestChapterString(t *testing.T) {
	tests := []struct {
		chapter  *Chapter
		expected string
	}{
		// Basic formatting
		{&Chapter{Start: 0, Title: "Introduction"}, "0:00 Introduction"},
		{&Chapter{Start: 59 * time.Second, Title: "Test"}, "0:59 Test"},
		{&Chapter{Start: 60 * time.Second, Title: "Test"}, "1:00 Test"},
		{&Chapter{Start: 90 * time.Second, Title: "Main Topic"}, "1:30 Main Topic"},
		{&Chapter{Start: 3599 * time.Second, Title: "Test"}, "59:59 Test"},
		{&Chapter{Start: 3600 * time.Second, Title: "Test"}, "1:00:00 Test"},
		{&Chapter{Start: 3661 * time.Second, Title: "Test"}, "1:01:01 Test"},
		{&Chapter{Start: 7322 * time.Second, Title: "Test"}, "2:02:02 Test"},

		// With milliseconds
		{&Chapter{Start: 90500 * time.Millisecond, Title: "Main Topic"}, "1:30.500 Main Topic"},
		{&Chapter{Start: 3750 * time.Second, Title: "Long Chapter"}, "1:02:30 Long Chapter"},
		{&Chapter{Start: (3750*time.Second + 123*time.Millisecond), Title: "Long Chapter"}, "1:02:30.123 Long Chapter"},
		{&Chapter{Start: (3661*time.Second + 123*time.Millisecond), Title: "Test"}, "1:01:01.123 Test"},
	}

	for _, tt := range tests {
		got := tt.chapter.String()
		if got != tt.expected {
			t.Errorf("Chapter.String() = %q, want %q", got, tt.expected)
		}
	}
}

func TestDumpWithoutFile(t *testing.T) {
	c := New("nonexistent.mp3")
	var buf bytes.Buffer
	err := c.Dump(&buf)
	if err == nil {
		t.Error("Dump should return error for nonexistent file")
	}
}

func TestChapterMarshalYAML(t *testing.T) {
	tests := []struct {
		chapter  *Chapter
		expected string
	}{
		{&Chapter{Start: 90 * time.Second, Title: "Main Topic"}, "1:30 Main Topic\n"},
		{&Chapter{Start: 90500 * time.Millisecond, Title: "Main Topic"}, "1:30.500 Main Topic\n"},
		{&Chapter{Start: 0, Title: "Introduction"}, "0:00 Introduction\n"},
	}

	for _, tt := range tests {
		yamlData, err := yaml.Marshal(tt.chapter)
		if err != nil {
			t.Fatalf("Failed to marshal chapter: %v", err)
		}

		yamlStr := string(yamlData)
		if yamlStr != tt.expected {
			t.Errorf("Expected %q, got %q", tt.expected, yamlStr)
		}
	}
}

func TestChapterUnmarshalYAML(t *testing.T) {
	tests := []struct {
		yamlStr   string
		wantStart time.Duration
		wantTitle string
	}{
		{"1:30 Main Topic", 90 * time.Second, "Main Topic"},
		{"1:30.500 Main Topic", 90500 * time.Millisecond, "Main Topic"},
		{"0:00 Introduction", 0, "Introduction"},
		// Test millisecond padding behavior
		{"1:30.5 Main Topic", 500*time.Millisecond + 90*time.Second, "Main Topic"},    // .5 → .500
		{"1:30.12 Main Topic", 120*time.Millisecond + 90*time.Second, "Main Topic"},   // .12 → .120
		{"1:30.1234 Main Topic", 123*time.Millisecond + 90*time.Second, "Main Topic"}, // .1234 → .123 (truncated)
		{"0:05.05 Short", 5050 * time.Millisecond, "Short"},                           // .05 → .050
	}

	for _, tt := range tests {
		var chapter Chapter
		err := yaml.Unmarshal([]byte(tt.yamlStr), &chapter)
		if err != nil {
			t.Fatalf("Failed to unmarshal chapter: %v", err)
		}

		if chapter.Start != tt.wantStart {
			t.Errorf("Expected start time %v, got %v", tt.wantStart, chapter.Start)
		}
		if chapter.Title != tt.wantTitle {
			t.Errorf("Expected title %q, got %q", tt.wantTitle, chapter.Title)
		}
	}
}

func TestChapterWithQuotes(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Normal Title", "0:00 Normal Title"},
		{"Title: With Colon", `"0:00 Title: With Colon"`},
		{"Title with # hash", `"0:00 Title with # hash"`},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			chapter := &Chapter{Start: 0, Title: tt.title}

			yamlData, err := yaml.Marshal(chapter)
			if err != nil {
				t.Fatalf("Failed to marshal chapter: %v", err)
			}

			yamlStr := strings.TrimSpace(string(yamlData))
			if yamlStr != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, yamlStr)
			}

			// Test round-trip
			var unmarshaledChapter Chapter
			err = yaml.Unmarshal(yamlData, &unmarshaledChapter)
			if err != nil {
				t.Fatalf("Failed to unmarshal chapter: %v", err)
			}

			if unmarshaledChapter.Title != tt.title {
				t.Errorf("Round-trip failed: expected title %q, got %q", tt.title, unmarshaledChapter.Title)
			}
		})
	}
}

func TestNumberInSet(t *testing.T) {
	tests := []struct {
		input    *NumberInSet
		expected string
	}{
		{&NumberInSet{Current: 1, Total: 0}, "1"},
		{&NumberInSet{Current: 3, Total: 10}, "3/10"},
		{&NumberInSet{Current: 1, Total: 2}, "1/2"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.expected {
			t.Errorf("NumberInSet.String() = %q, want %q", got, tt.expected)
		}

		// Test YAML marshaling
		yamlData, err := yaml.Marshal(tt.input)
		if err != nil {
			t.Fatalf("Failed to marshal NumberInSet: %v", err)
		}

		yamlStr := strings.TrimSpace(string(yamlData))
		if yamlStr != tt.expected {
			t.Errorf("YAML marshal = %q, want %q", yamlStr, tt.expected)
		}

		// Test YAML unmarshaling
		var unmarshaled NumberInSet
		err = yaml.Unmarshal([]byte(tt.expected), &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal NumberInSet: %v", err)
		}

		if unmarshaled.Current != tt.input.Current || unmarshaled.Total != tt.input.Total {
			t.Errorf("Unmarshal failed: got Current=%d, Total=%d, want Current=%d, Total=%d",
				unmarshaled.Current, unmarshaled.Total, tt.input.Current, tt.input.Total)
		}
	}
}

func TestTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected *Timestamp
		output   string
	}{
		{"2024", &Timestamp{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Precision: PrecisionYear}, "2024"},
		{"2024-08", &Timestamp{Time: time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC), Precision: PrecisionMonth}, "2024-08"},
		{"2024-08-15", &Timestamp{Time: time.Date(2024, 8, 15, 0, 0, 0, 0, time.UTC), Precision: PrecisionDay}, "2024-08-15"},
		{"2024-08-15T14", &Timestamp{Time: time.Date(2024, 8, 15, 14, 0, 0, 0, time.UTC), Precision: PrecisionHour}, "2024-08-15T14"},
		{"2024-08-15T14:30", &Timestamp{Time: time.Date(2024, 8, 15, 14, 30, 0, 0, time.UTC), Precision: PrecisionMinute}, "2024-08-15T14:30"},
		{"2024-08-15T14:30:45", &Timestamp{Time: time.Date(2024, 8, 15, 14, 30, 45, 0, time.UTC), Precision: PrecisionSecond}, "2024-08-15T14:30:45"},
	}

	for _, tt := range tests {
		// Test unmarshaling
		var ts Timestamp
		err := ts.UnmarshalYAML([]byte(tt.input))
		if err != nil {
			t.Fatalf("Failed to unmarshal Timestamp %q: %v", tt.input, err)
		}

		if !ts.Time.Equal(tt.expected.Time) || ts.Precision != tt.expected.Precision {
			t.Errorf("Unmarshal %q: got Time=%v, Precision=%v, want Time=%v, Precision=%v",
				tt.input, ts.Time, ts.Precision, tt.expected.Time, tt.expected.Precision)
		}

		// Test marshaling
		got := ts.String()
		if got != tt.output {
			t.Errorf("Timestamp.String() = %q, want %q", got, tt.output)
		}

		// Test YAML round-trip
		yamlData, err := yaml.Marshal(&ts)
		if err != nil {
			t.Fatalf("Failed to marshal Timestamp: %v", err)
		}

		yamlStr := strings.TrimSpace(string(yamlData))
		if yamlStr != tt.output {
			t.Errorf("YAML marshal = %q, want %q", yamlStr, tt.output)
		}
	}
}

func TestParseDataURI(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
		mimeType    string
	}{
		{"data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=", false, "image/jpeg"},
		{"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==", false, "image/png"},
		{"data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7", false, "image/gif"},
		{"invalid-data-uri", true, ""},
		{"data:image/jpeg,notbase64", true, ""},
	}

	for _, tt := range tests {
		data, mimeType, err := parseDataURI(tt.input)

		if tt.expectError {
			if err == nil {
				t.Errorf("parseDataURI(%q) should return error", tt.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("parseDataURI(%q) returned error: %v", tt.input, err)
			continue
		}

		if mimeType != tt.mimeType {
			t.Errorf("parseDataURI(%q) mimeType = %q, want %q", tt.input, mimeType, tt.mimeType)
		}

		if len(data) == 0 {
			t.Errorf("parseDataURI(%q) returned empty data", tt.input)
		}
	}
}

func TestParseArtwork(t *testing.T) {
	// Test data URI
	dataURI := "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
	_, mimeType, err := parseArtwork(dataURI)
	if err != nil {
		t.Errorf("parseArtwork with data URI failed: %v", err)
	}
	if mimeType != "image/jpeg" {
		t.Errorf("parseArtwork data URI mimeType = %q, want %q", mimeType, "image/jpeg")
	}

	// Test non-existent file path (should return error)
	_, _, err = parseArtwork("nonexistent.jpg")
	if err == nil {
		t.Error("parseArtwork with nonexistent file should return error")
	}
}

func TestParseHTTPURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{"invalid URL", "http://invalid-domain-that-does-not-exist.invalid", true},
		{"non-HTTP URL", "ftp://example.com/image.jpg", false}, // Should be treated as file path, not HTTP
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.url == "ftp://example.com/image.jpg" {
				// This should be treated as file path, not HTTP URL
				_, _, err := parseArtwork(tt.url)
				if err == nil {
					t.Error("parseArtwork with FTP URL should return error (treated as file path)")
				}
				return
			}

			_, _, err := parseHTTPURL(tt.url)
			if tt.expectError && err == nil {
				t.Errorf("parseHTTPURL(%q) should return error", tt.url)
			}
			if !tt.expectError && err != nil {
				t.Errorf("parseHTTPURL(%q) returned unexpected error: %v", tt.url, err)
			}
		})
	}
}

func TestGetMimeTypeFromExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".JPG", "image/jpeg"},
		{".png", "image/png"},
		{".PNG", "image/png"},
		{".gif", "image/gif"},
		{".bmp", "image/bmp"},
		{".webp", "image/webp"},
		{".txt", ""},
		{".unknown", ""},
	}

	for _, tt := range tests {
		got := getMimeTypeFromExt(tt.ext)
		if got != tt.expected {
			t.Errorf("getMimeTypeFromExt(%q) = %q, want %q", tt.ext, got, tt.expected)
		}
	}
}

func TestGetExtFromMimeType(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/bmp", ".bmp"},
		{"image/webp", ".webp"},
		{"text/plain", ""},
		{"unknown/type", ""},
	}

	for _, tt := range tests {
		got := getExtFromMimeType(tt.mimeType)
		if got != tt.expected {
			t.Errorf("getExtFromMimeType(%q) = %q, want %q", tt.mimeType, got, tt.expected)
		}
	}
}

func TestDumpWithArtwork(t *testing.T) {
	var buf bytes.Buffer

	// Test with HTTP URL
	c := New("nonexistent.mp3", "https://example.com/cover.jpg")
	err := c.Dump(&buf)
	if err == nil {
		t.Error("Dump should return error for nonexistent file")
	}

	// Test with file path that doesn't exist
	c = New("nonexistent.mp3", "/tmp/test-artwork.jpg")
	err = c.Dump(&buf)
	if err == nil {
		t.Error("Dump should return error for nonexistent file")
	}
}
