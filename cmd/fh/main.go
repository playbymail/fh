// Package main implements the Far Horizons CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
	"github.com/playbymail/fh/internal/cerrs"
	"github.com/playbymail/fh/internal/data/store"
)

func main() {
	rootFlags := ff.NewFlagSet("fh")

	rootCmd := &ff.Command{
		Name:        "fh",
		Usage:       "fh <subcommand> [flags]",
		ShortHelp:   "Far Horizons CLI",
		LongHelp:    "Far Horizons is a play-by-mail game engine rewritten in Go.",
		Flags:       rootFlags,
		Subcommands: buildSubcommands(rootFlags),
	}

	if err := rootCmd.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		if errors.Is(err, ff.ErrHelp) || errors.Is(err, ff.ErrNoExec) {
			fmt.Fprintf(os.Stderr, "%s\n", ffhelp.Command(rootCmd))
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// notImpl is the Exec function for unimplemented command stubs.
func notImpl(ctx context.Context, args []string) error {
	return cerrs.ErrNotImplemented
}

// stub builds an unimplemented command whose flag set is parented to parent.
func stub(parent *ff.FlagSet, name, usage, help string) *ff.Command {
	return &ff.Command{
		Name:      name,
		Usage:     usage,
		ShortHelp: help,
		Flags:     ff.NewFlagSet(name).SetParent(parent),
		Exec:      notImpl,
	}
}

// requireFlags returns an error if any of the named flags were not set.
func requireFlags(fs *ff.FlagSet, names ...string) error {
	for _, name := range names {
		f, ok := fs.GetFlag(name)
		if !ok || !f.IsSet() {
			return fmt.Errorf("--%s is required", name)
		}
	}
	return nil
}

// buildSubcommands assembles the full command tree under the root flag set.
func buildSubcommands(rootFlags *ff.FlagSet) []*ff.Command {
	return []*ff.Command{
		newCreateCmd(rootFlags),
		newExportCmd(rootFlags),
		newImportCmd(rootFlags),
		stub(rootFlags, "inspect", "fh inspect", "Inspect game state"),
		stub(rootFlags, "list", "fh list", "List game elements"),
		newRunCmd(rootFlags),
		stub(rootFlags, "scan", "fh scan", "Display a species-specific scan for a location"),
		stub(rootFlags, "scan-near", "fh scan-near", "Display all ships and colonies near a location"),
		newShowCmd(rootFlags),
		stub(rootFlags, "stats", "fh stats", "Show statistics"),
		stub(rootFlags, "turn", "fh turn", "Show the current turn number"),
		newUpdateCmd(rootFlags),
		newVersionCmd(rootFlags),
	}
}

// newCreateCmd builds the "create" command and its subcommands.
func newCreateCmd(rootFlags *ff.FlagSet) *ff.Command {
	createFlags := ff.NewFlagSet("create").SetParent(rootFlags)

	galaxyFlags := ff.NewFlagSet("galaxy").SetParent(createFlags)
	_ = galaxyFlags.IntLong("species", 0, "Number of species")
	_ = galaxyFlags.IntLong("stars", 0, "Number of stars")
	_ = galaxyFlags.IntLong("radius", 0, "Galactic radius in parsecs")
	_ = galaxyFlags.BoolLong("suggest-values", "Suggest appropriate values")
	_ = galaxyFlags.BoolLong("less-crowded", "Create a less crowded galaxy")
	galaxyCmd := &ff.Command{
		Name:      "galaxy",
		Usage:     "fh create galaxy [flags]",
		ShortHelp: "Create a new galaxy",
		Flags:     galaxyFlags,
		Exec:      notImpl,
	}

	speciesFlags := ff.NewFlagSet("species").SetParent(createFlags)
	_ = speciesFlags.StringLong("config", "", "Configuration file")
	_ = speciesFlags.IntLong("radius", 10, "Radius")
	speciesCmd := &ff.Command{
		Name:      "species",
		Usage:     "fh create species --config FILE [flags]",
		ShortHelp: "Create species",
		Flags:     speciesFlags,
		Exec: func(ctx context.Context, args []string) error {
			if err := requireFlags(speciesFlags, "config"); err != nil {
				return err
			}
			return cerrs.ErrNotImplemented
		},
	}

	return &ff.Command{
		Name:      "create",
		Usage:     "fh create <subcommand>",
		ShortHelp: "Create new game elements",
		Flags:     createFlags,
		Subcommands: []*ff.Command{
			newCreateGameCmd(createFlags),
			galaxyCmd,
			stub(createFlags, "home-system-templates", "fh create home-system-templates", "Create home system templates"),
			stub(createFlags, "locations", "fh create locations", "Create locations data file and update economic efficiency in planets data file"),
			newCreateReportsCmd(createFlags),
			speciesCmd,
		},
	}
}

// newExportCmd builds the "export" command and its subcommands.
func newExportCmd(rootFlags *ff.FlagSet) *ff.Command {
	exportFlags := ff.NewFlagSet("export").SetParent(rootFlags)

	snapshotFlags := ff.NewFlagSet("snapshot").SetParent(exportFlags)
	path := snapshotFlags.StringLong("path", "", "Path to the game directory")
	_ = snapshotFlags.IntLong("turn", 0, "Turn number")
	_ = snapshotFlags.StringLong("output", "", "Output directory for JSON files")
	snapshotCmd := &ff.Command{
		Name:      "snapshot",
		Usage:     "fh export snapshot --path GAMEDIR --turn N --output DIR",
		ShortHelp: "Export a game snapshot to JSON",
		Flags:     snapshotFlags,
		Exec: func(ctx context.Context, args []string) error {
			if err := requireFlags(snapshotFlags, "path", "turn", "output"); err != nil {
				return err
			}

			st, err := store.Open(ctx, *path)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer st.Close()

			return cerrs.ErrNotImplemented
		},
	}

	return &ff.Command{
		Name:        "export",
		Usage:       "fh export <subcommand>",
		ShortHelp:   "Export game data to JSON for testing",
		Flags:       exportFlags,
		Subcommands: []*ff.Command{snapshotCmd},
	}
}

// newRunCmd builds the "run" command and its subcommands.
func newRunCmd(rootFlags *ff.FlagSet) *ff.Command {
	runFlags := ff.NewFlagSet("run").SetParent(rootFlags)

	combatFlags := ff.NewFlagSet("combat").SetParent(runFlags)
	_ = combatFlags.BoolShort('s', "Set summary mode for battle reports")
	_ = combatFlags.BoolShort('p', "Prompt GM before saving results")
	_ = combatFlags.BoolShort('t', "Enable test mode")
	_ = combatFlags.BoolShort('v', "Enable verbose mode")
	_ = combatFlags.BoolLong("combat", "Run normal combat (default)")
	_ = combatFlags.BoolLong("strike", "Run strike combat")
	combatCmd := &ff.Command{
		Name:      "combat",
		Usage:     "fh run combat [flags]",
		ShortHelp: "Run combat phase",
		Flags:     combatFlags,
		Exec:      notImpl,
	}

	return &ff.Command{
		Name:      "run",
		Usage:     "fh run <subcommand>",
		ShortHelp: "Run a game phase",
		Flags:     runFlags,
		Subcommands: []*ff.Command{
			combatCmd,
			stub(runFlags, "finish", "fh run finish", "Run end of turn phase"),
			stub(runFlags, "jump", "fh run jump", "Run jump phase"),
			stub(runFlags, "post-arrival", "fh run post-arrival", "Run post-arrival phase"),
			stub(runFlags, "pre-departure", "fh run pre-departure", "Run pre-departure phase"),
			stub(runFlags, "production", "fh run production", "Run production phase"),
		},
	}
}

// newShowCmd builds the "show" command and its subcommands.
func newShowCmd(rootFlags *ff.FlagSet) *ff.Command {
	showFlags := ff.NewFlagSet("show").SetParent(rootFlags)

	return &ff.Command{
		Name:      "show",
		Usage:     "fh show <subcommand>",
		ShortHelp: "Show game information",
		Flags:     showFlags,
		Subcommands: []*ff.Command{
			stub(showFlags, "d-num-species", "fh show d-num-species", "Show maximum number of species"),
			stub(showFlags, "num-natural-wormholes", "fh show num-natural-wormholes", "Show number of natural wormholes in cluster"),
			stub(showFlags, "num-planets", "fh show num-planets", "Show number of planets in cluster"),
			stub(showFlags, "num-species", "fh show num-species", "Show number of species in cluster"),
			stub(showFlags, "num-stars", "fh show num-stars", "Show number of stars in cluster"),
			stub(showFlags, "radius", "fh show radius", "Show radius of cluster"),
		},
	}
}
