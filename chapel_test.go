package chapel

import (
	"bytes"
	"strings"
	"testing"

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
			{Start: 90, Title: "Main Topic"},
			{Start: 945, Title: "Conclusion"},
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
		t.Errorf("YAML should contain chapters")
	}
}

func TestParseChapter(t *testing.T) {
	tests := []struct {
		input     string
		wantTime  uint64
		wantTitle string
		wantErr   bool
	}{
		{"0:00 Introduction", 0, "Introduction", false},
		{"1:30 Main Topic", 90, "Main Topic", false},
		{"1:02:30 Long Chapter", 3750, "Long Chapter", false},
		{"invalid", 0, "", true},
		{"1:30", 0, "", true}, // Missing title
	}

	for _, tt := range tests {
		ch, err := ParseChapter(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseChapter(%q) should have returned error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseChapter(%q) returned unexpected error: %v", tt.input, err)
			continue
		}
		if ch.Start != tt.wantTime {
			t.Errorf("ParseChapter(%q) Start = %d, want %d", tt.input, ch.Start, tt.wantTime)
		}
		if ch.Title != tt.wantTitle {
			t.Errorf("ParseChapter(%q) Title = %q, want %q", tt.input, ch.Title, tt.wantTitle)
		}
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		seconds uint64
		want    string
	}{
		{0, "0:00"},
		{59, "0:59"},
		{60, "1:00"},
		{90, "1:30"},
		{3599, "59:59"},
		{3600, "1:00:00"},
		{3661, "1:01:01"},
		{7322, "2:02:02"},
	}

	for _, tt := range tests {
		got := FormatTime(tt.seconds)
		if got != tt.want {
			t.Errorf("FormatTime(%d) = %q, want %q", tt.seconds, got, tt.want)
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
	chapter := &Chapter{Start: 90, Title: "Main Topic"}

	yamlData, err := yaml.Marshal(chapter)
	if err != nil {
		t.Fatalf("Failed to marshal chapter: %v", err)
	}

	yamlStr := string(yamlData)
	expected := "1:30 Main Topic\n"
	if yamlStr != expected {
		t.Errorf("Expected %q, got %q", expected, yamlStr)
	}
}

func TestChapterUnmarshalYAML(t *testing.T) {
	yamlStr := "1:30 Main Topic"
	var chapter Chapter

	err := yaml.Unmarshal([]byte(yamlStr), &chapter)
	if err != nil {
		t.Fatalf("Failed to unmarshal chapter: %v", err)
	}

	if chapter.Start != 90 {
		t.Errorf("Expected start time 90, got %d", chapter.Start)
	}
	if chapter.Title != "Main Topic" {
		t.Errorf("Expected title 'Main Topic', got %q", chapter.Title)
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
