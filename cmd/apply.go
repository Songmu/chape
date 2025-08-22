package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Songmu/chapel"
)

var cmdApply = &command{
	Name: "apply",
	Run: func(ctx context.Context, argv []string, outStream, errStream io.Writer) error {
		fs := flag.NewFlagSet("chapel apply", flag.ContinueOnError)
		fs.SetOutput(errStream)
		yes := fs.Bool("y", false, "Skip confirmation prompts")
		if err := fs.Parse(argv); err != nil {
			return err
		}
		argv = fs.Args()
		if len(argv) < 1 {
			return fmt.Errorf("no args specified")
		}
		if strings.HasSuffix(argv[0], ".mp3") {
			return chapel.New(argv[0]).Apply(os.Stdin, *yes)
		}
		return fmt.Errorf("unknown file type %q", argv[0])
	},
}
