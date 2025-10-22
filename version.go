package main

import (
	"fmt"

	"github.com/maloquacious/semver"
	"github.com/spf13/cobra"
)

var (
	version = semver.Version{
		Major:      0,
		Minor:      10,
		Patch:      0,
		PreRelease: "alpha",
		Build:      semver.Commit(),
	}
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of fh",
	Long:  `All software has versions. This is fh's`,
	Run: func(cmd *cobra.Command, args []string) {
		if verbose, err := cmd.Flags().GetBool("verbose"); err == nil && verbose {
			fmt.Println(version.String())
			return
		}
		fmt.Println(version.Core())
	},
}
