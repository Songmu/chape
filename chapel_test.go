package chapel

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
)

func TestMetadataYAMLMarshal(t *testing.T) {
	metadata := &Metadata{
		Title:       "Test Song",
		Artist:      "Test Artist",
		Album:       "Test Album",
		AlbumArtist: "Test Album Artist",
		Date:        "2024",
		Track:       1,
		TotalTracks: 10,
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
	if !strings.Contains(yamlStr, "date: \"2024\"") {
		t.Errorf("YAML should contain date")
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
