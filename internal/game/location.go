package game

// Port of location.c.

import (
	"fmt"
	"os"
)

func add_location(x, y, z int) {
	for i := 0; i < num_locs; i++ {
		if loc[i].x == x && loc[i].y == y && loc[i].z == z && loc[i].s == species_number {
			return /* This location is already in list for this species. */
		}
	}

	/* Add new location to the list. */
	loc[num_locs].x = x
	loc[num_locs].y = y
	loc[num_locs].z = z
	loc[num_locs].s = species_number
	num_locs++
	if num_locs < MAX_LOCATIONS {
		return
	}
	fmt.Fprintf(os.Stderr, "\n\n\tInternal error. Overflow of 'loc' arrays!\n\n")
	os.Exit(255)
}

/* This routine will create the "loc" array based on current species' data. */
func do_locations() {
	num_locs = 0
	for species_number = 1; species_number <= galaxy.num_species; species_number++ {
		spidx := species_number - 1
		if data_in_memory[spidx] == FALSE {
			continue
		}

		species = &spec_data[spidx]
		nampla_base = namp_data[spidx]
		ship_base = ship_data[spidx]

		for i := 0; i < species.num_namplas; i++ {
			nampla = nampla_base[i]
			if nampla.pn == 99 {
				continue
			}
			if nampla.status&POPULATED != 0 {
				add_location(nampla.x, nampla.y, nampla.z)
			}
		}

		for i := 0; i < species.num_ships; i++ {
			ship = ship_base[i]
			if ship.pn == 99 {
				continue
			} else if ship.status == FORCED_JUMP || ship.status == JUMPED_IN_COMBAT {
				continue
			}
			add_location(ship.x, ship.y, ship.z)
		}
	}
}

// locationCommand creates the location data file from the current data.
// also updates the economic efficiency field in the planet data file.
// args mirrors the C argv: args[0] is the command name (the C dispatcher
// always passes "locations"); any extra arguments are ignored, as in C.
func locationCommand(args []string) int {
	cmdName := args[0]

	// load data used to derive locations
	fmt.Printf("fh: %s: loading   galaxy   data...\n", cmdName)
	get_galaxy_data()
	fmt.Printf("fh: %s: loading   planet   data...\n", cmdName)
	get_planet_data()
	fmt.Printf("fh: %s: loading   species  data...\n", cmdName)
	get_species_data()

	// allocate memory for array "total_econ_base"
	total_econ_base := make([]int, num_planets)

	// initialize total econ base for each planet
	for i := 0; i < num_planets; i++ {
		planet = planet_base[i]
		total_econ_base[i] = 0
	}

	// get total economic base for each planet from nampla data.
	for species_number = 1; species_number <= galaxy.num_species; species_number++ {
		if data_in_memory[species_number-1] == FALSE {
			continue
		}
		data_modified[species_number-1] = TRUE

		species = &spec_data[species_number-1]
		nampla_base = namp_data[species_number-1]

		for nampla_index = 0; nampla_index < species.num_namplas; nampla_index++ {
			nampla = nampla_base[nampla_index]
			if nampla.pn == 99 {
				continue
			}
			if (nampla.status & HOME_PLANET) == 0 {
				total_econ_base[nampla.planet_index] += nampla.mi_base + nampla.ma_base
			}
		}
	}

	// update economic efficiencies of all planets.
	for i := 0; i < num_planets; i++ {
		planet = planet_base[i]
		diff := total_econ_base[i] - 2000
		if diff <= 0 {
			planet.econ_efficiency = 100
		} else {
			planet.econ_efficiency = (100 * (diff/20 + 2000)) / total_econ_base[i]
		}
	}

	// create new locations data
	do_locations()

	// save the results
	fmt.Printf("fh: %s: saving    planet   data...\n", cmdName)
	save_planet_data()
	fmt.Printf("fh: %s: saving    location data...\n", cmdName)
	save_location_data()

	// clean up
	free_species_data()
	planet_base = nil

	return 0
}
