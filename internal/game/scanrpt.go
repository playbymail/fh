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

// Port of scan.c. The scan(x, y, z, printLSN) helper itself lives in
// star.go (ported from star.c); this file only ports the CLI entries.

import (
	"fmt"
	"math"
	"os"
	"strings"
)

type scan_system_t struct {
	star     *star_data_t
	distance float64
}

// scanCommand performs a scan on a system for a species.
// It is for use by the game master only.
func scanCommand(args []string) int {
	cmdName := args[0]
	if len(args) != 5 {
		fmt.Fprintf(os.Stderr, "usage: fh scan speciesNumber x y z\n")
		return 2
	}
	spno := cfgAtoi(args[1])
	spidx := spno - 1
	x := cfgAtoi(args[2])
	y := cfgAtoi(args[3])
	z := cfgAtoi(args[4])

	fmt.Printf("fh: %s: loading   galaxy   data...\n", cmdName)
	get_galaxy_data()
	if spno < 1 || spno > galaxy.num_species {
		fmt.Fprintf(os.Stderr, "error: invalid species number\n")
	} else if x < 0 || x > 2*galaxy.radius {
		fmt.Fprintf(os.Stderr, "error: invalid x coordinate\n")
	} else if y < 0 || y > 2*galaxy.radius {
		fmt.Fprintf(os.Stderr, "error: invalid y coordinate\n")
	} else if z < 0 || z > 2*galaxy.radius {
		fmt.Fprintf(os.Stderr, "error: invalid z coordinate\n")
	}
	fmt.Printf("fh: %s: loading   star     data...\n", cmdName)
	get_star_data()
	fmt.Printf("fh: %s: loading   planet   data...\n", cmdName)
	get_planet_data()
	fmt.Printf("fh: %s: loading   species  data...\n", cmdName)
	get_species_data()

	fmt.Printf("Scan for SP %s:\n", spec_data[spidx].name)

	// set external globals for the scan command
	ignore_field_distorters = TRUE
	log_file = os.Stdout
	species_number = spno
	species_index = spidx
	species = &spec_data[species_index]
	nampla_base = namp_data[species_index]

	// display scan for the location
	scan(x, y, z, TRUE)

	return 0
}

// scanNearCommand scans all systems near a location.
// It is for use by the game master only.
func scanNearCommand(args []string) int {
	cmdName := args[0]
	x := 0
	y := 0
	z := 0
	radius := 0
	systems := FALSE

	for i := 1; i < len(args); i++ {
		opt := args[i]
		// In C, the option is split on the first '=' and val is NULL when
		// there is no '=' or when nothing follows it.
		val := ""
		if idx := strings.IndexByte(opt, '='); idx >= 0 {
			val = opt[idx+1:]
			opt = opt[:idx]
		}
		haveVal := val != ""
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr,
				"fh: usage: %s --x=integer --y=integer --z=integer --radius=integer\n", cmdName)
			return 2
		} else if opt == "--radius" && haveVal {
			radius = cfgAtoi(val)
		} else if opt == "--x" && haveVal {
			x = cfgAtoi(val)
		} else if opt == "--y" && haveVal {
			y = cfgAtoi(val)
		} else if opt == "--z" && haveVal {
			z = cfgAtoi(val)
		} else if opt == "--systems" {
			systems = TRUE
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	radiusSquared := radius * radius

	// external globals?
	ignore_field_distorters = TRUE
	log_file = os.Stdout

	//printf("fh: %s: loading   galaxy   data...\n", cmdName);
	get_galaxy_data()
	if x < 0 || x > 2*galaxy.radius {
		fmt.Fprintf(os.Stderr, "error: invalid x coordinate\n")
	} else if y < 0 || y > 2*galaxy.radius {
		fmt.Fprintf(os.Stderr, "error: invalid y coordinate\n")
	} else if z < 0 || z > 2*galaxy.radius {
		fmt.Fprintf(os.Stderr, "error: invalid z coordinate\n")
	} else if radius < 0 || radius > galaxy.radius {
		fmt.Fprintf(os.Stderr, "error: invalid radius\n")
	}
	//printf("fh: %s: loading   star     data...\n", cmdName);
	get_star_data()
	//printf("fh: %s: loading   planet   data...\n", cmdName);
	get_planet_data()
	//printf("fh: %s: loading   species  data...\n", cmdName);
	get_species_data()

	/* Display scan. */
	if systems == FALSE {
		fmt.Printf("Ships and populated planets within %d parsecs of %d %d %d:\n", radius, x, y, z)
		for spidx := 0; spidx < galaxy.num_species; spidx++ {
			if data_in_memory[spidx] == FALSE {
				continue
			}
			species_printed := FALSE
			species_number = spidx + 1
			species = &spec_data[spidx]

			/* Set dest_x for all ships to zero.
			 * We will use this to prevent multiple listings of a ship. */
			for ship_index := 0; ship_index < species.num_ships; ship_index++ {
				// The C code computes sd = ship_data[spidx] + ship_index and
				// then writes sd[ship_index].dest_x, i.e. it only clears the
				// even-indexed ships (writes past num_ships land in the
				// allocation headroom and are never read). Replicate that.
				if ship_index+ship_index < len(ship_data[spidx]) {
					ship_data[spidx][ship_index+ship_index].dest_x = 0
				}
			}

			/* Check all namplas for this species. */
			for namplaIndex := 0; namplaIndex < species.num_namplas; namplaIndex++ {
				nd := namp_data[spidx][namplaIndex]
				if nd.status&POPULATED == 0 {
					continue
				}
				delta_x := x - nd.x
				delta_y := y - nd.y
				delta_z := z - nd.z
				distance_squared := (delta_x * delta_x) + (delta_y * delta_y) + (delta_z * delta_z)
				if distance_squared > radiusSquared {
					continue
				}
				if species_printed == FALSE {
					fmt.Printf("  Species #%d, SP %s:\n", species_number, species.name)
					species_printed = TRUE
				}
				fmt.Printf("    %2d %2d %2d #%d", nd.x, nd.y, nd.z, nd.pn)
				if nd.status&HOME_PLANET != 0 {
					fmt.Printf("  Home planet")
				} else if nd.status&MINING_COLONY != 0 {
					fmt.Printf("  Mining colony")
				} else if nd.status&RESORT_COLONY != 0 {
					fmt.Printf("  Resort colony")
				} else {
					fmt.Printf("  Normal colony")
				}
				fmt.Printf(" PL %s, EB = %d.%d, %d Yrds",
					nd.name,
					(nd.mi_base+nd.ma_base)/10, (nd.mi_base+nd.ma_base)%10,
					nd.shipyards)
				for i := 0; i < MAX_ITEMS; i++ {
					if nd.item_quantity[i] > 0 {
						fmt.Printf(", %d %s", nd.item_quantity[i], item_abbr[i])
					}
				}
				if nd.hidden != FALSE {
					fmt.Printf(", HIDING!")
				}
				fmt.Printf("\n")

				/* List ships at this colony. */
				for ship_index := 0; ship_index < species.num_ships; ship_index++ {
					sd := ship_data[spidx][ship_index]
					if sd.dest_x != 0 {
						/* Already listed. */
						continue
					} else if sd.x != nd.x || sd.y != nd.y || sd.z != nd.z ||
						sd.pn != nd.pn {
						// not at this colony
						continue
					}
					fmt.Printf("                 %s", ship_name(sd))
					for i := 0; i < MAX_ITEMS; i++ {
						if sd.item_quantity[i] > 0 {
							fmt.Printf(", %d %s", sd.item_quantity[i], item_abbr[i])
						}
					}
					fmt.Printf("\n")
					sd.dest_x = 99 /* Do not list this ship again. */
				}
			}

			for shipIndex := 0; shipIndex < species.num_ships; shipIndex++ {
				sd := ship_data[spidx][shipIndex]
				if sd.pn == 99 {
					// sometimes 99 means that the ship's slot is not used?
					continue
				} else if sd.dest_x != 0 {
					/* Already listed above. */
					continue
				}
				delta_x := x - sd.x
				delta_y := y - sd.y
				delta_z := z - sd.z
				distance_squared := (delta_x * delta_x) + (delta_y * delta_y) + (delta_z * delta_z)
				if distance_squared > radiusSquared {
					continue
				}
				if species_printed == FALSE {
					fmt.Printf("  Species #%d, SP %s:\n", species_number, species.name)
					species_printed = TRUE
				}
				fmt.Printf("    %2d %2d %2d", sd.x, sd.y, sd.z)
				fmt.Printf("     %s", ship_name(sd))
				for i := 0; i < MAX_ITEMS; i++ {
					if sd.item_quantity[i] > 0 {
						fmt.Printf(", %d %s", sd.item_quantity[i], item_abbr[i])
					}
				}
				fmt.Printf("\n")
				sd.dest_x = 99 /* Do not list this ship again. */
			}
		}
		return 0
	}

	fmt.Printf("Systems within %d parsecs of %d %d %d:\n", radius, x, y, z)
	ss := make([]*scan_system_t, num_stars)
	for i := 0; i < num_stars; i++ {
		ss[i] = &scan_system_t{}
		ss[i].star = star_base[i]
		delta_x := x - ss[i].star.x
		delta_y := y - ss[i].star.y
		delta_z := z - ss[i].star.z
		ss[i].distance = math.Sqrt(float64((delta_x * delta_x) + (delta_y * delta_y) + (delta_z * delta_z)))
	}
	// bubbly and proud
	for i := 0; i < num_stars; i++ {
		for j := i + 1; j < num_stars; j++ {
			if ss[i].distance > ss[j].distance {
				tmp := ss[i]
				ss[i] = ss[j]
				ss[j] = tmp
			}
		}
	}
	for i := 0; i < num_stars; i++ {
		if ss[i].distance < float64(radius) {
			star := ss[i].star
			fmt.Printf("System  x = %3d  y = %3d  z = %3d  %c%c%c %8.2f parsecs\n",
				star.x, star.y, star.z,
				type_char[star.star_type], color_char[star.color], size_char[star.size],
				ss[i].distance)
			if star.worm_here != FALSE {
				fmt.Printf("\t*** terminus of a natural wormhole.\n")
			}
			if star.num_planets == 0 {
				fmt.Printf("\t*** nova remnant, no planets.\n\n")
				continue
			}
			fmt.Printf("\t#  Dia  Grav  TempClass  PressClass  MiningDiff\n")
			fmt.Printf("\t-----------------------------------------------\n")
			for pn := 0; pn < star.num_planets; pn++ {
				// NOTE: the "- 1" below is in the original C code, which
				// reads one record before the system's first planet.
				planet := planet_base[star.planet_index+pn-1]
				orbit := pn + 1
				fmt.Printf("\t%d  %3d  %d.%02d  %9d  %10d  %7d.%02d\n",
					orbit,
					planet.diameter,
					planet.gravity/100, planet.gravity%100,
					planet.temperature_class,
					planet.pressure_class,
					planet.mining_difficulty/100, planet.mining_difficulty%100)
			}
		}
	}

	return 0
}
