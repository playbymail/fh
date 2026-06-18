package scripting

import (
	"fmt"
	"sort"

	lua "github.com/yuin/gopher-lua"
)

// Scope is the immutable access scope chosen at the command line; a script cannot
// change it, so an adversarial agent cannot widen its own view.
type Scope int

const (
	ScopeGM     Scope = iota // --gm: every species visible
	ScopePlayer              // --species <id>: only that species visible
)

// Host runs a sandboxed Lua script against a Game under a fixed scope. The Game
// supplies the data; the Host supplies the sandbox and the access policy.
type Host struct {
	game      Game
	scope     Scope
	speciesID int // valid only when scope == ScopePlayer
}

// NewHost returns a Host bound to a Game and the immutable scope. speciesID is
// meaningful only for ScopePlayer.
func NewHost(game Game, scope Scope, speciesID int) *Host {
	return &Host{game: game, scope: scope, speciesID: speciesID}
}

// Run builds the sandboxed interpreter, installs the fh query verbs, and executes
// the script file. The sandbox opens only the pure, deterministic stdlib
// (base/string/table/math) and removes the code-loading and nondeterminism
// globals, so a script reaches game data only through the scoped verbs.
func (h *Host) Run(scriptPath string) error {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()

	for _, lib := range []struct {
		name string
		open lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.StringLibName, lua.OpenString},
		{lua.TabLibName, lua.OpenTable},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.open))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	for _, g := range []string{"dofile", "load", "loadfile", "loadstring", "require", "package"} {
		L.SetGlobal(g, lua.LNil)
	}
	if mathTbl, ok := L.GetGlobal(lua.MathLibName).(*lua.LTable); ok {
		mathTbl.RawSetString("random", lua.LNil)
		mathTbl.RawSetString("randomseed", lua.LNil)
	}

	// The fh table is the script's only entry into game data: the read-only query
	// verbs.
	fhTbl := L.NewTable()
	fhTbl.RawSetString("current_turn", L.NewFunction(h.luaCurrentTurn))
	fhTbl.RawSetString("turn_status", L.NewFunction(h.luaTurnStatus))
	fhTbl.RawSetString("orders", L.NewFunction(h.luaOrders))
	fhTbl.RawSetString("report", L.NewFunction(h.luaReport))
	fhTbl.RawSetString("species_stats", L.NewFunction(h.luaSpeciesStats))
	L.SetGlobal("fh", fhTbl)

	return L.DoFile(scriptPath)
}

// resolveSpecies maps the optional Lua species argument to the species a
// per-species verb should act on, enforcing the scope. arg == 0 means the
// argument was omitted.
//
//   - Player scope: the species is fixed to the caller's id; an explicit id that
//     is not the caller's is denied.
//   - GM scope: a species id is required (there is no implied single species).
func (h *Host) resolveSpecies(arg int) (int, error) {
	if h.scope == ScopePlayer {
		if arg != 0 && arg != h.speciesID {
			return 0, fmt.Errorf("species %d is not visible in this scope", arg)
		}
		return h.speciesID, nil
	}
	if arg == 0 {
		return 0, fmt.Errorf("a species id is required under --gm")
	}
	return arg, nil
}

// visibleSpecies returns the species ids the caller may see, ascending: the
// whole roster under GM, just the caller's species under player scope.
func (h *Host) visibleSpecies() ([]int, error) {
	if h.scope == ScopePlayer {
		return []int{h.speciesID}, nil
	}
	ids, err := h.game.SpeciesIDs()
	if err != nil {
		return nil, err
	}
	out := append([]int(nil), ids...)
	sort.Ints(out)
	return out, nil
}

// luaCurrentTurn implements fh.current_turn() -> number.
func (h *Host) luaCurrentTurn(L *lua.LState) int {
	n, err := h.game.CurrentTurn()
	if err != nil {
		L.RaiseError("fh.current_turn: %v", err)
		return 0
	}
	L.Push(lua.LNumber(n))
	return 1
}

// luaTurnStatus implements fh.turn_status(turn) -> "pending" | "resolved".
func (h *Host) luaTurnStatus(L *lua.LState) int {
	turn := L.CheckInt(1)
	status, err := h.game.TurnStatus(turn)
	if err != nil {
		L.RaiseError("fh.turn_status: %v", err)
		return 0
	}
	L.Push(lua.LString(status))
	return 1
}

// luaOrders implements fh.orders(turn [, species]) -> string | nil. It returns
// the entire orders file, or nil when no orders were submitted.
func (h *Host) luaOrders(L *lua.LState) int {
	turn := L.CheckInt(1)
	species, err := h.resolveSpecies(L.OptInt(2, 0))
	if err != nil {
		L.RaiseError("fh.orders: %v", err)
		return 0
	}
	content, ok, err := h.game.Orders(turn, species)
	if err != nil {
		L.RaiseError("fh.orders: %v", err)
		return 0
	}
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(content))
	return 1
}

// luaReport implements fh.report(turn [, species]) -> string. It raises if the
// turn has not been resolved.
func (h *Host) luaReport(L *lua.LState) int {
	turn := L.CheckInt(1)
	species, err := h.resolveSpecies(L.OptInt(2, 0))
	if err != nil {
		L.RaiseError("fh.report: %v", err)
		return 0
	}
	content, err := h.game.Report(turn, species)
	if err != nil {
		L.RaiseError("fh.report: %v", err)
		return 0
	}
	L.Push(lua.LString(content))
	return 1
}

// luaSpeciesStats implements fh.species_stats(turn [, species]). With a species
// (always, under player scope) it returns one stats table; under GM scope with
// the species omitted it returns an array of every species' stats, ascending.
func (h *Host) luaSpeciesStats(L *lua.LState) int {
	turn := L.CheckInt(1)
	arg := L.OptInt(2, 0)

	if h.scope == ScopeGM && arg == 0 {
		ids, err := h.visibleSpecies()
		if err != nil {
			L.RaiseError("fh.species_stats: %v", err)
			return 0
		}
		arr := L.NewTable()
		for _, id := range ids {
			stats, err := h.game.SpeciesStats(turn, id)
			if err != nil {
				L.RaiseError("fh.species_stats: %v", err)
				return 0
			}
			arr.Append(statsToLuaTable(L, stats))
		}
		L.Push(arr)
		return 1
	}

	species, err := h.resolveSpecies(arg)
	if err != nil {
		L.RaiseError("fh.species_stats: %v", err)
		return 0
	}
	stats, err := h.game.SpeciesStats(turn, species)
	if err != nil {
		L.RaiseError("fh.species_stats: %v", err)
		return 0
	}
	L.Push(statsToLuaTable(L, stats))
	return 1
}

// statsToLuaTable converts a SpeciesStats into the Lua table the script sees.
func statsToLuaTable(L *lua.LState, s SpeciesStats) *lua.LTable {
	tbl := L.NewTable()
	tbl.RawSetString("species", lua.LNumber(s.Species))
	tbl.RawSetString("name", lua.LString(s.Name))
	tbl.RawSetString("total_production", lua.LNumber(s.TotalProduction))
	tbl.RawSetString("num_planets", lua.LNumber(s.NumPlanets))
	tbl.RawSetString("num_ships", lua.LNumber(s.NumShips))
	tbl.RawSetString("num_shipyards", lua.LNumber(s.NumShipyards))
	tbl.RawSetString("offensive_power", lua.LNumber(s.OffensivePower))
	tbl.RawSetString("defensive_power", lua.LNumber(s.DefensivePower))
	tbl.RawSetString("econ_units", lua.LNumber(s.EconUnits))

	tech := L.NewTable()
	for name, level := range s.Tech {
		tech.RawSetString(name, lua.LNumber(level))
	}
	tbl.RawSetString("tech", tech)
	return tbl
}
