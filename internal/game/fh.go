package game

import (
	"fmt"
	"os"
)

// CommandRunner is a faithful port of fh.c's main() dispatcher. It dispatches
// to the selected command and terminates the process with the C engine's exit
// code. main() simply calls this and lets the process exit normally on the
// help paths (which the C main() returns showHelp()==0 for).
func CommandRunner(argv []string) {
	argc := len(argv)
	if argc == 1 {
		showHelp()
		return
	}

	// runCommand terminates the process with the C engine's exit code for a
	// dispatched command: 0 on success, 2 on any error. This mirrors fh.c's
	// `return cmd(argc - i, argv + i)` — the C main() exits with the command's
	// return value, which is always 0 or 2. Errors are reported by the command
	// itself (to stderr), so there is nothing more to print here.
	runCommand := func(rv int) {
		if rv != 0 {
			os.Exit(2)
		}
		os.Exit(0)
	}

	for i := 1; i < argc; i++ {
		// Pass argv from the command name onward (mirrors C's
		// `cmd(argc - i, argv + i)`); the command functions expect
		// args[0] to be the command name.
		arg, args := argv[i], argv[i:]

		if arg == "?" || arg == "-?" || arg == "--help" {
			showHelp()
			return
		} else if arg == "-t" {
			test_mode = TRUE
		} else if arg == "-v" {
			verbose_mode = TRUE
		} else if arg == "combat" {
			runCommand(combatCommand(args))
		} else if arg == "create" {
			runCommand(createCommand(args))
		} else if arg == "export" {
			runCommand(exportCommand(args))
		} else if arg == "finish" {
			runCommand(finishCommand(args))
		} else if arg == "import" {
			runCommand(importCommand(args))
		} else if arg == "inspect" {
			runCommand(inspectCommand(args))
		} else if arg == "jump" {
			runCommand(jumpCommand(args))
		} else if arg == "list" {
			runCommand(listCommand(args))
		} else if arg == "locations" {
			runCommand(locationCommand(args))
		} else if arg == "logrnd" {
			// Mirrors C engine.c logRandomCommand; the Go port writes to
			// an io.Writer for testability, so pass os.Stdout here. The C
			// command ignores its args and always returns 0.
			runCommand(logRandomCommand(os.Stdout))
		} else if arg == "post-arrival" {
			runCommand(postArrivalCommand(args))
		} else if arg == "pre-departure" {
			runCommand(preDepartureCommand(args))
		} else if arg == "production" {
			runCommand(productionCommand(args))
		} else if arg == "report" {
			runCommand(reportCommand(args))
		} else if arg == "scan" {
			runCommand(scanCommand(args))
		} else if arg == "scan-near" {
			runCommand(scanNearCommand(args))
		} else if arg == "show" {
			runCommand(showCommand(args))
		} else if arg == "stats" {
			runCommand(statsCommand(args))
		} else if arg == "turn" {
			runCommand(turnCommand(args))
		} else if arg == "update" {
			runCommand(updateCommand(args))
		} else if arg == "version" {
			// versionCommand prints the engine version and returns 2 even on
			// its success path (a quirk of version.c); runCommand turns that
			// into the matching exit code 2, exactly as the C main() does.
			runCommand(versionCommand(args))
		} else {
			fmt.Fprintf(os.Stderr, "fh: unknown option '%s'\n", arg)
			os.Exit(2)
		}
	}
	fmt.Fprintf(os.Stdout, "fh: try `fh --help` for instructions\n")
	os.Exit(2)
}
