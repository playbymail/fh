package game

import (
	"fmt"
	"os"
)

func CommandRunner(argv []string) error {
	argc := len(argv)
	if argc == 1 {
		showHelp()
		return nil
	}

	for i := 1; i < argc; i++ {
		// Pass argv from the command name onward (mirrors C's
		// `cmd(argc - i, argv + i)`); the command functions expect
		// args[0] to be the command name.
		arg, args := argv[i], argv[i:]

		if arg == "?" || arg == "-?" || arg == "--help" {
			showHelp()
			return nil
		} else if arg == "-t" {
			test_mode = TRUE
		} else if arg == "-v" {
			verbose_mode = TRUE
		} else if arg == "combat" {
			rv := combatCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "create" {
			rv := createCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "export" {
			rv := exportCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "finish" {
			rv := finishCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "import" {
			rv := importCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "inspect" {
			rv := inspectCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "jump" {
			rv := jumpCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "list" {
			rv := listCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "locations" {
			rv := locationCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "logrnd" {
			// Mirrors C engine.c logRandomCommand; the Go port writes to
			// an io.Writer for testability, so pass os.Stdout here. The C
			// command ignores its args and always returns 0.
			rv := logRandomCommand(os.Stdout)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "post-arrival" {
			rv := postArrivalCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "pre-departure" {
			rv := preDepartureCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "production" {
			rv := productionCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "report" {
			rv := reportCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "scan" {
			rv := scanCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "scan-near" {
			rv := scanNearCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "show" {
			rv := showCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "stats" {
			rv := statsCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "turn" {
			rv := turnCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "update" {
			rv := updateCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else if arg == "version" {
			// versionCommand prints the C engine version and returns 2 on
			// its success path (a deliberate quirk of version.c). The C
			// main() propagates that as the process exit code, so `fh
			// version` exits 2. CommandRunner can only signal "non-zero" as
			// an error here, so fhc currently exits 1 instead — see the
			// known exit-code discrepancy noted in the port report.
			rv := versionCommand(args)
			if rv != 0 {
				return fmt.Errorf("fh: %s: exit %d", arg, rv)
			}
			return nil
		} else {
			return fmt.Errorf("fh: unknown option '%s'\n", arg)
		}
	}
	return fmt.Errorf("fh: try `fh --help` for instructions")
}
