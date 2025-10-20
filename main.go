// Package main implements the Far Horizons CLI.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "fh",
		Short: "Far Horizons CLI",
		Long:  `Far Horizons is a play-by-mail game engine rewritten in Go.`,
	}

	versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version information")
	rootCmd.AddCommand(versionCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
