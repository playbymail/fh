package game

import (
	"fmt"
	"os"
)

// cEngineVersion is the version string printed by the C engine's `version`
// command (version.c). It tracks the ../Far-Horizons target tag — currently
// v7.5.12 — and MUST be bumped whenever that target version bumps. This is the
// C engine's version, deliberately distinct from the Go port's own version
// (see ../../version.go); the `version` command reproduces the C output, not
// the Go port's release number.
const cEngineVersion = "7.5.12"

// versionCommand is a faithful port of version.c's versionCommand. It prints
// the C engine version to stdout and returns 0 on success (v7.5.12 changed the
// success path from the historical `return 2` to `return 0`). --help/-h/-? and
// unknown options return 2 after writing usage/error text to stderr.
func versionCommand(args []string) int {
	for i := 1; i < len(args); i++ {
		opt, val, hasVal := splitOptVal(args[i])

		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "usage: fh version\n")
			fmt.Fprintf(os.Stderr, "       display version of this program\n")
			return 2
		} else {
			// Mirrors version.c's `'%s%s%s'` with opt, (val ? "=" : ""), and
			// (val ? val : "") — the v7.5.12 NULL-%s fix, which prints nothing
			// (not "(null)") when there is no value.
			if hasVal {
				fmt.Fprintf(os.Stderr, "error: unknown option '%s=%s'\n", opt, val)
			} else {
				fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			}
			return 2
		}
	}

	fmt.Printf("%s\n", cEngineVersion)

	return 0
}
