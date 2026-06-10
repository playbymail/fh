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

// Port of star.c.

import (
	"fmt"
	"math"
	"os"
)

// color_char, size_char, and type_char are declared in tables.go.

// changeSystemToHomeSystem replaces the planets in a system with ones
// from the related homesystem template. the template flags one of the
// planets in the system as a home planet.
func changeSystemToHomeSystem(star *star_data_t) int {
	if star == nil {
		fmt.Fprintf(os.Stderr, "error: changeSystemToHomeSystem: internal error, star is NULL\n")
		os.Exit(2)
	}

	fmt.Printf(" info: updating system id %4d at %3d %3d %3d planet index %4d\n",
		star.id, star.x, star.y, star.z, star.planet_index)

	// load the home system template
	filename := fmt.Sprintf("homesystem%d.dat", star.num_planets)
	fmt.Printf(" info: loading template '%s'\n", filename)
	templateSystem := getPlanetData(0, filename)
	if templateSystem == nil {
		fmt.Fprintf(os.Stderr, "error: changeSystemToHomeSystem: unable to load template '%s'\n", filename)
		os.Exit(2)
	}
	fmt.Printf(" info: loaded template from '%s'\n", filename)

	// make minor random modifications to the template
	for pn := 0; pn < star.num_planets; pn++ {
		planet := templateSystem[pn]
		if planet.special == 1 {
			fmt.Printf(" info: randomizing home world: orbit %d\n", pn+1)
		} else {
			fmt.Printf(" info: randomizing planet    : orbit %d\n", pn+1)
		}
		if planet.temperature_class > 12 {
			planet.temperature_class -= rnd(3) - 1
		} else if planet.temperature_class > 0 {
			planet.temperature_class += rnd(3) - 1
		}
		if planet.pressure_class > 12 {
			planet.pressure_class -= rnd(3) - 1
		} else if planet.pressure_class > 0 {
			planet.pressure_class += rnd(3) - 1
		}
		if planet.gas[2] > 0 {
			roll := rnd(25) + 10
			if planet.gas_percent[2] > 50 {
				planet.gas_percent[1] += roll
				planet.gas_percent[2] -= roll
			} else if planet.gas_percent[1] > 50 {
				planet.gas_percent[1] -= roll
				planet.gas_percent[2] += roll
			}
		}
		if planet.diameter > 12 {
			planet.diameter -= rnd(3) - 1
		} else if planet.diameter > 0 {
			planet.diameter += rnd(3) - 1
		}
		if planet.gravity > 100 {
			planet.gravity -= rnd(10)
		} else if planet.gravity > 0 {
			planet.gravity += rnd(10)
		}
		if planet.mining_difficulty > 100 {
			planet.mining_difficulty -= rnd(10)
		} else if planet.mining_difficulty > 0 {
			planet.mining_difficulty += rnd(10)
		}
	}

	// copy from the template into the system's planet data
	for pn := 0; pn < star.num_planets; pn++ {
		p := planet_base[star.planet_index+pn]
		pd := templateSystem[pn]
		p.temperature_class = pd.temperature_class
		p.pressure_class = pd.pressure_class
		p.special = pd.special
		for g := 0; g < 4; g++ {
			p.gas[g] = pd.gas[g]
			p.gas_percent[g] = pd.gas_percent[g]
		}
		p.diameter = pd.diameter
		p.gravity = pd.gravity
		p.mining_difficulty = pd.mining_difficulty
		p.econ_efficiency = pd.econ_efficiency
		p.md_increase = pd.md_increase
		p.message = pd.message
		p.isValid = pd.isValid
	}
	star.home_system = TRUE

	return 0
}

func chToStarColor(ch byte) int {
	// space is supposed to be invalid, so work around
	for i := 1; i < len(color_char); i++ {
		if color_char[i] == ch {
			return i
		}
	}
	// the C loop bound is sizeof(color_char), which includes the NUL
	// terminator at index 8.
	if ch == 0 {
		return len(color_char)
	}
	return 0
}

func chToStarType(ch byte) int {
	// space is valid for two types of star, so work around
	for i := 1; i < len(type_char); i++ {
		if type_char[i] == ch {
			return i
		}
	}
	// the C loop bound is sizeof(type_char), which includes the NUL
	// terminator at index 5.
	if ch == 0 {
		return len(type_char)
	}
	return 0
}

func closest_unvisited_star(ship *ship_data_t) {
	found := FALSE
	var shx, shy, shz int
	var stx, sty, stz int
	closest_distance := 999999
	var temp_distance int
	var star *star_data_t
	var closest_star *star_data_t

	/* Get array index and bit mask. */
	species_array_index := (species_number - 1) / 32
	species_bit_number := (species_number - 1) % 32
	species_bit_mask := uint32(1) << uint(species_bit_number)

	shx = ship.x
	shy = ship.y
	shz = ship.z

	x = -1

	for i := 0; i < num_stars; i++ {
		star = star_base[i]

		/* Check if bit is already set. */
		if star.visited_by[species_array_index]&species_bit_mask != 0 {
			continue
		}

		stx = star.x
		sty = star.y
		stz = star.z
		temp_distance = ((shx - stx) * (shx - stx)) + ((shy - sty) * (shy - sty)) + ((shz - stz) * (shz - stz))

		if temp_distance < closest_distance {
			x = stx
			y = sty
			z = stz
			closest_distance = temp_distance
			closest_star = star
			found = TRUE
		}
	}

	if found != FALSE {
		fmt.Fprintf(orders_file, "%d %d %d", x, y, z)
		/* So that we don't send more than one ship to the same place. */
		closest_star.visited_by[species_array_index] |= species_bit_mask
	} else {
		fmt.Fprintf(orders_file, "???")
	}
}

// closest_unvisited_star_report is just slight different? why?
func closest_unvisited_star_report(ship *ship_data_t, fp *os.File) {
	var found int
	var shx, shy, shz, stx, sty, stz, closest_distance, temp_distance int
	var star, closest_star *star_data_t

	/* Get array index and bit mask. */
	species_array_index := (species_number - 1) / 32
	species_bit_number := (species_number - 1) % 32
	species_bit_mask := uint32(1) << uint(species_bit_number)

	shx = ship.x
	shy = ship.y
	shz = ship.z

	x = 9999
	closest_distance = 999999

	found = FALSE
	for i := 0; i < num_stars; i++ {
		star = star_base[i]

		/* Check if bit is already set. */
		if star.visited_by[species_array_index]&species_bit_mask != 0 {
			continue
		}

		stx = star.x
		sty = star.y
		stz = star.z

		temp_distance = ((shx - stx) * (shx - stx)) + ((shy - sty) * (shy - sty)) + ((shz - stz) * (shz - stz))

		if temp_distance < closest_distance {
			x = stx
			y = sty
			z = stz
			closest_distance = temp_distance
			closest_star = star
			found = TRUE
		}
	}

	if found != FALSE {
		fmt.Fprintf(fp, "%d %d %d", x, y, z)
		closest_star.visited_by[species_array_index] |= species_bit_mask
		/* So that we don't send more than one ship to the same place. */
	} else {
		fmt.Fprintf(fp, "???")
	}
}

func distanceBetween(s1, s2 *star_data_t) float64 {
	dX := float64(s1.x - s2.x)
	dY := float64(s1.y - s2.y)
	dZ := float64(s1.z - s2.z)
	return math.Sqrt(dX*dX + dY*dY + dZ*dZ)
}

// findHomeSystemCandidate returns a randomly picked system that has at least 3 planets,
// is not currently a home system, is not a worm_hole endpoint, and is at least the
// minimum distance from any existing home system. it returns NULL if there are no such systems.
func findHomeSystemCandidate(radius int) *star_data_t {
	candidates := make([]*star_data_t, num_stars+1)
	sidx := 0
	for i := 0; i < num_stars; i++ {
		candidate := star_base[i]
		if candidate.num_planets >= 3 && candidate.home_system == FALSE && candidate.worm_here == FALSE {
			candidates[sidx] = candidate
			sidx++
		}
	}
	if sidx == 0 {
		// fprintf(stderr, "error: findHomeSystemCandidate: no candidates meet the criteria for home system!\n");
		return nil
	}
	// pick one at random by shuffling the list, then iterating through it until we find a match.
	// Fisher and Yates shuffle, updated
	// -- To shuffle an array A of n elements (indices 0..n-1):
	//    for i from n−1 downto 1 do
	//        j ← random integer such that 0 ≤ j ≤ i
	//        swap(A[i], A[j])
	for i := sidx - 1; i > 0; i-- {
		// rnd(i)         returns 1 ≤ x ≤ i
		// rnd(i + 1)     returns 1 ≤ x ≤ i+1
		// rnd(i + 1) - 1 returns 0 ≤ x ≤ i
		j := rnd(i+1) - 1
		tmp := candidates[j]
		candidates[j] = candidates[i]
		candidates[i] = tmp
	}

	// return the first system from the list of candidates that meets the minimum distance criteria.
	for i := 0; candidates[i] != nil; i++ {
		if hasHomeSystemNeighbor(candidates[i], radius) == FALSE {
			candidate := candidates[i]
			return candidate
		}
	}
	// fprintf(stderr, "error: findHomeSystemCandidate: no candidates meet the criteria for radius of %d!\n", radius);

	return nil
}

// hasHomeSystemNeighbor returns TRUE if the star has a neighbor within the given radius that is a home system.
func hasHomeSystemNeighbor(star *star_data_t, radius int) int {
	radiusSquared := radius * radius
	for i := 0; i < num_stars; i++ {
		star2 := star_base[i]
		if star2.home_system == FALSE {
			continue
		}
		dx := star.x - star2.x
		dy := star.y - star2.y
		dz := star.z - star2.z
		if dx*dx+dy*dy+dz*dz <= radiusSquared {
			return TRUE
		}
	}
	return FALSE
}

func scan(x, y, z, printLSN int) {
	var i, n, found, num_gases, ls_needed int
	var filename string
	var star *star_data_t
	var planet, home_planet *planet_data_t
	var home_nampla *nampla_data_t

	/* Find star. */
	found = FALSE
	for i = 0; i < num_stars; i++ {
		star = star_base[i]
		if star.x == x && star.y == y && star.z == z {
			found = TRUE
			break
		}
	}

	if found == FALSE {
		fmt.Fprintf(log_file, "Scan Report: There is no star system at x = %d, y = %d, z = %d.\n", x, y, z)
		return
	}

	/* Print data for star, */
	fmt.Fprintf(log_file, "Coordinates:\tx = %d\ty = %d\tz = %d", x, y, z)
	fmt.Fprintf(log_file, "\tstellar type = %c%c%c", type_char[star.star_type], color_char[star.color], size_char[star.size])
	fmt.Fprintf(log_file, "   %d planets.\n\n", star.num_planets)

	if star.worm_here != FALSE {
		fmt.Fprintf(log_file, "This star system is the terminus of a natural wormhole.\n\n")
	}

	/* Print header. */
	fmt.Fprintf(log_file, "               Temp  Press Mining\n")
	fmt.Fprintf(log_file, "  #  Dia  Grav Class Class  Diff  LSN  Atmosphere\n")
	fmt.Fprintf(log_file, " ---------------------------------------------------------------------\n")

	/* Check for nova. */
	if star.num_planets == 0 {
		fmt.Fprintf(log_file, "\n\tThis star is a nova remnant. Any planets it may have once\n")
		fmt.Fprintf(log_file, "\thad have been blown away.\n\n")
		return
	}

	/* Print data for each planet. */
	planet_index := star.planet_index
	planet = planet_base[planet_index]
	if printLSN != FALSE {
		home_nampla = nampla_base[0]
		home_planet = planet_base[home_nampla.planet_index]
	}

	for i = 1; i <= star.num_planets; i++ {
		/* Get life support tech level needed. */
		if printLSN != FALSE {
			ls_needed = life_support_needed(species, home_planet, planet)
		} else {
			ls_needed = 99
		}

		fmt.Fprintf(log_file, "  %d  %3d  %d.%02d  %2d    %2d    %d.%02d %4d  ",
			i,
			planet.diameter,
			planet.gravity/100, planet.gravity%100,
			planet.temperature_class,
			planet.pressure_class,
			planet.mining_difficulty/100, planet.mining_difficulty%100,
			ls_needed)

		num_gases = 0
		for n = 0; n < 4; n++ {
			if planet.gas_percent[n] > 0 {
				if num_gases > 0 {
					fmt.Fprintf(log_file, ",")
				}
				fmt.Fprintf(log_file, "%s(%d%%)", gas_string[planet.gas[n]], planet.gas_percent[n])
				num_gases++
			}
		}
		if num_gases == 0 {
			fmt.Fprintf(log_file, "No atmosphere")
		}
		fmt.Fprintf(log_file, "\n")

		planet_index++
		if planet_index < len(planet_base) {
			planet = planet_base[planet_index]
		}
	}

	if star.message != 0 {
		/* There is a message that must be logged whenever this star system is scanned. */
		filename = fmt.Sprintf("message%d.txt", star.message)
		log_message(filename)
	}
}

func star_color(c int) byte {
	if 0 <= c && c <= 7 {
		return color_char[c]
	}
	return '?'
}

func star_size(c int) byte {
	if 0 <= c && c <= 9 {
		return size_char[c]
	}
	return '?'
}

func star_type(c int) byte {
	if 0 <= c && c <= 4 {
		return type_char[c]
	}
	return '?'
}

/* The following routine will check if coordinates x-y-z contain a star and,
 * if so, will set the appropriate bit in the "visited_by" variable for the star.
 * If the star exists, TRUE will be returned; otherwise, FALSE will be returned. */
func star_visited(x, y, z int) int {
	found := FALSE

	/* Get array index and bit mask. */
	species_array_index := (species_number - 1) / 32
	species_bit_number := (species_number - 1) % 32
	species_bit_mask := uint32(1) << uint(species_bit_number)

	for i := 0; i < num_stars; i++ {
		star := star_base[i]
		if x != star.x {
			continue
		}
		if y != star.y {
			continue
		}
		if z != star.z {
			continue
		}
		found = TRUE
		/* Check if bit is already set. */
		if star.visited_by[species_array_index]&species_bit_mask != 0 {
			break
		}
		/* Set the appropriate bit. */
		star.visited_by[species_array_index] |= species_bit_mask
		star_data_modified = TRUE
		break
	}

	return found
}
