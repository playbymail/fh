// Package main implements the Far Horizons CLI.
package main

import (
	"os"

	"github.com/playbymail/fh/internal/game"
)

func main() {
	// CommandRunner dispatches the command and terminates the process with the
	// C engine's exit code (0 on success, 2 on any error); the help paths
	// return here and the process exits 0 normally.
	game.CommandRunner(os.Args)
}
