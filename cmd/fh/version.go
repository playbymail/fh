package main

import (
	"context"
	"fmt"

	"github.com/peterbourgon/ff/v4"
	"github.com/playbymail/fh"
)

// newVersionCmd returns the "version" command.
func newVersionCmd(parent *ff.FlagSet) *ff.Command {
	fs := ff.NewFlagSet("version").SetParent(parent)
	verbose := fs.BoolShort('v', "Show detailed version information")

	return &ff.Command{
		Name:      "version",
		Usage:     "fh version [-v]",
		ShortHelp: "Print the version number of fh",
		Flags:     fs,
		Exec: func(ctx context.Context, args []string) error {
			if *verbose {
				fmt.Println(fh.Version().String())
				return nil
			}
			fmt.Println(fh.Version().Core())
			return nil
		},
	}
}
