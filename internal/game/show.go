package game

import (
	"fmt"
	"os"
)

// star_here mirrors show.c's file-static int star_here[MAX_DIAMETER][MAX_DIAMETER].
// Both map routines fully reinitialize it before use, so its persistence between
// calls is irrelevant; it is declared here (not reset by ResetState) to match the
// C static.
var star_here [MAX_DIAMETER][MAX_DIAMETER]int

// showCommand is a faithful port of show.c's showCommand.
func showCommand(args []string) int {
	sep := ""

	for i := 1; i < len(args); i++ {
		opt, _, hasVal := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "usage: show (galaxy | help | version | _game_value_)\n")
			fmt.Fprintf(os.Stderr, "  opt: _game_value_    shows the current value of a game setting\n")
			fmt.Fprintf(os.Stderr, "         d_num_species          maximum number of species\n")
			fmt.Fprintf(os.Stderr, "         num_planets            number of planets in cluster\n")
			fmt.Fprintf(os.Stderr, "         num_species            number of planets in cluster\n")
			fmt.Fprintf(os.Stderr, "         num_stars              number of stars in cluster\n")
			fmt.Fprintf(os.Stderr, "         num_natural_wormholes  number of natural wormholes in cluster\n")
			fmt.Fprintf(os.Stderr, "         radius                 radius of cluster\n")
			fmt.Fprintf(os.Stderr, "         turn_number            current turn number\n")
			return 2
		} else if opt == "-v" && !hasVal {
			verbose_mode = TRUE
		} else if opt == "d_num_species" && !hasVal {
			get_galaxy_data()
			fmt.Printf("%s%d", sep, galaxy.d_num_species)
			sep = " "
		} else if opt == "galaxy" && !hasVal {
			return showGalaxyCommand(args[1:])
		} else if opt == "help" && !hasVal {
			return showHelp()
		} else if opt == "num_planets" && !hasVal {
			get_galaxy_data()
			get_star_data()
			get_planet_data()
			fmt.Printf("%s%d", sep, num_planets)
			sep = " "
		} else if opt == "num_species" && !hasVal {
			get_galaxy_data()
			fmt.Printf("%s%d", sep, galaxy.num_species)
			sep = " "
		} else if opt == "num_stars" && !hasVal {
			get_galaxy_data()
			get_star_data()
			fmt.Printf("%s%d", sep, num_stars)
			sep = " "
		} else if opt == "num_natural_wormholes" && !hasVal {
			get_galaxy_data()
			get_star_data()
			fmt.Printf("%s%d", sep, num_natural_wormholes)
			sep = " "
		} else if opt == "radius" && !hasVal {
			get_galaxy_data()
			fmt.Printf("%s%d", sep, galaxy.radius)
			sep = " "
		} else if opt == "turn_number" && !hasVal {
			get_galaxy_data()
			fmt.Printf("%s%d", sep, galaxy.turn_number)
			sep = " "
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	if len(sep) > 0 {
		fmt.Printf("\n")
	}

	return 0
}

// showGalaxyCommand is a faithful port of show.c's showGalaxyCommand.
func showGalaxyCommand(args []string) int {
	for i := 1; i < len(args); i++ {
		opt, _, hasVal := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "usage: show galaxy [--ascii]\n")
			fmt.Fprintf(os.Stderr, "ascii: display a crude ASCII map of the galaxy  with the relative positions\n")
			fmt.Fprintf(os.Stderr, "       of home planets, ideal colonies, and other star systems. the GM may\n")
			fmt.Fprintf(os.Stderr, "       run this program after creating a new galaxy to visually confirm the\n")
			fmt.Fprintf(os.Stderr, "       distribution is not too lopsided.\n")
			fmt.Fprintf(os.Stderr, "       in the map, 'H' indicates ideal home planet, 'C' ideal colony, and '*'\n")
			fmt.Fprintf(os.Stderr, "       is used for all other stars.\n")
			return 2
		} else if opt == "--ascii" && !hasVal {
			return showGalaxyAsciiMap()
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	return showGalaxyMap()
}

// showGalaxyAsciiMap is a faithful port of show.c's showGalaxyAsciiMap.
func showGalaxyAsciiMap() int {
	/* Get all the raw data. */
	get_galaxy_data()
	get_star_data()
	get_planet_data()

	galactic_diameter := 2 * galaxy.radius

	/* For each star, set corresponding element of star_here[] to index into star array. */
	for x := 0; x < galactic_diameter; x++ {
		/* Initialize array. */
		for y := 0; y < galactic_diameter; y++ {
			star_here[x][y] = -1
		}
	}

	// bug: if multiple systems have the same x and y, only one system's index gets saved
	for star_index := 0; star_index < num_stars; star_index++ {
		star := star_base[star_index]
		star_here[star.x][star.y] = star_index
	}

	fmt.Printf("+")
	for i := 0; i < galactic_diameter; i++ {
		fmt.Printf("-")
	}
	fmt.Printf("+\n")

	/* Outermost loop will control y-coordinates. */
	for y := galactic_diameter - 1; y >= 0; y-- {
		/* Innermost loop will control x-coordinate. */
		for x := 0; x <= galactic_diameter; x++ {
			if x == 0 {
				fmt.Printf("|")
			} else if x == galactic_diameter {
				fmt.Printf("|\n")
				continue
			}

			star_index := star_here[x][y]
			if star_index == -1 {
				fmt.Printf(" ")
			} else {
				star := star_base[star_index]
				special := 0
				pi := star.planet_index
				for i := 0; special == 0 && i < star.num_planets; i++ {
					planet := planet_base[pi]
					if planet.special != 0 {
						special = planet.special
					}
					pi++
				}

				switch special {
				case 0:
					fmt.Printf(".")
				case 1:
					fmt.Printf("H")
				case 2:
					fmt.Printf("C")
				case 3:
					fmt.Printf("R")
				default:
					// C: printf("%d", planet->special) using the planet pointer
					// left one past the last inspected planet.
					fmt.Printf("%d", planet_base[pi].special)
				}
			}
		}
	}

	fmt.Printf("+")
	for i := 0; i < galactic_diameter; i++ {
		fmt.Printf("-")
	}
	fmt.Printf("+\n")

	fmt.Printf("    H - ideal home planet\n")
	fmt.Printf("    C - ideal colony\n")
	fmt.Printf("    . - all other star systems\n")

	return 0
}

// showGalaxyMap is a faithful port of show.c's showGalaxyMap. It writes the map
// to the file galaxy.map and prints the page count to stdout.
func showGalaxyMap() int {
	/* Get all the raw data. */
	get_galaxy_data()
	get_star_data()

	galactic_diameter := 2 * galaxy.radius

	/* Determine number of pages that will be needed to contain the complete map. */
	n_columns := 132

	x_increment := (n_columns - 4) / 6 /* 4 columns for left margin plus 6 per star. */
	page_count := (2*galaxy.radius + x_increment - 1) / x_increment

	fmt.Printf("\nI will generate %d page(s).\n\n", page_count)

	// initialize array to hold -1, which means "no system here"
	for x := 0; x < MAX_DIAMETER; x++ {
		for y := 0; y < MAX_DIAMETER; y++ {
			star_here[x][y] = -1
		}
	}

	/* For each star, set corresponding element of star_here[] to index into star array. */
	for star_index := 0; star_index < num_stars; star_index++ {
		star := star_base[star_index]
		star_here[star.x][star.y] = star_index
	}

	/* Create output file. */
	outfile, err := os.Create("galaxy.map")
	if err != nil {
		fmt.Fprintf(os.Stderr, "showGalaxyMap:: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create file galaxy.map!\n")
		os.Exit(2)
	}

	/* Outermost loop will count pages. */
	left_x := 0
	for page := 1; page <= page_count; page++ {
		/* Next-to-outermost loop will control y-coordinates. */
		for y := 2*galaxy.radius - 1; y >= 0; y-- {
			/* Next-to-innermost loop will count the 4 lines that make up each star box.
			 * Fifth and sixth lines are generated only at the very bottom of the page. */
			for line := 1; line <= 6; line++ {
				x := left_x

				/* Do left margin of first page. */
				if x == 0 && page == 1 {
					switch line {
					case 1:
						fmt.Fprintf(outfile, "   -")
					case 2:
						fmt.Fprintf(outfile, "   |")
					case 3:
						fmt.Fprintf(outfile, "%2d |", y)
					case 4:
						if n_columns < 100 {
							fmt.Fprintf(outfile, "   |")
						}
					case 5:
						if y == 0 {
							fmt.Fprintf(outfile, " Y -")
						}
					case 6:
						if y == 0 {
							fmt.Fprintf(outfile, "  X ")
						}
					}
				}

				/* Innermost loop will control x-coordinate. */
				for x_count := 1; x_count <= x_increment; x_count++ {
					if x == galactic_diameter {
						break
					}
					star_index := star_here[x][y]
					star := star_base[0]
					if star_index > 0 {
						star = star_base[star_index]
					}

					switch line {
					case 1:
						fmt.Fprintf(outfile, "------")
					case 2:
						if star_index >= 0 {
							z := star.z
							if z < 10 {
								fmt.Fprintf(outfile, "%3d  |", z)
							} else {
								fmt.Fprintf(outfile, "%4d |", z)
							}
						} else {
							fmt.Fprintf(outfile, "     |")
						}
					case 3:
						if star_index >= 0 {
							fmt.Fprintf(outfile, " %c%c%c |", type_char[star.star_type], color_char[star.color],
								size_char[star.size])
						} else {
							fmt.Fprintf(outfile, "     |")
						}
					case 4:
						if n_columns < 100 {
							fmt.Fprintf(outfile, "     |")
						}
					case 5:
						if y == 0 {
							fmt.Fprintf(outfile, "------")
						}
					case 6:
						if y == 0 {
							fmt.Fprintf(outfile, "  %2d  ", x)
						}
					}
					x++
				}

				if (line < 4) || (line == 4 && n_columns < 100) {
					/* End of line. */
					fmt.Fprintf(outfile, "\n")
				}

				if y == 0 && line == 5 {
					fmt.Fprintf(outfile, "\n")
				}
			}
		}

		fmt.Fprintf(outfile, "\n\f") /* Formfeed character. */
		left_x += x_increment
	}

	/* Clean up and exit. */
	_ = outfile.Close()
	return 0
}

// showHelp is a faithful port of show.c's showHelp.
func showHelp() int {
	fmt.Printf("usage: fh [options...] command [arguments...]\n")
	fmt.Printf("  opt: --help          show this helpful text and exit\n")
	fmt.Printf("       -t | --test     enable test mode\n")
	fmt.Printf("       -v | --verbose  enable verbose mode\n")
	fmt.Printf("  cmd: turn            display the current turn number\n")
	fmt.Printf("       locations       create locations data file and update\n")
	fmt.Printf("                       economic efficiency in planets data file\n")
	fmt.Printf("       combat          run combat commands\n")
	fmt.Printf("       pre-departure   run pre-departure commands\n")
	fmt.Printf("       jump            run jump commands\n")
	fmt.Printf("       production      run production commands\n")
	fmt.Printf("       post-arrival    run post-arrival commands\n")
	fmt.Printf("       finish          run end of turn logic\n")
	fmt.Printf("       report          create end of turn reports\n")
	fmt.Printf("       stats           display statistics\n")
	fmt.Printf("       create          create a new galaxy, home system templates\n")
	fmt.Printf("       convert         convert between binary and json formats\n")
	fmt.Printf("       export          convert binary .dat to json or s-expression\n")
	fmt.Printf("       logrnd          display a list of random values for testing the PRNG\n")
	fmt.Printf("       scan            display a species-specific scan for a location\n")
	fmt.Printf("       scan-near       display all ships and colonies near a location\n")
	fmt.Printf("       set             update values for planet, species, or star\n")
	fmt.Printf("       version         display version of this program\n")

	return 0
}
