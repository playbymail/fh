package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/playbymail/fh/internal/cerrs"
)

// ConfigFileName is the per-game configuration file written at the root of a
// game directory. It records which store backend the game uses so the CLI can
// open the right one without the gamemaster having to remember.
const ConfigFileName = "far-horizons.json"

// ConfigVersion is the current far-horizons.json schema version.
const ConfigVersion = 1

// StoreType identifies a persistence backend.
type StoreType string

const (
	// StoreBinary persists game state as the classic Far Horizons binary
	// .dat files (galaxy.dat, stars.dat, planets.dat, sp%02d.dat, ...).
	StoreBinary StoreType = "binary"
	// StoreJSON persists game state as the fhc JSON export files
	// (galaxy.json, systems.json, species.%03d.json, ...).
	StoreJSON StoreType = "json"
	// StoreSQLite persists game state in a SQLite database (game.db).
	StoreSQLite StoreType = "sqlite"
)

// Valid reports whether t is a known store type.
func (t StoreType) Valid() bool {
	switch t {
	case StoreBinary, StoreJSON, StoreSQLite:
		return true
	default:
		return false
	}
}

// GameConfig is the parsed far-horizons.json document.
type GameConfig struct {
	Version int          `json:"version"`
	Game    GameMeta     `json:"game"`
	Store   StoreSection `json:"store"`
}

// GameMeta holds the game's identity.
type GameMeta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"` // RFC 3339
}

// StoreSection records the chosen backend.
type StoreSection struct {
	Type StoreType `json:"type"`
}

// ReadConfig loads and validates far-horizons.json from a game directory.
func ReadConfig(dir string) (*GameConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, cerrs.ErrNotExist
		}
		return nil, err
	}
	var cfg GameConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigFileName, err)
	}
	if !cfg.Store.Type.Valid() {
		return nil, fmt.Errorf("%s: invalid store type %q", ConfigFileName, cfg.Store.Type)
	}
	return &cfg, nil
}

// WriteConfig writes far-horizons.json to a game directory.
func WriteConfig(dir string, cfg *GameConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, ConfigFileName), data, 0o644)
}
