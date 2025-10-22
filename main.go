// Package main implements the Far Horizons CLI.
package main

import (
	"fmt"
	"os"

	"github.com/playbymail/fh/internal/cerrs"
	"github.com/playbymail/fh/internal/data/store"
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
	combatCmd.Flags().BoolP("summary", "s", false, "Set summary mode for battle reports")
	combatCmd.Flags().BoolP("prompt", "p", false, "Prompt GM before saving results")
	combatCmd.Flags().BoolP("test", "t", false, "Enable test mode")
	combatCmd.Flags().BoolP("verbose", "v", false, "Enable verbose mode")
	combatCmd.Flags().Bool("combat", false, "Run normal combat (default)")
	combatCmd.Flags().Bool("strike", false, "Run strike combat")
	rootCmd.AddCommand(combatCmd)

	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create new game elements",
	}

	var createGalaxyCmd = &cobra.Command{
		Use:   "galaxy",
		Short: "Create a new galaxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	createGalaxyCmd.Flags().Int("species", 0, "Number of species")
	createGalaxyCmd.Flags().Int("stars", 0, "Number of stars")
	createGalaxyCmd.Flags().Int("radius", 0, "Galactic radius in parsecs")
	createGalaxyCmd.Flags().Bool("suggest-values", false, "Suggest appropriate values")
	createGalaxyCmd.Flags().Bool("less-crowded", false, "Create a less crowded galaxy")
	createCmd.AddCommand(createGalaxyCmd)

	var createHomeSystemTemplatesCmd = &cobra.Command{
		Use:   "home-system-templates",
		Short: "Create home system templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	createCmd.AddCommand(createHomeSystemTemplatesCmd)

	var createSpeciesCmd = &cobra.Command{
		Use:   "species",
		Short: "Create species",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	createSpeciesCmd.Flags().String("config", "", "Configuration file")
	createSpeciesCmd.Flags().Int("radius", 10, "Radius")
	createSpeciesCmd.MarkFlagRequired("config")
	createCmd.AddCommand(createSpeciesCmd)

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

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize commands",
	}

	var initGameCmd = &cobra.Command{
		Use:   "game",
		Short: "Initialize the data store for a new game",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := cmd.Flags().GetString("path")
			force, _ := cmd.Flags().GetBool("force")
			storeType, _ := cmd.Flags().GetString("store-type")

			var st store.Store
			var err error
			switch storeType {
			case "json":
				st, err = store.NewJSONStore(path, force)
			case "sql":
				st, err = store.NewSQLiteStore(path, force)
			default:
				return fmt.Errorf("invalid store type: %s", storeType)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to initialize store: %v\n", err)
				os.Exit(1)
			}
			defer st.Close()

			return nil
		},
	}
	initGameCmd.Flags().String("path", "", "Path to the data store")
	initGameCmd.Flags().String("game-id", "", "Game ID")
	initGameCmd.Flags().Bool("force", false, "Force overwriting existing store")
	initGameCmd.Flags().String("store-type", "json", "Type of store (json or sql)")
	initGameCmd.MarkFlagRequired("path")
	initGameCmd.MarkFlagRequired("game-id")
	initCmd.AddCommand(initGameCmd)

	rootCmd.AddCommand(initCmd)

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
	}

	var showDNumSpeciesCmd = &cobra.Command{
		Use:   "d-num-species",
		Short: "Show maximum number of species",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showDNumSpeciesCmd)

	var showNumPlanetsCmd = &cobra.Command{
		Use:   "num-planets",
		Short: "Show number of planets in cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showNumPlanetsCmd)

	var showNumSpeciesCmd = &cobra.Command{
		Use:   "num-species",
		Short: "Show number of species in cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showNumSpeciesCmd)

	var showNumStarsCmd = &cobra.Command{
		Use:   "num-stars",
		Short: "Show number of stars in cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showNumStarsCmd)

	var showNumNaturalWormholesCmd = &cobra.Command{
		Use:   "num-natural-wormholes",
		Short: "Show number of natural wormholes in cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showNumNaturalWormholesCmd)

	var showRadiusCmd = &cobra.Command{
		Use:   "radius",
		Short: "Show radius of cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showRadiusCmd)

	var showTurnNumberCmd = &cobra.Command{
		Use:   "turn-number",
		Short: "Show current turn number",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showTurnNumberCmd)

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
