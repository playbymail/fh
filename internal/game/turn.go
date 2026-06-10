package game

// Port of turn.c.

import (
	"fmt"
	"os"
)

// turnCommand displays the current turn number.
// args mirrors the C argv: args[0] is the command name (the C dispatcher
// always passes "turn"); any additional argument is an error, as in C.
func turnCommand(args []string) int {
	cmdName := args[0]

	// check for valid command line
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "fh: %s: invalid option '%s'\n", cmdName, args[1])
		return 2
	}

	// load the galaxy data and then print the current turn number
	get_galaxy_data()
	fmt.Printf("%d\n", galaxy.turn_number)

	return 0
}
