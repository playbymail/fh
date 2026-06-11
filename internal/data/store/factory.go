package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/playbymail/fh/internal/cerrs"
)

// sqliteDBName is the SQLite backing file inside a game directory.
const sqliteDBName = "game.db"

// CreateOptions configures a new game directory.
type CreateOptions struct {
	ID    string    // game identifier (required)
	Name  string    // human-readable name (defaults to ID)
	Type  StoreType // backend; defaults to StoreBinary
	Force bool      // overwrite an existing game in the directory
}

// Create initializes a new game in dir: it writes far-horizons.json recording
// the chosen backend and prepares that backend's backing store. The directory
// is created if it does not exist.
func Create(ctx context.Context, dir string, opts CreateOptions) (Store, error) {
	if opts.ID == "" {
		return nil, fmt.Errorf("game id is required")
	}
	if opts.Type == "" {
		opts.Type = StoreBinary
	}
	if !opts.Type.Valid() {
		return nil, fmt.Errorf("invalid store type %q", opts.Type)
	}

	switch info, err := os.Stat(dir); {
	case err == nil:
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", dir)
		}
		if _, err := os.Stat(filepath.Join(dir, ConfigFileName)); err == nil && !opts.Force {
			return nil, cerrs.ErrExists
		}
	case os.IsNotExist(err):
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	name := opts.Name
	if name == "" {
		name = opts.ID
	}
	cfg := &GameConfig{
		Version: ConfigVersion,
		Game:    GameMeta{ID: opts.ID, Name: name, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		Store:   StoreSection{Type: opts.Type},
	}
	if err := WriteConfig(dir, cfg); err != nil {
		return nil, err
	}

	switch opts.Type {
	case StoreSQLite:
		st, err := NewSQLiteStore(filepath.Join(dir, sqliteDBName), opts.Force)
		if err != nil {
			return nil, err
		}
		if err := st.CreateGame(ctx, opts.ID, name); err != nil {
			st.Close()
			return nil, fmt.Errorf("create game row: %w", err)
		}
		return st, nil
	case StoreJSON:
		return newJSONStore(dir), nil
	default: // StoreBinary
		return newBinaryStore(dir), nil
	}
}

// Open resolves the backend recorded in dir's far-horizons.json and opens it.
func Open(_ context.Context, dir string) (Store, error) {
	cfg, err := ReadConfig(dir)
	if err != nil {
		return nil, err
	}
	switch cfg.Store.Type {
	case StoreSQLite:
		return OpenSQLiteStore(filepath.Join(dir, sqliteDBName))
	case StoreJSON:
		return newJSONStore(dir), nil
	case StoreBinary:
		return newBinaryStore(dir), nil
	default:
		return nil, fmt.Errorf("invalid store type %q", cfg.Store.Type)
	}
}
