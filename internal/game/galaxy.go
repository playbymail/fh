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

// Port of galaxy.c.

import (
	"fmt"
	"os"
)

func createGalaxy(galacticRadius, desiredNumStars, desiredNumSpecies int) int {
	if galacticRadius < MIN_RADIUS || galacticRadius > MAX_RADIUS {
		fmt.Fprintf(os.Stderr, "error: galaxy must have a radius between %d and %d parsecs.\n", MIN_RADIUS, MAX_RADIUS)
		return 2
	}
	if desiredNumStars < MIN_STARS || desiredNumStars > MAX_STARS {
		fmt.Fprintf(os.Stderr, "error: galaxy must have between %d and %d star systems\n", MIN_STARS, MAX_STARS)
		return 2
	}
	if desiredNumSpecies < MIN_SPECIES || desiredNumSpecies > MAX_SPECIES {
		fmt.Fprintf(os.Stderr, "error: galaxy must have between %d and %d species\n", MIN_SPECIES, MAX_SPECIES)
		return 2
	}

	fmt.Printf(" info: radius      %6d\n", galacticRadius)
	fmt.Printf(" info: stars       %6d\n", desiredNumStars)
	fmt.Printf(" info: species     %6d\n", desiredNumSpecies)

	/* Get the number of cubic parsecs within a sphere with a radius of galacticRadius parsecs.
	 * Again, use long values to prevent loss of data by compilers that use 16-bit ints. */
	galactic_diameter := 2 * galacticRadius
	galactic_volume := (4 * 314 * galacticRadius * galacticRadius * galacticRadius) / 300

	/* THe chance_of_star is the probability of a star system existing at any particular
	 * set of x,y,z coordinates. It's the volume of the cluster divided by the desired
	 * number of stars. */
	chance_of_star := galactic_volume / desiredNumStars
	if chance_of_star < 50 {
		fmt.Fprintf(os.Stderr, "error: galactic radius is too small for %d stars\n", desiredNumStars)
		fmt.Fprintf(os.Stderr, "       galactic_volume   == %6d\n", galactic_volume)
		fmt.Fprintf(os.Stderr, "       desiredNumStars   == %6d\n", desiredNumStars)
		fmt.Fprintf(os.Stderr, "       chance_of_star    == %6d\n", chance_of_star)
		return 2
	} else if chance_of_star > 3200 {
		fmt.Fprintf(os.Stderr, "error: galactic radius is too large for %d stars\n", desiredNumStars)
		fmt.Fprintf(os.Stderr, "       galactic_volume   == %6d\n", galactic_volume)
		fmt.Fprintf(os.Stderr, "       desiredNumStars   == %6d\n", desiredNumStars)
		fmt.Fprintf(os.Stderr, "       chance_of_star    == %6d\n", chance_of_star)
		return 2
	}

	/* Initialize star location data.
	 * Set the z-coordinate to -1 as a flag that there's no star at the location. */
	var star_here [MAX_DIAMETER][MAX_DIAMETER]int
	for x := 0; x < galactic_diameter; x++ {
		for y := 0; y < galactic_diameter; y++ {
			star_here[x][y] = -1 // flag an invalid z-coordinate
		}
	}

	// randomly assign stars to locations within the galactic cluster.
	// this loop can take a very, very long time to complete since each
	// loop rolls against the chance_of_star to see if the system has a star.
	// a better solution might be to create a list of x,y,z, shuffle the list,
	// then pick from the list until we have the desired number of stars.
	for i := 0; i < desiredNumStars; {
		for { // loop until we place a star
			x := rnd(galactic_diameter) - 1
			y := rnd(galactic_diameter) - 1
			z := rnd(galactic_diameter) - 1

			real_x := x - galacticRadius
			real_y := y - galacticRadius
			real_z := z - galacticRadius

			// check that the coordinate is within the galactic boundary.
			sq_distance_from_center := (real_x * real_x) + (real_y * real_y) + (real_z * real_z)
			if sq_distance_from_center < galacticRadius*galacticRadius {
				// create a new star here if there's not already a star here.
				if star_here[x][y] == -1 {
					star_here[x][y] = z // z-coordinate
					i++
					break // break from this inner loop and add the next star
				}
			}
		}
	}

	galaxy.d_num_species = desiredNumSpecies
	galaxy.num_species = 0
	galaxy.radius = galacticRadius
	galaxy.turn_number = 0

	/* Allocate enough memory for star and planet data. */
	star_base = make([]*star_data_t, desiredNumStars)
	for i := range star_base {
		star_base[i] = &star_data_t{}
	}

	max_planets := 9 * desiredNumStars /* Maximum number possible. */
	planet_base = make([]*planet_data_t, max_planets)
	for i := range planet_base {
		planet_base[i] = &planet_data_t{}
	}

	// printing to stdout for backspace trick later...
	fmt.Fprintf(os.Stdout, "\nGenerating star number     ")

	pl_index := 0
	st_index := 0
	for x := 0; x < galactic_diameter; x++ {
		for y := 0; y < galactic_diameter; y++ {
			if star_here[x][y] == -1 {
				// no star at this location
				continue
			}

			star := star_base[st_index]
			*star = star_data_t{}

			/* Set coordinates. */
			star.x = x
			star.y = y
			star.z = star_here[x][y]

			/* Determine type of star. Make MAIN_SEQUENCE the most common star type. */
			star_type := rnd(GIANT + 6)
			if star_type > GIANT {
				star_type = MAIN_SEQUENCE
			}
			star.star_type = star_type

			/* Color and size of star are totally random. */
			star_color := rnd(RED)
			star.color = star_color
			star_size := rnd(10) - 1
			star.size = star_size

			/* Determine the number of planets in orbit around the star.
			 * The algorithm is something I tweaked until I liked it.
			 * It's weird, but it works. */
			d := RED + 2 - star_color
			/* Size of die.
			 * Big stars (blue, blue-white) roll bigger dice. Smaller stars (orange, red) roll smaller dice. */
			roll := star_type
			if roll > 2 {
				roll -= 1
			}
			/* Number of rolls:
			 *  - dwarves have 1 roll
			 *  - degenerates and main sequence stars have 2 rolls
			 *  - giants have 3 rolls. */
			star_num_planets := -2
			for i := 1; i <= roll; i++ {
				star_num_planets += rnd(d)
			}
			/* Trim down if too many. */
			for star_num_planets > 9 {
				star_num_planets -= rnd(3)
			}
			if star_num_planets < 1 {
				star_num_planets = 1
			}
			star.num_planets = star_num_planets

			/* Determine pl_index of first planet in file "planets.dat". */
			star.planet_index = pl_index

			/* Generate planets and write to file "planets.dat". */
			//star_data_t *current_star = star;
			generate_planets(planet_base[pl_index:], star_num_planets, FALSE, FALSE)

			star.home_system = FALSE
			star.worm_here = FALSE

			/* Update pointers and indices. */
			pl_index += star_num_planets

			st_index++
			if st_index%10 == 0 {
				fmt.Fprintf(os.Stdout, "\b\b\b\b%4d", st_index)
			}
		}
	}

	fmt.Fprintf(os.Stdout, "\b\b\b\b%4d\n", st_index)

	/* Allocate natural wormholes. */
	num_wormholes := 0
	for i := 0; i < desiredNumStars; i++ {
		star := star_base[i]
		if rnd(100) >= 92 && star.home_system == FALSE && star.worm_here == FALSE {
			/* There is a wormhole here. Get coordinates of other end. */
			var worm_star *star_data_t
			for worm_star == nil {
				endPoint := star_base[rnd(desiredNumStars)-1]
				if endPoint != star && endPoint.home_system == FALSE && endPoint.worm_here == FALSE {
					worm_star = endPoint
				}
			}

			// eliminate wormholes less than 20 parsecs in length
			dx := star.x - worm_star.x
			dy := star.y - worm_star.y
			dz := star.z - worm_star.z
			if (dx*dx)+(dy*dy)+(dz*dz) < 400 {
				continue
			}

			star.worm_here = TRUE
			star.worm_x = worm_star.x
			star.worm_y = worm_star.y
			star.worm_z = worm_star.z

			worm_star.worm_here = TRUE
			worm_star.worm_x = star.x
			worm_star.worm_y = star.y
			worm_star.worm_z = star.z

			num_wormholes++
		}
	}

	if st_index != desiredNumStars {
		fmt.Fprintf(os.Stderr, "error: internal consistency check #1 failed!!!\n")
		return 2
	}

	// set the global. ugh. required for the save_xxxx routines.
	num_stars = desiredNumStars
	num_planets = pl_index

	fmt.Printf(" info: this galaxy contains a total of %d stars and %d planets.\n", num_stars, num_planets)
	fmt.Printf("       the galaxy contains %d natural wormholes.\n", num_wormholes)

	// save data
	save_galaxy_data()
	fp, err := os.Create("galaxy.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "fh: export: sexpr:: %s\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create new version of file 'galaxy.txt'!\n")
		return 2
	}
	galaxyDataAsSexpr(fp)
	fp.Close()

	save_star_data()
	fp, err = os.Create("stars.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "fh: export: sexpr:: %s\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create new version of file 'stars.txt'!\n")
		return 2
	}
	starDataAsSExpr(star_base, num_stars, fp)
	fp.Close()

	save_planet_data()
	fp, err = os.Create("planets.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "fh: export: sexpr:: %s\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create new version of file 'planets.txt'!\n")
		return 2
	}
	planetDataAsSExpr(planet_base, num_planets, fp)
	fp.Close()

	star_base = nil
	planet_base = nil

	return 0
}
