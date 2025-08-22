# Chapel

[![Test Status](https://github.com/Songmu/chapel/actions/workflows/test.yaml/badge.svg?branch=main)][actions]
[![Coverage Status](https://codecov.io/gh/Songmu/chapel/branch/main/graph/badge.svg)][codecov]
[![MIT License](https://img.shields.io/github/license/Songmu/chapel)][license]
[![PkgGoDev](https://pkg.go.dev/badge/github.com/Songmu/chapel)][PkgGoDev]

[actions]: https://github.com/Songmu/chapel/actions?workflow=test
[codecov]: https://codecov.io/gh/Songmu/chapel
[license]: https://github.com/Songmu/chapel/blob/main/LICENSE
[PkgGoDev]: https://pkg.go.dev/github.com/Songmu/chapel

Chapel is a powerful command-line tool for editing MP3 metadata using YAML format. It provides an intuitive way to manage ID3v2 tags, chapters, and artwork in MP3 files.

## Features
- **YAML-based metadata editing**: Edit MP3 metadata using familiar YAML syntax
- **Chapter support**: Manage chapter markers for audiobooks and podcasts
- **Artwork management**: Handle embedded artwork with support for local files, URLs, and data URIs
- **Interactive editing**: Built-in editor support for seamless workflow

## Usage

### Basic Commands

**Interactive editing with your `EDITOR`:**
```bash
chapel audio.mp3
```

**Dump metadata to YAML:**
```bash
chapel dump audio.mp3 > metadata.yaml
```

**Apply YAML metadata to MP3:**
```bash
chapel apply audio.mp3 < metadata.yaml
```

### Options
- `-y`: Skip confirmation prompts (useful for automation)
- `--artwork <path>`: Override artwork with local file path or HTTP/HTTPS URL

### Examples

**Edit metadata interactively:**
```bash
chapel my-audiobook.mp3
```

**Edit metadata with custom artwork:**
```bash
chapel --artwork cover.jpg my-audiobook.mp3
```

**Batch processing with automation:**
```bash
chapel apply -y audio.mp3 < batch-metadata.yaml
```

## YAML Format

Chapel uses a structured YAML format for metadata:

```yaml
title: "My Audiobook"
artist: "Author Name"
album: "Book Series"
albumArtist: "Publisher"
date: "2024"
track: "1/12"
disc: "1/2"
genre: "Audiobook"
comment: "A great book"
composer: "Author Name"
publisher: "Publisher Name"
bpm: 120
artwork: "cover.jpg"
lyrics: |
  Chapter content here...
chapters:
- 0:00 Introduction
- 5:30 Chapter 1: Getting Started
- 15:45 Chapter 2: Advanced Topics
- 28:20 Chapter 3: Conclusion
```

### Metadata Fields

| Field | Description | ID3v2 Tag |
|-------|-------------|-----------|
| `title` | Song/track title (podcast: episode title) | TIT2 |
| `artist` | Primary artist (podcast: host name) | TPE1 |
| `album` | Album title (podcast: show name) | TALB |
| `albumArtist` | Album artist (podcast: network/publisher) | TPE2 |
| `date` | Recording date | TDRC |
| `track` | Track number (podcast: episode number) | TRCK |
| `disc` | Disc number (podcast: season number) | TPOS |
| `genre` | Music genre (podcast: "Podcast" or category) | TCON |
| `comment` | Comments (podcast: episode description) | COMM |
| `composer` | Composer (podcast: producer) | TCOM |
| `publisher` | Publisher (podcast: network/platform) | TPUB |
| `bpm` | Beats per minute | TBPM |
| `artwork` | Artwork (file path, URL, or data URI) | APIC |
| `lyrics` | Lyrics text (podcast: transcript) | USLT |
| `chapters` | Chapter markers with timestamps | CHAP |

### Date Format

The `date` field supports ISO 8601 format with varying precision:
- `2024` (year only)
- `2024-03` (year-month)
- `2024-03-15` (year-month-day)
- `2024-03-15T14:30` (with time)
- `2024-03-15T14:30:45` (with seconds)

### Chapter Format

Chapters use WebVTT-style time format with titles:
```yaml
chapters:
- 0:00 Introduction
- 1:05:30 Long chapter (over 1 hour)
- 1:23.500 Chapter with milliseconds
```

### Artwork Sources

Chapel supports multiple artwork sources:

1. **Local file paths**: `artwork: "cover.jpg"`
2. **HTTP/HTTPS URLs**: `artwork: "https://example.com/cover.jpg"`
3. **Data URIs**: `artwork: "data:image/jpeg;base64,/9j/4AAQ..."`

When you specify an artwork path that doesn't exist, Chapel will:
1. Check if the MP3 has embedded artwork
2. Automatically extract and save it to the specified path
3. Update the metadata to reference the new file

## Advanced Features

### Interactive Editor Integration

Chapel integrates with your preferred text editor for seamless metadata editing:

```bash
# Uses $EDITOR environment variable
export EDITOR=nano
chapel audio.mp3
```

### Automation and Scripting

Use the `-y` flag for non-interactive batch processing:

```bash
#!/bin/bash
for file in *.mp3; do
    echo "title: $(basename "$file" .mp3)" | chapel apply -y "$file"
done
```

### Artwork Management

Extract artwork from MP3 files:
```bash
# This will extract artwork to cover.jpg if it doesn't exist
echo 'artwork: cover.jpg' | chapel apply audio.mp3
```

Override artwork source:
```bash
chapel --artwork https://example.com/new-cover.jpg audio.mp3
```

## Installation

```console
# Install the latest version. (Install it into ./bin/ by default).
% curl -sfL https://raw.githubusercontent.com/Songmu/chapel/main/install.sh | sh -s

# Specify installation directory ($(go env GOPATH)/bin/) and version.
% curl -sfL https://raw.githubusercontent.com/Songmu/chapel/main/install.sh | sh -s -- -b $(go env GOPATH)/bin [vX.Y.Z]

# In alpine linux (as it does not come with curl by default)
% wget -O - -q https://raw.githubusercontent.com/Songmu/chapel/main/install.sh | sh -s [vX.Y.Z]

# go install
% go install github.com/Songmu/chapel/cmd/chapel@latest
```

## Author

[Songmu](https://github.com/Songmu)
