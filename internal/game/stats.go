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

package game

import (
	"fmt"
)

func statsCommand(args []string) int {
	cmdName := args[0]

	fmt.Printf("fh: %s: loading   galaxy   data...\n", cmdName)
	get_galaxy_data()
	fmt.Printf("fh: %s: loading   planet   data...\n", cmdName)
	get_planet_data()
	fmt.Printf("fh: %s: loading   species  data...\n", cmdName)
	get_species_data()

	/* Initialize data. */
	var all_production int = 0
	var all_starbase_tons int = 0
	var all_tech_level [6]int
	var all_transport_tons int = 0
	var all_warship_tons int = 0
	var avg_pop_pl int = 0
	var avg_production int
	var avg_starbase_tons int
	var avg_tech_level int = 0
	var avg_transport_tons int
	var avg_warship_tons int
	var avg_yards int = 0
	var fleet_percent_cost int = 0
	var i int = 0
	var ls_needed int = 0
	var m int = 0
	var max_pop_pl int = 0
	var max_production int = 0
	var max_starbases int = 0
	var max_tech_level [6]int
	var max_transports int = 0
	var max_warships int = 0
	var max_yards int = 0
	var min_pop_pl int = 32000
	var min_production int = 1000000000
	var min_starbases int = 32000
	var min_tech_level [6]int
	var min_transports int = 32000
	var min_warships int = 32000
	var min_yards int = 32000
	var n_pop_pl int = 0
	var n_species int = 0
	var n_warships int = 0
	var n_starbases int = 0
	var n_transports int = 0
	var n_yards int = 0
	var n1 int
	var n2 int
	var n3 int
	var nba int = 0
	var ntr int = 0
	var num_pop_planets int = 0
	var num_ships int = 0
	var num_yards int = 0
	var nwa int = 0
	var production_capacity int = 0
	var production_penalty int = 0
	var raw_material_units int = 0
	var tons int = 0
	var total_defensive_power int
	var total_offensive_power int
	var total_production int
	var total_tonnage int

	var totalBankedEconUnits, minBankedEconUnits, maxBankedEconUnits, avgBankedEconUnits int

	var nampla_index int = 0
	var ship_index int = 0

	for i := 0; i < 6; i++ {
		all_tech_level[i] = 0
		min_tech_level[i] = 32000
		max_tech_level[i] = 0
	}

	/* Print header. */
	fmt.Printf("SP Species               Tech Levels        Total  Num Num  Num  Offen.  Defen.  Econ\n")
	fmt.Printf(" # Name             MI  MA  ML  GV  LS  BI  Prod.  Pls Shps Yrds  Power   Power  Units\n")
	fmt.Printf("----------------------------------------------------------------------------------------\n")

	/* Main loop. For each species, take appropriate action. */
	for species_number = 1; species_number <= galaxy.num_species; species_number++ {
		if data_in_memory[species_number-1] == FALSE {
			continue
		}
		n_species++

		species = &spec_data[species_number-1]
		nampla_base = namp_data[species_number-1]
		ship_base = ship_data[species_number-1]

		/* Get fleet maintenance cost. */
		fleet_percent_cost = species.fleet_percent_cost

		if fleet_percent_cost > 10000 {
			fleet_percent_cost = 10000
		}

		/* Print species data. */
		fmt.Printf("%2d", species_number)
		fmt.Printf(" %-15.15s", species.name)

		for i := 0; i < 6; i++ {
			fmt.Printf("%4d", species.tech_level[i])
			all_tech_level[i] += species.tech_level[i]
			if species.tech_level[i] < min_tech_level[i] {
				min_tech_level[i] = species.tech_level[i]
			}
			if species.tech_level[i] > max_tech_level[i] {
				max_tech_level[i] = species.tech_level[i]
			}
		}

		/* Get stats for namplas. */
		total_production = 0
		total_defensive_power = 0
		num_yards = 0
		num_pop_planets = 0
		home_planet = planet_base[nampla_base[0].planet_index]
		for nampla_index = 0; nampla_index < species.num_namplas; nampla_index++ {
			nampla = nampla_base[nampla_index]
			if nampla.pn == 99 {
				continue
			}

			num_yards += nampla.shipyards
			n_yards += nampla.shipyards

			planet = planet_base[nampla.planet_index]

			raw_material_units =
				(10 * species.tech_level[MI] * nampla.mi_base) / planet.mining_difficulty

			production_capacity = (species.tech_level[MA] * nampla.ma_base) / 10

			ls_needed = life_support_needed(species, home_planet, planet)

			if ls_needed == 0 {
				production_penalty = 0
			} else {
				production_penalty = (100 * ls_needed) / species.tech_level[LS]
			}

			raw_material_units -= (production_penalty * raw_material_units) / 100

			raw_material_units = ((planet.econ_efficiency * raw_material_units) + 50) / 100

			production_capacity -= (production_penalty * production_capacity) / 100

			production_capacity = ((planet.econ_efficiency * production_capacity) + 50) / 100

			if nampla.status&MINING_COLONY != 0 {
				n1 = (2 * raw_material_units) / 3
			} else if nampla.status&RESORT_COLONY != 0 {
				n1 = (2 * production_capacity) / 3
			} else {
				if production_capacity > raw_material_units {
					n1 = raw_material_units
				} else {
					n1 = production_capacity
				}
			}

			n2 = ((fleet_percent_cost * n1) + 5000) / 10000
			n3 = n1 - n2
			total_production += n3

			tons = nampla.item_quantity[PD] / 200
			if tons < 1 && nampla.item_quantity[PD] > 0 {
				tons = 1
			}
			total_defensive_power += power(tons)

			if nampla.status&POPULATED != 0 {
				n_pop_pl++
				num_pop_planets++
			}
		}

		fmt.Printf("%7d%4d", total_production, num_pop_planets)

		if total_production < min_production {
			min_production = total_production
		}
		if total_production > max_production {
			max_production = total_production
		}

		if num_pop_planets < min_pop_pl {
			min_pop_pl = num_pop_planets
		}
		if num_pop_planets > max_pop_pl {
			max_pop_pl = num_pop_planets
		}

		if num_yards < min_yards {
			min_yards = num_yards
		}
		if num_yards > max_yards {
			max_yards = num_yards
		}

		all_production += total_production

		/* Get stats for ships. */
		num_ships = 0
		ntr = 0
		nba = 0
		nwa = 0
		total_tonnage = 0
		total_offensive_power = 0
		for ship_index = 0; ship_index < species.num_ships; ship_index++ {
			ship = ship_base[ship_index]
			if ship.pn == 99 {
				continue
			} else if ship.status == UNDER_CONSTRUCTION {
				continue
			}

			num_ships++
			total_tonnage += ship.tonnage

			if ship.ship_type == STARBASE {
				total_defensive_power += power(ship.tonnage)
				all_starbase_tons += ship.tonnage
				n_starbases++
				nba++
			} else if ship.class == TR {
				all_transport_tons += ship.tonnage
				n_transports++
				ntr++
			} else {
				if ship.ship_type == SUB_LIGHT {
					total_defensive_power += power(ship.tonnage)
				} else {
					total_offensive_power += power(ship.tonnage)
				}
				all_warship_tons += ship.tonnage
				n_warships++
				nwa++
			}
		}

		if nwa < min_warships {
			min_warships = nwa
		}
		if nwa > max_warships {
			max_warships = nwa
		}

		if nba < min_starbases {
			min_starbases = nba
		}
		if nba > max_starbases {
			max_starbases = nba
		}

		if ntr < min_transports {
			min_transports = ntr
		}
		if ntr > max_transports {
			max_transports = ntr
		}

		total_offensive_power +=
			(species.tech_level[ML] * total_offensive_power) / 50

		total_defensive_power +=
			(species.tech_level[ML] * total_defensive_power) / 50

		if species.tech_level[ML] == 0 {
			total_defensive_power = 0
			total_offensive_power = 0
		}

		total_offensive_power /= 10
		total_defensive_power /= 10

		fmt.Printf("%5d", num_ships)
		fmt.Printf("%5d", num_yards)
		fmt.Printf("%8d%8d", total_offensive_power, total_defensive_power)

		totalBankedEconUnits += species.econ_units
		if species_number == 1 {
			minBankedEconUnits = species.econ_units
			maxBankedEconUnits = species.econ_units
		} else {
			if minBankedEconUnits > species.econ_units {
				minBankedEconUnits = species.econ_units
			}
			if maxBankedEconUnits < species.econ_units {
				maxBankedEconUnits = species.econ_units
			}
		}
		fmt.Printf("%9d\n", species.econ_units)
	}

	m = n_species / 2
	fmt.Printf("\n")
	for i := 0; i < 6; i++ {
		avg_tech_level = (all_tech_level[i] + m) / n_species
		fmt.Printf("Average %s tech level = %d (min = %d, max = %d)\n",
			tech_name[i], avg_tech_level, min_tech_level[i], max_tech_level[i])
	}

	i = ((10 * n_warships) + m) / n_species
	fmt.Printf("\nAverage number of warships per species = %d.%d (min = %d, max = %d)\n",
		i/10, i%10, min_warships, max_warships)

	if n_warships == 0 {
		n_warships = 1
	}
	avg_warship_tons = (10000 * all_warship_tons) / n_warships
	avg_warship_tons = 1000 * ((avg_warship_tons + 500) / 1000)
	fmt.Printf("Average warship size = %s tons\n", commas(avg_warship_tons))

	avg_warship_tons = (10000 * all_warship_tons) / n_species
	avg_warship_tons = 1000 * ((avg_warship_tons + 500) / 1000)
	fmt.Printf("Average total warship tonnage per species = %s tons\n", commas(avg_warship_tons))

	i = ((10 * n_starbases) + m) / n_species
	fmt.Printf("\nAverage number of starbases per species = %d.%d (min = %d, max = %d)\n",
		i/10, i%10, min_starbases, max_starbases)

	if n_starbases == 0 {
		n_starbases = 1
	}
	avg_starbase_tons = (10000 * all_starbase_tons) / n_starbases
	avg_starbase_tons = 1000 * ((avg_starbase_tons + 500) / 1000)
	fmt.Printf("Average starbase size = %s tons\n", commas(avg_starbase_tons))

	avg_starbase_tons = (10000 * all_starbase_tons) / n_species
	avg_starbase_tons = 1000 * ((avg_starbase_tons + 500) / 1000)
	fmt.Printf("Average total starbase tonnage per species = %s tons\n", commas(avg_starbase_tons))

	i = ((10 * n_transports) + m) / n_species
	fmt.Printf("\nAverage number of transports per species = %d.%d (min = %d, max = %d)\n",
		i/10, i%10, min_transports, max_transports)

	if n_transports == 0 {
		n_transports = 1
	}
	avg_transport_tons = (10000 * all_transport_tons) / n_transports
	avg_transport_tons = 1000 * ((avg_transport_tons + 500) / 1000)
	fmt.Printf("Average transport size = %s tons\n", commas(avg_transport_tons))

	avg_transport_tons = (10000 * all_transport_tons) / n_species
	avg_transport_tons = 1000 * ((avg_transport_tons + 500) / 1000)
	fmt.Printf("Average total transport tonnage per species = %s tons\n", commas(avg_transport_tons))

	avg_yards = ((10 * n_yards) + m) / n_species
	fmt.Printf("\nAverage number of shipyards per species = %d.%d (min = %d, max = %d)\n",
		avg_yards/10, avg_yards%10, min_yards, max_yards)

	avg_pop_pl = ((10 * n_pop_pl) + m) / n_species
	fmt.Printf("\nAverage number of populated planets per species = %d.%d (min = %d, max = %d)\n",
		avg_pop_pl/10, avg_pop_pl%10, min_pop_pl, max_pop_pl)

	avg_production = (all_production + m) / n_species
	fmt.Printf("Average total production per species = %d (min = %d, max = %d)\n",
		avg_production, min_production, max_production)

	avgBankedEconUnits = (totalBankedEconUnits + m) / n_species
	fmt.Printf("\nAverage banked economic units per species = %d (min = %d, max = %d)\n",
		avgBankedEconUnits, minBankedEconUnits, maxBankedEconUnits)

	return 0
}
