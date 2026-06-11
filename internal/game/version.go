package game

import (
	"fmt"
	"os"
)

// cEngineVersion is the version string printed by the C engine's `version`
// command (version.c). It tracks the ../Far-Horizons target tag — currently
// v7.5.11 — and MUST be bumped whenever that target version bumps. This is the
// C engine's version, deliberately distinct from the Go port's own version
// (see ../../version.go); the `version` command reproduces the C output, not
// the Go port's release number.
const cEngineVersion = "7.5.11"

// versionCommand is a faithful port of version.c's versionCommand. It prints
// the C engine version to stdout and returns 2. The C command returns 2 even
// on the success path (a quirk of the original); --help/-h/-? and unknown
// options also return 2 after writing usage/error text to stderr.
func versionCommand(args []string) int {
	for i := 1; i < len(args); i++ {
		opt, val, hasVal := splitOptVal(args[i])

		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "usage: fh version\n")
			fmt.Fprintf(os.Stderr, "       display version of this program\n")
			return 2
		} else {
			if hasVal {
				fmt.Fprintf(os.Stderr, "error: unknown option '%s=%s'\n", opt, val)
			} else {
				fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			}
			return 2
		}
	}

	fmt.Printf("%s\n", cEngineVersion)

	return 2
}
