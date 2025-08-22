package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/Songmu/chapel"
)

const cmdName = "chapel"

// Run the chapel
func Run(ctx context.Context, argv []string, outStream, errStream io.Writer) error {
	log.SetOutput(errStream)
	log.SetPrefix(fmt.Sprintf("[%s] ", cmdName))
	nameAndVer := fmt.Sprintf("%s (v%s ref:%s)", cmdName, chapel.Version, chapel.Revision)
	fs := flag.NewFlagSet(
		fmt.Sprintf("%s (v%s rev:%s)", cmdName, chapel.Version, chapel.Revision), flag.ContinueOnError)
	fs.SetOutput(errStream)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage of %s:\n", nameAndVer)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\nCommands:\n")
		formatCommands(fs.Output())
	}
	ver := fs.Bool("version", false, "display version")
	yes := fs.Bool("y", false, "skip confirmation prompts")
	var artworkPath string
	fs.StringVar(&artworkPath, "artwork", "", "path or URL for artwork (extracts from MP3 if file doesn't exist)")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if *ver {
		return printVersion(outStream)
	}
	argv = fs.Args()
	if len(argv) < 1 {
		return fmt.Errorf("no args specified")
	}
	if strings.HasSuffix(argv[0], ".mp3") {
		return chapel.New(argv[0], artworkPath).Edit(*yes)
	}
	if cmd, ok := cmder.dispatch[argv[0]]; ok {
		return cmd.Run(ctx, argv[1:], outStream, errStream)
	}
	return fmt.Errorf("unknown command %q", argv[0])
}

func printVersion(out io.Writer) error {
	_, err := fmt.Fprintf(out, "%s v%s (rev:%s)\n", cmdName, chapel.Version, chapel.Revision)
	return err
}
