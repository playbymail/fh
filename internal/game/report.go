package game

// Far Horizons Game Engine
// Copyright (C) 2022 Michael D Henderson
// Copyright (C) 2021 Raven Zachary
// Copyright (C) 2019 Casey Link, Adam Piggott
// Copyright (C) 1999 Richard A. Morneau
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Port of report.c.

import (
	"fmt"
	"os"
)

// File-level variables ported from report.c's statics. Each one is fully
// (re)initialized by reportCommand before use, so no reset function is
// required between runs.
var fleet_percent_cost int
var nampla1_base []*nampla_data_t
var nampla2_base []*nampla_data_t
var printing_alien int
var report_file *os.File
var ship_already_listed [5000]int
var ship1_base []*ship_data_t
var ship2_base []*ship_data_t

func do_planet_report(nampla *nampla_data_t, s_base []*ship_data_t, species *species_data_t) {
	var i, j, ship_index, header_printed, ls_needed, production_penalty int
	var n1, n2, n3, raw_material_units, production_capacity, available_to_spend, n, ib, ab, current_base, md, denom int
	var ship *ship_data_t

	/* Print type of planet, name and coordinates. */
	fmt.Fprintf(report_file, "\n\n")

	if nampla.status&HOME_PLANET != 0 {
		fmt.Fprintf(report_file, "HOME PLANET")
	} else if nampla.status&MINING_COLONY != 0 {
		fmt.Fprintf(report_file, "MINING COLONY")
	} else if nampla.status&RESORT_COLONY != 0 {
		fmt.Fprintf(report_file, "RESORT COLONY")
	} else if nampla.status&POPULATED != 0 {
		fmt.Fprintf(report_file, "COLONY PLANET")
	} else {
		fmt.Fprintf(report_file, "PLANET")
	}

	fmt.Fprintf(report_file, ": PL %s", nampla.name)

	fmt.Fprintf(report_file, "\n   Coordinates: x = %d, y = %d, z = %d, planet number %d\n", nampla.x, nampla.y,
		nampla.z, nampla.pn)

	if nampla.status&HOME_PLANET != 0 {
		ib = nampla.mi_base
		ab = nampla.ma_base
		current_base = ib + ab
		if current_base < species.hp_original_base {
			n = species.hp_original_base - current_base /* Number of CUs needed. */

			md = home_planet.mining_difficulty

			denom = 100 + md
			j = (100*(n+ib) - (md * ab) + denom/2) / denom
			i = n - j

			if i < 0 {
				j = n
				i = 0
			}
			if j < 0 {
				i = n
				j = 0
			}

			fmt.Fprintf(report_file, "\nWARNING! Home planet has not yet completely recovered from bombardment!\n")
			fmt.Fprintf(report_file, "         %d IUs and %d AUs will have to be installed for complete recovery.\n", i, j)
		}
	}

	if nampla.status&POPULATED == 0 {
		goto do_inventory
	}

	/* Print available population. */
	if nampla.status&(MINING_COLONY|RESORT_COLONY) != 0 {
		// do nothing
	} else {
		fmt.Fprintf(report_file, "\nAvailable population units = %d\n", nampla.pop_units)
	}

	if nampla.siege_eff != 0 {
		fmt.Fprintf(report_file, "\nWARNING!  This planet is currently under siege and will remain\n")
		fmt.Fprintf(report_file, "  under siege until the combat phase of the next turn!\n")
	}

	if nampla.use_on_ambush > 0 {
		fmt.Fprintf(report_file, "\nIMPORTANT!  This planet has made preparations for an ambush!\n")
	}

	if nampla.hidden != FALSE {
		fmt.Fprintf(report_file, "\nIMPORTANT!  This planet is actively hiding from alien observation!\n")
	}

	/* Print what will be produced this turn. */
	raw_material_units = (10 * species.tech_level[MI] * nampla.mi_base) / planet.mining_difficulty
	production_capacity = (species.tech_level[MA] * nampla.ma_base) / 10

	ls_needed = life_support_needed(species, home_planet, planet)

	if ls_needed == 0 {
		production_penalty = 0
	} else {
		production_penalty = (100 * ls_needed) / species.tech_level[LS]
	}

	fmt.Fprintf(report_file, "\nProduction penalty = %d%% (LSN = %d)\n", production_penalty, ls_needed)

	fmt.Fprintf(report_file, "\nEconomic efficiency = %d%%\n", planet.econ_efficiency)

	raw_material_units -= (production_penalty * raw_material_units) / 100
	raw_material_units = ((planet.econ_efficiency * raw_material_units) + 50) / 100
	production_capacity -= (production_penalty * production_capacity) / 100
	production_capacity = ((planet.econ_efficiency * production_capacity) + 50) / 100

	if nampla.mi_base > 0 {
		fmt.Fprintf(report_file, "\nMining base = %d.%d", nampla.mi_base/10, nampla.mi_base%10)
		fmt.Fprintf(report_file, " (MI = %d, MD = %d.%02d)\n", species.tech_level[MI], planet.mining_difficulty/100,
			planet.mining_difficulty%100)

		/* For mining colonies, print economic units that will be produced. */
		if nampla.status&MINING_COLONY != 0 {
			n1 = (2 * raw_material_units) / 3
			n2 = ((fleet_percent_cost * n1) + 5000) / 10000
			n3 = n1 - n2
			fmt.Fprintf(report_file, "   This mining colony will generate %d - %d = %d economic units this turn.\n", n1,
				n2, n3)

			nampla.use_on_ambush = n3 /* Temporary use only. */
		} else {
			fmt.Fprintf(report_file, "   %d raw material units will be produced this turn.\n", raw_material_units)
		}
	}

	if nampla.ma_base > 0 {
		if nampla.status&RESORT_COLONY != 0 {
			fmt.Fprintf(report_file, "\n")
		}

		fmt.Fprintf(report_file, "Manufacturing base = %d.%d", nampla.ma_base/10, nampla.ma_base%10)
		fmt.Fprintf(report_file, " (MA = %d)\n", species.tech_level[MA])

		/* For resort colonies, print economic units that will be produced. */
		if nampla.status&RESORT_COLONY != 0 {
			n1 = (2 * production_capacity) / 3
			n2 = ((fleet_percent_cost * n1) + 5000) / 10000
			n3 = n1 - n2
			fmt.Fprintf(report_file, "   This resort colony will generate %d - %d = %d economic units this turn.\n", n1,
				n2, n3)

			nampla.use_on_ambush = n3 /* Temporary use only. */
		} else {
			fmt.Fprintf(report_file, "   Production capacity this turn will be %d.\n", production_capacity)
		}
	}

	if nampla.item_quantity[RM] > 0 {
		fmt.Fprintf(report_file, "\n%ss (%s,C%d) carried over from last turn = %d\n",
			item_name[RM], item_abbr[RM], item_carry_capacity[RM], nampla.item_quantity[RM])
	}

	/* Print what can be spent this turn. */
	raw_material_units += nampla.item_quantity[RM]
	if raw_material_units > production_capacity {
		available_to_spend = production_capacity
		nampla.special = raw_material_units - production_capacity
		/* Excess raw material units that may be recycled in AUTO mode. */
	} else {
		available_to_spend = raw_material_units
		nampla.special = 0
	}

	/* Don't print spendable amount for mining and resort colonies. */
	n1 = available_to_spend
	n2 = ((fleet_percent_cost * n1) + 5000) / 10000
	n3 = n1 - n2
	if nampla.status&MINING_COLONY == 0 && nampla.status&RESORT_COLONY == 0 {
		fmt.Fprintf(report_file, "\nTotal available for spending this turn = %d - %d = %d\n", n1, n2, n3)
		nampla.use_on_ambush = n3 /* Temporary use only. */

		fmt.Fprintf(report_file, "\nShipyard capacity = %d\n", nampla.shipyards)
	}

do_inventory:

	header_printed = FALSE

	for i = 0; i < MAX_ITEMS; i++ {
		if nampla.item_quantity[i] > 0 && i != RM {
			if header_printed == FALSE {
				header_printed = TRUE
				fmt.Fprintf(report_file, "\nPlanetary inventory:\n")
			}

			fmt.Fprintf(report_file, "   %ss (%s,C%d) = %d", item_name[i], item_abbr[i], item_carry_capacity[i],
				nampla.item_quantity[i])
			if i == PD {
				fmt.Fprintf(report_file, " (warship equivalence = %d tons)", 50*nampla.item_quantity[PD])
			}
			fmt.Fprintf(report_file, "\n")
		}
	}

	/* Print all ships that are under construction on, on the surface of, or in orbit around this planet. */
	printing_alien = FALSE
	header_printed = FALSE
	for ship_index = 0; ship_index < species.num_ships; ship_index++ {
		ship = s_base[ship_index]

		if nampla.x != ship.x {
			continue
		}
		if nampla.y != ship.y {
			continue
		}
		if nampla.z != ship.z {
			continue
		}
		if nampla.pn != ship.pn {
			continue
		}
		if ship.class != BA {
			continue
		}

		if header_printed == FALSE {
			fmt.Fprintf(report_file, "\nShips at PL %s:\n", nampla.name)
			print_ship_header()
		}
		header_printed = TRUE

		print_ship(ship, species, species_number)

		ship_already_listed[ship_index] = TRUE
	}

	for ship_index = 0; ship_index < species.num_ships; ship_index++ {
		ship = s_base[ship_index]

		if nampla.x != ship.x {
			continue
		}
		if nampla.y != ship.y {
			continue
		}
		if nampla.z != ship.z {
			continue
		}
		if nampla.pn != ship.pn {
			continue
		}
		if ship.class != TR {
			continue
		}

		if header_printed == FALSE {
			fmt.Fprintf(report_file, "\nShips at PL %s:\n", nampla.name)
			print_ship_header()
		}
		header_printed = TRUE

		print_ship(ship, species, species_number)

		ship_already_listed[ship_index] = TRUE
	}

	for ship_index = 0; ship_index < species.num_ships; ship_index++ {
		ship = s_base[ship_index]

		if nampla.x != ship.x {
			continue
		}
		if nampla.y != ship.y {
			continue
		}
		if nampla.z != ship.z {
			continue
		}
		if nampla.pn != ship.pn {
			continue
		}
		if ship_already_listed[ship_index] != FALSE {
			continue
		}

		if header_printed == FALSE {
			fmt.Fprintf(report_file, "\nShips at PL %s:\n", nampla.name)
			print_ship_header()
		}
		header_printed = TRUE

		print_ship(ship, species, species_number)

		ship_already_listed[ship_index] = TRUE
	}
}

func print_ship_header() {
	fmt.Fprintf(report_file, "  Name                          ")
	if printing_alien != FALSE {
		fmt.Fprintf(report_file, "                     Species\n")
	} else {
		fmt.Fprintf(report_file, "                 Cap. Cargo\n")
	}
	fmt.Fprintf(report_file, " ---------------------------------------")
	fmt.Fprintf(report_file, "-------------------------------------\n")
}

func print_mishap_chance(ship *ship_data_t, destx, desty, destz int) {
	var mishap_GV, mishap_age int
	var x, y, z, mishap_chance, success_chance int

	if destx == 9999 {
		fmt.Fprintf(report_file, "Mishap chance = ???")
		return
	}

	mishap_GV = species.tech_level[GV]
	mishap_age = ship.age

	x = destx
	y = desty
	z = destz
	mishap_chance = (100 * (((x - ship.x) * (x - ship.x)) + ((y - ship.y) * (y - ship.y)) +
		((z - ship.z) * (z - ship.z)))) / mishap_GV

	if mishap_age > 0 && mishap_chance < 10000 {
		success_chance = 10000 - mishap_chance
		success_chance -= (2 * mishap_age * success_chance) / 100
		mishap_chance = 10000 - success_chance
	}

	if mishap_chance > 10000 {
		mishap_chance = 10000
	}

	fmt.Fprintf(report_file, "mishap chance = %d.%02d%%",
		mishap_chance/100, mishap_chance%100)
}

func print_ship(ship *ship_data_t, species *species_data_t, species_number int) {
	var i, n, length, capacity, need_comma int

	if printing_alien != FALSE {
		ignore_field_distorters = FALSE
	} else {
		ignore_field_distorters = TRUE
	}

	fmt.Fprintf(report_file, "  %s", ship_name(ship))

	length = len(full_ship_id)
	if printing_alien != FALSE {
		n = 50
	} else {
		n = 46
	}

	for i = 0; i < (n - length); i++ {
		fmt.Fprintf(report_file, " ")
	}

	if ship.class == BA {
		capacity = 10 * ship.tonnage
	} else if ship.class == TR {
		capacity = (10 + (ship.tonnage / 2)) * ship.tonnage
	} else {
		capacity = ship.tonnage
	}

	if printing_alien != FALSE {
		fmt.Fprintf(report_file, " ")
	} else {
		fmt.Fprintf(report_file, "%4d  ", capacity)
		if ship.status == UNDER_CONSTRUCTION {
			fmt.Fprintf(report_file, "Left to pay = %d\n", ship.remaining_cost)
			return
		}
	}

	if printing_alien != FALSE {
		if ship.status == ON_SURFACE || ship.item_quantity[FD] != ship.tonnage {
			fmt.Fprintf(report_file, "SP %s", species.name)
		} else {
			fmt.Fprintf(report_file, "SP %d", distorted(species_number))
		}
	} else {
		need_comma = FALSE
		for i = 0; i < MAX_ITEMS; i++ {
			if ship.item_quantity[i] > 0 {
				if need_comma != FALSE {
					fmt.Fprintf(report_file, ",")
				}
				fmt.Fprintf(report_file, "%d %s", ship.item_quantity[i], item_abbr[i])
				need_comma = TRUE
			}
		}
	}

	fmt.Fprintf(report_file, "\n")
}

func reportCommand(args []string) int {
	cmdName := args[0]

	var i, j, k, ship_index, my_loc_index, its_loc_index int
	var industry int
	var header_printed, alien_can_hide, sp_index int
	var array_index, bit_number, we_have_colony_here, nampla_index int
	var we_have_planet_here, found int
	var temp_ignore_field_distorters int
	var filename, log_line, temp2 string
	var n, nn int
	var bit_mask uint32
	var alien *species_data_t
	var nampla, alien_nampla, our_nampla, temp_nampla *nampla_data_t
	var ship, ship2, alien_ship *ship_data_t
	var my_loc, its_loc *sp_loc_data_t

	// consolidate logic for reporting and logging flags
	// by default, log and report on all species
	logSpecies := 1
	var reportSpecies [MAX_SPECIES + 1]int
	for j = 0; j <= MAX_SPECIES; j++ {
		reportSpecies[j] = 1
	}

	// process the arguments and reset flags as needed
	for i = 1; i < len(args); i++ {
		if args[i] == "--help" || args[i] == "-h" || args[i] == "-?" {
			fmt.Fprintf(os.Stderr, "usage: report [--skip-log] [list-of-species]\n")
			fmt.Fprintf(os.Stderr, "\t--skip-log       do not include prior turn results in report\n")
			fmt.Fprintf(os.Stderr, "\tlist-of-species  you may specify individual species numbers to report on\n")
			return 2
		} else if args[i] == "--skip-log" {
			// turn off logging for all species
			logSpecies = 0
		} else { // should be the species to report
			speciesNo := cfgAtoi(args[i])
			if speciesNo < 1 || speciesNo > MAX_SPECIES {
				fmt.Fprintf(os.Stderr, "error: unknown species '%s'\n", args[i])
				return 2
			} else {
				// ugly hack to reset the flags.
				if reportSpecies[0] == 1 {
					// reset the flags because we're not reporting on all
					for j = 0; j <= MAX_SPECIES; j++ {
						reportSpecies[j] = 0
					}
				}
				// set the flag for this species
				reportSpecies[speciesNo] = 1
			}
		}
	}

	/* Get all necessary data. */
	fmt.Printf("fh: %s: loading   galaxy   file...\n", cmdName)
	get_galaxy_data()
	fmt.Printf("fh: %s: loading   star     file...\n", cmdName)
	get_star_data()
	fmt.Printf("fh: %s: loading   planet   file...\n", cmdName)
	get_planet_data()
	fmt.Printf("fh: %s: loading   species  file...\n", cmdName)
	get_species_data()
	fmt.Printf("fh: %s: loading   location file...\n", cmdName)
	get_location_data()

	turn_number := galaxy.turn_number

	/* Generate a report for each species. */
	alien_number := 0 /* Pointers to alien data not yet assigned. */
	for species_number = 1; species_number <= galaxy.num_species; species_number++ {
		/* Check if we are doing all species, or just one or more specified ones. */
		if reportSpecies[species_number] != 1 {
			continue
		}

		/* Check if this species is still in the game. */
		if data_in_memory[species_number-1] == FALSE {
			if len(args) == 1 {
				/* This species is no longer in the game. */
				continue
			}

			fmt.Fprintf(os.Stderr, "\n\tCannot open data file for species #%d!\n\n", species_number)
			os.Exit(255)
		}

		species = &spec_data[species_number-1]
		nampla_base = namp_data[species_number-1]
		nampla1_base = nampla_base
		ship_base = ship_data[species_number-1]
		ship1_base = ship_base
		home_planet = planet_base[nampla1_base[0].planet_index]

		/* Print message for gamemaster. */
		if verbose_mode != FALSE {
			fmt.Printf("Generating turn %d report for species #%d, SP %s...\n",
				turn_number, species_number, species.name)
		}

		/* Open report file for writing. */
		filename = fmt.Sprintf("sp%02d.rpt.t%d", species_number, turn_number)
		var err error
		report_file, err = os.Create(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n\tCannot open '%s' for writing!\n\n", filename)
			os.Exit(255)
		}

		/* Copy log file, if any, to output file. */
		if logSpecies == 1 {
			filename = fmt.Sprintf("sp%02d.log", species_number)
			// The C code reuses the log_file global as a read handle here;
			// the Go port uses a local cfile for reading instead.
			species_log := fopen_r(filename)
			if species_log != nil {
				if turn_number > 1 {
					fmt.Fprintf(report_file, "\n\n\t\t\tEVENT LOG FOR TURN %d\n", turn_number-1)
				}

				for {
					var ok bool
					log_line, ok = readln(species_log, 256)
					if !ok {
						break
					}
					fmt.Fprint(report_file, log_line)
				}

				fmt.Fprintf(report_file, "\n\n")

				species_log.fclose()
			}
		}

		/* Print header for status report. */
		fmt.Fprintf(report_file, "\n\t\t\t SPECIES STATUS\n\n\t\t\tSTART OF TURN %d\n\n", turn_number)
		fmt.Fprintf(report_file, "Species name: %s\n", species.name)
		fmt.Fprintf(report_file, "Government name: %s\n", species.govt_name)
		fmt.Fprintf(report_file, "Government type: %s\n", species.govt_type)

		fmt.Fprintf(report_file, "\nTech Levels:\n")
		for i = 0; i < 6; i++ {
			fmt.Fprintf(report_file, "   %s = %d", tech_name[i], species.tech_level[i])
			if species.tech_knowledge[i] > species.tech_level[i] {
				fmt.Fprintf(report_file, "/%d", species.tech_knowledge[i])
			}
			fmt.Fprintf(report_file, "\n")
		}

		fmt.Fprintf(report_file, "\nAtmospheric Requirement: %d%%-%d%% %s", species.required_gas_min,
			species.required_gas_max, gas_string[species.required_gas])
		fmt.Fprintf(report_file, "\nNeutral Gases:")
		for i = 0; i < 6; i++ {
			if i != 0 {
				fmt.Fprintf(report_file, ",")
			}
			fmt.Fprintf(report_file, " %s", gas_string[species.neutral_gas[i]])
		}
		fmt.Fprintf(report_file, "\nPoisonous Gases:")
		for i = 0; i < 6; i++ {
			if i != 0 {
				fmt.Fprintf(report_file, ",")
			}
			fmt.Fprintf(report_file, " %s", gas_string[species.poison_gas[i]])
		}
		fmt.Fprintf(report_file, "\n")

		/* List fleet maintenance cost and its percentage of total production. */
		fleet_percent_cost = species.fleet_percent_cost

		fmt.Fprintf(report_file, "\nFleet maintenance cost = %d (%d.%02d%% of total production)\n",
			species.fleet_cost, fleet_percent_cost/100, fleet_percent_cost%100)

		if fleet_percent_cost > 10000 {
			fleet_percent_cost = 10000
		}

		/* List species that have been met. */
		n = 0
		log_file = report_file /* Use log utils for this. */
		log_stdout = FALSE
		header_printed = FALSE
		for sp_index = 0; sp_index < galaxy.num_species; sp_index++ {
			if data_in_memory[sp_index] == FALSE {
				continue
			}

			array_index = (sp_index) / 32
			bit_number = (sp_index) % 32
			bit_mask = uint32(1) << uint(bit_number)
			if species.contact[array_index]&bit_mask == 0 {
				continue
			}

			if header_printed == FALSE {
				log_string("\nSpecies met: ")
				header_printed = TRUE
			}

			if n > 0 {
				log_string(", ")
			}
			log_string("SP ")
			log_string(spec_data[sp_index].name)
			n++
		}
		if n > 0 {
			log_char('\n')
		}

		/* List declared allies. */
		n = 0
		header_printed = FALSE
		for sp_index = 0; sp_index < galaxy.num_species; sp_index++ {
			if data_in_memory[sp_index] == FALSE {
				continue
			}

			array_index = (sp_index) / 32
			bit_number = (sp_index) % 32
			bit_mask = uint32(1) << uint(bit_number)
			if species.ally[array_index]&bit_mask == 0 {
				continue
			}
			if species.contact[array_index]&bit_mask == 0 {
				continue
			}

			if header_printed == FALSE {
				log_string("\nAllies: ")
				header_printed = TRUE
			}

			if n > 0 {
				log_string(", ")
			}
			log_string("SP ")
			log_string(spec_data[sp_index].name)
			n++
		}
		if n > 0 {
			log_char('\n')
		}

		/* List declared enemies that have been met. */
		n = 0
		header_printed = FALSE
		for sp_index = 0; sp_index < galaxy.num_species; sp_index++ {
			if data_in_memory[sp_index] == FALSE {
				continue
			}

			array_index = (sp_index) / 32
			bit_number = (sp_index) % 32
			bit_mask = uint32(1) << uint(bit_number)
			if species.enemy[array_index]&bit_mask == 0 {
				continue
			}
			if species.contact[array_index]&bit_mask == 0 {
				continue
			}

			if header_printed == FALSE {
				log_string("\nEnemies: ")
				header_printed = TRUE
			}

			if n > 0 {
				log_string(", ")
			}
			log_string("SP ")
			log_string(spec_data[sp_index].name)
			n++
		}
		if n > 0 {
			log_char('\n')
		}

		fmt.Fprintf(report_file, "\nEconomic units = %d\n", species.econ_units)

		/* Initialize flag. */
		for i = 0; i < species.num_ships; i++ {
			ship_already_listed[i] = FALSE
		}

		/* Print report for each producing planet. */
		for i = 0; i < species.num_namplas; i++ {
			nampla = nampla1_base[i]

			if nampla.pn == 99 {
				continue
			}
			if nampla.mi_base == 0 && nampla.ma_base == 0 && nampla.status&HOME_PLANET == 0 {
				continue
			}

			planet = planet_base[nampla.planet_index]
			fmt.Fprintf(report_file, "\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")
			do_planet_report(nampla, ship1_base, species)
		}

		/* Give only a one-line listing for other planets. */
		printing_alien = FALSE
		header_printed = FALSE
		for i = 0; i < species.num_namplas; i++ {
			nampla = nampla1_base[i]

			if nampla.pn == 99 {
				continue
			}
			if nampla.mi_base > 0 || nampla.ma_base > 0 || nampla.status&HOME_PLANET != 0 {
				continue
			}

			if header_printed == FALSE {
				fmt.Fprintf(report_file, "\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")
				fmt.Fprintf(report_file, "\n\nOther planets and ships:\n\n")
				header_printed = TRUE
			}
			fmt.Fprintf(report_file, "%4d%3d%3d #%d\tPL %s", nampla.x, nampla.y, nampla.z, nampla.pn, nampla.name)

			for j = 0; j < MAX_ITEMS; j++ {
				if nampla.item_quantity[j] > 0 {
					fmt.Fprintf(report_file, ", %d %s", nampla.item_quantity[j], item_abbr[j])
				}
			}
			fmt.Fprintf(report_file, "\n")

			/* Print any ships at this planet. */
			for ship_index = 0; ship_index < species.num_ships; ship_index++ {
				ship = ship1_base[ship_index]

				if ship_already_listed[ship_index] != FALSE {
					continue
				}

				if ship.x != nampla.x {
					continue
				}
				if ship.y != nampla.y {
					continue
				}
				if ship.z != nampla.z {
					continue
				}
				if ship.pn != nampla.pn {
					continue
				}

				fmt.Fprintf(report_file, "\t\t%s", ship_name(ship))
				for j = 0; j < MAX_ITEMS; j++ {
					if ship.item_quantity[j] > 0 {
						fmt.Fprintf(report_file, ", %d %s", ship.item_quantity[j], item_abbr[j])
					}
				}
				fmt.Fprintf(report_file, "\n")

				ship_already_listed[ship_index] = TRUE
			}
		}

		/* Report ships that are not associated with a planet. */
		for ship_index = 0; ship_index < species.num_ships; ship_index++ {
			ship = ship1_base[ship_index]

			ship.special = 0

			if ship_already_listed[ship_index] != FALSE {
				continue
			}

			ship_already_listed[ship_index] = TRUE

			if ship.pn == 99 {
				continue
			}

			if header_printed == FALSE {
				fmt.Fprintf(report_file, "\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")
				fmt.Fprintf(report_file, "\n\nOther planets and ships:\n\n")
				header_printed = TRUE
			}

			if ship.status == JUMPED_IN_COMBAT || ship.status == FORCED_JUMP {
				fmt.Fprintf(report_file, "  ?? ?? ??\t%s", ship_name(ship))
			} else if test_mode != FALSE && ship.arrived_via_wormhole != FALSE {
				fmt.Fprintf(report_file, "  ?? ?? ??\t%s", ship_name(ship))
			} else {
				fmt.Fprintf(report_file, "%4d%3d%3d\t%s", ship.x, ship.y, ship.z, ship_name(ship))
			}

			for i = 0; i < MAX_ITEMS; i++ {
				if ship.item_quantity[i] > 0 {
					fmt.Fprintf(report_file, ", %d %s", ship.item_quantity[i], item_abbr[i])
				}
			}
			fmt.Fprintf(report_file, "\n")

			if ship.status == JUMPED_IN_COMBAT || ship.status == FORCED_JUMP {
				continue
			}

			if test_mode != FALSE && ship.arrived_via_wormhole != FALSE {
				continue
			}

			/* Print other ships at the same location. */
			for i = ship_index + 1; i < species.num_ships; i++ {
				ship2 = ship1_base[i]

				if ship_already_listed[i] != FALSE {
					continue
				}
				if ship2.pn == 99 {
					continue
				}
				if ship2.x != ship.x {
					continue
				}
				if ship2.y != ship.y {
					continue
				}
				if ship2.z != ship.z {
					continue
				}

				fmt.Fprintf(report_file, "\t\t%s", ship_name(ship2))
				for j = 0; j < MAX_ITEMS; j++ {
					if ship2.item_quantity[j] > 0 {
						fmt.Fprintf(report_file, ", %d %s", ship2.item_quantity[j], item_abbr[j])
					}
				}
				fmt.Fprintf(report_file, "\n")

				ship_already_listed[i] = TRUE
			}
		}

		fmt.Fprintf(report_file, "\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")

		/* Report aliens at locations where current species has inhabited planets or ships. */
		printing_alien = TRUE
		for my_loc_index = 0; my_loc_index < num_locs; my_loc_index++ {
			my_loc = &loc[my_loc_index]
			if my_loc.s != species_number {
				continue
			}

			header_printed = FALSE
			for its_loc_index = 0; its_loc_index < num_locs; its_loc_index++ {
				its_loc = &loc[its_loc_index]
				if its_loc.s == species_number {
					continue
				}
				if my_loc.x != its_loc.x {
					continue
				}
				if my_loc.y != its_loc.y {
					continue
				}
				if my_loc.z != its_loc.z {
					continue
				}

				/* There is an alien here. Check if pointers for data for this alien have been assigned yet. */
				if its_loc.s != alien_number {
					alien_number = its_loc.s
					if data_in_memory[alien_number-1] == FALSE {
						fmt.Fprintf(os.Stderr, "\n\nWarning! Data for alien #%d is needed but is not in memory!\n\n",
							alien_number)
						continue
					}
					alien = &spec_data[alien_number-1]
					nampla2_base = namp_data[alien_number-1]
					ship2_base = ship_data[alien_number-1]
				}

				/* Check if we have a named planet in this system. If so, use it when you print the header. */
				we_have_planet_here = FALSE
				for i = 0; i < species.num_namplas; i++ {
					nampla = nampla1_base[i]

					if nampla.x != my_loc.x {
						continue
					}
					if nampla.y != my_loc.y {
						continue
					}
					if nampla.z != my_loc.z {
						continue
					}
					if nampla.pn == 99 {
						continue
					}

					we_have_planet_here = TRUE
					our_nampla = nampla

					break
				}

				/* Print all inhabited alien namplas at this location. */
				for i = 0; i < alien.num_namplas; i++ {
					alien_nampla = nampla2_base[i]

					if my_loc.x != alien_nampla.x {
						continue
					}
					if my_loc.y != alien_nampla.y {
						continue
					}
					if my_loc.z != alien_nampla.z {
						continue
					}
					if alien_nampla.status&POPULATED == 0 {
						continue
					}

					/* Check if current species has a colony on the same planet. */
					we_have_colony_here = FALSE
					for j = 0; j < species.num_namplas; j++ {
						nampla = nampla1_base[j]

						if alien_nampla.x != nampla.x {
							continue
						}
						if alien_nampla.y != nampla.y {
							continue
						}
						if alien_nampla.z != nampla.z {
							continue
						}
						if alien_nampla.pn != nampla.pn {
							continue
						}
						if nampla.status&POPULATED == 0 {
							continue
						}

						we_have_colony_here = TRUE

						break
					}

					if alien_nampla.hidden != FALSE && we_have_colony_here == FALSE {
						continue
					}

					if header_printed == FALSE {
						fmt.Fprintf(report_file, "\n\nAliens at x = %d, y = %d, z = %d", my_loc.x, my_loc.y, my_loc.z)

						if we_have_planet_here != FALSE {
							fmt.Fprintf(report_file, " (PL %s star system)", our_nampla.name)
						}

						fmt.Fprintf(report_file, ":\n")
						header_printed = TRUE
					}

					industry = alien_nampla.mi_base + alien_nampla.ma_base

					var temp1 string
					if alien_nampla.status&MINING_COLONY != 0 {
						temp1 = "Mining colony"
					} else if alien_nampla.status&RESORT_COLONY != 0 {
						temp1 = "Resort colony"
					} else if alien_nampla.status&HOME_PLANET != 0 {
						temp1 = "Home planet"
					} else if industry > 0 {
						temp1 = "Colony planet"
					} else {
						temp1 = "Uncolonized planet"
					}

					temp2 = fmt.Sprintf("  %s PL %s (pl #%d)", temp1, alien_nampla.name, alien_nampla.pn)
					n = 53 - len(temp2)
					for j = 0; j < n; j++ {
						temp2 += " "
					}
					fmt.Fprintf(report_file, "%sSP %s\n", temp2, alien.name)

					j = industry
					if industry < 100 {
						industry = (industry + 5) / 10
					} else {
						industry = ((industry + 50) / 100) * 10
					}

					if j == 0 {
						fmt.Fprintf(report_file, "      (No economic base.)\n")
					} else {
						fmt.Fprintf(report_file, "      (Economic base is approximately %d.)\n", industry)
					}

					/* If current species has a colony on the same planet, report any PDs and any shipyards. */
					if we_have_colony_here != FALSE {
						if alien_nampla.item_quantity[PD] == 1 {
							fmt.Fprintf(report_file, "      (There is 1 %s on the planet.)\n", item_name[PD])
						} else if alien_nampla.item_quantity[PD] > 1 {
							fmt.Fprintf(report_file, "      (There are %d %ss on the planet.)\n",
								alien_nampla.item_quantity[PD], item_name[PD])
						}

						if alien_nampla.shipyards == 1 {
							fmt.Fprintf(report_file, "      (There is 1 shipyard on the planet.)\n")
						} else if alien_nampla.shipyards > 1 {
							fmt.Fprintf(report_file, "      (There are %d shipyards on the planet.)\n",
								alien_nampla.shipyards)
						}
					}

					/* Also report if alien colony is actively hiding. */
					if alien_nampla.hidden != FALSE {
						fmt.Fprintf(report_file, "      (Colony is actively hiding from alien observation.)\n")
					}
				}

				/* Print all alien ships at this location. */
				for i = 0; i < alien.num_ships; i++ {
					alien_ship = ship2_base[i]

					if alien_ship.pn == 99 {
						continue
					}
					if my_loc.x != alien_ship.x {
						continue
					}
					if my_loc.y != alien_ship.y {
						continue
					}
					if my_loc.z != alien_ship.z {
						continue
					}

					/* An alien ship cannot hide if it lands on the surface of a planet populated by the current species. */
					alien_can_hide = TRUE
					for j = 0; j < species.num_namplas; j++ {
						nampla = nampla1_base[j]

						if alien_ship.x != nampla.x {
							continue
						}
						if alien_ship.y != nampla.y {
							continue
						}
						if alien_ship.z != nampla.z {
							continue
						}
						if alien_ship.pn != nampla.pn {
							continue
						}
						if nampla.status&POPULATED != 0 {
							alien_can_hide = FALSE
							break
						}
					}

					if alien_can_hide != FALSE && alien_ship.status == ON_SURFACE {
						continue
					}

					if alien_can_hide != FALSE && alien_ship.status == UNDER_CONSTRUCTION {
						continue
					}

					if header_printed == FALSE {
						fmt.Fprintf(report_file, "\n\nAliens at x = %d, y = %d, z = %d", my_loc.x, my_loc.y, my_loc.z)

						if we_have_planet_here != FALSE {
							fmt.Fprintf(report_file, " (PL %s star system)", our_nampla.name)
						}

						fmt.Fprintf(report_file, ":\n")
						header_printed = TRUE
					}

					print_ship(alien_ship, alien, alien_number)
				}
			}
		}

		printing_alien = FALSE

		/* The C code uses "goto done_report" when in test mode; the order
		   section below is wrapped in this if statement instead. */
		if test_mode == FALSE {
			/* Generate order section. */
			truncate_name = TRUE
			temp_ignore_field_distorters = ignore_field_distorters
			ignore_field_distorters = TRUE

			fmt.Fprintf(report_file, "\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")

			fmt.Fprintf(report_file, "\n\nORDER SECTION. Remove these two lines and everything above\n")
			fmt.Fprintf(report_file, "  them, and submit only the orders below.\n\n")

			fmt.Fprintf(report_file, "START COMBAT\n")
			fmt.Fprintf(report_file, "; Place combat orders here.\n\n")
			fmt.Fprintf(report_file, "END\n\n")

			fmt.Fprintf(report_file, "START PRE-DEPARTURE\n")
			fmt.Fprintf(report_file, "; Place pre-departure orders here.\n\n")

			for nampla_index = 0; nampla_index < species.num_namplas; nampla_index++ {
				nampla = nampla_base[nampla_index]
				if nampla.pn == 99 {
					continue
				}

				/* Generate auto-installs for colonies that were loaded via the DEVELOP command. */
				if nampla.auto_IUs != 0 {
					fmt.Fprintf(report_file, "\tInstall\t%d IU\tPL %s\n", nampla.auto_IUs, nampla.name)
				}
				if nampla.auto_AUs != 0 {
					fmt.Fprintf(report_file, "\tInstall\t%d AU\tPL %s\n", nampla.auto_AUs, nampla.name)
				}
				if nampla.auto_IUs != 0 || nampla.auto_AUs != 0 {
					fmt.Fprintf(report_file, "\n")
				}

				if species.auto_orders == FALSE {
					continue
				}

				/* Generate auto UNLOAD orders for transports at this nampla. */
				for j = 0; j < species.num_ships; j++ {
					ship = ship_base[j]
					if ship.pn == 99 {
						continue
					}
					if ship.x != nampla.x {
						continue
					}
					if ship.y != nampla.y {
						continue
					}
					if ship.z != nampla.z {
						continue
					}
					if ship.pn != nampla.pn {
						continue
					}
					if ship.status == JUMPED_IN_COMBAT {
						continue
					}
					if ship.status == FORCED_JUMP {
						continue
					}
					if ship.class != TR {
						continue
					}
					if ship.item_quantity[CU] < 1 {
						continue
					}

					/* New colonies will never be started automatically unless ship was loaded via a DEVELOP order. */
					if ship.loading_point != 0 {
						/* Check if transport is at specified unloading point. */
						n = ship.unloading_point
						if n == nampla_index || (n == 9999 && nampla_index == 0) {
							goto unload_ship
						}
					}

					if nampla.status&POPULATED == 0 {
						continue
					}

					if (nampla.mi_base + nampla.ma_base) >= 2000 {
						continue
					}

					if nampla.x == nampla_base[0].x && nampla.y == nampla_base[0].y && nampla.z == nampla_base[0].z {
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

					fmt.Fprintf(report_file, "\tUnload\tTR%d%s %s\n\n", ship.tonnage, ship_type[ship.ship_type], ship.name)

					ship.special = ship.loading_point
					n = nampla_index /* C: n = nampla - nampla_base; */
					if n == 0 {
						n = 9999
					}
					ship.unloading_point = n
				}
			}

			fmt.Fprintf(report_file, "END\n\n")

			fmt.Fprintf(report_file, "START JUMPS\n")
			fmt.Fprintf(report_file, "; Place jump orders here.\n\n")

			/* Generate auto-jumps for ships that were loaded via the DEVELOP command or which were UNLOADed because of the AUTO command. */
			for i = 0; i < species.num_ships; i++ {
				ship = ship_base[i]

				ship.just_jumped = FALSE

				if ship.pn == 99 {
					continue
				}
				if ship.status == JUMPED_IN_COMBAT {
					continue
				}
				if ship.status == FORCED_JUMP {
					continue
				}

				j = ship.special
				if j != 0 {
					if j == 9999 {
						/* Home planet. */
						j = 0
					}
					temp_nampla = nampla_base[j]

					fmt.Fprintf(report_file, "\tJump\t%s, PL %s\t; Age %d, ", ship_name(ship), temp_nampla.name, ship.age)

					print_mishap_chance(ship, temp_nampla.x, temp_nampla.y, temp_nampla.z)

					fmt.Fprintf(report_file, "\n\n")

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

					fmt.Fprintf(report_file, "\tJump\t%s, PL %s\t; ", ship_name(ship), temp_nampla.name)

					print_mishap_chance(ship, temp_nampla.x, temp_nampla.y, temp_nampla.z)

					fmt.Fprintf(report_file, "\n\n")

					ship.just_jumped = TRUE
				}
			}

			if species.auto_orders == FALSE {
				goto jump_end
			}

			/* Generate JUMP orders for all ships that have not yet been given orders. */
			for i = 0; i < species.num_ships; i++ {
				ship = ship_base[i]
				if ship.pn == 99 {
					continue
				}
				if ship.just_jumped != FALSE {
					continue
				}
				if ship.status == UNDER_CONSTRUCTION {
					continue
				}
				if ship.status == JUMPED_IN_COMBAT {
					continue
				}
				if ship.status == FORCED_JUMP {
					continue
				}

				if ship.ship_type == FTL {
					fmt.Fprintf(report_file, "\tJump\t%s, ", ship_name(ship))
					if ship.class == TR && ship.tonnage == 1 {
						closest_unvisited_star_report(ship, report_file)
						fmt.Fprintf(report_file, "\n\t\t\t; Age %d, now at %d %d %d, ", ship.age, ship.x, ship.y, ship.z)

						if ship.status == IN_ORBIT {
							fmt.Fprintf(report_file, "O%d, ", ship.pn)
						} else if ship.status == ON_SURFACE {
							fmt.Fprintf(report_file, "L%d, ", ship.pn)
						} else {
							fmt.Fprintf(report_file, "D, ")
						}

						print_mishap_chance(ship, x, y, z)
					} else {
						fmt.Fprintf(report_file, "???\t; Age %d, now at %d %d %d", ship.age, ship.x, ship.y, ship.z)

						if ship.status == IN_ORBIT {
							fmt.Fprintf(report_file, ", O%d", ship.pn)
						} else if ship.status == ON_SURFACE {
							fmt.Fprintf(report_file, ", L%d", ship.pn)
						} else {
							fmt.Fprintf(report_file, ", D")
						}

						x = 9999
					}

					fmt.Fprintf(report_file, "\n")

					/* Save destination so that we can check later if it needs to be scanned. */
					if x == 9999 {
						ship.dest_x = -1
					} else {
						ship.dest_x = x
						ship.dest_y = y
						ship.dest_z = z
					}
				}
			}

		jump_end:
			fmt.Fprintf(report_file, "END\n\n")

			fmt.Fprintf(report_file, "START PRODUCTION\n\n")

			fmt.Fprintf(report_file, ";   Economic units at start of turn = %d\n\n", species.econ_units)

			/* Generate a PRODUCTION order for each planet that can produce. */
			for nampla_index = species.num_namplas - 1; nampla_index >= 0; nampla_index-- {
				nampla = nampla1_base[nampla_index]
				if nampla.pn == 99 {
					continue
				}

				if nampla.mi_base == 0 && nampla.status&RESORT_COLONY == 0 {
					continue
				}
				if nampla.ma_base == 0 && nampla.status&MINING_COLONY == 0 {
					continue
				}

				fmt.Fprintf(report_file, "    PRODUCTION PL %s\n", nampla.name)

				if nampla.status&MINING_COLONY != 0 {
					fmt.Fprintf(report_file, "    ; The above PRODUCTION order is required for this mining colony, even\n")
					fmt.Fprintf(report_file, "    ;  if no other production orders are given for it. This mining colony\n")
					fmt.Fprintf(report_file, "    ;  will generate %d economic units this turn.\n", nampla.use_on_ambush)
				} else if nampla.status&RESORT_COLONY != 0 {
					fmt.Fprintf(report_file, "    ; The above PRODUCTION order is required for this resort colony, even\n")
					fmt.Fprintf(report_file, "    ;  though no other production orders can be given for it.  This resort\n")
					fmt.Fprintf(report_file, "    ;  colony will generate %d economic units this turn.\n",
						nampla.use_on_ambush)
				} else {
					fmt.Fprintf(report_file, "    ; Place production orders here for planet %s", nampla.name)
					fmt.Fprintf(report_file, " (sector %d %d %d #%d).\n", nampla.x, nampla.y, nampla.z, nampla.pn)
					fmt.Fprintf(report_file, "    ;  Avail pop = %d, shipyards = %d, to spend = %d",
						nampla.pop_units, nampla.shipyards, nampla.use_on_ambush)

					n = nampla.use_on_ambush
					if nampla.status&HOME_PLANET != 0 {
						if species.hp_original_base != 0 {
							fmt.Fprintf(report_file, " (max = %d)", 5*n)
						} else {
							fmt.Fprintf(report_file, " (max = no limit)")
						}
					} else {
						fmt.Fprintf(report_file, " (max = %d)", 2*n)
					}

					fmt.Fprintf(report_file, ".\n\n")
				}

				/* Build IUs and AUs for incoming ships with CUs. */
				if nampla.IUs_needed != 0 {
					fmt.Fprintf(report_file, "\tBuild\t%d IU\n", nampla.IUs_needed)
				}
				if nampla.AUs_needed != 0 {
					fmt.Fprintf(report_file, "\tBuild\t%d AU\n", nampla.AUs_needed)
				}
				if nampla.IUs_needed != 0 || nampla.AUs_needed != 0 {
					fmt.Fprintf(report_file, "\n")
				}

				if species.auto_orders == FALSE {
					continue
				}
				if nampla.status&MINING_COLONY != 0 {
					continue
				}
				if nampla.status&RESORT_COLONY != 0 {
					continue
				}

				/* See if there are any RMs to recycle. */
				n = nampla.special / 5
				if n > 0 {
					fmt.Fprintf(report_file, "\tRecycle\t%d RM\n\n", 5*n)
				}

				/* Generate DEVELOP commands for ships arriving here because of
				AUTO command. */
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
						k = 0
					} /* Home planet. */

					if nampla != nampla_base[k] {
						continue
					}

					k = ship.unloading_point
					if k == 9999 {
						k = 0
					}
					temp_nampla = nampla_base[k]

					fmt.Fprintf(report_file, "\tDevelop\tPL %s, TR%d%s %s\n\n", temp_nampla.name, ship.tonnage,
						ship_type[ship.ship_type], ship.name)
				}

				/* Give orders to continue construction of unfinished ships and starbases. */
				for i = 0; i < species.num_ships; i++ {
					ship = ship_base[i]
					if ship.pn == 99 {
						continue
					}

					if ship.x != nampla.x {
						continue
					}
					if ship.y != nampla.y {
						continue
					}
					if ship.z != nampla.z {
						continue
					}
					if ship.pn != nampla.pn {
						continue
					}

					if ship.status == UNDER_CONSTRUCTION {
						fmt.Fprintf(report_file,
							"\tContinue\t%s, %d\t; Left to pay = %d\n\n",
							ship_name(ship), ship.remaining_cost,
							ship.remaining_cost)

						continue
					}

					if ship.ship_type != STARBASE {
						continue
					}

					j = (species.tech_level[MA] / 2) - ship.tonnage
					if j < 1 {
						continue
					}

					fmt.Fprintf(report_file, "\tContinue\tBAS %s, %d\t; Current tonnage = %s\n\n", ship.name, 100*j,
						commas(10000*ship.tonnage))
				}

				/* Generate DEVELOP command if this is a colony with an economic base less than 200. */
				n = nampla.mi_base + nampla.ma_base + nampla.IUs_needed + nampla.AUs_needed
				nn = nampla.item_quantity[CU]
				for i = 0; i < species.num_ships; i++ {
					/* Get CUs on transports at planet. */
					ship = ship_base[i]
					if ship.x != nampla.x {
						continue
					}
					if ship.y != nampla.y {
						continue
					}
					if ship.z != nampla.z {
						continue
					}
					if ship.pn != nampla.pn {
						continue
					}
					nn += ship.item_quantity[CU]
				}
				n += nn
				if nampla.status&COLONY != 0 && n < 2000 && nampla.pop_units > 0 {
					if nampla.pop_units > (2000 - n) {
						nn = 2000 - n
					} else {
						nn = nampla.pop_units
					}

					fmt.Fprintf(report_file, "\tDevelop\t%d\n\n", 2*nn)

					nampla.IUs_needed += nn
				}

				/* For home planets and any colonies that have an economic base of
				 * at least 200, check if there are other colonized planets in
				 * the same sector that are not self-sufficient.
				 * If so, DEVELOP them. */
				if n >= 2000 || nampla.status&HOME_PLANET != 0 {
					/* Skip home planet. */
					for i = 1; i < species.num_namplas; i++ {
						if i == nampla_index {
							continue
						}

						temp_nampla = nampla_base[i]

						if temp_nampla.pn == 99 {
							continue
						}
						if temp_nampla.x != nampla.x {
							continue
						}
						if temp_nampla.y != nampla.y {
							continue
						}
						if temp_nampla.z != nampla.z {
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

						fmt.Fprintf(report_file, "\tDevelop\t%d\tPL %s\n\n", 2*nn, temp_nampla.name)

						temp_nampla.AUs_needed += nn
					}
				}
			}

			fmt.Fprintf(report_file, "END\n\n")

			fmt.Fprintf(report_file, "START POST-ARRIVAL\n")
			fmt.Fprintf(report_file, "; Place post-arrival orders here.\n\n")

			if species.auto_orders == FALSE {
				goto post_end
			}

			/* Generate an AUTO command. */
			fmt.Fprintf(report_file, "\tAuto\n\n")

			/* Generate SCAN orders for all TR1s that are jumping to sectors which current species does not inhabit. */
			for i = 0; i < species.num_ships; i++ {
				ship = ship_base[i]
				if ship.pn == 99 {
					continue
				}
				if ship.status == UNDER_CONSTRUCTION {
					continue
				}
				if ship.class != TR {
					continue
				}
				if ship.tonnage != 1 {
					continue
				}
				if ship.ship_type != FTL {
					continue
				}

				found = FALSE
				for j = 0; j < species.num_namplas; j++ {
					if ship.dest_x == -1 {
						break
					}

					nampla = nampla_base[j]
					if nampla.pn == 99 {
						continue
					}
					if nampla.x != ship.dest_x {
						continue
					}
					if nampla.y != ship.dest_y {
						continue
					}
					if nampla.z != ship.dest_z {
						continue
					}

					if nampla.status&POPULATED != 0 {
						found = TRUE
						break
					}
				}
				if found == FALSE {
					fmt.Fprintf(report_file, "\tScan\tTR1 %s\n", ship.name)
				}
			}

		post_end:
			fmt.Fprintf(report_file, "END\n\n")

			fmt.Fprintf(report_file, "START STRIKES\n")
			fmt.Fprintf(report_file, "; Place strike orders here.\n\n")
			fmt.Fprintf(report_file, "END\n")

			truncate_name = FALSE
			ignore_field_distorters = temp_ignore_field_distorters
		}

		/* done_report: Clean up for this species. */
		report_file.Close()
	}

	/* Clean up and exit. */
	free_species_data()

	return 0
}
