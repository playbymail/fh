// Package scripting is the engine-agnostic Far Horizons scripting engine. It
// embeds Lua, applies the GM/player scope, exposes the read-only query verbs and
// the GM turn-lifecycle verbs, and drives the host-side turn lifecycle — the
// single source of process truth, in Go, mirroring the validated shell scripts
// (tools/*.sh, see docs/project-ultron/turn-lifecycle.md).
//
// The engine-specific operations the lifecycle needs — read a turn's number,
// create a game, run a turn — are abstracted behind the Engine interface. fhc
// and fh each implement Engine in their own terms (fhc by driving the C-port
// engine, fh via its SQLite store), so this package and the lifecycle logic
// above it never change when a new engine is added.
package scripting

// Engine is the port the scripting engine drives; fhc and fh implement it in
// their own terms but both satisfy the same interface. All methods operate on a
// turn directory: the per-turn working directory under the data root
// (<data-root>/<turn>/).
type Engine interface {
	// TurnNumber returns the authoritative turn_number for the game in turnDir
	// (the count of turns resolved so far). exists is false when turnDir holds no
	// game, which the lifecycle predicate reads as the "absent" state. The engine
	// owns where that number lives (fhc: galaxy.dat; fh: its store).
	TurnNumber(turnDir string) (n int, exists bool, err error)

	// Genesis creates a new game in turnDir — galaxy, home-system templates, and
	// numSpecies species — deterministically seeded, leaving turn_number == 0.
	Genesis(turnDir string, seed uint64, numSpecies int) error

	// RunTurn runs the full turn pipeline to completion in turnDir: it stages the
	// per-species orders found under turnDir, resolves the turn, advances
	// turn_number by one, and writes the player reports.
	RunTurn(turnDir string) error
}

// TurnState is the lifecycle state of a turn folder, derived from its
// turn_number relative to its folder name N (see turn-lifecycle.md). A folder is
// resolved when turn_number == N and pending (orders open, not yet run) when it
// is exactly one behind.
type TurnState string

const (
	TurnAbsent    TurnState = "absent"    // no game in the folder
	TurnPending   TurnState = "pending"   // turn_number == N-1: orders open, not yet run
	TurnResolved  TurnState = "resolved"  // turn_number == N: pipeline has run
	TurnAnomalous TurnState = "anomalous" // neither: corrupt or hand-edited
)
