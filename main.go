// Package main implements the Far Horizons CLI.
package main

import (
	"fmt"
	"log"
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

	// Command stubs for Far Horizons

	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create new game elements",
	}
	rootCmd.AddCommand(createCmd)

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

	var createLocationsCmd = &cobra.Command{
		Use:   "locations",
		Short: "Create locations data file and update economic efficiency in planets data file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	createCmd.AddCommand(createLocationsCmd)

	var createReportsCmd = &cobra.Command{
		Use:   "reports",
		Short: "Create turn reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	createCmd.AddCommand(createReportsCmd)

	var createSpeciesCmd = &cobra.Command{
		Use:   "species",
		Short: "Create species",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	createSpeciesCmd.Flags().String("config", "", "Configuration file")
	createSpeciesCmd.Flags().Int("radius", 10, "Radius")
	if err := createSpeciesCmd.MarkFlagRequired("config"); err != nil {
		log.Fatalf("create species --config: %v\n", err)
	}
	createCmd.AddCommand(createSpeciesCmd)

	var exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Export game data to JSON for testing",
	}
	rootCmd.AddCommand(exportCmd)

	var exportSnapshotCmd = &cobra.Command{
		Use:   "snapshot",
		Short: "Export a game snapshot to JSON",
		Run: func(cmd *cobra.Command, args []string) {
			storePath, _ := cmd.Flags().GetString("store")
			gameID, _ := cmd.Flags().GetString("game")
			turnNum, _ := cmd.Flags().GetInt("turn")
			outputPath, _ := cmd.Flags().GetString("output")
			_, _, _ = gameID, turnNum, outputPath

			st, err := store.OpenSQLiteStore(storePath)
			if err != nil {
				log.Fatalf("failed to open store: %v\n", err)
			}
			defer st.Close()

			fmt.Printf("not implemented")
			os.Exit(1)
		},
	}
	exportSnapshotCmd.Flags().String("store", "", "Path to SQLite store")
	exportSnapshotCmd.Flags().String("game", "", "Game ID")
	exportSnapshotCmd.Flags().Int("turn", 0, "Turn number")
	exportSnapshotCmd.Flags().String("output", "", "Output directory for JSON files")
	if err := exportSnapshotCmd.MarkFlagRequired("store"); err != nil {
		log.Fatalf("config species --store: %v\n", err)
	}
	if err := exportSnapshotCmd.MarkFlagRequired("game"); err != nil {
		log.Fatalf("config species --game: %v\n", err)
	}
	if err := exportSnapshotCmd.MarkFlagRequired("turn"); err != nil {
		log.Fatalf("config species --turn: %v\n", err)
	}
	if err := exportSnapshotCmd.MarkFlagRequired("output"); err != nil {
		log.Fatalf("config species --output: %v\n", err)
	}
	exportCmd.AddCommand(exportSnapshotCmd)

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
	rootCmd.AddCommand(initCmd)

	var initGameCmd = &cobra.Command{
		Use:   "game",
		Short: "Initialize the data store for a new game",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, _ := cmd.Flags().GetString("path")
			force, _ := cmd.Flags().GetBool("force")

			st, err := store.NewSQLiteStore(path, force)
			if err != nil {
				log.Fatalf("failed to initialize store: %v\n", err)
			}
			defer st.Close()

			return nil
		},
	}
	initGameCmd.Flags().String("path", ".", "Path to the data store")
	initGameCmd.Flags().String("id", "", "Game ID")
	initGameCmd.Flags().Bool("force", false, "Force overwriting existing store")
	if err := initGameCmd.MarkFlagRequired("id"); err != nil {
		log.Fatalf("init game --id")
	}
	initCmd.AddCommand(initGameCmd)

	var inspectCmd = &cobra.Command{
		Use:   "inspect",
		Short: "Inspect game state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(inspectCmd)

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List game elements",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(listCmd)

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run a game phase",
	}
	rootCmd.AddCommand(runCmd)

	var runCombatCmd = &cobra.Command{
		Use:   "combat",
		Short: "Run combat phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	runCombatCmd.Flags().BoolP("summary", "s", false, "Set summary mode for battle reports")
	runCombatCmd.Flags().BoolP("prompt", "p", false, "Prompt GM before saving results")
	runCombatCmd.Flags().BoolP("test", "t", false, "Enable test mode")
	runCombatCmd.Flags().BoolP("verbose", "v", false, "Enable verbose mode")
	runCombatCmd.Flags().Bool("combat", false, "Run normal combat (default)")
	runCombatCmd.Flags().Bool("strike", false, "Run strike combat")
	runCmd.AddCommand(runCombatCmd)

	var runFinishCmd = &cobra.Command{
		Use:   "finish",
		Short: "Run end of turn phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	runCmd.AddCommand(runFinishCmd)

	var runJumpCmd = &cobra.Command{
		Use:   "jump",
		Short: "Run jump phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	runCmd.AddCommand(runJumpCmd)

	var runPostArrivalCmd = &cobra.Command{
		Use:   "post-arrival",
		Short: "Run post-arrival phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	runCmd.AddCommand(runPostArrivalCmd)

	var runPreDepartureCmd = &cobra.Command{
		Use:   "pre-departure",
		Short: "Run pre-departure phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	runCmd.AddCommand(runPreDepartureCmd)

	var runProductionCmd = &cobra.Command{
		Use:   "production",
		Short: "Run production phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	runCmd.AddCommand(runProductionCmd)

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
	rootCmd.AddCommand(showCmd)

	var showDNumSpeciesCmd = &cobra.Command{
		Use:   "d-num-species",
		Short: "Show maximum number of species",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showDNumSpeciesCmd)

	var showNumNaturalWormholesCmd = &cobra.Command{
		Use:   "num-natural-wormholes",
		Short: "Show number of natural wormholes in cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showNumNaturalWormholesCmd)

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

	var showRadiusCmd = &cobra.Command{
		Use:   "radius",
		Short: "Show radius of cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	showCmd.AddCommand(showRadiusCmd)

	var showStatsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(showStatsCmd)

	var showTurnCmd = &cobra.Command{
		Use:   "turn",
		Short: "Show the current turn number",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cerrs.ErrNotImplemented
		},
	}
	rootCmd.AddCommand(showTurnCmd)

	updateGoldenCmd.AddCommand(updateGoldenRngCmd)
	updateCmd.AddCommand(updateGoldenCmd)
	rootCmd.AddCommand(updateCmd)

	versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version information")
	rootCmd.AddCommand(versionCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
