package chape

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

type Chape struct {
	audio   string
	artwork string
}

func New(audio string, artwork ...string) *Chape {
	c := &Chape{
		audio: audio,
	}
	if len(artwork) > 0 {
		c.artwork = artwork[0]
	}
	return c
}

func (c *Chape) Edit(yes bool) error {
	// Create a temporary YAML file with current metadata
	tempFile, err := os.CreateTemp("", "chape-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Dump current metadata to temp file with artwork handling
	err = c.Dump(tempFile)
	if err != nil {
		return fmt.Errorf("failed to dump metadata: %w", err)
	}

	// Close file before opening with editor
	tempFile.Close()

	// Get editor command
	editor := getEditor()

	// Build command - use shell only if editor contains whitespace
	var cmd *exec.Cmd
	if strings.ContainsFunc(editor, unicode.IsSpace) {
		// Editor has arguments (e.g., "code --wait"), use shell
		if runtime.GOOS == "windows" {
			// Windows: use cmd /c with proper quoting
			quotedPath := strconv.Quote(tempFile.Name())
			cmd = exec.Command("cmd", "/c", editor+" "+quotedPath)
		} else {
			// Unix-like: use sh -c with proper shell escaping
			// Use single quotes for safety unless the path contains single quotes
			escapedPath := tempFile.Name()
			if strings.Contains(escapedPath, "'") {
				escapedPath = strconv.Quote(escapedPath)
			} else {
				escapedPath = "'" + escapedPath + "'"
			}
			cmd = exec.Command("sh", "-c", editor+" "+escapedPath)
		}
	} else {
		// Simple editor command, execute directly
		cmd = exec.Command(editor, tempFile.Name())
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("editor command failed: %w", err)
	}

	// Read edited content back
	editedFile, err := os.Open(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}
	defer editedFile.Close()

	// Apply the edited metadata
	err = c.Apply(editedFile, yes)
	if err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	return nil
}

// getEditor returns the editor command to use
func getEditor() string {
	// Check environment variables in order of preference
	if editor := os.Getenv("CHAPE_EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}

	// Default editors by platform
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}
