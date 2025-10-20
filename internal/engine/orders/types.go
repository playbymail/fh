// Package orders implements order parsing, schema, and validators.
package orders

import (
	"github.com/playbymail/fh/internal/engine/rng"
)

// Context holds execution context for orders.
type Context struct {
	GameID string
	Turn   int
	Phase  string // e.g., "Economic", "Movement", "Combat"
	Actor  string // player/faction issuing the order
	Rng    rng.Scoped
}

// Order represents a parsed player order.
type Order interface {
	Key() string           // stable key for seeding RNG
	Actor() string         // which faction
	Validate(w ReadOnly) error
	Dependencies(w ReadOnly) []string // IDs this order reads/writes
	Execute(w ReadWrite, ctx Context) (Effect, error)
}

// ReadOnly is a read-only world view for validation.
type ReadOnly interface {
	GetEntity(id string) (interface{}, bool)
	// TODO: Add query methods
}

// ReadWrite is a mutable world view for execution.
type ReadWrite interface {
	ReadOnly
	Upsert(id string, entity interface{})
	Delete(id string)
}

// Effect describes changes from order execution.
type Effect interface {
	Targets() []string
}
