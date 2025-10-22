// Package main implements the Far Horizons CLI.
package main

import (
	"os"

	"github.com/playbymail/fh/internal/cerrs"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "fh",
		Short: "Far Horizons CLI",
		Long:  `Far Horizons is a play-by-mail game engine rewritten in Go.`,
	}

	updateGoldenCmd.AddCommand(updateGoldenRngCmd)
	updateCmd.AddCommand(updateGoldenCmd)
	rootCmd.AddCommand(updateCmd)

	versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version information")
	rootCmd.AddCommand(versionCmd)

	// Command stubs for Far Horizons
	var combatCmd = &cobra.Command{
		Use:   "combat",
		Short: "Run combat commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(combatCmd)

	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new galaxy and home system templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(createCmd)

	var exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Convert binary .dat to json or s-expression",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(exportCmd)

	var finishCmd = &cobra.Command{
		Use:   "finish",
		Short: "Run end of turn logic",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(finishCmd)

	var importCmd = &cobra.Command{
		Use:   "import",
		Short: "Import game data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(importCmd)

	var inspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Inspect game state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(inspectCmd)

	var jumpCmd = &cobra.Command{
		Use:   "jump",
		Short: "Run jump commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(jumpCmd)

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List game elements",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(listCmd)

	var locationsCmd = &cobra.Command{
		Use:   "locations",
		Short: "Create locations data file and update economic efficiency in planets data file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(locationsCmd)

	var postArrivalCmd = &cobra.Command{
		Use:   "post-arrival",
		Short: "Run post-arrival commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(postArrivalCmd)

	var preDepartureCmd = &cobra.Command{
		Use:   "pre-departure",
		Short: "Run pre-departure commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(preDepartureCmd)

	var productionCmd = &cobra.Command{
		Use:   "production",
		Short: "Run production commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(productionCmd)

	var reportCmd = &cobra.Command{
		Use:   "report",
		Short: "Create end of turn reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(reportCmd)

	var scanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Display a species-specific scan for a location",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(scanCmd)

	var scanNearCmd = &cobra.Command{
		Use:   "scan-near",
		Short: "Display all ships and colonies near a location",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(scanNearCmd)

	var showCmd = &cobra.Command{
		Use:   "show",
		Short: "Show game information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(showCmd)

	var statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Display statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(statsCmd)

	var turnCmd = &cobra.Command{
		Use:   "turn",
		Short: "Display the current turn number",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(turnCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
