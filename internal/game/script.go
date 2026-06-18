package game

// Host for the `fhc script` subcommand — an embedded GopherLua interpreter
// that will expose a read-only, scope-controlled view of a Far Horizons game
// to Lua scripts (the Ultron harness). The host lives inside package game so
// later slices can reach the unexported loaders, globals, and structs
// directly (see docs/project-ultron/fhc-script-design.md).
//
// This first slice (#39) only stands up the CLI surface and the GopherLua
// state: it parses and validates the flags, records the immutable scope, and
// runs the named script file end-to-end. The stdlib sandbox, the `fh.load{}`
// scan, turn selection, and the scoped query API are later issues (#40–#45),
// so nothing here loads a .dat file or touches the PRNG — the subcommand is
// byte-neutral on the golden trees.

import (
	"fmt"
	"os"
	"path/filepath"
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
// the scan results and the loaded turn off this struct; for #39 it only holds
// the scope and the data root.
type scriptHost struct {
	dataRoot  string      // --data-root: dir of turn folders
	scope     scriptScope // GM or player, set once at invocation
	speciesID int         // caller's species id; valid only when scope == scopePlayer
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

	if err := L.DoFile(scriptPath); err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}
	return 0
}
