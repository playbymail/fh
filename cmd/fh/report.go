package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/peterbourgon/ff/v4"
	"github.com/playbymail/fh/internal/data/store"
	"github.com/playbymail/fh/internal/engine"
)

// newImportCmd builds the "import" command: it ingests an fhc state directory
// (its export-json output plus per-species logs and the locations index) into a
// SQLite store as a game's world state.
func newImportCmd(rootFlags *ff.FlagSet) *ff.Command {
	importFlags := ff.NewFlagSet("import").SetParent(rootFlags)

	jsonFlags := ff.NewFlagSet("json").SetParent(importFlags)
	storePath := jsonFlags.StringLong("store", "", "Path to SQLite store")
	gameID := jsonFlags.StringLong("game", "", "Game ID")
	dir := jsonFlags.StringLong("dir", ".", "Directory holding galaxy.json, systems.json, species.NNN.json, sp0X.log")
	force := jsonFlags.BoolLong("force", "Overwrite an existing store")
	jsonCmd := &ff.Command{
		Name:      "json",
		Usage:     "fh import json --store PATH --game ID --dir DIR",
		ShortHelp: "Ingest an fhc JSON state directory into a SQLite store",
		Flags:     jsonFlags,
		Exec: func(ctx context.Context, args []string) error {
			if err := requireFlags(jsonFlags, "store", "game"); err != nil {
				return err
			}
			st, err := openOrCreateStore(*storePath, *force)
			if err != nil {
				return err
			}
			defer st.Close()

			if err := st.CreateGame(ctx, *gameID, *gameID); err != nil {
				// A game row may already exist on re-import; ingest replaces
				// the domain rows regardless, so a duplicate game is benign.
				_ = err
			}
			eng := engine.New(st, *gameID)
			if err := eng.IngestDir(ctx, *dir); err != nil {
				return fmt.Errorf("import: %w", err)
			}
			fmt.Printf("fh: import: ingested %q into game %q\n", *dir, *gameID)
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
// from a SQLite store, byte-identical to fhc's sp%02d.rpt.t%d.
func newCreateReportsCmd(createFlags *ff.FlagSet) *ff.Command {
	fs := ff.NewFlagSet("reports").SetParent(createFlags)
	storePath := fs.StringLong("store", "", "Path to SQLite store")
	gameID := fs.StringLong("game", "", "Game ID")
	out := fs.StringLong("out", ".", "Output directory for reports")
	species := fs.IntLong("species", 0, "Render only this species number (0 = all)")
	skipLog := fs.BoolLong("skip-log", "Omit the prepended prior-turn event log")

	return &ff.Command{
		Name:      "reports",
		Usage:     "fh create reports --store PATH --game ID [--out DIR] [--species N]",
		ShortHelp: "Create turn reports from a SQLite store",
		Flags:     fs,
		Exec: func(ctx context.Context, args []string) error {
			if err := requireFlags(fs, "store", "game"); err != nil {
				return err
			}
			st, err := store.OpenSQLiteStore(*storePath)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer st.Close()

			w, err := st.LoadWorld(ctx, *gameID)
			if err != nil {
				return fmt.Errorf("load world: %w", err)
			}
			eng := engine.New(st, *gameID)
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

// openOrCreateStore opens an existing store or creates a new one.
func openOrCreateStore(path string, force bool) (*store.SQLiteStore, error) {
	if _, err := os.Stat(path); err == nil && !force {
		st, err := store.OpenSQLiteStore(path)
		if err != nil {
			return nil, fmt.Errorf("open store: %w", err)
		}
		return st, nil
	}
	st, err := store.NewSQLiteStore(path, force)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}
	return st, nil
}
