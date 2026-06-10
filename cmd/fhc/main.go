// Package main implements the Far Horizons CLI.
package main

import (
	"fmt"
	"os"

	"github.com/playbymail/fh/internal/game"
)

func main() {
	if err := game.CommandRunner(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
