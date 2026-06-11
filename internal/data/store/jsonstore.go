package store

import (
	"context"

	"github.com/playbymail/fh/internal/data/fhjson"
	"github.com/playbymail/fh/internal/model"
)

// JSONStore persists a game as the Far Horizons JSON export files in a
// directory: galaxy.json, systems.json, species.%03d.json, the sp%02d.log
// event-log sidecars, and the locations.dat index. The encoding is shared with
// the engine's fhc-import path (internal/data/fhjson).
//
// The store is bound to a single game directory; the gameID arguments on the
// Store interface are accepted for compatibility and otherwise unused.
type JSONStore struct {
	dir string
}

// newJSONStore binds a JSONStore to a game directory.
func newJSONStore(dir string) *JSONStore { return &JSONStore{dir: dir} }

// Close releases resources (none for a file-backed store).
func (j *JSONStore) Close() error { return nil }

// IngestWorld writes the complete World as JSON export files.
func (j *JSONStore) IngestWorld(_ context.Context, _ string, w *model.World) error {
	return fhjson.WriteWorld(j.dir, w)
}

// LoadWorld reads the JSON export files back into a model.World.
func (j *JSONStore) LoadWorld(_ context.Context, _ string) (*model.World, error) {
	return fhjson.ReadWorld(j.dir)
}
