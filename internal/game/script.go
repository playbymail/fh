package game

// `fhc script` subcommand — the CLI boundary for the embedded scripting engine.
// This file is now a thin shim: it parses and validates the flags, fixes the
// immutable scope, and hands off to the engine-agnostic scripting host in
// interface/scripting, wired with the fhc adapter (interface/game). The host,
// the sandbox, the fh.load{} scan, and the GM lifecycle verbs live in those
// packages so fh can reuse the same scripting engine with its own Engine
// implementation. See docs/project-ultron/{fhc-script-design,turn-lifecycle}.md.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	gamescript "github.com/playbymail/fh/interface/game"
	"github.com/playbymail/fh/interface/scripting"
)

// scriptUsage is the one-line usage string, mirroring the CLI shape in the
// design doc.
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

	// Resolve the script path against the original cwd now, before the lifecycle
	// verbs chdir into turn directories, so turn selection cannot change which
	// file we run.
	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}

	// The fhc adapter drives the engine by invoking this very fhc binary's
	// subcommands (so it matches the shell-script oracle and keeps the C-port
	// engine frozen).
	fhcPath, err := os.Executable()
	if err != nil {
		fhcPath = "fhc"
	}

	host := scripting.NewHost(gamescript.New(fhcPath), dataRoot, scope, speciesID)
	if err := host.Run(absScript); err != nil {
		fmt.Fprintf(os.Stderr, "fh: script: %v\n", err)
		return 2
	}
	return 0
}
