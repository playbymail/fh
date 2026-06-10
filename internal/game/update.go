package game

// Port of update.c.

import (
	"fmt"
	"os"
)

func updateCommand(args []string) int {
	cmdName := args[0]
	get_galaxy_data()
	get_star_data()
	get_planet_data()
	for i := 1; i < len(args); i++ {
		// fprintf(stderr, "fh: %s: argc %2d argv '%s'\n", cmdName, i, argv[i]);
		opt, _, _ := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr,
				"fh: usage: update (home-system | planet | ship | species | star)\n")
			return 2
		} else if opt == "home-system" {
			return updateHomeSystem(args[i:])
		} else if opt == "planet" {
			return updatePlanet(args[i:])
		} else if opt == "ship" {
			return updateShip(args[i:])
		} else if opt == "species" {
			return updateSpecies(args[i:])
		} else if opt == "star" {
			return updateStar(args[i:])
		} else {
			fmt.Fprintf(os.Stderr, "fh: %s: unknown option '%s'\n", cmdName, opt)
			return 2
		}
	}
	return 0
}

// updateHomeSystem changes the type of a planet in an existing star system to HOME_PLANET
// and then replaces the existing planets in the system with ones from the related homesystem
// template.
// by default, this function chooses a system at random from the set of systems that:
//
//	are at least some minimum distance from other home systems
//	do not already contain a home planet
//
// the user may specify the minimum distance and/or the locations of a system to use.
// if a system is specified, the use may pass in a flag to ignore the minimum distance
// check described above. this is useful if the gamemaster intends to force species to
// be close neighbors.
// there is no way to force multiple planets in a system to be home planets.
func updateHomeSystem(args []string) int {
	cmdName := args[0]
	chooseRandomly := TRUE
	force := FALSE
	radius := 10
	x := 0
	y := 0
	z := 0

	for i := 1; i < len(args); i++ {
		// fprintf(stderr, "fh: %s: argc %2d argv '%s'\n", cmdName, i, argv[i]);
		opt, val, hasVal := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "fh: usage: update home-system options...\n")
			fmt.Fprintf(os.Stderr, "           applies a home system template to a system\n")
			fmt.Fprintf(os.Stderr, "      opt: --radius=n      minimum distance between home systems [default 10]\n")
			fmt.Fprintf(os.Stderr,
				"           --system=x,y,z  coordinates of system to update       [defaults to random system]\n")
			fmt.Fprintf(os.Stderr, "      opt: --force         ignore radius and home system check\n")
			return 2
		} else if opt == "--force" && !hasVal {
			force = TRUE
		} else if opt == "--radius" && hasVal {
			radius = cfgAtoi(val)
			if radius < 1 || radius > galaxy.radius {
				fmt.Fprintf(os.Stderr, "error: invalid radius '%s'\n", val)
				return 2
			}
		} else if opt == "--system" && hasVal {
			vi := 0
			for ; vi < len(val) && val[vi] != ','; vi++ {
				if !isdigit(val[vi]) {
					fmt.Fprintf(os.Stderr, "error: system coordinates must be numeric\n")
					return 2
				}
				x = x*10 + int(val[vi]-'0')
			}
			if at(val, vi) != ',' {
				fmt.Fprintf(os.Stderr, "error: system coordinates must be separated by commas\n")
				return 2
			}
			vi++
			for ; vi < len(val) && val[vi] != ','; vi++ {
				if !isdigit(val[vi]) {
					fmt.Fprintf(os.Stderr, "error: system coordinates must be numeric\n")
					return 2
				}
				y = y*10 + int(val[vi]-'0')
			}
			if at(val, vi) != ',' {
				fmt.Fprintf(os.Stderr, "error: system coordinates must be separated by commas\n")
				return 2
			}
			vi++
			for ; vi < len(val); vi++ {
				if !isdigit(val[vi]) {
					fmt.Fprintf(os.Stderr, "error: system coordinates must be numeric\n")
					return 2
				}
				z = z*10 + int(val[vi]-'0')
			}
			chooseRandomly = FALSE
		} else {
			fmt.Fprintf(os.Stderr, "fh: %s: unknown option '%s'\n", cmdName, opt)
			return 2
		}
	}

	var star *star_data_t
	if chooseRandomly == FALSE {
		for i := 0; i < num_stars; i++ {
			star2 := star_base[i]
			if star2.x == x && star2.y == y && star2.z == z {
				star = star2
				break
			}
		}
		if star == nil {
			fmt.Fprintf(os.Stderr, "error: could not find non-home system at %d %d %d\n", x, y, z)
			return 2
		} else if star.num_planets < 3 {
			fmt.Fprintf(os.Stderr, "error: system at %d %d %d has only %d planets!\n", x, y, z, star.num_planets)
		} else if hasHomeSystemNeighbor(star, radius) != FALSE {
			if force == FALSE {
				fmt.Fprintf(os.Stderr, "error: system at %d %d %d has home system neighbor within %d parsecs\n",
					x, y, z, radius)
				return 2
			}
			fmt.Printf(" warn: system at %d %d %d has home system neighbor within %d parsecs\n", x, y, z, radius)
		}
		fmt.Printf(" info: given  system %3d %3d %3d\n", star.x, star.y, star.z)
	} else {
		star = findHomeSystemCandidate(radius)
		if star == nil {
			fmt.Fprintf(os.Stderr, "error: no systems meet the criteria for home systems!\n")
			return 2
		}
		fmt.Printf(" info: random system %3d %3d %3d\n", star.x, star.y, star.z)
	}

	if changeSystemToHomeSystem(star) != 0 {
		fmt.Fprintf(os.Stderr, "error: updateHomeSystem: failed to update the system\n")
		return 2
	}

	// save the updated data
	save_star_data()
	fp, err := os.Create("stars.hs.txt")
	if err != nil {
		cfgPerror("changeSystemToHomeSystem:", err)
		os.Exit(2)
	}
	starDataAsSExpr(star_base, num_stars, fp)
	fp.Close()

	save_planet_data()
	fp, err = os.Create("planets.hs.txt")
	if err != nil {
		cfgPerror("changeSystemToHomeSystem:", err)
		os.Exit(2)
	}
	planetDataAsSExpr(planet_base, num_planets, fp)
	fp.Close()

	return 0
}

func updatePlanet(args []string) int {
	cmdName := args[0]
	_ = cmdName
	return 0
}

func updateShip(args []string) int {
	var sp *species_data_t
	spno := 0
	spidx := -1
	var ship *ship_data_t

	fmt.Printf("fh: update: loading   species  data...\n")
	get_species_data()

	for i := 1; i < len(args); i++ {
		fmt.Fprintf(os.Stderr, "fh: update ship: argc %2d argv '%s'\n", i, args[i])
		opt, val, hasVal := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "fh: usage: update ship spNo shipName [field value]\n")
			fmt.Fprintf(os.Stderr, "    where: spNo is a valid species number (no leading zeroes)\n")
			fmt.Fprintf(os.Stderr, "    where: shipName is a valid ship name (case sensitive, no type)\n")
			fmt.Fprintf(os.Stderr, "    where: field is age\n")
			fmt.Fprintf(os.Stderr, "      and: value is an integer between 1 and 50\n")
			fmt.Fprintf(os.Stderr, "    where: opt is --class=ship_class\n")
			fmt.Fprintf(os.Stderr, "      and: ship_class is an valid ship class (PB, DD, etc)\n")
			fmt.Fprintf(os.Stderr, "    where: opt is --ftl\n")
			fmt.Fprintf(os.Stderr, "    where: opt is --name=new_name\n")
			fmt.Fprintf(os.Stderr, "    where: opt is --sub-light\n")
			fmt.Fprintf(os.Stderr, "    where: opt is --tonnage value\n")
			fmt.Fprintf(os.Stderr, "      and: value is an valid tonnage value\n")
			return 2
		} else if spno == 0 {
			spno = cfgAtoi(opt)
			spidx = spno - 1
			if !(1 <= spno && spno <= galaxy.num_species) {
				fmt.Fprintf(os.Stderr, "error: invalid species number '%s'\n", opt)
				return 2
			} else if data_in_memory[spidx] == FALSE {
				fmt.Fprintf(os.Stderr, "error: species %d is not loaded\n", spno)
				return 2
			}
			fmt.Printf("fh: update ship: species number is %3d\n", spno)
			sp = &spec_data[spidx]
			ship_base = ship_data[spidx]
		} else if ship == nil {
			for j := 0; j < sp.num_ships; j++ {
				if ship_base[j].name == opt {
					ship = ship_base[j]
					break
				}
			}
			if ship == nil {
				fmt.Fprintf(os.Stderr, "error: species %d has no ship named '%s'\n", spno, opt)
				return 2
			}
		} else if opt == "--age" && hasVal {
			value := cfgAtoi(val)
			if value < 0 || value > 50 {
				fmt.Fprintf(os.Stderr, "error: invalid age value '%s'\n", val)
				return 2
			}
			fmt.Printf("fh: update ship: species %d name '%s' age from %4d to %4d\n", spno, ship.name, ship.age, value)
			ship.age = value
			data_modified[spidx] = TRUE
		} else if opt == "--class" && hasVal {
			if val == "BC" {
				ship.class = BC
			} else if val == "BM" {
				ship.class = BM
			} else if val == "BR" {
				ship.class = BR
			} else if val == "BS" {
				ship.class = BS
			} else if val == "BW" {
				ship.class = BW
			} else if val == "CA" {
				ship.class = CA
			} else if val == "CC" {
				ship.class = CC
			} else if val == "CL" {
				ship.class = CL
			} else if val == "CT" {
				ship.class = CT
			} else if val == "DD" {
				ship.class = DD
			} else if val == "DN" {
				ship.class = DN
			} else if val == "ES" {
				ship.class = ES
			} else if val == "FF" {
				ship.class = FF
			} else if val == "PB" {
				ship.class = PB
			} else if val == "SD" {
				ship.class = SD
			} else if val == "TR" {
				ship.class = TR
			} else {
				fmt.Fprintf(os.Stderr, "error: unknown ship class '%s'\n", val)
				return 2
			}
			fmt.Printf("fh: update ship: species %d name '%s' class to %s\n", spno, ship.name, val)
			ship.tonnage = ship_tonnage[ship.class]
			ship.ship_type = FTL
			data_modified[spidx] = TRUE
		} else if opt == "--ftl" && hasVal {
			if val == "yes" {
				fmt.Printf("fh: update ship: species %d name '%s' to ftl\n", spno, ship.name)
				ship.ship_type = FTL
				data_modified[spidx] = TRUE
			} else if val == "no" {
				fmt.Printf("fh: update ship: species %d name '%s' to sub-light\n", spno, ship.name)
				ship.ship_type = SUB_LIGHT
				data_modified[spidx] = TRUE
			} else {
				fmt.Fprintf(os.Stderr, "error: ftl value must be yes or no\n")
				return 2
			}
		} else if opt == "--item" && hasVal {
			newQuantity := 0
			if val == "FS" {
				ship.item_quantity[FS]++
				newQuantity = ship.item_quantity[FS]
			} else if val == "GU1" {
				ship.item_quantity[GU1]++
				newQuantity = ship.item_quantity[GU1]
			} else if val == "GU2" {
				ship.item_quantity[GU2]++
				newQuantity = ship.item_quantity[GU2]
			} else if val == "GU3" {
				ship.item_quantity[GU3]++
				newQuantity = ship.item_quantity[GU3]
			} else if val == "GU4" {
				ship.item_quantity[GU4]++
				newQuantity = ship.item_quantity[GU4]
			} else if val == "GU5" {
				ship.item_quantity[GU5]++
				newQuantity = ship.item_quantity[GU5]
			} else if val == "GU6" {
				ship.item_quantity[GU6]++
				newQuantity = ship.item_quantity[GU6]
			} else if val == "GU7" {
				ship.item_quantity[GU7]++
				newQuantity = ship.item_quantity[GU7]
			} else if val == "GU8" {
				ship.item_quantity[GU8]++
				newQuantity = ship.item_quantity[GU8]
			} else if val == "GU9" {
				ship.item_quantity[GU9]++
				newQuantity = ship.item_quantity[GU9]
			} else if val == "SG1" {
				ship.item_quantity[SG1]++
				newQuantity = ship.item_quantity[SG1]
			} else if val == "SG2" {
				ship.item_quantity[SG2]++
				newQuantity = ship.item_quantity[SG2]
			} else if val == "SG3" {
				ship.item_quantity[SG3]++
				newQuantity = ship.item_quantity[SG3]
			} else if val == "SG4" {
				ship.item_quantity[SG4]++
				newQuantity = ship.item_quantity[SG4]
			} else if val == "SG5" {
				ship.item_quantity[SG5]++
				newQuantity = ship.item_quantity[SG5]
			} else if val == "SG6" {
				ship.item_quantity[SG6]++
				newQuantity = ship.item_quantity[SG6]
			} else if val == "SG7" {
				ship.item_quantity[SG7]++
				newQuantity = ship.item_quantity[SG7]
			} else if val == "SG8" {
				ship.item_quantity[SG8]++
				newQuantity = ship.item_quantity[SG8]
			} else if val == "SG9" {
				ship.item_quantity[SG9]++
				newQuantity = ship.item_quantity[SG9]
			} else if val == "GW" {
				ship.item_quantity[GW]++
				newQuantity = ship.item_quantity[GW]
			} else {
				fmt.Fprintf(os.Stderr, "error: unknown inventory item '%s'\n", val)
				return 2
			}
			fmt.Printf("fh: update ship: species %d name '%s' inventory %s to %d\n", spno, ship.name, val, newQuantity)
			data_modified[spidx] = TRUE
		} else if opt == "--name" && hasVal {
			if len(val) < 5 {
				fmt.Fprintf(os.Stderr, "error: ship name is too short\n")
				return 2
			} else if len(val) > 30 {
				fmt.Fprintf(os.Stderr, "error: ship name is too long\n")
				return 2
			}
			fmt.Printf("fh: update ship: species %d name '%s' to '%s'\n", spno, ship.name, val)
			ship.name = val
			data_modified[spidx] = TRUE
		} else if opt == "--reset-inventory" && !hasVal {
			for item := 0; item < MAX_ITEMS; item++ {
				ship.item_quantity[item] = 0
			}
		} else if opt == "--tonnage" && hasVal {
			if ship.class != TR {
				fmt.Fprintf(os.Stderr, "error: tonnage is valid only for transports\n")
				return 2
			}
			value := cfgAtoi(val)
			if value < 1 {
				fmt.Fprintf(os.Stderr, "error: invalid tonnage value '%s'\n", val)
				return 2
			} else if value > 5*sp.tech_level[MA] {
				fmt.Fprintf(os.Stderr, "error: invalid tonnage value '%s' (exceeds MA)\n", val)
				return 2
			}
			fmt.Printf("fh: update ship: species %d name '%s' tonnage from %d to %d\n",
				spno, ship.name, ship.tonnage, value)
			ship.tonnage = value
			data_modified[spidx] = TRUE
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	if sp == nil || data_modified[spidx] == FALSE {
		fmt.Printf("fh: update ship: no changes to save\n")
	} else {
		fmt.Printf("fh: update: saving    species  data...\n")
		save_species_data()
	}
	return 0
}

func updateSpecies(args []string) int {
	var sp *species_data_t
	spno := 0
	spidx := -1

	fmt.Printf("fh: update: loading   species  data...\n")
	get_species_data()

	for i := 1; i < len(args); i++ {
		fmt.Fprintf(os.Stderr, "fh: update species: argc %2d argv '%s'\n", i, args[i])
		opt := args[i]
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "fh: usage: update species spNo [field value]\n")
			fmt.Fprintf(os.Stderr, "    where: spNo is a valid species number (no leading zeroes)\n")
			fmt.Fprintf(os.Stderr, "    where: field is govt-type\n")
			fmt.Fprintf(os.Stderr, "      and: value is between 1 and 31 characters\n")
			return 2
		} else if spno == 0 {
			spno = cfgAtoi(opt)
			spidx = spno - 1
			if !(1 <= spno && spno <= galaxy.num_species) {
				fmt.Fprintf(os.Stderr, "error: invalid species number '%s'\n", opt)
				return 2
			} else if data_in_memory[spidx] == FALSE {
				fmt.Fprintf(os.Stderr, "error: species %d is not loaded\n", spno)
				return 2
			}
			fmt.Printf("fh: update species: species number is %3d\n", spno)
			sp = &spec_data[spidx]
		} else if opt == "bi" || opt == "gv" || opt == "ls" ||
			opt == "ma" || opt == "mi" || opt == "ml" {
			tech := opt
			if i+1 == len(args) || len(args[i+1]) == 0 {
				fmt.Fprintf(os.Stderr, "error: missing tech level value\n")
				return 2
			}
			i++
			value := cfgAtoi(args[i])
			if value < 0 {
				fmt.Fprintf(os.Stderr, "error: invalid tech level value\n")
				return 2
			}
			var code int
			if tech == "bi" {
				code = BI
			} else if tech == "gv" {
				code = GV
			} else if tech == "ls" {
				code = LS
			} else if tech == "ma" {
				code = MA
			} else if tech == "mi" {
				code = MI
			} else {
				code = ML
			}
			fmt.Printf("fh: update species: %s from %4d to %4d\n", tech, sp.tech_level[code], value)
			sp.tech_level[code] = value
			data_modified[spidx] = TRUE
		} else if opt == "eu" {
			if i+1 == len(args) || len(args[i+1]) == 0 {
				fmt.Fprintf(os.Stderr, "error: missing economic units value\n")
				return 2
			}
			i++
			value := cfgAtoi(args[i])
			if value < 0 {
				fmt.Fprintf(os.Stderr, "error: invalid economic units value\n")
				return 2
			}
			fmt.Printf("fh: update species: eu from %4d to %4d\n", sp.econ_units, value)
			sp.econ_units = value
			data_modified[spidx] = TRUE
		} else if opt == "govt-type" {
			if i+1 == len(args) || len(args[i+1]) == 0 {
				fmt.Fprintf(os.Stderr, "error: missing government type value\n")
				return 2
			}
			i++
			value := args[i]
			if !(len(value) < 32) {
				fmt.Fprintf(os.Stderr, "error: invalid government type\n")
				return 2
			}
			fmt.Printf("fh: update species: govt-type from \"%s\" to \"%s\"\n", sp.govt_type, value)
			sp.govt_type = value
			data_modified[spidx] = TRUE
		} else if opt == "hp" {
			if i+1 == len(args) || len(args[i+1]) == 0 {
				fmt.Fprintf(os.Stderr, "error: missing hp economic base value\n")
				return 2
			}
			i++
			value := cfgAtoi(args[i])
			if value < 0 {
				fmt.Fprintf(os.Stderr, "error: invalid hp economic base value\n")
				return 2
			}
			fmt.Printf("fh: update species: hp from %4d to %4d\n", sp.hp_original_base, value)
			sp.hp_original_base = value
			data_modified[spidx] = TRUE
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}
	if sp == nil || data_modified[spidx] == FALSE {
		fmt.Printf("fh: update species: no changes to save\n")
	} else {
		fmt.Printf("fh: update: saving    species  data...\n")
		save_species_data()
	}
	return 0
}

func updateStar(args []string) int {
	cmdName := args[0]
	_ = cmdName
	return 0
}
