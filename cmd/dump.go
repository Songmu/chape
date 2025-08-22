package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Songmu/chapel"
)

var cmdDump = &command{
	Name: "dump",
	Run: func(ctx context.Context, argv []string, outStream, errStream io.Writer) error {
		fs := flag.NewFlagSet("chapel dump", flag.ContinueOnError)
		fs.SetOutput(errStream)
		var artworkPath string
		fs.StringVar(&artworkPath, "artwork", "", "path or URL for artwork (extracts from MP3 if file doesn't exist)")
		if err := fs.Parse(argv); err != nil {
			return err
		}
		argv = fs.Args()
		if len(argv) < 1 {
			return fmt.Errorf("no args specified")
		}
		if strings.HasSuffix(argv[0], ".mp3") {
			return chapel.New(argv[0], artworkPath).Dump(outStream)
		}
		return fmt.Errorf("unknown file type %q", argv[0])
	},
}
