package game

// Port of species.c.

func alien_is_visible(x, y, z, species_number, alien_number int) int {
	/* Check if the alien has a ship or starbase here that is in orbit or in deep space. */
	alien := &spec_data[alien_number-1]
	for i := 0; i < alien.num_ships; i++ {
		alien_ship := ship_data[alien_number-1][i]

		if alien_ship.x != x {
			continue
		}
		if alien_ship.y != y {
			continue
		}
		if alien_ship.z != z {
			continue
		}
		if alien_ship.item_quantity[FD] == alien_ship.tonnage {
			continue
		}

		if alien_ship.status == IN_ORBIT || alien_ship.status == IN_DEEP_SPACE {
			return TRUE
		}
	}

	/* Check if alien has a planet that is not hidden. */
	for i := 0; i < alien.num_namplas; i++ {
		alien_nampla := namp_data[alien_number-1][i]

		if alien_nampla.x != x {
			continue
		}
		if alien_nampla.y != y {
			continue
		}
		if alien_nampla.z != z {
			continue
		}
		if alien_nampla.status&POPULATED == 0 {
			continue
		}

		if alien_nampla.hidden == FALSE {
			return TRUE
		}

		/* The colony is hidden. See if we have population on the same planet. */
		species := &spec_data[species_number-1]
		for j := 0; j < species.num_namplas; j++ {
			nampla := namp_data[species_number-1][j]

			if nampla.x != x {
				continue
			}
			if nampla.y != y {
				continue
			}
			if nampla.z != z {
				continue
			}
			if nampla.pn != alien_nampla.pn {
				continue
			}
			if nampla.status&POPULATED == 0 {
				continue
			}

			/* We have population on the same planet, so the alien cannot hide. */
			return TRUE
		}
	}

	return FALSE
}

/*
The following routine provides the 'distorted' species number used to

	identify a species that uses field distortion units. The input
	variable 'species_number' is the same number used in filename
	creation for the species.
*/
func distorted(species_number int) int {
	/* We must use the LS tech level at the start of the turn because
	   the distorted species number must be the same throughout the
	   turn, even if the tech level changes during production. */
	ls := spec_data[species_number-1].init_tech_level[LS]
	i := species_number & 0x000F        /* Lower four bits. */
	j := (species_number >> 4) & 0x000F /* Upper four bits. */
	return (ls%5+3)*(4*i+j) + (ls%11 + 7)
}

// free_species_data will free memory used for all species data
func free_species_data() {
	for species_index := 0; species_index < galaxy.num_species; species_index++ {
		if namp_data[species_index] != nil {
			namp_data[species_index] = nil
		}
		if ship_data[species_index] != nil {
			ship_data[species_index] = nil
		}
		data_in_memory[species_index] = FALSE
		data_modified[species_index] = FALSE
	}
}

/* Get life support tech level needed. */
func life_support_needed(species *species_data_t, home, colony *planet_data_t) int {
	if species == nil || home == nil || colony == nil {
		return 99
	}

	// temperature class requires 3 points of LS per point of difference
	tc := colony.temperature_class - home.temperature_class
	if tc < 0 {
		tc = -tc
	}

	// pressure class requires 3 points of LS per point of difference
	pc := colony.pressure_class - home.pressure_class
	if pc < 0 {
		pc = -pc
	}

	/* Assuming required gas is NOT present. */
	hasRequiredGas := FALSE

	/* Check for poison gases on planet. */
	poisonGases := 0
	for j := 0; j < 4; j++ {
		// check if the slot has gas
		if colony.gas_percent[j] != 0 {
			// check if required gas is present
			if colony.gas[j] == species.required_gas {
				// and in the right amount
				if species.required_gas_min <= colony.gas_percent[j] &&
					colony.gas_percent[j] <= species.required_gas_max {
					hasRequiredGas = TRUE
				}
			} else {
				// check if it is a poisonous gas
				for i := 0; i < 6; i++ {
					if colony.gas[j] == species.poison_gas[i] {
						poisonGases++
						break
					}
				}
			}
		}
	}

	// each point of difference and each poisonous gas requires 3 points of life support
	ls_needed := 3 * (tc + pc + poisonGases)
	// add 3 more if the required gas is not present in the right amounts
	if hasRequiredGas == FALSE {
		ls_needed += 3
	}

	return ls_needed
}

func undistorted(distorted_species_number int) int {
	for i := 0; i < MAX_SPECIES; i++ {
		species_number := i + 1
		if distorted(species_number) == distorted_species_number {
			return species_number
		}
	}
	return 0 /* Not a legitimate species. */
}
