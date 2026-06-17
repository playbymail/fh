// Package store implements persistence interfaces for Far Horizons.
package store

import (
	"context"

	"github.com/playbymail/fh/internal/model"
)

// Store is the interface every game-data backend implements. It is
// deliberately small: the whole game World is loaded and saved as a unit, so
// the report renderer is agnostic to whether the backing store is binary .dat
// files, JSON files, or SQLite. Backends are constructed through Open / Create
// (see factory.go), which resolve the backend from a game directory's
// far-horizons.json config.
//
// Backend-specific operations (SQLite schema versioning, turn/snapshot/order
// tables, etc.) are NOT part of this interface; they remain concrete methods on
// the implementations that support them.
type Store interface {
	// IngestWorld persists a complete game World, replacing any prior state.
	IngestWorld(ctx context.Context, gameID string, w *model.World) error
	// LoadWorld reads the complete game World back into memory.
	LoadWorld(ctx context.Context, gameID string) (*model.World, error)
	// Close releases any resources held by the store.
	Close() error
}

// Game represents a game instance.
type Game struct {
	ID        string
	Name      string
	CreatedAt string // ISO 8601
}

// Turn represents a game turn.
type Turn struct {
	GameID    string
	Num       int
	Phase     string
	StartedAt string
	EndedAt   string
}

// Entity represents a world entity (serialized).
type Entity struct {
	ID   string
	Kind string
	Data []byte // JSON or msgpack
}

// Order represents a player order (serialized).
type Order struct {
	Seq        int
	Raw        string
	Normalized string // JSON
	Status     string
	Error      string
}
