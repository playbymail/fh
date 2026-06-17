package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/peterbourgon/ff/v4"
	"github.com/playbymail/fh/internal/engine"
)

// newImportCmd builds the "import" command: it ingests an fhc state directory
// (its export-json output plus per-species logs and the locations index) into
// an existing game's store as that game's world state. The game directory must
// already have been created with "fh create game".
func newImportCmd(rootFlags *ff.FlagSet) *ff.Command {
	importFlags := ff.NewFlagSet("import").SetParent(rootFlags)

	jsonFlags := ff.NewFlagSet("json").SetParent(importFlags)
	path := jsonFlags.StringLong("path", "", "Path to the game directory (created by 'fh create game')")
	dir := jsonFlags.StringLong("dir", ".", "Source directory holding galaxy.json, systems.json, species.NNN.json, sp0X.log")
	jsonCmd := &ff.Command{
		Name:      "json",
		Usage:     "fh import json --path GAMEDIR --dir SRC",
		ShortHelp: "Ingest an fhc JSON state directory into a game's store",
		Flags:     jsonFlags,
		Exec: func(ctx context.Context, args []string) error {
			if err := requireFlags(jsonFlags, "path", "dir"); err != nil {
				return err
			}
			st, gameID, err := openGame(ctx, *path)
			if err != nil {
				return err
			}
			defer st.Close()

			eng := engine.New(st, gameID)
			if err := eng.IngestDir(ctx, *dir); err != nil {
				return fmt.Errorf("import: %w", err)
			}
			fmt.Printf("fh: import: ingested %q into game %q\n", *dir, gameID)
			return nil
		},
	}

	return &ff.Command{
		Name:        "import",
		Usage:       "fh import <subcommand>",
		ShortHelp:   "Import game data",
		Flags:       importFlags,
		Subcommands: []*ff.Command{jsonCmd},
	}
}

// newCreateReportsCmd builds "create reports": render per-species turn reports
// from a game's store, byte-identical to fhc's sp%02d.rpt.t%d.
func newCreateReportsCmd(createFlags *ff.FlagSet) *ff.Command {
	fs := ff.NewFlagSet("reports").SetParent(createFlags)
	path := fs.StringLong("path", "", "Path to the game directory")
	out := fs.StringLong("out", ".", "Output directory for reports")
	species := fs.IntLong("species", 0, "Render only this species number (0 = all)")
	skipLog := fs.BoolLong("skip-log", "Omit the prepended prior-turn event log")

	return &ff.Command{
		Name:      "reports",
		Usage:     "fh create reports --path GAMEDIR [--out DIR] [--species N]",
		ShortHelp: "Create turn reports from a game's store",
		Flags:     fs,
		Exec: func(ctx context.Context, args []string) error {
			if err := requireFlags(fs, "path"); err != nil {
				return err
			}
			st, gameID, err := openGame(ctx, *path)
			if err != nil {
				return err
			}
			defer st.Close()

			w, err := st.LoadWorld(ctx, gameID)
			if err != nil {
				return fmt.Errorf("load world: %w", err)
			}
			eng := engine.New(st, gameID)
			opts := engine.ReportOptions{SkipLog: *skipLog}
			turn := w.Galaxy.TurnNumber

			for _, sp := range w.Species {
				if *species != 0 && sp.ID != *species {
					continue
				}
				body, err := eng.RenderReport(ctx, sp.ID, opts)
				if err != nil {
					return fmt.Errorf("render sp%02d: %w", sp.ID, err)
				}
				name := filepath.Join(*out, fmt.Sprintf("sp%02d.rpt.t%d", sp.ID, turn))
				if err := os.WriteFile(name, body, 0o644); err != nil {
					return fmt.Errorf("write %s: %w", name, err)
				}
				fmt.Printf("fh: create reports: wrote %s\n", name)
			}
			return nil
		},
	}
}
