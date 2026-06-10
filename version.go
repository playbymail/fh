package main

import (
	"context"
	"fmt"

	"github.com/maloquacious/semver"
	"github.com/peterbourgon/ff/v4"
)

var (
	version = semver.Version{
		Major:      0,
		Minor:      17,
		Patch:      0,
		PreRelease: "alpha",
		Build:      semver.Commit(),
	}
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
				fmt.Println(version.String())
				return nil
			}
			fmt.Println(version.Core())
			return nil
		},
	}
}
