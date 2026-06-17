package main

import (
	"context"
	"fmt"

	"github.com/peterbourgon/ff/v4"
	"github.com/playbymail/fh/internal/data/store"
)

// newCreateGameCmd builds "create game": create a new game directory, write its
// far-horizons.json config, and initialize the chosen store backend. The store
// type defaults to binary; gamemasters pick binary, json, or sqlite.
func newCreateGameCmd(parent *ff.FlagSet) *ff.Command {
	fs := ff.NewFlagSet("game").SetParent(parent)
	path := fs.StringLong("path", "", "Path to the game directory to create")
	id := fs.StringLong("id", "", "Game ID")
	name := fs.StringLong("name", "", "Human-readable game name (defaults to ID)")
	storeType := fs.StringLong("store-type", string(store.StoreBinary), "Store backend: binary, json, or sqlite")
	force := fs.BoolLong("force", "Overwrite an existing game in the directory")

	return &ff.Command{
		Name:      "game",
		Usage:     "fh create game --path DIR --id ID [--store-type binary|json|sqlite]",
		ShortHelp: "Create a new game directory and data store",
		Flags:     fs,
		Exec: func(ctx context.Context, args []string) error {
			if err := requireFlags(fs, "path", "id"); err != nil {
				return err
			}
			st, err := store.Create(ctx, *path, store.CreateOptions{
				ID:    *id,
				Name:  *name,
				Type:  store.StoreType(*storeType),
				Force: *force,
			})
			if err != nil {
				return fmt.Errorf("create game: %w", err)
			}
			defer st.Close()
			fmt.Printf("fh: create game: initialized %s store for game %q in %s\n", *storeType, *id, *path)
			return nil
		},
	}
}

// openGame opens the store recorded in a game directory's far-horizons.json and
// returns it along with the game's ID (needed to address the store).
func openGame(ctx context.Context, dir string) (store.Store, string, error) {
	cfg, err := store.ReadConfig(dir)
	if err != nil {
		return nil, "", fmt.Errorf("read game config: %w", err)
	}
	st, err := store.Open(ctx, dir)
	if err != nil {
		return nil, "", fmt.Errorf("open store: %w", err)
	}
	return st, cfg.Game.ID, nil
}
