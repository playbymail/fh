// Package engine implements the Far Horizons game engine.
package engine

import (
	"github.com/playbymail/fh/internal/data/store"
	"github.com/playbymail/fh/internal/engine/rng"
)

// Engine coordinates game execution.
type Engine struct {
	store   store.Store
	rng     rng.Factory
	// TODO: Add planner, etc.
}

// New creates a new engine instance.
func New(store store.Store, rng rng.Factory) *Engine {
	return &Engine{
		store: store,
		rng:   rng,
	}
}