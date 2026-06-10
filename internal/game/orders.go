package game

// Port of orders.c.

import (
	"fmt"
	"os"
)

// createOrders creates an orders file for every species that does
// not currently have one.
// `advanced` and `reminder` are not currently used.
func createOrders(advanced, reminder int) int {
	_ = advanced
	_ = reminder

	/* Get all necessary data. */
	get_galaxy_data()
	get_star_data()
	get_planet_data()
	get_species_data()
	get_location_data()

	truncate_name = TRUE

	/* Major loop. Check each species in the game. */
	for species_index = 0; species_index < galaxy.num_species; species_index++ {
		species_number = species_index + 1

		// check if this species is still in the game
		if data_in_memory[species_index] == FALSE {
			fmt.Fprintf(os.Stderr, " warn: createOrders: species %2d is not in memory\n", species_number)
			continue
		}

		// check if we have an orders file for this species
		filename := fmt.Sprintf("sp%02d.ord", species_number)
		if _, err := os.Stat(filename); err == nil {
			// file exists
			continue
		}

		// no file, so do our thing, whatever that is
		NoOrdersForSpecies()
	}

	return 0
}

func NoOrdersForSpecies() {
	var i int
	var j int
	var k int
	var nampla_index int
	var found int
	var n int
	var nn int
	// The C code declares a local FILE *log_file that shadows the global;
	// the local nampla and ship pointers shadow the globals the same way.
	var log_file *os.File
	var nampla *nampla_data_t
	var home_nampla *nampla_data_t
	var temp_nampla *nampla_data_t
	var ship *ship_data_t

	species = &spec_data[species_index]
	nampla_base = namp_data[species_index]
	ship_base = ship_data[species_index]
	home_nampla = nampla_base[0]
	home_planet = planet_base[home_nampla.planet_index]

	for i = 0; i < species.num_ships; i++ {
		ship = ship_base[i]
		ship.special = 0
	}

	/* Print message for gamemaster. */
	fmt.Printf("Generating orders for species #%02d, SP %s...\n", species_number, species.name)

	/* Open message file. */
	filename := "noorders.txt"
	message_file := fopen_r(filename)
	if message_file == nil {
		fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for reading!\n\n", filename)
		os.Exit(2)
	}

	/* Open log file. */
	filename = fmt.Sprintf("sp%02d.log", species_number)
	log_file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for appending!\n\n", filename)
		os.Exit(2)
	}

	/* Copy message to log file. */
	for {
		message_line, ok := readln(message_file, 131)
		if !ok {
			break
		}
		fmt.Fprint(log_file, message_line)
	}

	message_file.fclose()
	log_file.Close()

	/* Open orders file for writing. */
	filename = fmt.Sprintf("sp%02d.ord", species_number)
	orders_file, err = os.Create(filename)
	if err != nil {
		orders_file = nil
		fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for writing!\n\n", filename)
		os.Exit(2)
	}

	/* Issue PRE-DEPARTURE orders. */
	fmt.Fprintf(orders_file, "START PRE-DEPARTURE\n")
	fmt.Fprintf(orders_file, "; Place pre-departure orders here.\n\n")

	for nampla_index = 0; nampla_index < species.num_namplas; nampla_index++ {
		nampla = nampla_base[nampla_index]
		if nampla.pn == 99 {
			continue
		}

		/* Generate auto-installs for colonies that were loaded via the DEVELOP command. */
		if nampla.auto_IUs != 0 {
			fmt.Fprintf(orders_file, "\tInstall\t%d IU\tPL %s\n", nampla.auto_IUs, nampla.name)
		}
		if nampla.auto_AUs != 0 {
			fmt.Fprintf(orders_file, "\tInstall\t%d AU\tPL %s\n", nampla.auto_AUs, nampla.name)
		}
		if nampla.auto_IUs != 0 || nampla.auto_AUs != 0 {
			fmt.Fprintf(orders_file, "\n")
		}

		nampla.item_quantity[CU] -= nampla.auto_IUs + nampla.auto_AUs

		/* Generate auto UNLOAD orders for transports at this nampla. */
		for j = 0; j < species.num_ships; j++ {
			ship = ship_base[j]
			if ship.pn == 99 {
				continue
			} else if ship.x != nampla.x {
				continue
			} else if ship.y != nampla.y {
				continue
			} else if ship.z != nampla.z {
				continue
			} else if ship.pn != nampla.pn {
				continue
			} else if ship.status == JUMPED_IN_COMBAT {
				continue
			} else if ship.status == FORCED_JUMP {
				continue
			} else if ship.class != TR {
				continue
			} else if ship.item_quantity[CU] < 1 {
				continue
			}

			/* New colonies will never be started automatically unless ship was loaded via a DEVELOP order. */
			if ship.loading_point != 0 { /* Check if transport is at specified unloading point. */
				n = ship.unloading_point
				if n == nampla_index || (n == 9999 && nampla_index == 0) {
					goto unload_ship
				}
			}

			if (nampla.status & POPULATED) == 0 {
				continue
			} else if (nampla.mi_base + nampla.ma_base) >= 2000 {
				continue
			} else if nampla.x == nampla_base[0].x && nampla.y == nampla_base[0].y && nampla.z == nampla_base[0].z {
				/* Home sector. */
				continue
			}

		unload_ship:

			n = ship.loading_point
			if n == 9999 {
				/* Home planet. */
				n = 0
			}
			if n == nampla_index {
				/* Ship was just loaded here. */
				continue
			}

			fmt.Fprintf(orders_file, "\tUnload\tTR%d%s %s\n\n", ship.tonnage, ship_type[ship.ship_type], ship.name)

			nampla.item_quantity[CU] = 0

			ship.special = ship.loading_point
			n = nampla_index /* n = nampla - nampla_base */
			if n == 0 {
				n = 9999
			}
			ship.unloading_point = n
		}

		if nampla.status&HOME_PLANET != 0 {
			continue
		} else if nampla.item_quantity[CU] == 0 {
			continue
		} else if nampla.item_quantity[IU] == 0 && nampla.item_quantity[AU] == 0 {
			continue
		}

		if nampla.item_quantity[IU] > 0 {
			fmt.Fprintf(orders_file, "\tInstall\t0 IU\tPL %s\n", nampla.name)
		}
		if nampla.item_quantity[AU] > 0 {
			fmt.Fprintf(orders_file, "\tInstall\t0 AU\tPL %s\n\n", nampla.name)
		}
	}

	fmt.Fprintf(orders_file, "END\n\n")

	fmt.Fprintf(orders_file, "START JUMPS\n")
	fmt.Fprintf(orders_file, "; Place jump orders here.\n\n")

	/* Initialize to make sure ships are not given more than one JUMP order. */
	for i = 0; i < species.num_ships; i++ {
		ship = ship_base[i]
		ship.just_jumped = FALSE
	}

	/* Generate auto-jumps for ships that were loaded via the DEVELOP command or which were UNLOADed because of the AUTO command. */
	for i = 0; i < species.num_ships; i++ {
		ship = ship_base[i]

		if ship.status == JUMPED_IN_COMBAT {
			continue
		} else if ship.status == FORCED_JUMP {
			continue
		} else if ship.pn == 99 {
			continue
		} else if ship.just_jumped != FALSE {
			continue
		}

		j = ship.special
		if j != 0 {
			if j == 9999 {
				/* Home planet. */
				j = 0
			}
			temp_nampla = nampla_base[j]
			fmt.Fprintf(orders_file, "\tJump\t%s, PL %s\t; ", ship_name(ship), temp_nampla.name)
			printMishapChanceToOrders(ship, temp_nampla.x, temp_nampla.y, temp_nampla.z)
			fmt.Fprintf(orders_file, "\n\n")
			ship.just_jumped = TRUE
			continue
		}

		n = ship.unloading_point
		if n != 0 {
			if n == 9999 {
				/* Home planet. */
				n = 0
			}
			temp_nampla = nampla_base[n]
			if temp_nampla.x == ship.x && temp_nampla.y == ship.y && temp_nampla.z == ship.z {
				continue
			}
			fmt.Fprintf(orders_file, "\tJump\t%s, PL %s\t; ", ship_name(ship), temp_nampla.name)
			printMishapChanceToOrders(ship, temp_nampla.x, temp_nampla.y, temp_nampla.z)
			fmt.Fprintf(orders_file, "\n\n")
			ship.just_jumped = TRUE
		}
	}

	/* Generate JUMP orders for all TR1s. */
	for i = 0; i < species.num_ships; i++ {
		ship = ship_base[i]
		if ship.pn == 99 {
			continue
		} else if ship.status == UNDER_CONSTRUCTION {
			continue
		} else if ship.status == JUMPED_IN_COMBAT {
			continue
		} else if ship.status == FORCED_JUMP {
			continue
		} else if ship.just_jumped != FALSE {
			continue
		}

		if ship.class == TR && ship.tonnage == 1 && ship.ship_type == FTL {
			fmt.Fprintf(orders_file, "\tJump\tTR1 %s, ", ship.name)
			closest_unvisited_star(ship)
			fmt.Fprintf(orders_file, "\n\t\t\t; Age %d, now at %d %d %d, ", ship.age, ship.x, ship.y, ship.z)
			printMishapChanceToOrders(ship, x, y, z)
			ship.dest_x = x
			ship.dest_y = y
			ship.dest_z = z
			fmt.Fprintf(orders_file, "\n\n")
			ship.just_jumped = TRUE
		}
	}

	fmt.Fprintf(orders_file, "END\n\n")

	fmt.Fprintf(orders_file, "START PRODUCTION\n")

	/* Generate a PRODUCTION order for each planet that can produce. */
	for nampla_index = species.num_namplas - 1; nampla_index >= 0; nampla_index-- {
		nampla = nampla_base[nampla_index]
		if nampla.pn == 99 {
			continue
		} else if nampla.mi_base == 0 && (nampla.status&RESORT_COLONY) == 0 {
			continue
		} else if nampla.ma_base == 0 && (nampla.status&MINING_COLONY) == 0 {
			continue
		}
		fmt.Fprintf(orders_file, "    PRODUCTION PL %s\n", nampla.name)
		if nampla.status&MINING_COLONY != 0 {
			fmt.Fprintf(orders_file, "    ; The above PRODUCTION order is required for this mining colony, even\n")
			fmt.Fprintf(orders_file, "    ;  if no other production orders are given for it.\n")
		} else if nampla.status&RESORT_COLONY != 0 {
			fmt.Fprintf(orders_file, "    ; The above PRODUCTION order is required for this resort colony, even\n")
			fmt.Fprintf(orders_file, "    ;  though no other production orders can be given for it.\n")
		} else {
			fmt.Fprintf(orders_file, "    ; Place production orders here for planet %s.\n\n", nampla.name)
		}

		/* Build IUs and AUs for incoming ships with CUs. */
		if nampla.IUs_needed != 0 {
			fmt.Fprintf(orders_file, "\tBuild\t%d IU\n", nampla.IUs_needed)
		}
		if nampla.AUs_needed != 0 {
			fmt.Fprintf(orders_file, "\tBuild\t%d AU\n", nampla.AUs_needed)
		}
		if nampla.IUs_needed != 0 || nampla.AUs_needed != 0 {
			fmt.Fprintf(orders_file, "\n")
		}
		if nampla.status&MINING_COLONY != 0 {
			continue
		} else if nampla.status&RESORT_COLONY != 0 {
			continue
		}

		/* See if there are any RMs to recycle. */
		n = nampla.special / 5
		if n > 0 {
			fmt.Fprintf(orders_file, "\tRecycle\t%d RM\n\n", 5*n)
		}

		/* Generate DEVELOP commands for ships arriving here because of AUTO command. */
		for i = 0; i < species.num_ships; i++ {
			ship = ship_base[i]
			if ship.pn == 99 {
				continue
			}
			k = ship.special
			if k == 0 {
				continue
			}
			if k == 9999 {
				/* Home planet. */
				k = 0
			}
			if nampla != nampla_base[k] {
				continue
			}
			k = ship.unloading_point
			if k == 9999 {
				/* Home planet. */
				k = 0
			}
			temp_nampla = nampla_base[k]
			fmt.Fprintf(orders_file, "\tDevelop\tPL %s, TR%d%s %s\n\n", temp_nampla.name, ship.tonnage,
				ship_type[ship.ship_type], ship.name)
		}

		/* Give orders to continue construction of unfinished ships and starbases. */
		for i = 0; i < species.num_ships; i++ {
			ship = ship_base[i]
			if ship.pn == 99 {
				continue
			} else if ship.x != nampla.x {
				continue
			} else if ship.y != nampla.y {
				continue
			} else if ship.z != nampla.z {
				continue
			} else if ship.pn != nampla.pn {
				continue
			} else if ship.status == UNDER_CONSTRUCTION {
				fmt.Fprintf(orders_file, "\tContinue\t%s, %d\t; Left to pay = %d\n\n", ship_name(ship),
					ship.remaining_cost, ship.remaining_cost)
				continue
			} else if ship.ship_type != STARBASE {
				continue
			}
			j = (species.tech_level[MA] / 2) - ship.tonnage
			if j < 1 {
				continue
			}
			fmt.Fprintf(orders_file, "\tContinue\tBAS %s, %d\t; Current tonnage = %s\n\n", ship.name, 100*j,
				commas(10000*ship.tonnage))
		}

		/* Generate DEVELOP command if this is a colony with an economic base less than 200. */
		n = nampla.mi_base + nampla.ma_base + nampla.IUs_needed + nampla.AUs_needed
		if (nampla.status&COLONY) != 0 && n < 2000 && nampla.pop_units > 0 {
			if nampla.pop_units > (2000 - n) {
				nn = 2000 - n
			} else {
				nn = nampla.pop_units
			}
			fmt.Fprintf(orders_file, "\tDevelop\t%d\n\n", 2*nn)
			nampla.IUs_needed += nn
		}

		/* For home planets and any colonies that have an economic base of at least 200, check if there are other colonized planets in the same sector that are not self-sufficient.  If so, DEVELOP them. */
		if n >= 2000 || (nampla.status&HOME_PLANET) != 0 {
			for i = 1; i < species.num_namplas; i++ { /* Skip HP. */
				if i == nampla_index {
					continue
				}
				temp_nampla = nampla_base[i]
				if temp_nampla.pn == 99 {
					continue
				} else if temp_nampla.x != nampla.x {
					continue
				} else if temp_nampla.y != nampla.y {
					continue
				} else if temp_nampla.z != nampla.z {
					continue
				}
				n = temp_nampla.mi_base + temp_nampla.ma_base + temp_nampla.IUs_needed + temp_nampla.AUs_needed
				if n == 0 {
					continue
				}
				nn = temp_nampla.item_quantity[IU] + temp_nampla.item_quantity[AU]
				if nn > temp_nampla.item_quantity[CU] {
					nn = temp_nampla.item_quantity[CU]
				}
				n += nn
				if n >= 2000 {
					continue
				}
				nn = 2000 - n
				if nn > nampla.pop_units {
					nn = nampla.pop_units
				}
				fmt.Fprintf(orders_file, "\tDevelop\t%d\tPL %s\n\n", 2*nn, temp_nampla.name)
				temp_nampla.AUs_needed += nn
			}
		}
	}

	fmt.Fprintf(orders_file, "END\n\n")

	fmt.Fprintf(orders_file, "START POST-ARRIVAL\n")
	fmt.Fprintf(orders_file, "; Place post-arrival orders here.\n\n")

	/* Generate an AUTO command. */
	fmt.Fprintf(orders_file, "\tAuto\n\n")

	/* Generate SCAN orders for all TR1s in sectors that current species does not inhabit. */
	for i = 0; i < species.num_ships; i++ {
		ship = ship_base[i]
		if ship.pn == 99 {
			continue
		} else if ship.status == UNDER_CONSTRUCTION {
			continue
		} else if ship.class != TR {
			continue
		} else if ship.tonnage != 1 {
			continue
		} else if ship.ship_type != FTL {
			continue
		} else if ship.dest_x == -1 {
			/* Not jumping anywhere. */
			continue
		}
		found = FALSE
		for j = 1; j < species.num_namplas; j++ {
			/* Skip home sector. */
			nampla = nampla_base[j]
			if nampla.pn == 99 {
				continue
			} else if nampla.x != ship.dest_x {
				continue
			} else if nampla.y != ship.dest_y {
				continue
			} else if nampla.z != ship.dest_z {
				continue
			} else if nampla.status&POPULATED != 0 {
				found = TRUE
				break
			}
		}
		if found == FALSE {
			fmt.Fprintf(orders_file, "\tScan\tTR1 %s\n", ship.name)
		}
	}

	fmt.Fprintf(orders_file, "END\n\n")

	/* Clean up for this species. */
	orders_file.Close()
}
