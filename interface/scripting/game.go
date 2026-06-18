// Package scripting is the engine-agnostic Far Horizons scripting host. It
// embeds Lua (GopherLua), applies a sandbox and the immutable GM/player scope,
// and exposes a small, read-only set of query verbs that Ultron and the GM use
// to inspect game state.
//
// The engine-specific data access the verbs need is abstracted behind the Game
// interface. fhc and fh each implement Game in their own terms (fhc by reading
// its flat .dat turn directories; fh via its SQLite store), so this package
// never changes when a new engine is added. Policy — who may see which species —
// lives only in the host; a Game implementation serves any (turn, species) and
// the host gates access against the scope fixed at the command line.
package scripting

// Game is the read-only query surface the Lua verbs call; fhc and fh implement
// it in-process. A game is addressed the way Ultron thinks about it: a turn
// number N (N > 0) and a 1-based species id. Where the data physically lives is
// the implementation's concern.
type Game interface {
	// CurrentTurn returns the highest turn number for the game (the active turn,
	// N > 0). It is an error if the game has no turns yet.
	CurrentTurn() (n int, err error)

	// TurnStatus returns the lifecycle status of turn n: TurnResolved once the
	// turn's pipeline has run, otherwise TurnPending. It is an error if turn n
	// does not exist.
	TurnStatus(turn int) (status string, err error)

	// SpeciesIDs returns the species roster (ascending). The host uses it for the
	// GM all-species view and to validate a player's scope.
	SpeciesIDs() (ids []int, err error)

	// Orders returns the entire orders file for (turn, species). ok is false when
	// no orders have been submitted for that species and turn.
	Orders(turn, species int) (content string, ok bool, err error)

	// Report returns the entire turn report for (turn, species). It is an error
	// if the turn has not been resolved.
	Report(turn, species int) (content string, err error)

	// SpeciesStats computes the per-species statistics for (turn, species) on
	// demand and returns them in structured form.
	SpeciesStats(turn, species int) (SpeciesStats, error)
}

// Turn status values returned by Game.TurnStatus.
const (
	TurnPending  = "pending"  // orders open, pipeline not yet run (turn_number == N-1)
	TurnResolved = "resolved" // pipeline has run, reports written (turn_number == N)
)

// SpeciesStats is the structured per-species snapshot Ultron ingests. The fields
// mirror the per-species columns of the engine's stats report so the scripted
// view and the stats command never drift.
type SpeciesStats struct {
	Species         int            // 1-based species id
	Name            string         // species name
	Tech            map[string]int // tech levels keyed MI, MA, ML, GV, LS, BI
	TotalProduction int            // total production across all colonies
	NumPlanets      int            // number of populated planets
	NumShips        int            // number of completed ships
	NumShipyards    int            // number of shipyards
	OffensivePower  int            // total offensive power
	DefensivePower  int            // total defensive power
	EconUnits       int            // banked economic units
}
