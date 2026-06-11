package game

import (
	"fmt"
	"os"
)

// listCommand is a faithful port of list.c's listCommand.
func listCommand(args []string) int {
	_ = args[0] // cmdName (unused, as in C)

	// load data used to derive locations
	get_galaxy_data()
	get_star_data()
	get_planet_data()
	get_species_data()

	list_planets := TRUE
	list_wormholes := FALSE
	listGalaxy := FALSE
	listScanned := FALSE

	spno := 0
	spidx := -1
	var home_nampla *nampla_data_t = nil
	var home_planet *planet_data_t = nil

	for i := 1; i < len(args); i++ {
		opt, val, hasVal := splitOptVal(args[i])

		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "fh: usage: list galaxy options...\n")
			fmt.Fprintf(os.Stderr, "           list all systems in the galaxy\n")
			fmt.Fprintf(os.Stderr, "      opt: --planets=bool     list planets   [default is false]\n")
			fmt.Fprintf(os.Stderr, "           --wormholes=bool   list wormholes [default is false]\n")
			fmt.Fprintf(os.Stderr, "fh: usage: list scanned options...\n")
			fmt.Fprintf(os.Stderr, "           list all systems scanned by a species\n")
			fmt.Fprintf(os.Stderr, "      opt: --species=integer  species number to scan for [required]\n")
			return 2
		} else if opt == "--planets" && hasVal && listGalaxy != FALSE {
			if val == "true" {
				list_planets = TRUE
			} else if val == "false" {
				list_planets = FALSE
			} else {
				fmt.Fprintf(os.Stderr, "error: --planets requires either 'true' or 'false'\n")
				return 2
			}
		} else if opt == "--wormholes" && hasVal && listGalaxy != FALSE {
			if val == "true" {
				list_wormholes = TRUE
			} else if val == "false" {
				list_wormholes = FALSE
			} else {
				fmt.Fprintf(os.Stderr, "error: --wormholes requires either 'true' or 'false'\n")
				return 2
			}
		} else if opt == "--species" && hasVal {
			spno = cfgAtoi(val)
			spidx = spno - 1
			if !(1 <= spno && spno <= galaxy.num_species) {
				fmt.Fprintf(os.Stderr, "error: invalid species number '%s'\n", opt)
				return 2
			} else if data_in_memory[spidx] == FALSE {
				fmt.Fprintf(os.Stderr, "error: species %d is not loaded\n", spno)
				return 2
			}
		} else if opt == "galaxy" && !hasVal {
			if listScanned != FALSE {
				fmt.Fprintf(os.Stderr, "error: you must not specify both galaxy and scanned\n")
				return 2
			}
			listGalaxy = TRUE
		} else if opt == "scanned" && !hasVal {
			if listGalaxy != FALSE {
				fmt.Fprintf(os.Stderr, "error: you must not specify both galaxy and scanned\n")
				return 2
			}
			listScanned = TRUE
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	if listGalaxy == TRUE {
		/* Initialize counts. */
		total_planets := 0
		total_wormstars := 0
		var type_count [10]int
		for i := DWARF; i <= GIANT; i++ {
			type_count[i] = 0
		}

		/* For each star, list info. */
		// C: planet_data_t *planet = planet_base; (running pointer advanced
		// only inside the list_planets printing loop). We track its index.
		planetIdx := 0
		for star_index := 0; star_index < num_stars; star_index++ {
			star := star_base[star_index]
			if list_wormholes == FALSE {
				if list_planets != FALSE {
					fmt.Printf("System #%d:\t", star_index+1)
				}
				fmt.Printf("x = %d\ty = %d\tz = %d", star.x, star.y, star.z)
				fmt.Printf("\tstellar type = %c%c%c",
					type_char[star.star_type], color_char[star.color], size_char[star.size])
				if list_planets != FALSE {
					fmt.Printf("\t%d planets.", star.num_planets)
				}
				fmt.Printf("\n")
				if star.num_planets == 0 {
					fmt.Printf("\tStar #%d went nova! All planets were blown away!\n", star_index+1)
				}
			}

			total_planets += star.num_planets
			type_count[star.star_type] += 1

			if star.worm_here != FALSE {
				total_wormstars++
				if list_planets != FALSE {
					fmt.Printf("!!! Natural wormhole from here to %d %d %d\n",
						star.worm_x, star.worm_y, star.worm_z)
				} else if list_wormholes != FALSE {
					fmt.Printf("Wormhole #%d: from %d %d %d to %d %d %d\n",
						total_wormstars, star.x, star.y, star.z,
						star.worm_x, star.worm_y, star.worm_z)
					for i := 0; i < num_stars; i++ {
						worm_star := star_base[i]
						if star.worm_x == worm_star.x && star.worm_y == worm_star.y &&
							star.worm_z == worm_star.z {
							worm_star.worm_here = FALSE
							break
						}
					}
				}
			}

			home_system := FALSE
			// C: planet_data_t *home_planet = planet; (shadows the outer one)
			var gb_home_planet *planet_data_t = planet_base[planetIdx]
			if list_planets != FALSE {
				/* Check if system has a home planet. */
				// C bug, preserved: this reads planet+1 .. planet+num_planets
				// (skipping this star's first planet and including the next
				// star's first planet). On the last star the final read falls
				// into C's zeroed NUM_EXTRA_PLANETS headroom (special == 0);
				// the Go planet_base has no headroom, so an out-of-range index
				// yields the same "no match" result.
				for i := 1; i <= star.num_planets; i++ {
					idx := planetIdx + i
					if idx < num_planets {
						gb_home_planet = planet_base[idx]
						if gb_home_planet.special == 1 || gb_home_planet.special == 2 {
							home_system = TRUE
							break
						}
					}
				}
			}

			if list_planets != FALSE {
				for i := 1; i <= star.num_planets; i++ {
					planet := planet_base[planetIdx]
					switch planet.special {
					case 0:
						fmt.Printf("     ")
					case 1:
						fmt.Printf(" HOM ")
					case 2:
						fmt.Printf(" COL ")
					}
					fmt.Printf("#%d dia=%3d g=%d.%02d tc=%2d pc=%2d md=%d.%02d", i,
						planet.diameter,
						planet.gravity/100, planet.gravity%100,
						planet.temperature_class,
						planet.pressure_class,
						planet.mining_difficulty/100, planet.mining_difficulty%100)

					if home_system == FALSE {
						fmt.Printf("  ")
					} else {
						print_LSN(planet, gb_home_planet)
					}

					num_gases := 0
					for n := 0; n < 4; n++ {
						if planet.gas_percent[n] > 0 {
							if num_gases > 0 {
								fmt.Printf(",")
							}
							fmt.Printf("%s(%d%%)", gas_string[planet.gas[n]], planet.gas_percent[n])
							num_gases++
						}
					}
					if num_gases == 0 {
						fmt.Printf("No atmosphere")
					}
					fmt.Printf("\n")
					planetIdx++
				}
			}
			if list_planets != FALSE {
				fmt.Printf("\n")
			}
		}
		if list_wormholes != FALSE {
			return 0
		}
		/* Print summary. */
		fmt.Printf("\nThe galaxy has a radius of %d parsecs.\n", galaxy.radius)
		fmt.Printf("It contains %d dwarf stars, %d degenerate stars, ",
			type_count[DWARF], type_count[DEGENERATE])
		fmt.Printf("%d main sequence stars,\n    and %d giant stars, ",
			type_count[MAIN_SEQUENCE], type_count[GIANT])
		fmt.Printf("for a total of %d stars.\n", num_stars)
		if list_planets != FALSE {
			fmt.Printf("The total number of planets in the galaxy is %d.\n", total_planets)
			fmt.Printf("The total number of natural wormholes in the galaxy is %d.\n",
				total_wormstars/2)
			fmt.Printf("The galaxy was designed for %d species.\n", galaxy.d_num_species)
			fmt.Printf("A total of %d species have been designated so far.\n\n",
				galaxy.num_species)
		}
		return 0
	}

	if listScanned == TRUE {
		if spno == 0 {
			fmt.Fprintf(os.Stderr, "error: you must specify species to scan for\n")
			return 2
		}
		species = &spec_data[spidx]
		species_number = spno
		nampla_base = namp_data[spidx]
		ship_base = ship_data[spidx]
		home_nampla = namp_data[spidx][0]
		home_planet = planet_base[home_nampla.planet_index]
		for j := 0; j < num_stars; j++ {
			star := star_base[j]
			// list only if the species has visited this system
			if (star.visited_by[spidx/32] & (1 << (spidx % 32))) == 0 {
				continue
			}
			fmt.Printf("System  x = %d  y = %d  z = %d\n", star.x, star.y, star.z)
			if star.worm_here == TRUE {
				fmt.Printf("\tThis star system is the terminus of a natural wormhole.\n")
			}
			/* Check for nova. */
			if star.num_planets == 0 {
				fmt.Printf("\tThis star is a nova remnant.\n")
				fmt.Printf("\t Any planets it may have once had have been blown away.\n\n")
			} else {
				// list all the planets and colonies in the system
				fmt.Printf("\t#  Grav  MIDiff  LSN   Details\n")
				fmt.Printf("\t-----------------------------------------------------\n")
				for pidx := 0; pidx < star.num_planets; pidx++ {
					planet := planet_base[star.planet_index+pidx]
					pn := pidx + 1
					lsFlag := byte(' ')
					lsNeeded := life_support_needed(species, home_planet, planet)
					if lsNeeded <= species.tech_level[LS] {
						lsFlag = '*'
					}
					fmt.Printf("\t%d  %d.%02d  %3d.%02d %4d%c  ",
						pn,
						planet.gravity/100, planet.gravity%100,
						planet.mining_difficulty/100, planet.mining_difficulty%100,
						lsNeeded, lsFlag)
					// is there a colony here? print the colony and inventory.
					for npidx := 0; npidx < species.num_namplas; npidx++ {
						colony := nampla_base[npidx]
						if star.x != colony.x || star.y != colony.y || star.z != colony.z ||
							pn != colony.pn {
							continue
						}
						fmt.Printf("PL %s", colony.name)
						for item := 0; item < MAX_ITEMS; item++ {
							if colony.item_quantity[item] > 0 {
								fmt.Printf("\n\t                       %s %s %d",
									item_abbr[item], item_name[item], colony.item_quantity[item])
							}
						}
					}
					// are there any ships here? print the ship and inventory.
					for shidx := 0; shidx < species.num_ships; shidx++ {
						ship := ship_base[shidx]
						if star.x != ship.x || star.y != ship.y || star.z != ship.z || pn != ship.pn {
							continue
						}
						fmt.Printf("\n\t                    -- %s", ship_name(ship))
						for item := 0; item < MAX_ITEMS; item++ {
							if ship.item_quantity[item] > 0 {
								fmt.Printf("\n\t                       %s %s %d",
									item_abbr[item], item_name[item], ship.item_quantity[item])
							}
						}
						if ship.loading_point != 0 {
							var colony *nampla_data_t = nil
							if ship.loading_point == 9999 {
								// use homeworld
								colony = nampla_base[0]
							} else if 0 < ship.loading_point && ship.loading_point < species.num_namplas {
								colony = nampla_base[ship.loading_point]
							}
							if colony == nil {
								fmt.Printf("\n\t                       load   %5d  ***internal error***",
									ship.loading_point)
							} else {
								fmt.Printf("\n\t                       load   PL %s", colony.name)
							}
						}
						if ship.unloading_point != 0 {
							var colony *nampla_data_t = nil
							if ship.unloading_point == 9999 {
								// use homeworld
								colony = nampla_base[0]
							} else if 0 < ship.unloading_point && ship.unloading_point < species.num_namplas {
								colony = nampla_base[ship.unloading_point]
							}
							if colony == nil {
								fmt.Printf("\n\t                       unload %5d  ***internal error***",
									ship.unloading_point)
							} else {
								fmt.Printf("\n\t                       unload PL %s", colony.name)
							}
						}
					}
					fmt.Printf("\n")
				}
				// are there any ships in deep orbit? print the ship and inventory.
				foundDeepOrbiters := FALSE
				for shidx := 0; shidx < species.num_ships; shidx++ {
					ship := ship_base[shidx]
					if star.x != ship.x || star.y != ship.y || star.z != ship.z || ship.pn != 0 {
						continue
					}
					if foundDeepOrbiters == FALSE {
						fmt.Printf("\n\tShips in deep orbit -------------------------------")
						foundDeepOrbiters = TRUE
					}
					fmt.Printf("\n\t                    -- %s", ship_name(ship))
					for item := 0; item < MAX_ITEMS; item++ {
						if ship.item_quantity[item] > 0 {
							fmt.Printf("\n\t                       %s %s %d",
								item_abbr[item], item_name[item], ship.item_quantity[item])
						}
					}
				}
				// are there any aliens here? print something?
				fmt.Printf("\n")
			}
		}
		return 0
	}

	fmt.Fprintf(os.Stderr, "error: you must specify galaxy or scanned\n")
	return 2
}

// print_LSN is a faithful port of list.c's print_LSN. It provides an
// approximate LSN for a planet, assuming oxygen is required and any gas that
// does not appear on the home planet is poisonous.
func print_LSN(planet *planet_data_t, home_planet *planet_data_t) {
	j := planet.temperature_class - home_planet.temperature_class
	if j < 0 {
		j = -j
	}
	ls_needed := 2 * j /* Temperature class. */

	j = planet.pressure_class - home_planet.pressure_class
	if j < 0 {
		j = -j
	}
	ls_needed += 2 * j /* Pressure class. */

	/* Check gases. */
	ls_needed += 2 // assumes oxygen is not present
	for j = 0; j < 4; j++ {
		if planet.gas[j] != 0 {
			if planet.gas[j] == O2 {
				ls_needed -= 2
			}
			poison := TRUE
			for k := 0; k < 4; k++ {
				/* Compare with home planet. */
				if planet.gas[j] == home_planet.gas[k] {
					poison = FALSE
					break
				}
			}
			if poison != FALSE {
				ls_needed += 2
			}
		}
	}
	fmt.Printf("%4d ", ls_needed)
}
