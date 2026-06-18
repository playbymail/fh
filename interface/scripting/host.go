package scripting

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// Host is the embedded Lua scripting engine: it builds a sandboxed interpreter,
// applies the immutable GM/player scope, exposes the read-only fh.load{} verb and
// the GM lifecycle verbs (fh.gm()), and runs a script file. The engine-specific
// work is delegated to the Engine, so the same Host drives fhc today and fh
// later. See docs/project-ultron/{fhc-script-design,turn-lifecycle}.md.

// Scope is the immutable access scope chosen at the command line; a script cannot
// change it, so an adversarial agent cannot widen its own view.
type Scope int

const (
	ScopeGM     Scope = iota // --gm: all turns, all species, GM lifecycle verbs
	ScopePlayer              // --species <id>: all turns, only that species, no GM verbs
)

// Host carries the validated invocation state and the scoped scan results.
type Host struct {
	eng       Engine
	dataRoot  string
	scope     Scope
	speciesID int // valid only when scope == ScopePlayer

	// scoped fh.load{} scan results.
	turns   []int
	species []int
}

// NewHost returns a Host bound to an Engine, a data root, and the immutable scope.
func NewHost(eng Engine, dataRoot string, scope Scope, speciesID int) *Host {
	return &Host{eng: eng, dataRoot: dataRoot, scope: scope, speciesID: speciesID}
}

// Run builds the sandboxed interpreter, installs the fh verbs, and executes the
// script file. The sandbox (security layer 1) opens only the pure, deterministic
// stdlib (base/string/table/math) and removes the code-loading and
// nondeterminism globals, so a script reaches game data and the engine only
// through the scoped verbs.
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

	// The fh table is the script's only entry into game data and control: load{}
	// (read-only scan) and gm() (the GM lifecycle handle, nil under player scope).
	fhTbl := L.NewTable()
	fhTbl.RawSetString("load", L.NewFunction(h.luaLoad))
	fhTbl.RawSetString("gm", L.NewFunction(h.luaGM))
	L.SetGlobal("fh", fhTbl)

	return L.DoFile(scriptPath)
}

// scan enumerates the data-root: integer-named turn dirs → turn list; integer
// species subdirs (union across turns) → species roster. Directory enumeration
// only; reads no game state. Turn 0 is rejected here (it is genesis, not a
// player-addressable turn) — distinct from the lifecycle's turn scan.
func (h *Host) scan() (turns []int, species []int, err error) {
	turnEntries, err := os.ReadDir(h.dataRoot)
	if err != nil {
		return nil, nil, err
	}
	speciesSet := make(map[int]bool)
	for _, te := range turnEntries {
		if !te.IsDir() {
			continue
		}
		turn, ok := parsePositiveInt(te.Name())
		if !ok {
			continue
		}
		turns = append(turns, turn)
		spEntries, err := os.ReadDir(filepath.Join(h.dataRoot, te.Name()))
		if err != nil {
			continue
		}
		for _, se := range spEntries {
			if !se.IsDir() {
				continue
			}
			if id, ok := parsePositiveInt(se.Name()); ok {
				speciesSet[id] = true
			}
		}
	}
	for id := range speciesSet {
		species = append(species, id)
	}
	sort.Ints(turns)
	sort.Ints(species)
	return turns, species, nil
}

// parsePositiveInt parses a bare directory name as a positive integer id (>= 1).
func parsePositiveInt(name string) (int, bool) {
	n, err := strconv.Atoi(name)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// luaLoad implements fh.load{}: scan, apply scope, cache the scoped lists, and
// return handle, turns, species. Under player scope the roster is filtered to
// the caller's species (error if absent from the scan).
func (h *Host) luaLoad(L *lua.LState) int {
	turns, species, err := h.scan()
	if err != nil {
		L.RaiseError("fh.load: cannot scan data-root: %v", err)
		return 0
	}
	if h.scope == ScopePlayer {
		found := false
		for _, id := range species {
			if id == h.speciesID {
				found = true
				break
			}
		}
		if !found {
			L.RaiseError("fh.load: species %d not present in data-root", h.speciesID)
			return 0
		}
		species = []int{h.speciesID}
	}

	h.turns = append([]int(nil), turns...)
	h.species = append([]int(nil), species...)

	turnsTbl := intSliceToLuaTable(L, h.turns)
	speciesTbl := intSliceToLuaTable(L, h.species)
	handle := L.NewTable()
	handle.RawSetString("turns", turnsTbl)
	handle.RawSetString("species", speciesTbl)

	L.Push(handle)
	L.Push(turnsTbl)
	L.Push(speciesTbl)
	return 3
}

// luaGM implements fh.gm(): returns the GM lifecycle handle (a table of verbs)
// under GM scope, or nil under player scope so a player script cannot reach the
// mutating verbs. The scope rides on the host, never on the handle.
func (h *Host) luaGM(L *lua.LState) int {
	if h.scope != ScopeGM {
		L.Push(lua.LNil)
		return 1
	}
	gm := L.NewTable()
	gm.RawSetString("genesis", L.NewFunction(h.luaGenesis))
	gm.RawSetString("freeze_and_forward", L.NewFunction(h.luaFreezeAndForward))
	gm.RawSetString("run_turn", L.NewFunction(h.luaRunTurn))
	L.Push(gm)
	return 1
}

// luaGenesis implements gm:genesis{ seed = N, species = M }. seed defaults to 0
// (the engine's historical default); species is required (>= 1).
func (h *Host) luaGenesis(L *lua.LState) int {
	opts := optTable(L, 2)
	seed := uint64(0)
	if n, ok := opts.RawGetString("seed").(lua.LNumber); ok {
		seed = uint64(n)
	}
	species := 0
	if n, ok := opts.RawGetString("species").(lua.LNumber); ok {
		species = int(n)
	}
	if species < 1 {
		L.RaiseError("gm:genesis requires species >= 1")
		return 0
	}
	if err := Genesis(h.eng, h.dataRoot, seed, species); err != nil {
		L.RaiseError("gm:genesis: %v", err)
		return 0
	}
	return 0
}

// luaFreezeAndForward implements gm:freeze_and_forward(); returns the new turn id.
func (h *Host) luaFreezeAndForward(L *lua.LState) int {
	next, err := FreezeAndForward(h.eng, h.dataRoot)
	if err != nil {
		L.RaiseError("gm:freeze_and_forward: %v", err)
		return 0
	}
	L.Push(lua.LNumber(next))
	return 1
}

// luaRunTurn implements gm:run_turn(); returns the resolved turn id.
func (h *Host) luaRunTurn(L *lua.LState) int {
	resolved, err := RunTurn(h.eng, h.dataRoot)
	if err != nil {
		L.RaiseError("gm:run_turn: %v", err)
		return 0
	}
	L.Push(lua.LNumber(resolved))
	return 1
}

// optTable returns the table argument at position n (gm:verb{...} passes the gm
// handle as arg 1 and the options table as arg 2), or an empty table.
func optTable(L *lua.LState, n int) *lua.LTable {
	if t, ok := L.Get(n).(*lua.LTable); ok {
		return t
	}
	return L.NewTable()
}

// intSliceToLuaTable builds a 1-based, ipairs-iterable Lua array table of numbers.
func intSliceToLuaTable(L *lua.LState, xs []int) *lua.LTable {
	tbl := L.NewTable()
	for _, x := range xs {
		tbl.Append(lua.LNumber(x))
	}
	return tbl
}
