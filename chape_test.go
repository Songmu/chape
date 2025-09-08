package chape_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Songmu/chape"
	"github.com/goccy/go-yaml"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// createDummyMP3 creates a dummy MP3 file with approximately the specified duration
func createDummyMP3(t *testing.T, duration time.Duration) string {
	t.Helper()

	tmpDir := t.TempDir()
	mp3Path := filepath.Join(tmpDir, "chape_test.mp3")
	tmpFile, err := os.Create(mp3Path)
	if err != nil {
		t.Fatalf("Failed to create MP3 file: %v", err)
	}
	defer tmpFile.Close()

	// Write ID3v2 header (empty)
	id3v2Header := []byte{
		0x49, 0x44, 0x33, // "ID3"
		0x04, 0x00, // Version 2.4.0
		0x00,                   // Flags
		0x00, 0x00, 0x00, 0x00, // Size (0, will be updated by id3v2 library)
	}
	if _, err := tmpFile.Write(id3v2Header); err != nil {
		t.Fatalf("Failed to write ID3v2 header: %v", err)
	}

	// Calculate approximate number of frames needed
	// MP3 frame duration is typically around 26ms for 44.1kHz
	frameDurationMs := 26
	totalFrames := int(duration.Milliseconds()) / frameDurationMs

	// MP3 frame header for 44.1kHz, 128kbps, stereo
	frameHeader := []byte{0xFF, 0xFB, 0x90, 0x00}

	// Create frames with minimal data
	frameSize := 417 // Typical frame size for 128kbps
	frameData := make([]byte, frameSize)
	copy(frameData, frameHeader)
	// Fill rest with pattern to make it look like audio data
	for i := 4; i < frameSize; i++ {
		frameData[i] = byte(i % 256)
	}

	for i := range totalFrames {
		if _, err := tmpFile.Write(frameData); err != nil {
			t.Fatalf("Failed to write MP3 frame %d: %v", i, err)
		}
	}
	return mp3Path
}

// normalizeYAMLForComparison normalizes YAML content like apply.go does
func normalizeYAMLForComparison(t *testing.T, yamlContent string) string {
	t.Helper()

	// Parse and re-marshal to normalize like apply.go does
	var metadata chape.Metadata
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}
	normalizedYAMLData, err := yaml.Marshal(&metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}
	return string(normalizedYAMLData)
}

func TestIntegration(t *testing.T) {
	// Find all YAML test files
	testFiles, err := filepath.Glob("testdata/*.yaml")
	if err != nil {
		t.Fatalf("Failed to find test files: %v", err)
	}
	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			// Read original YAML
			originalYAML, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file %s: %v", testFile, err)
			}

			// Create dummy MP3 file (10 minutes)
			mp3File := createDummyMP3(t, 10*time.Minute)
			defer os.Remove(mp3File)

			// Apply YAML to MP3
			chape := chape.New(mp3File)
			originalReader := bytes.NewReader(originalYAML)

			err = chape.Apply(originalReader, true) // Use -y flag to skip prompts
			if err != nil {
				t.Fatalf("Failed to apply YAML to MP3: %v", err)
			}

			// Dump metadata back to YAML
			var dumpedYAML bytes.Buffer
			err = chape.Dump(&dumpedYAML)
			if err != nil {
				t.Fatalf("Failed to dump metadata from MP3: %v", err)
			}

			// Normalize both YAMLs for comparison like apply.go does
			originalNormalized := normalizeYAMLForComparison(t, string(originalYAML))
			dumpedNormalized := normalizeYAMLForComparison(t, dumpedYAML.String())

			// Compare normalized content
			if originalNormalized != dumpedNormalized {
				// Generate diff for better error reporting
				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(originalNormalized, dumpedNormalized, false)
				diffText := dmp.DiffPrettyText(diffs)

				t.Errorf("YAML content mismatch for %s:\n%s", testFile, diffText)
				t.Logf("\nOriginal normalized:\n%s\n\nDumped normalized:\n%s", originalNormalized, dumpedNormalized)
			}
		})
	}
}

func TestIntegrationWithArtwork(t *testing.T) {
	// Create a dummy MP3
	mp3File := createDummyMP3(t, 5*time.Minute)

	// Create test artwork file
	tmpDir := t.TempDir()
	artworkPath := filepath.Join(tmpDir, "artwork.jpg")
	artworkFile, err := os.Create(artworkPath)
	if err != nil {
		t.Fatalf("Failed to create artwork file: %v", err)
	}

	// Write minimal JPEG data
	jpegHeader := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, // JPEG header
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, // JFIF data
		0xFF, 0xD9, // JPEG end marker
	}
	if _, err := artworkFile.Write(jpegHeader); err != nil {
		t.Fatalf("Failed to write artwork data: %v", err)
	}
	artworkFile.Close()

	// Test YAML with artwork
	yamlWithArtwork := `title: "Artwork Test"
artist: "Test Artist"
artwork: "` + artworkPath + `"`

	chape := chape.New(mp3File)

	// Apply YAML
	err = chape.Apply(strings.NewReader(yamlWithArtwork), true)
	if err != nil {
		t.Fatalf("Failed to apply YAML with artwork: %v", err)
	}

	// First dump - should contain the original path because file exists
	var dumpedYAML1 bytes.Buffer
	err = chape.Dump(&dumpedYAML1)
	if err != nil {
		t.Fatalf("Failed to dump metadata: %v", err)
	}

	dumpedContent1 := dumpedYAML1.String()
	// Should contain the original artwork path since file still exists
	if !strings.Contains(dumpedContent1, "artwork: "+artworkPath) {
		t.Errorf("First dump should contain original artwork path")
		t.Errorf("Dumped content:\n%s", dumpedContent1)
	}

	// Remove the artwork file to test CHAPE_SOURCE recovery
	err = os.Remove(artworkPath)
	if err != nil {
		t.Fatalf("Failed to remove artwork file: %v", err)
	}

	// Second dump - should extract from embedded artwork since original file is missing
	var dumpedYAML2 bytes.Buffer
	err = chape.Dump(&dumpedYAML2)
	if err != nil {
		t.Fatalf("Failed to dump metadata after file removal: %v", err)
	}

	dumpedContent2 := dumpedYAML2.String()
	// After removing original file, should still contain the path (and file should be recreated)
	if !strings.Contains(dumpedContent2, "artwork: "+artworkPath) {
		t.Errorf("Second dump should still contain artwork path and recreate file")
		t.Errorf("Dumped content:\n%s", dumpedContent2)
	}

	// Verify the file was recreated
	if _, err := os.Stat(artworkPath); os.IsNotExist(err) {
		t.Errorf("Artwork file should have been recreated at %s", artworkPath)
	} else {
		// Verify the recreated file is identical to the original
		recreatedData, err := os.ReadFile(artworkPath)
		if err != nil {
			t.Fatalf("Failed to read recreated artwork file: %v", err)
		}

		if !bytes.Equal(jpegHeader, recreatedData) {
			t.Errorf("Recreated artwork file does not match original data")
			t.Errorf("Original size: %d bytes, Recreated size: %d bytes", len(jpegHeader), len(recreatedData))

			// Show hex dump for debugging if files are small
			if len(jpegHeader) <= 32 && len(recreatedData) <= 32 {
				t.Errorf("Original data: % x", jpegHeader)
				t.Errorf("Recreated data: % x", recreatedData)
			}
		}
	}
}

func TestIntegrationEmptyMP3(t *testing.T) {
	// Test with minimal MP3 file
	mp3File := createDummyMP3(t, 1*time.Second)
	defer os.Remove(mp3File)

	chape := chape.New(mp3File)

	// Dump empty MP3 metadata
	var initialDump bytes.Buffer
	err := chape.Dump(&initialDump)
	if err != nil {
		t.Fatalf("Failed to dump initial metadata: %v", err)
	}

	// Should contain at least the schema comment and minimal structure
	dumpContent := initialDump.String()
	if !strings.Contains(dumpContent, "yaml-language-server") {
		t.Error("Dumped YAML should contain schema comment")
	}
}
