// Package store implements persistence interfaces for Far Horizons.
package store

import (
	"context"
	"io"
)

// Store is the interface for game data persistence.
// Implementations can be JSON files, SQLite, etc.
type Store interface {
	// Schema management
	GetSchemaVersion(ctx context.Context) (string, error)
	UpgradeSchema(ctx context.Context) error

	// Game management
	CreateGame(ctx context.Context, id, name string) error
	GetGame(ctx context.Context, id string) (*Game, error)

	// Turn management
	CreateTurn(ctx context.Context, gameID string, turnNum int, phase string) error
	GetCurrentTurn(ctx context.Context, gameID string) (*Turn, error)

	// World snapshots
	SaveSnapshot(ctx context.Context, gameID string, turnNum int, entities []Entity) error
	LoadSnapshot(ctx context.Context, gameID string, turnNum int) ([]Entity, error)

	// Orders
	SaveOrders(ctx context.Context, gameID string, turnNum int, actor string, orders []Order) error
	GetOrders(ctx context.Context, gameID string, turnNum int, actor string) ([]Order, error)

	// Reports
	SaveReport(ctx context.Context, gameID string, turnNum int, actor string, mime string, body io.Reader) error
	GetReport(ctx context.Context, gameID string, turnNum int, actor string, mime string) (io.ReadCloser, error)

	// Close the store
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
