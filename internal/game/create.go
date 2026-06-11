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

// Port of create.c — the CLI entries for 'fh create' plus species
// creation. The speciesDataAsSExpr exporter (from speciesio.c) is ported
// at the bottom of this file because speciesio.go skipped it; the
// "species%03d.txt" snapshot the C saveSpeciesData writes is reproduced
// here in createSpeciesCommand immediately after save_species_data().

import (
	"fmt"
	"os"
)

// splitOptVal mirrors the C option parsing loop in every *Command
// function: the argument is split at the first '='; if there is no '='
// or nothing follows it, val is NULL (hasVal == false).
func splitOptVal(arg string) (opt string, val string, hasVal bool) {
	for i := 0; i < len(arg); i++ {
		if arg[i] == '=' {
			opt, val = arg[:i], arg[i+1:]
			if len(val) == 0 {
				return opt, "", false
			}
			return opt, val, true
		}
	}
	return arg, "", false
}

// strcasecmp reproduces the C library's byte-wise ASCII case-insensitive
// comparison.
func strcasecmp(s1, s2 string) int {
	for i := 0; ; i++ {
		var c1, c2 byte
		if i < len(s1) {
			c1 = s1[i]
		}
		if i < len(s2) {
			c2 = s2[i]
		}
		l1 := cfgToLower(c1)
		l2 := cfgToLower(c2)
		if l1 != l2 {
			return int(l1) - int(l2)
		}
		if c1 == 0 {
			return 0
		}
	}
}

func createCommand(args []string) int {
	cmdName := args[0]
	for i := 1; i < len(args); i++ {
		// fprintf(stderr, "fh: %s: argc %2d argv '%s'\n", cmdName, i, argv[i]);
		opt, _, _ := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "fh: usage: create (galaxy | home-system-templates | species)\n")
			return 2
		} else if opt == "galaxy" {
			return createGalaxyCommand(args[i:])
		} else if opt == "home-system-templates" {
			return createHomeSystemTemplatesCommand(args[i:])
		} else if opt == "orders" {
			return createOrdersCommand(args[i:])
		} else if opt == "species" {
			return createSpeciesCommand(args[i:])
		} else {
			fmt.Fprintf(os.Stderr, "fh: %s: unknown option '%s'\n", cmdName, opt)
			return 2
		}
	}
	return 0
}

func createGalaxyCommand(args []string) int {
	desiredNumSpecies := 0
	desiredNumStars := 0
	galacticRadius := 0
	lessCrowded := FALSE
	suggestValues := FALSE

	for i := 1; i < len(args); i++ {
		opt, val, hasVal := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr,
				"fh: usage: create galaxy --species=integer [--stars=integer] [--radius=integer] [--suggest-values]\n")
			return 2
		} else if opt == "--less-crowded" {
			lessCrowded = TRUE
		} else if opt == "--radius" && hasVal {
			galacticRadius = cfgAtoi(val)
			if galacticRadius < MIN_RADIUS || galacticRadius > MAX_RADIUS {
				fmt.Fprintf(os.Stderr, "error: radius must be between %d and %d parsecs.\n", MIN_RADIUS, MAX_RADIUS)
				return 2
			}
		} else if opt == "--species" && hasVal {
			desiredNumSpecies = cfgAtoi(val)
			if desiredNumSpecies < MIN_SPECIES || desiredNumSpecies > MAX_SPECIES {
				fmt.Fprintf(os.Stderr, "error: species must be between %d and %d.\n", MIN_SPECIES, MAX_SPECIES)
				return 2
			}
		} else if opt == "--stars" && hasVal {
			desiredNumStars = cfgAtoi(val)
			if desiredNumStars < MIN_STARS || desiredNumStars > MAX_STARS {
				fmt.Fprintf(os.Stderr, "error: stars must be between %d and %d.\n", MIN_STARS, MAX_STARS)
				return 2
			}
		} else if opt == "--suggest-values" {
			suggestValues = TRUE
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	if desiredNumSpecies == 0 {
		fmt.Fprintf(os.Stderr, "error: you must supply the desired number of species.\n")
		return 2
	}

	if desiredNumStars == 0 || suggestValues != FALSE {
		if lessCrowded == FALSE {
			desiredNumStars = (desiredNumSpecies * STANDARD_NUMBER_OF_STAR_SYSTEMS) / STANDARD_NUMBER_OF_SPECIES
		} else {
			// bump the number of stars by 50% to make it take longer to encounter other species.
			desiredNumStars =
				(3 * desiredNumSpecies * STANDARD_NUMBER_OF_STAR_SYSTEMS) / (2 * STANDARD_NUMBER_OF_SPECIES)
		}
		if desiredNumStars > MAX_STARS {
			fmt.Fprintf(os.Stderr, "error: calculation results in a number greater than %d stars.\n", MAX_STARS)
			return 2
		}
	}

	if galacticRadius == 0 || suggestValues != FALSE {
		minVolume := desiredNumStars *
			STANDARD_GALACTIC_RADIUS * STANDARD_GALACTIC_RADIUS * STANDARD_GALACTIC_RADIUS /
			STANDARD_NUMBER_OF_STAR_SYSTEMS
		for galacticRadius = MIN_RADIUS; galacticRadius*galacticRadius*galacticRadius < minVolume; {
			galacticRadius++
		}
		if galacticRadius > MAX_RADIUS {
			fmt.Fprintf(os.Stderr, "error: calculation results in a radius greater than %d parsecs.\n", MAX_RADIUS)
			return 2
		}
	}

	if suggestValues != FALSE {
		crowded := ""
		if lessCrowded != FALSE {
			crowded = "less crowded "
		}
		fmt.Printf(" info: for %d species, a %sgalaxy needs about %d star systems.\n",
			desiredNumSpecies, crowded, desiredNumStars)
		fmt.Printf(" info: for %d stars, the galaxy should have a radius of about %d parsecs.\n",
			desiredNumStars, galacticRadius)
		return 0
	}

	return createGalaxy(galacticRadius, desiredNumStars, desiredNumSpecies)
}

func createHomeSystemTemplatesCommand(args []string) int {
	for i := 1; i < len(args); i++ {
		opt, _, _ := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr,
				"fh: usage: create home-system-templates...\n")
			return 2
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	return createHomeSystemTemplates()
}

func createOrdersCommand(args []string) int {
	advanced := FALSE
	reminder := FALSE

	for i := 1; i < len(args); i++ {
		opt, _, hasVal := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "fh: usage: create orders opt\n")
			fmt.Fprintf(os.Stderr, "           creates orders files for species that have not yet submitted theirs\n")
			fmt.Fprintf(os.Stderr, "      opt: --add-reminder    insert a reminder into the orders\n")
			fmt.Fprintf(os.Stderr, "         : --auto            generate a better set of orders\n")
			fmt.Fprintf(os.Stderr, "         : --default         generate the minimal set of orders   [default]\n")
			return 2
		} else if opt == "--add-reminder" && !hasVal {
			reminder = TRUE
		} else if opt == "--auto" && !hasVal {
			advanced = TRUE
		} else if opt == "--default" && !hasVal {
			advanced = FALSE
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	return createOrders(advanced, reminder)
}

func createSpeciesCommand(args []string) int {
	// fprintf(stderr, "%s:%s:%d\n", __FILE_NAME__, __FUNCTION__, __LINE__);
	configFile := ""
	radius := 10 // default minimum distance between home systems

	for i := 1; i < len(args); i++ {
		opt, val, hasVal := splitOptVal(args[i])
		if opt == "--help" || opt == "-h" || opt == "-?" {
			fmt.Fprintf(os.Stderr, "fh: usage: create species...\n")
			return 2
		} else if opt == "--config" && hasVal {
			configFile = val
		} else if opt == "--radius" && hasVal {
			radius = cfgAtoi(val)
			if radius < 1 {
				fmt.Fprintf(os.Stderr, "error: invalid radius '%s'\n", val)
				return 2
			}
		} else {
			fmt.Fprintf(os.Stderr, "error: unknown option '%s'\n", opt)
			return 2
		}
	}

	if configFile == "" {
		fmt.Fprintf(os.Stderr, "error: you must supply a configuration file name\n")
		return 2
	}
	cfg := cfgSpeciesFromFile(configFile)
	if cfg == nil || cfg[0] == nil {
		fmt.Fprintf(os.Stderr, "error: %s: no species in config file\n", configFile)
		return 2
	}

	get_galaxy_data()
	get_star_data()
	get_planet_data()
	get_species_data()

	for spidx := 0; spidx < galaxy.num_species; spidx++ {
		if data_in_memory[spidx] == FALSE {
			fmt.Fprintf(os.Stderr, "error: internal error: createSpeciesCommand: sp %d data not in memory\n", spidx+1)
			os.Exit(2)
		}
	}

	if radius > galaxy.radius/2 {
		fmt.Fprintf(os.Stderr, "error: radius must be between 1 and %d for this galaxy\n", galaxy.radius/2)
		return 2
	}

	for n := 0; cfg[n] != nil; n++ {
		c := cfg[n]
		species_index = galaxy.num_species
		species_number = species_index + 1

		if species_number > galaxy.d_num_species {
			fmt.Fprintf(os.Stderr, "error: galaxy limit is %d species\n", galaxy.d_num_species)
			return 2
		} else if data_in_memory[species_index] != FALSE {
			fmt.Fprintf(os.Stderr, "error: createSpeciesCommand: internal error: data_in_memory[%d] is TRUE\n", species_index)
			os.Exit(2)
		} else if data_modified[species_index] != FALSE {
			fmt.Fprintf(os.Stderr, "error: createSpeciesCommand: internal error: data_modified[%d] is TRUE\n", species_index)
			os.Exit(2)
		}

		if c.name == "" {
			fmt.Fprintf(os.Stderr, "error: section missing species name\n")
			return 2
		} else if len(c.name) < 5 {
			fmt.Fprintf(os.Stderr, "error: species name '%s' must be at least 5 characters long.\n", c.name)
			return 2
		} else if len(c.name) > 31 {
			fmt.Fprintf(os.Stderr, "error: species name '%s' must be less than 32 characters long.\n", c.name)
			return 2
		} else {
			for spidx := 0; spidx < galaxy.num_species; spidx++ {
				if strcasecmp(c.name, spec_data[spidx].name) == 0 {
					fmt.Fprintf(os.Stderr, "error: species name '%s' is not unique\n", c.name)
					return 2
				}
			}
		}
		if c.govtname == "" {
			fmt.Fprintf(os.Stderr, "error: section missing species govtname\n")
			return 2
		} else if len(c.govtname) < 5 {
			fmt.Fprintf(os.Stderr, "error: species govtname '%s' must be at least 5 characters long.\n", c.govtname)
			return 2
		} else if len(c.govtname) > 31 {
			fmt.Fprintf(os.Stderr, "error: species govtname '%s' must be less than 32 characters long.\n", c.govtname)
			return 2
		}
		if c.govttype == "" {
			fmt.Fprintf(os.Stderr, "error: section missing species govttype\n")
			return 2
		} else if len(c.govttype) < 5 {
			fmt.Fprintf(os.Stderr, "error: species govttype '%s' must be at least 5 characters long.\n", c.govttype)
			return 2
		} else if len(c.govttype) > 31 {
			fmt.Fprintf(os.Stderr, "error: species govttype '%s' must be less than 32 characters long.\n", c.govttype)
			return 2
		}
		if c.homeworld == "" {
			fmt.Fprintf(os.Stderr, "error: section missing species homeworld\n")
			return 2
		} else if len(c.homeworld) < 5 {
			fmt.Fprintf(os.Stderr, "error: species homeworld '%s' must be at least 5 characters long.\n", c.homeworld)
			return 2
		} else if len(c.homeworld) > 31 {
			fmt.Fprintf(os.Stderr, "error: species homeworld '%s' must be less than 32 characters long.\n", c.homeworld)
			return 2
		}
		if c.bi+c.gv+c.ls+c.ml > 15 {
			fmt.Fprintf(os.Stderr, "error: species tech levels must sum to less than 16.\n")
			return 2
		}

		// clear out the species data, just in case
		spec_data[species_index] = species_data_t{}
		sp := &spec_data[species_index]

		sp.id = species_number
		sp.index = species_index
		sp.name = c.name
		sp.govt_name = c.govtname
		sp.govt_type = c.govttype
		sp.tech_level[MA] = 10
		if c.experimental.tech_ma > sp.tech_level[MA] {
			sp.tech_level[MA] = c.experimental.tech_ma
		}
		sp.tech_level[MI] = 10
		if c.experimental.tech_mi > sp.tech_level[MI] {
			sp.tech_level[MI] = c.experimental.tech_mi
		}
		sp.tech_level[ML] = c.ml
		if c.experimental.tech_ml > sp.tech_level[ML] {
			sp.tech_level[ML] = c.experimental.tech_ml
		}
		sp.tech_level[GV] = c.gv
		if c.experimental.tech_gv > sp.tech_level[GV] {
			sp.tech_level[GV] = c.experimental.tech_gv
		}
		sp.tech_level[LS] = c.ls
		if c.experimental.tech_ls > sp.tech_level[LS] {
			sp.tech_level[LS] = c.experimental.tech_ls
		}
		sp.tech_level[BI] = c.bi
		if c.experimental.tech_bi > sp.tech_level[BI] {
			sp.tech_level[BI] = c.experimental.tech_bi
		}
		// initialize other tech stuff
		for t := MI; t <= BI; t++ {
			sp.tech_knowledge[t] = sp.tech_level[t]
			sp.init_tech_level[t] = sp.tech_level[t]
			sp.tech_eps[t] = 0
		}
		sp.econ_units = 0
		if c.experimental.econ_units > sp.econ_units {
			sp.econ_units = c.experimental.econ_units
		}
		// make O2 a required gas for the species
		sp.required_gas = O2

		// find candidate home systems
		var candidateSystems [MAX_SPECIES + 1]*star_data_t
		numCandidates := 0 // index into candidateSystems
		for s := 0; s < num_stars; s++ {
			star := star_base[s]
			for p := 0; p < star.num_planets; p++ {
				planet := planet_base[star.planet_index+p]
				if planet.special != HOME_PLANET {
					continue
				}
				var claimedBy *species_data_t
				for spidx := 0; spidx < galaxy.num_species && claimedBy == nil; spidx++ {
					sp := &spec_data[spidx]
					if sp.x == star.x && sp.y == star.y && sp.z == star.z {
						// would be cruel to have two species share a system
						claimedBy = sp
					}
				}
				if claimedBy == nil {
					// printf("%s:%d: unclaimed system: %d,%d,%d\n", __FUNCTION__, __LINE__, star->x, star->y, star->z);
					candidateSystems[numCandidates] = star
					numCandidates++
				}
			}
		}
		if numCandidates == 0 {
			// no candidates, so create one
			candidateSystems[0] = findHomeSystemCandidate(radius)
			if candidateSystems[0] == nil {
				fmt.Fprintf(os.Stderr, "error: createSpeciesCommand: no systems meet the criteria for radius of %d!\n",
					radius)
				return 2
			}
			if changeSystemToHomeSystem(candidateSystems[0]) != 0 {
				fmt.Fprintf(os.Stderr, "error: createSpeciesCommand: failed to change system to home system\n")
				return 2
			}
			numCandidates++
		}
		// randomly choose a home system from the list of candidates
		homeSystem := candidateSystems[rnd(numCandidates)-1]
		// fetch the home planet in the home system
		var home_planet *planet_data_t
		for pn := 0; pn < homeSystem.num_planets; pn++ {
			p := planet_base[homeSystem.planet_index+pn]
			if p.special == HOME_PLANET {
				home_planet = p
				break
			}
		}
		sp.x = homeSystem.x
		sp.y = homeSystem.y
		sp.z = homeSystem.z
		sp.pn = home_planet.orbit

		if c.experimental.make_bridges != 0 {
			fmt.Printf(" warn: engaging experimental hook 'make-bridges'\n")
			fmt.Printf(" hook: values before running hook\n")
			fmt.Printf("       system %d: worm_here %s\n", homeSystem.id, trueFalse(homeSystem.worm_here))
			// set all other species wormholes to exit in this system
			var alienHomeSystem *star_data_t
			for alienIndex := 0; alienIndex < MAX_SPECIES; alienIndex++ {
				if data_in_memory[alienIndex] == FALSE {
					continue
				}
				alien := &spec_data[alienIndex]
				alienNamedPlanets := namp_data[alienIndex]
				alienHomeSystem = alienNamedPlanets[0].star
				fmt.Printf("       hsp %3d: creating wormhole from %d,%d,%d\n", alien.id, alienHomeSystem.x,
					alienHomeSystem.y, alienHomeSystem.z)
				alienHomeSystem.worm_here = TRUE
				alienHomeSystem.worm_x = homeSystem.x
				alienHomeSystem.worm_y = homeSystem.y
				alienHomeSystem.worm_z = homeSystem.z
			}
			if alienHomeSystem == nil {
				fmt.Printf("       no alien species found, not creating wormhole\n")
			} else {
				fmt.Printf("       creating wormhole to %d,%d,%d\n", alienHomeSystem.x, alienHomeSystem.y,
					alienHomeSystem.z)
				homeSystem.worm_here = TRUE
				homeSystem.worm_x = alienHomeSystem.x
				homeSystem.worm_y = alienHomeSystem.y
				homeSystem.worm_z = alienHomeSystem.z
			}
			fmt.Printf(" hook: values after running hook\n")
			fmt.Printf("       system %d: worm_here %s\n", homeSystem.id, trueFalse(homeSystem.worm_here))
		}

		home_nampla := &nampla_data_t{}
		namp_data[species_index] = []*nampla_data_t{home_nampla}
		home_nampla.name = c.homeworld
		home_nampla.star = homeSystem
		home_nampla.planet = home_planet
		home_nampla.x = homeSystem.x
		home_nampla.y = homeSystem.y
		home_nampla.z = homeSystem.z
		home_nampla.pn = home_planet.orbit
		home_nampla.planet_index = home_planet.index

		// verify that planet has oxygen and initialize the good gases
		var good_gas [14]int
		for i := 0; i < 14; i++ {
			good_gas[i] = FALSE
		}
		foundOxygen := FALSE
		num_neutral := 0
		for i := 0; i < 4; i++ {
			if home_planet.gas[i] == O2 {
				foundOxygen = TRUE
				sp.required_gas_min = home_planet.gas_percent[i] / 2
				if sp.required_gas_min < 1 {
					sp.required_gas_min = 1
				}
				sp.required_gas_max = 2 * home_planet.gas_percent[i]
				if sp.required_gas_max < 20 {
					sp.required_gas_max += 20
				} else if sp.required_gas_max > 100 {
					sp.required_gas_max = 100
				}
			}
			if home_planet.gas[i] > 0 {
				// all home planet gases are either required or neutral
				good_gas[home_planet.gas[i]] = TRUE
				num_neutral++
			}
		}
		if foundOxygen == FALSE {
			fmt.Fprintf(os.Stderr, "error: createSpeciesCommand: internal error: planet id %4d does not have %s(%d)!\n",
				home_planet.id, gas_string[O2], O2)
			os.Exit(2)
		}

		// Helium must always be neutral since it is a noble gas
		if good_gas[HE] == FALSE {
			good_gas[HE] = TRUE
			num_neutral++
		}
		// this game is biased towards oxygen breathers, so make H2O neutral
		if good_gas[H2O] == FALSE {
			good_gas[H2O] = TRUE
			num_neutral++
		}
		// initialize neutral gases for the species.
		// start with the good_gas array and add neutral gases until there are exactly seven of them.
		// one of the seven gases will be the required gas (currently hard-coded to Oxygen).
		for roll := rnd(13); num_neutral < 7; roll = rnd(13) {
			if good_gas[roll] == FALSE {
				good_gas[roll] = TRUE
				num_neutral++
			}
		}
		// add the list of neutral gases to the species data
		g := 0                    // index into neutral gases array
		for i := 1; i < 14; i++ { // start at one, ugh, first slot is ignored
			if good_gas[i] != FALSE && i != O2 {
				sp.neutral_gas[g] = i
				g++
			}
		}
		// same for poison gases
		g = 0
		for i := 1; i < 14; i++ { // start at one, ugh, first slot is ignored
			if good_gas[i] == FALSE {
				sp.poison_gas[g] = i
				g++
			}
		}

		/* Do mining and manufacturing bases of home planet.
		 * Initial mining and production capacity will be 25 times sum of MI and MA plus a small random amount.
		 * Mining and manufacturing base will be  reverse-calculated from the capacity. */
		base := sp.tech_level[MI] + sp.tech_level[MA]
		base = (25 * base) + rnd(base) + rnd(base) + rnd(base)
		home_nampla.mi_base = home_planet.mining_difficulty * base / (10 * sp.tech_level[MI])
		if c.experimental.mi_base > home_nampla.mi_base {
			home_nampla.mi_base = c.experimental.mi_base
		}
		home_nampla.ma_base = 10 * base / sp.tech_level[MA]
		if c.experimental.ma_base > home_nampla.ma_base {
			home_nampla.ma_base = c.experimental.ma_base
		}

		// fill out the rest
		sp.num_namplas = 1 // just the home planet for now ("nampla" means "named planet")
		home_nampla.status = HOME_PLANET | POPULATED
		home_nampla.pop_units = HP_AVAILABLE_POP
		home_nampla.shipyards = 1
		if c.experimental.ship_yards > home_nampla.shipyards {
			home_nampla.shipyards = c.experimental.ship_yards
		}
		// everything else was initialized to zero in the earlier call to 'delete_nampla'

		/* Print summary. */
		fmt.Printf("\n  Summary for species #%d:\n", species_number)

		fmt.Printf("\tName of species: %s\n", sp.name)
		fmt.Printf("\tName of home planet: %s\n", home_nampla.name)
		fmt.Printf("\t\tCoordinates: %d %d %d #%d\n", sp.x, sp.y, sp.z, sp.pn)
		fmt.Printf("\tName of government: %s\n", sp.govt_name)
		fmt.Printf("\tType of government: %s\n\n", sp.govt_type)

		fmt.Printf("\tTech levels: ")
		for i := 0; i < 6; i++ {
			fmt.Printf("%s = %d", tech_name[i], sp.tech_level[i])
			if i == 2 {
				fmt.Printf("\n\t             ")
			} else if i < 5 {
				fmt.Printf(",  ")
			}
		}

		fmt.Printf("\n\n\tFor this species, the required gas is %s (%d%%-%d%%).\n",
			gas_string[sp.required_gas], sp.required_gas_min, sp.required_gas_max)

		fmt.Printf("\tGases neutral to species:")
		for i := 0; i < 6; i++ {
			fmt.Printf(" %s ", gas_string[sp.neutral_gas[i]])
		}

		fmt.Printf("\n\tGases poisonous to species:")
		for i := 0; i < 6; i++ {
			fmt.Printf(" %s ", gas_string[sp.poison_gas[i]])
		}

		fmt.Printf("\n\n\tInitial mining base = %d.%d. Initial manufacturing base = %d.%d.\n",
			home_nampla.mi_base/10, home_nampla.mi_base%10,
			home_nampla.ma_base/10, home_nampla.ma_base%10)
		fmt.Printf("\tIn the first turn, %d raw material units will be produced,\n",
			(10*sp.tech_level[MI]*home_nampla.mi_base)/home_planet.mining_difficulty)
		fmt.Printf("\tand the total production capacity will be %d.\n\n",
			(sp.tech_level[MA]*home_nampla.ma_base)/10)

		// update galaxy
		galaxy.num_species++

		// set visited_by bit in star data
		species_array_index := (species_number - 1) / 32
		species_bit_number := (species_number - 1) % 32
		species_bit_mask := uint32(1) << uint(species_bit_number)
		homeSystem.visited_by[species_array_index] |= species_bit_mask

		data_in_memory[species_index] = TRUE
		data_modified[species_index] = TRUE

		/* Create log file for first turn. Write home star system data to it. */
		filename := fmt.Sprintf("sp%02d.log", species_number)
		var err error
		log_file, err = os.Create(filename)
		if err != nil {
			cfgPerror("createSpeciesCommand:", err)
			fmt.Fprintf(os.Stderr, "error: cannot open '%s' for writing!\n\n", filename)
			os.Exit(2)
		}

		fmt.Fprintf(log_file, "\nScan of home star system for SP %s:\n\n", sp.name)
		species = sp                           // species is required by the scan() function
		nampla_base = namp_data[species_index] // nampla_base is required by the scan() function
		scan(home_nampla.x, home_nampla.y, home_nampla.z, TRUE)

		log_file.Close()
	}

	// save the updated data
	save_galaxy_data()
	fp, err := os.Create("galaxy.hs.txt")
	if err != nil {
		cfgPerror("changeSystemToHomeSystem:", err)
		os.Exit(2)
	}
	galaxyDataAsSexpr(fp)
	fp.Close()

	save_star_data()
	fp, err = os.Create("stars.hs.txt")
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

	// save_species_data writes each species' sp%02d.dat record and the
	// "species%03d.txt" s-expression snapshot (via saveSpeciesData).
	save_species_data()

	return 0
}

// trueFalse mirrors the C idiom `flag ? "true" : "false"`.
func trueFalse(flag int) string {
	if flag != 0 {
		return "true"
	}
	return "false"
}

// speciesDataAsSExpr writes the current species data to a text file as an
// s-expression. Ported from speciesio.c because speciesio.go skipped it.
func speciesDataAsSExpr(sp *species_data_t, fp *os.File) {
	fmt.Fprintf(fp, "(species (id %3d) (name '%s') (auto %s)", sp.id, sp.name, trueFalse(sp.auto_orders))
	fmt.Fprintf(fp, "\n         (government (name '%s') (type '%s'))", sp.govt_name, sp.govt_type)
	fmt.Fprintf(fp, "\n         (homeworld (x %3d) (y %3d) (z %3d) (orbit %d) (hp_base %d))",
		sp.x, sp.y, sp.z, sp.pn, sp.hp_original_base)
	fmt.Fprintf(fp, "\n         (atmosphere")
	fmt.Fprintf(fp, "\n           (required (gas %2d) (min %3d) (max %3d))",
		sp.required_gas, sp.required_gas_min, sp.required_gas_max)
	fmt.Fprintf(fp, "\n           (neutral %2d %2d %2d %2d %2d %2d)",
		sp.neutral_gas[0], sp.neutral_gas[1], sp.neutral_gas[2],
		sp.neutral_gas[3], sp.neutral_gas[4], sp.neutral_gas[5])
	fmt.Fprintf(fp, "\n           (poison  %2d %2d %2d %2d %2d %2d)",
		sp.poison_gas[0], sp.poison_gas[1], sp.poison_gas[2],
		sp.poison_gas[3], sp.poison_gas[4], sp.poison_gas[5])
	fmt.Fprintf(fp, ")")
	fmt.Fprintf(fp, "\n         (technology")
	for i := 0; i < 6; i++ {
		fmt.Fprintf(fp, "\n           (tech (code '%s') (level %3d) (knowledge %3d) (init %2d) (xp %5d))",
			tech_level_names[i], sp.tech_level[i], sp.tech_knowledge[i], sp.init_tech_level[i], sp.tech_eps[i])
	}
	fmt.Fprintf(fp, ")")
	fmt.Fprintf(fp, "\n         (fleet (num_ships %5d) (maintenance (cost %9d) (percent %6d)))",
		sp.num_ships, sp.fleet_cost, sp.fleet_percent_cost)
	fmt.Fprintf(fp, "\n         (num_namplas %7d)", sp.num_namplas)
	fmt.Fprintf(fp, "\n         (banked_eu %9d)", sp.econ_units)
	fmt.Fprintf(fp, "\n         (contacts")
	for spidx := 0; spidx < galaxy.num_species; spidx++ {
		if (sp.contact[spidx/32] & (uint32(1) << uint(spidx%32))) != 0 {
			fmt.Fprintf(fp, " %3d", spidx+1)
		}
	}
	fmt.Fprintf(fp, ")")
	fmt.Fprintf(fp, "\n         (allies  ")
	for spidx := 0; spidx < galaxy.num_species; spidx++ {
		if (sp.ally[spidx/32] & (uint32(1) << uint(spidx%32))) != 0 {
			fmt.Fprintf(fp, " %3d", spidx+1)
		}
	}
	fmt.Fprintf(fp, ")")
	fmt.Fprintf(fp, "\n         (enemies ")
	for spidx := 0; spidx < galaxy.num_species; spidx++ {
		if (sp.enemy[spidx/32] & (uint32(1) << uint(spidx%32))) != 0 {
			fmt.Fprintf(fp, " %3d", spidx+1)
		}
	}
	fmt.Fprintf(fp, ")")
	fmt.Fprintf(fp, ")\n")
}
