package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
)

type commander struct {
	cmdNames             []string
	dispatch             map[string]*command
	maxSubcommandNameLen int
}

func (co *commander) register(rnrs ...*command) {
	for _, r := range rnrs {
		n := r.Name
		if co.dispatch == nil {
			co.dispatch = map[string]*command{}
		}
		if _, ok := co.dispatch[n]; ok {
			log.Fatalf("subcommand %q already registered", n)
		}
		co.dispatch[n] = r
		co.cmdNames = append(co.cmdNames, n)
		if co.maxSubcommandNameLen < len(n) {
			co.maxSubcommandNameLen = len(n)
		}
	}
}

var cmder = &commander{}

func init() {
	cmder.register(
		cmdApply,
		cmdDump,
	)
}

func formatCommands(out io.Writer) {
	format := fmt.Sprintf("  %%-%ds  %%s\n", cmder.maxSubcommandNameLen)
	for _, n := range cmder.cmdNames {
		r := cmder.dispatch[n]
		fmt.Fprintf(out, format, r.Name, r.Description)
	}
}

type command struct {
	Name        string
	Description string
	Run         func(context.Context, []string, io.Writer, io.Writer) error
}
