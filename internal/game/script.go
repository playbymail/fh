package game

// Host for the `fhc script` subcommand — an embedded GopherLua interpreter
// that will expose a read-only, scope-controlled view of a Far Horizons game
// to Lua scripts (the Ultron harness). The host lives inside package game so
// later slices can reach the unexported loaders, globals, and structs
// directly (see docs/project-ultron/fhc-script-design.md).
//
// The CLI surface + GopherLua state (#39), the stdlib sandbox (#40), and the
// fh.load{} scan/scope verb (#41) are in place: the host parses and validates
// the flags, records the immutable scope, sandboxes the interpreter, and
// exposes fh.load{} (scan the data-root tree, apply scope, return the turn and
// species lists). Turn selection (g:turn(id)) and the scoped query API are
// later issues (#42–#45). Nothing here loads a .dat file or touches the PRNG —
// the scan is directory enumeration only, so the subcommand stays byte-neutral
// on the golden trees and leaves prngSeed at 0.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// scriptScope is the immutable access scope chosen at the command line. It is
// fixed by the --gm / --species flag and cannot be changed from inside a
// script, so an adversarial Ultron agent cannot widen its own view.
type scriptScope int

const (
	scopeGM     scriptScope = iota // --gm: all turns, all species
	scopePlayer                    // --species <id>: all turns, only that species
)

// scriptHost carries the parsed, validated invocation state. Later slices hang
// the loaded turn off this struct; #41 caches the scoped scan results (turns,
// species) so the script handle and #42's g:turn(id) share one source of truth.
type scriptHost struct {
	dataRoot  string      // --data-root: dir of turn folders
	scope     scriptScope // GM or player, set once at invocation
	speciesID int         // caller's species id; valid only when scope == scopePlayer

	// Scoped scan results, populated by fh.load{} (luaLoad). turns is the full
	// ascending turn list (unrestricted in both scopes); species is the roster
	// after scope has been applied (full under GM, {speciesID} under player).
	// Cached on the host — never on the script-visible handle — so a script
	// cannot widen its own view, and so #42's g:turn(id) can validate turn ids.
	turns   []int
	species []int
}

// scriptUsage is the one-line usage string, mirroring the CLI shape in the
// design doc.
const scriptUsage = "fh: usage: script --data-root=<dir> (--gm | --species=<id>) <file.lua>"

// scriptCommand is the `script` entry point; args[0] is the command name. It
// parses the flags hand-rolled (the internal/game convention, not ff/v4),
// validates them before any Lua runs, then executes the named script file.
// Returns 0 on success and 2 on any error, like every other command.
func scriptCommand(args []string) int {
	var (
		dataRoot    string
		scriptPath  string
		gm          bool
		species     string
		haveSpecies bool
	)

	for i := 1; i < len(args); i++ {
		opt, val, hasVal := splitOptVal(args[i])

		switch {
		case opt == "--help" || opt == "-h" || opt == "-?":
			fmt.Fprintln(os.Stderr, scriptUsage)
			return 2
		case opt == "--data-root":
			if !hasVal {
				fmt.Fprintf(os.Stderr, "fh: script: --data-root requires a directory\n")
				return 2
			}
			dataRoot = val
		case opt == "--gm":
			gm = true
		case opt == "--species":
			if !hasVal {
				fmt.Fprintf(os.Stderr, "fh: script: --species requires an id\n")
				return 2
			}
			species = val
			haveSpecies = true
		case len(opt) > 0 && opt[0] == '-':
			fmt.Fprintf(os.Stderr, "fh: script: unknown option '%s'\n", opt)
			return 2
		case scriptPath == "":
			// First bare argument is the script file.
			scriptPath = args[i]
		default:
			fmt.Fprintf(os.Stderr, "fh: script: unexpected argument '%s'\n", args[i])
			return 2
		}
	}

	// Validate the flags before any Lua runs.
	if dataRoot == "" {
		fmt.Fprintf(os.Stderr, "fh: script: --data-root is required\n%s\n", scriptUsage)
		return 2
	}
	if gm && haveSpecies {
		fmt.Fprintf(os.Stderr, "fh: script: --gm and --species are mutually exclusive\n%s\n", scriptUsage)
		return 2
	}
	if !gm && !haveSpecies {
		fmt.Fprintf(os.Stderr, "fh: script: exactly one of --gm or --species is required\n%s\n", scriptUsage)
		return 2
	}
	if scriptPath == "" {
		fmt.Fprintf(os.Stderr, "fh: script: a <file.lua> script path is required\n%s\n", scriptUsage)
		return 2
	}

	host := &scriptHost{dataRoot: dataRoot}
	if gm {
		host.scope = scopeGM
	} else {
		id, err := strconv.Atoi(species)
		if err != nil || id < 1 {
			fmt.Fprintf(os.Stderr, "fh: script: invalid --species id '%s'\n", species)
			return 2
		}
		host.scope = scopePlayer
		host.speciesID = id
	}

	// Resolve the script path against the original cwd now, before any later
	// slice chdir's into a turn directory, so turn selection cannot change
	// which file we run.
	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}

	return host.run(absScript)
}

// run builds a sandboxed GopherLua state and executes the script file. Later
// slices install the `fh` query API on this state; the scoped query API is the
// only path a script has to game data.
//
// Security layer 1 (#40): an untrusted Ultron agent script must not reach the
// filesystem or any source of nondeterminism. We open the interpreter with
// SkipOpenLibs so nothing is available by default, then open only the pure,
// deterministic libraries (base, string, table, math). Leaving package, io,
// os, debug, coroutine, and channel unopened is what makes os, io, debug, and
// require absent. We then nil out the base globals that load code (dofile,
// load, loadfile, loadstring) and the two nondeterministic math entry points
// (math.random, math.randomseed). The determinism invariant — same data in,
// same report out — is what lets fh be validated against fhc, so the script
// layer must not introduce a hidden RNG. See
// docs/project-ultron/fhc-script-design.md ("Security model", "Sandboxing").
func (h *scriptHost) run(scriptPath string) int {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()

	// Open only the pure, deterministic stdlib pieces, via the loader-function
	// call pattern (push the open func + lib name, then Call(1, 0)).
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

	// OpenBase registers code-loading globals; remove them so a script cannot
	// pull in arbitrary code or files. require/package are never opened, but
	// nil them belt-and-suspenders.
	for _, g := range []string{"dofile", "load", "loadfile", "loadstring", "require", "package"} {
		L.SetGlobal(g, lua.LNil)
	}

	// Neutralize the nondeterministic math entry points; everything else in
	// math is a pure function.
	if mathTbl, ok := L.GetGlobal(lua.MathLibName).(*lua.LTable); ok {
		mathTbl.RawSetString("random", lua.LNil)
		mathTbl.RawSetString("randomseed", lua.LNil)
	}

	// Register the single host verb, fh.load{}. The `fh` table is the script's
	// only entry into game data; its load field is a Go closure bound to this
	// host (so the immutable scope rides on h, captured here, not on anything
	// the script can reach). Installed after the sandbox is built and before the
	// script runs.
	fhTbl := L.NewTable()
	fhTbl.RawSetString("load", L.NewFunction(h.luaLoad))
	L.SetGlobal("fh", fhTbl)

	if err := L.DoFile(scriptPath); err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}
	return 0
}

// scan enumerates the data-root directory tree to discover the game's turns and
// species roster. It is the host-side half of fh.load{}, kept pure (no Lua, no
// globals) so it is directly unit-testable.
//
// Contract (see docs/project-ultron/fhc-script-design.md, "Load mechanism"):
// this is directory enumeration ONLY — it reads no .dat and opens no game state
// (that is #42's g:turn(id)). Integer-named subdirectories of the data root are
// turns; within each turn dir, integer-named subdirectories are species ids.
// The species roster is the union across all turn dirs (the roster is stable
// across turns, but the union is the deterministic, defensive choice).
// Non-integer-named entries and plain files are ignored; a missing or unreadable
// data root is an error. Both lists are returned sorted ascending.
func (h *scriptHost) scan() (turns []int, species []int, err error) {
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

		// Enumerate species subdirs within this turn dir. An unreadable turn dir
		// is skipped rather than fatal — a turn folder may legitimately be empty.
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

// parsePositiveInt parses a bare directory name as a positive integer id. It
// rejects anything strconv.Atoi would not accept as a plain integer and any
// value < 1, so "0", "-1", "3x", and "" are all ignored. (No leading-zero
// normalization is needed: turn/species dirs are written as bare integers.)
func parsePositiveInt(name string) (int, bool) {
	n, err := strconv.Atoi(name)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// luaLoad implements fh.load{} — the script's one-shot "open this game" verb.
// It scans the data root, applies the immutable CLI scope to the species
// roster, caches the scoped lists on the host, and returns three values to Lua:
// the game handle, the turn list, and the species list.
//
// Scope (see the design doc, "Load mechanism" step 2): GM sees the full roster;
// player scope filters the roster to {h.speciesID} and errors if that id is
// absent from the scan (the CLI already checked id >= 1; this is the
// scan-membership check). The turn list is unrestricted in both scopes.
//
// The handle is a fresh Lua table exposing g.turns and g.species (the same
// lists). The scope itself stays on h, captured by this closure, and is never
// written onto the handle — a script must not be able to widen its own view.
// fh.load{} takes one table argument (e.g. {}); this slice accepts and ignores
// it. Loading no .dat here, the verb leaves prngSeed at 0.
func (h *scriptHost) luaLoad(L *lua.LState) int {
	turns, species, err := h.scan()
	if err != nil {
		L.RaiseError("fh.load: cannot scan data-root: %v", err)
		return 0
	}

	// Apply the immutable scope to the species roster.
	switch h.scope {
	case scopePlayer:
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

	// Cache the scoped lists on the host for #42's g:turn(id) to validate
	// against. Copy into fresh slices so the returned Lua tables and the host's
	// cache never alias the scan's working slices.
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

// intSliceToLuaTable builds a 1-based, ipairs-iterable Lua array table of
// numbers from an int slice, preserving order. Returning explicit sequences
// makes ordering a guarantee, so scripted output stays deterministic.
func intSliceToLuaTable(L *lua.LState, xs []int) *lua.LTable {
	tbl := L.NewTable()
	for _, x := range xs {
		tbl.Append(lua.LNumber(x))
	}
	return tbl
}
