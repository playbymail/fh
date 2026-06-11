// Package engine implements the idiomatic Far Horizons game engine (fh).
//
// Unlike internal/game (the byte-faithful C port), the engine holds no
// package-level mutable game state: all state hangs off the Engine struct and
// is threaded explicitly. State is persisted in a SQLite store (see
// internal/data/store) rather than the original binary .dat files.
package engine

import (
	"context"

	"github.com/playbymail/fh/internal/data/store"
	"github.com/playbymail/fh/internal/engine/rng"
	"github.com/playbymail/fh/internal/model"
)

// Engine coordinates game execution for a single game.
type Engine struct {
	store  store.Store
	rng    rng.Factory
	gameID string
}

// New creates a new engine bound to a store and game id.
func New(st store.Store, gameID string) *Engine {
	return &Engine{store: st, gameID: gameID}
}

// WithRNG attaches a deterministic RNG factory (used when the engine generates
// state rather than ingesting it).
func (e *Engine) WithRNG(f rng.Factory) *Engine {
	e.rng = f
	return e
}

// GameID returns the engine's game identifier.
func (e *Engine) GameID() string { return e.gameID }

// IngestWorld persists a fully-built World as this engine's game state.
func (e *Engine) IngestWorld(ctx context.Context, w *model.World) error {
	return e.store.IngestWorld(ctx, e.gameID, w)
}

// IngestDir reads an fhc state directory (its export-json output, per-species
// logs, and locations index) and persists it as this engine's game state.
func (e *Engine) IngestDir(ctx context.Context, dir string) error {
	w, err := LoadWorldFromDir(dir)
	if err != nil {
		return err
	}
	return e.IngestWorld(ctx, w)
}

// LoadWorld reads this engine's game state back from the store into memory.
func (e *Engine) LoadWorld(ctx context.Context) (*model.World, error) {
	return e.store.LoadWorld(ctx, e.gameID)
}

// RenderReport loads game state from the store and renders the turn report for
// the given 1-based species number, byte-identical to fhc's sp%02d.rpt.t%d.
func (e *Engine) RenderReport(ctx context.Context, speciesNumber int, opts ReportOptions) ([]byte, error) {
	w, err := e.LoadWorld(ctx)
	if err != nil {
		return nil, err
	}
	sp := w.SpeciesByNumber(speciesNumber)
	if sp == nil {
		return nil, &SpeciesNotFoundError{Number: speciesNumber}
	}
	return RenderReport(w, sp, opts), nil
}

// SpeciesNotFoundError reports a request for a species not present in the game.
type SpeciesNotFoundError struct{ Number int }

func (e *SpeciesNotFoundError) Error() string {
	return "engine: species not found: " + itoa(e.Number)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
