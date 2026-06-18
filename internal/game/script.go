package game

// `fhc script` subcommand — the CLI boundary for the embedded scripting engine.
// This is a thin shim: it parses and validates the flags, fixes the immutable
// scope, wires the fhc Game implementation (script_game.go) to the
// engine-agnostic scripting host (interface/scripting), and runs a script file.
// The host, sandbox, and read-only query verbs live in interface/scripting so
// fh can reuse the same scripting engine with its own Game implementation.
// See docs/project-ultron/fhc-script-design.md.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/playbymail/fh/interface/scripting"
)

// scriptUsage is the one-line usage string.
const scriptUsage = "fh: usage: script --data-root=<dir> (--gm | --species=<id>) <file.lua>"

// scriptCommand is the `script` entry point; args[0] is the command name. It
// parses the flags hand-rolled (the internal/game convention, not ff/v4),
// validates them before any Lua runs, then runs the named script through the
// scripting host. Returns 0 on success and 2 on any error, like every command.
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

	scope := scripting.ScopeGM
	speciesID := 0
	if !gm {
		id, err := strconv.Atoi(species)
		if err != nil || id < 1 {
			fmt.Fprintf(os.Stderr, "fh: script: invalid --species id '%s'\n", species)
			return 2
		}
		scope = scripting.ScopePlayer
		speciesID = id
	}

	// Resolve the script and data-root paths against the original cwd now, before
	// SpeciesStats chdirs into turn directories, so turn selection cannot change
	// which file we run or where we read game state.
	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}
	absRoot, err := filepath.Abs(dataRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}

	host := scripting.NewHost(newFHCGame(absRoot), scope, speciesID)
	if err := host.Run(absScript); err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}
	return 0
}
