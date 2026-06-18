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

		// takeVal returns the value for an option that needs one, accepting
		// both --opt=val and --opt val (the design doc writes the space
		// form). It advances i past a consumed following argument.
		takeVal := func() (string, bool) {
			if hasVal {
				return val, true
			}
			if i+1 < len(args) {
				i++
				return args[i], true
			}
			return "", false
		}

		switch {
		case opt == "--help" || opt == "-h" || opt == "-?":
			fmt.Fprintln(os.Stderr, scriptUsage)
			return 2
		case opt == "--data-root":
			v, ok := takeVal()
			if !ok {
				fmt.Fprintf(os.Stderr, "fh: script: --data-root requires a directory\n")
				return 2
			}
			dataRoot = v
		case opt == "--gm":
			gm = true
		case opt == "--species":
			v, ok := takeVal()
			if !ok {
				fmt.Fprintf(os.Stderr, "fh: script: --species requires an id\n")
				return 2
			}
			species = v
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

// run builds a GopherLua state and executes the script file. Later slices
// install the stdlib sandbox and the `fh` query API on this state; for now it
// is a stock interpreter so a trivial print("hello") script runs end-to-end.
func (h *scriptHost) run(scriptPath string) int {
	L := lua.NewState()
	defer L.Close()

	if err := L.DoFile(scriptPath); err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}
	return 0
}
