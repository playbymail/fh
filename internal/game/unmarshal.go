package game

// Port of unmarshal.c — reads the JSON produced by marshal.c back into
// engine state (the import/convert path). It relies on the cJSON
// stand-in and the helpers.c subset in marshal.go.
//
// Deviation from C: the C writes the imported namplas and ships into
// the headroom (extra_namplas/extra_ships) that get_species_data
// allocated. The Go slices have no headroom, so unmarshalNamedPlanets,
// unmarshalShips and unmarshalSpeciesFile grow the slices with append
// and return them; import.go stores the returned slices back into
// namp_data/ship_data.

import (
	"fmt"
	"os"
)

func unmarshalAtmosphericGas(obj *cJSON, code *int, pct *int) {
	if obj == nil {
		*code = 0
		*pct = 0
	} else if !cJSON_IsObject(obj) {
		fmt.Fprintf(os.Stderr, "error: gas: element must be an object\n")
		os.Exit(2)
	}
	*code = jsonGetInt(obj, "code")
	*pct = jsonGetInt(obj, "percent")
}

func unmarshalAtmosphericGases(obj *cJSON, codes []int, pcts []int) {
	if obj == nil {
		for i := 0; i < 4; i++ {
			codes[i] = 0
			pcts[i] = 0
		}
	} else if !cJSON_IsArray(obj) {
		fmt.Fprintf(os.Stderr, "error: atmosphere: property must be an array\n")
		os.Exit(2)
	} else if cJSON_GetArraySize(obj) != 4 {
		fmt.Fprintf(os.Stderr, "error: atmosphere: want 4 elements, got %d\n", cJSON_GetArraySize(obj))
		os.Exit(2)
	}
	gas_index := 0
	for _, elem := range obj.children() {
		unmarshalAtmosphericGas(elem, &codes[gas_index], &pcts[gas_index])
		gas_index++
	}
}

func unmarshalConsUnits(obj *cJSON, num_needed *int, num_autos *int, num_install *int) {
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: cons_units: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(obj) {
		fmt.Fprintf(os.Stderr, "error: cons_units: property must be an object\n")
		os.Exit(2)
	}
	*num_needed = jsonGetInt(obj, "needed")
	*num_autos = jsonGetInt(obj, "auto")
	*num_install = jsonGetInt(obj, "install")
}

func unmarshalCoords(root *cJSON, x *int, y *int, z *int) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: coords: element must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: coords: element must be an object\n")
		os.Exit(2)
	}
	*x = jsonGetInt(root, "x")
	*y = jsonGetInt(root, "y")
	*z = jsonGetInt(root, "z")
}

func unmarshalCoordsWithOrbit(root *cJSON, x *int, y *int, z *int, orbit *int) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: coords_with_orbit: element must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: coords_with_orbit: element must be an object\n")
		os.Exit(2)
	}
	*x = jsonGetInt(root, "x")
	*y = jsonGetInt(root, "y")
	*z = jsonGetInt(root, "z")
	*orbit = jsonGetInt(root, "orbit")
}

func unmarshalGalaxy(root *cJSON, g *galaxy_data_t) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: galaxy: property 'galaxy' must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: galaxy: property 'galaxy' must be an object\n")
		os.Exit(2)
	}
	var prop *cJSON
	prop = cJSON_GetObjectItemCaseSensitive(root, "d_num_species")
	if prop != nil && cJSON_IsNumber(prop) {
		g.d_num_species = prop.valueint
	}
	prop = cJSON_GetObjectItemCaseSensitive(root, "num_species")
	if prop != nil && cJSON_IsNumber(prop) {
		g.num_species = prop.valueint
	}
	prop = cJSON_GetObjectItemCaseSensitive(root, "radius")
	if prop != nil && cJSON_IsNumber(prop) {
		g.radius = prop.valueint
	}
	prop = cJSON_GetObjectItemCaseSensitive(root, "turn_number")
	if prop != nil && cJSON_IsNumber(prop) {
		g.turn_number = prop.valueint
	}
}

func unmarshalGalaxyFile(root *cJSON, g *galaxy_data_t) {
	version := unmarshalVersion(cJSON_GetObjectItem(root, "version"))
	if version != 1 {
		fmt.Fprintf(os.Stderr, "error: galaxy: version: want 1, got %d\n", version)
		os.Exit(2)
	}
	unmarshalGalaxy(cJSON_GetObjectItem(root, "galaxy"), g)
}

func unmarshalGases(array *cJSON, gases []int) {
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: gases: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsArray(array) {
		fmt.Fprintf(os.Stderr, "error: gases: property must be an array\n")
		os.Exit(2)
	} else if cJSON_GetArraySize(array) != 6 {
		fmt.Fprintf(os.Stderr, "error: gases: array must contain exactly 6 elements\n")
		os.Exit(2)
	}
	i := 0
	for _, elem := range array.children() {
		if !cJSON_IsNumber(elem) {
			fmt.Fprintf(os.Stderr, "error: gases: elements must be integers\n")
			os.Exit(2)
		}
		gases[i] = elem.valueint
		i = i + 1
	}
}

func unmarshalGovernment(root *cJSON, sp *species_data_t) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: species: property 'government' must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: species: property 'government' must be an object\n")
		os.Exit(2)
	}
	sp.govt_name = jsonGetString(root, "name", 32)
	sp.govt_type = jsonGetString(root, "type", 32)
}

func unmarshalItems(array *cJSON, items []int) {
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: items: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsArray(array) {
		fmt.Fprintf(os.Stderr, "error: items: property must be an array\n")
		os.Exit(2)
	}
	for _, elem := range array.children() {
		if !cJSON_IsObject(elem) {
			fmt.Fprintf(os.Stderr, "error: items: all elements must be objects\n")
			os.Exit(2)
		}
		code := jsonGetInt(elem, "code")
		if 0 <= code && code < MAX_ITEMS {
			items[code] = jsonGetInt(elem, "qty")
		}
	}
}

func unmarshalMiningDifficulty(obj *cJSON, base *int, increase *int) {
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: mining_difficulty: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(obj) {
		fmt.Fprintf(os.Stderr, "error: mining_difficulty: element must be an object\n")
		os.Exit(2)
	}
	*base = jsonGetInt(obj, "base")
	*increase = jsonGetInt(obj, "increase")
}

func unmarshalNamedPlanet(root *cJSON, npd *nampla_data_t) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: named_planet: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: named_planet: property must be an object\n")
		os.Exit(2)
	}
	npd.name = jsonGetString(root, "name", 32)
	unmarshalCoordsWithOrbit(cJSON_GetObjectItem(root, "location"), &npd.x, &npd.y, &npd.z, &npd.pn)
	npd.status = jsonGetInt(root, "status")
	npd.hiding = jsonGetInt(root, "hiding")
	npd.hidden = jsonGetInt(root, "hidden")
	npd.siege_eff = jsonGetInt(root, "siege_eff")
	npd.shipyards = jsonGetInt(root, "shipyards")
	unmarshalConsUnits(cJSON_GetObjectItem(root, "ius"), &npd.IUs_needed, &npd.auto_IUs, &npd.IUs_to_install)
	unmarshalConsUnits(cJSON_GetObjectItem(root, "aus"), &npd.AUs_needed, &npd.auto_AUs, &npd.AUs_to_install)
	npd.mi_base = jsonGetInt(root, "mi_base")
	npd.ma_base = jsonGetInt(root, "ma_base")
	npd.pop_units = jsonGetInt(root, "pop_units")
	npd.use_on_ambush = jsonGetInt(root, "use_on_ambush")
	unmarshalItems(cJSON_GetObjectItem(root, "items"), npd.item_quantity[:])
	npd.message = jsonGetInt(root, "message")
	npd.special = jsonGetInt(root, "special")

	// find star and planet
	for i := 0; i < num_stars; i++ {
		sd := star_base[i]
		if sd.x == npd.x && sd.y == npd.y && sd.z == npd.z {
			npd.star = sd
			if !(0 < npd.pn && npd.pn <= sd.num_planets) {
				fmt.Fprintf(os.Stderr, "error: named_planet: orbit %d is not in range 1..%d\n", npd.pn, sd.num_planets)
				os.Exit(2)
			}
			npd.planet = planet_base[sd.planet_index+npd.pn-1]
			npd.planet_index = npd.planet.index
		}
	}
}

func unmarshalNamedPlanets(array *cJSON, sp *species_data_t, npa []*nampla_data_t) []*nampla_data_t {
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: named_planets: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsArray(array) {
		fmt.Fprintf(os.Stderr, "error: named_planets: property must be an array\n")
		os.Exit(2)
	}
	sp.num_namplas = 0
	for _, elem := range array.children() {
		var npd *nampla_data_t
		if sp.num_namplas < len(npa) {
			npd = npa[sp.num_namplas]
			*npd = nampla_data_t{}
		} else {
			// C writes into the extra_namplas headroom; Go grows the slice
			npd = &nampla_data_t{}
			npa = append(npa, npd)
		}
		npd.id = sp.num_namplas + 1
		unmarshalNamedPlanet(elem, npd)
		sp.num_namplas++
	}
	return npa
}

func unmarshalPlanet(root *cJSON, pd *planet_data_t) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: planets: planet element must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: planets: planet element must be an object\n")
		os.Exit(2)
	}
	pd.temperature_class = jsonGetInt(root, "temperature_class")
	pd.pressure_class = jsonGetInt(root, "pressure_class")
	pd.special = jsonGetInt(root, "special")
	unmarshalAtmosphericGases(cJSON_GetObjectItem(root, "atmosphere"), pd.gas[:], pd.gas_percent[:])
	pd.diameter = jsonGetInt(root, "diameter")
	pd.gravity = jsonGetInt(root, "gravity")
	unmarshalMiningDifficulty(cJSON_GetObjectItem(root, "mining_difficulty"), &pd.mining_difficulty, &pd.md_increase)
	pd.econ_efficiency = jsonGetInt(root, "econ_efficiency")
	pd.isValid = 0
	pd.message = jsonGetInt(root, "message")
}

func unmarshalPlanets(root *cJSON, sd *star_data_t, pa []*planet_data_t) int {
	if root == nil {
		return 0
	} else if !cJSON_IsArray(root) {
		fmt.Fprintf(os.Stderr, "error: planets: property must be an array\n")
		os.Exit(2)
	}
	orbit := 0
	for _, elem := range root.children() {
		pd := pa[orbit]
		pd.id = sd.planet_index + orbit + 1
		pd.index = sd.planet_index + orbit
		pd.star = sd
		pd.orbit = orbit + 1
		unmarshalPlanet(elem, pd)
		pd.isValid = TRUE
		orbit = orbit + 1
	}
	return orbit
}

func unmarshalRequireAtmosphericGas(root *cJSON, code *int, min_pct *int, max_pct *int) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: species: required_gas must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: species: required_gas must be an object\n")
		os.Exit(2)
	}
	*code = jsonGetInt(root, "code")
	*min_pct = jsonGetInt(root, "min_pct")
	*max_pct = jsonGetInt(root, "max_pct")
}

func unmarshalShip(root *cJSON, sd *ship_data_t) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: ship: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: ship: property must be an object\n")
		os.Exit(2)
	}
	sd.name = jsonGetString(root, "name", 32)
	unmarshalCoordsWithOrbit(cJSON_GetObjectItem(root, "location"), &sd.x, &sd.y, &sd.z, &sd.pn)
	sd.status = jsonGetInt(root, "status")
	sd.ship_type = jsonGetInt(root, "type")
	unmarshalCoords(cJSON_GetObjectItem(root, "dest"), &sd.dest_x, &sd.dest_y, &sd.dest_z)
	sd.just_jumped = jsonGetInt(root, "just_jumped")
	sd.arrived_via_wormhole = jsonGetBool(root, "arrived_via_wormhole")
	sd.class = jsonGetInt(root, "class")
	sd.tonnage = jsonGetInt(root, "tonnage")
	unmarshalItems(cJSON_GetObjectItem(root, "cargo"), sd.item_quantity[:])
	sd.age = jsonGetInt(root, "age")
	sd.remaining_cost = jsonGetInt(root, "remaining_cost")
	sd.loading_point = jsonGetInt(root, "loading_point")
	sd.unloading_point = jsonGetInt(root, "unloading_point")
	sd.special = jsonGetInt(root, "special")
}

func unmarshalShips(array *cJSON, sp *species_data_t, sa []*ship_data_t) []*ship_data_t {
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: ships: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsArray(array) {
		fmt.Fprintf(os.Stderr, "error: ships: property must be an array\n")
		os.Exit(2)
	}
	sp.num_ships = 0
	for _, elem := range array.children() {
		var sd *ship_data_t
		if sp.num_ships < len(sa) {
			sd = sa[sp.num_ships]
			*sd = ship_data_t{}
		} else {
			// C writes into the extra_ships headroom; Go grows the slice
			sd = &ship_data_t{}
			sa = append(sa, sd)
		}
		sd.id = sp.num_ships + 1
		unmarshalShip(elem, sd)
		sp.num_ships++
	}
	return sa
}

func unmarshalSpecies(root *cJSON, sp *species_data_t) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: species: property 'species' must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "species: property 'species' must be an object\n")
		os.Exit(2)
	}
	*sp = species_data_t{}
	sp.id = jsonGetInt(root, "id")
	sp.name = jsonGetString(root, "name", 32)
	unmarshalGovernment(cJSON_GetObjectItem(root, "government"), sp)
	unmarshalCoordsWithOrbit(cJSON_GetObjectItem(root, "home_world"), &sp.x, &sp.y, &sp.z, &sp.pn)
	unmarshalSpeciesAtmosphere(cJSON_GetObjectItem(root, "atmosphere"), sp)
	sp.auto_orders = jsonGetBool(root, "auto_orders")
	unmarshalTechnologies(cJSON_GetObjectItem(root, "tech"), tech_level_names[:], sp.tech_level[:], sp.tech_knowledge[:], sp.tech_eps[:], sp.init_tech_level[:])
	sp.hp_original_base = jsonGetInt(root, "hp_original_base")
	sp.econ_units = jsonGetInt(root, "econ_units")
	sp.fleet_cost = jsonGetInt(root, "fleet_cost")
	sp.fleet_percent_cost = jsonGetInt(root, "fleet_percent_cost")
	unmarshalSpeciesBitfield(cJSON_GetObjectItem(root, "contacts"), sp.contact[:])
	unmarshalSpeciesBitfield(cJSON_GetObjectItem(root, "allies"), sp.ally[:])
	unmarshalSpeciesBitfield(cJSON_GetObjectItem(root, "enemies"), sp.enemy[:])
}

func unmarshalSpeciesAtmosphere(root *cJSON, sp *species_data_t) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: species: property 'atmosphere' must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "species: property 'atmosphere' must be an object\n")
		os.Exit(2)
	}
	unmarshalRequireAtmosphericGas(cJSON_GetObjectItem(root, "required"), &sp.required_gas, &sp.required_gas_min, &sp.required_gas_max)
	unmarshalGases(cJSON_GetObjectItem(root, "neutral"), sp.neutral_gas[:])
	unmarshalGases(cJSON_GetObjectItem(root, "poison"), sp.poison_gas[:])
}

func unmarshalSpeciesBitfield(array *cJSON, bits []uint32) {
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: species_bits: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsArray(array) {
		fmt.Fprintf(os.Stderr, "error: species_bits: property must be an array\n")
		os.Exit(2)
	} else if cJSON_GetArraySize(array) == 0 {
		// nothing to do
		return
	}
	for _, elem := range array.children() {
		if !cJSON_IsNumber(elem) {
			fmt.Fprintf(os.Stderr, "error: species_bits: elements must be numeric\n")
			os.Exit(2)
		}
		alien := elem.valueint // alien is 1..MAX_SPECIES
		if !(0 < alien && alien <= MAX_SPECIES) {
			fmt.Fprintf(os.Stderr, "error: species_bits: elements must be integers in range 1..%d\n", MAX_SPECIES)
			os.Exit(2)
		}
		word := (alien - 1) / 32
		bit := (alien - 1) % 32
		mask := uint32(1) << bit
		bits[word] |= mask
	}
}

func unmarshalSpeciesFile(root *cJSON, sp *species_data_t, npa []*nampla_data_t, sa []*ship_data_t) ([]*nampla_data_t, []*ship_data_t) {
	version := unmarshalVersion(cJSON_GetObjectItem(root, "version"))
	if version != 1 {
		fmt.Fprintf(os.Stderr, "error: species: version: want 1, got %d\n", version)
		os.Exit(2)
	}
	unmarshalSpecies(cJSON_GetObjectItem(root, "species"), sp)
	npa = unmarshalNamedPlanets(cJSON_GetObjectItem(root, "named_planets"), sp, npa)
	sa = unmarshalShips(cJSON_GetObjectItem(root, "ships"), sp, sa)
	sp.home.nampla = npa[0]
	sp.home.planet = sp.home.nampla.planet
	sp.home.star = sp.home.nampla.star
	if sp.home.planet.star != sp.home.nampla.star {
		fmt.Fprintf(os.Stderr, "error: species: %d: home.planet.star != home.nampla.star\n", sp.id)
		os.Exit(2)
	}
	fmt.Printf(" info: species %3d: name %s planet %s\n", sp.id, sp.name, sp.home.nampla.name)
	return npa, sa
}

func unmarshalStarColor(obj *cJSON) int {
	if obj == nil || !cJSON_IsString(obj) {
		return 0
	}
	code := obj.valuestring
	if len(code) != 1 {
		return 0
	}
	return chToStarColor(code[0])
}

func unmarshalStarType(obj *cJSON) int {
	if obj == nil || !cJSON_IsString(obj) {
		return 0
	}
	code := obj.valuestring
	if len(code) != 1 {
		return 0
	}
	return chToStarType(code[0])
}

func unmarshalSystem(root *cJSON, sd *star_data_t, pa []*planet_data_t) {
	sd.x = jsonGetInt(root, "x")
	sd.y = jsonGetInt(root, "y")
	sd.z = jsonGetInt(root, "z")
	sd.star_type = unmarshalStarType(cJSON_GetObjectItem(root, "type"))
	sd.color = unmarshalStarColor(cJSON_GetObjectItem(root, "color"))
	sd.size = jsonGetInt(root, "size")
	sd.num_planets = unmarshalPlanets(cJSON_GetObjectItem(root, "planets"), sd, pa)
	sd.home_system = jsonGetBool(root, "home_system")
	sd.worm_here = jsonGetBool(root, "worm_here")
	sd.worm_x = jsonGetInt(root, "worm_x")
	sd.worm_y = jsonGetInt(root, "worm_y")
	sd.worm_z = jsonGetInt(root, "worm_z")
	unmarshalSpeciesBitfield(cJSON_GetObjectItem(root, "visited_by"), sd.visited_by[:])
	sd.message = jsonGetInt(root, "message")
}

func unmarshalSystems(root *cJSON, sa []*star_data_t, pa []*planet_data_t) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: systems: property 'systems' must not be null")
		os.Exit(2)
	} else if !cJSON_IsArray(root) {
		fmt.Fprintf(os.Stderr, "error: systems: property 'systems' must be an array")
		os.Exit(2)
	} else if cJSON_GetArraySize(root) != num_stars {
		fmt.Fprintf(os.Stderr, "error: systems: array should contain %d stars\n", num_stars)
		os.Exit(2)
	}
	star_index := 0
	planet_index := 0
	for _, elem := range root.children() {
		sd := sa[star_index]
		sd.id = star_index + 1
		sd.index = star_index
		sd.planet_index = planet_index
		unmarshalSystem(elem, sd, pa[planet_index:])
		star_index = star_index + 1
		planet_index = planet_index + sd.num_planets
	}
	fmt.Printf("unmarshal: systems: found %8d stars\n", star_index)
	fmt.Printf("unmarshal: systems: found %8d planets\n", planet_index)
	for i := 0; i < num_stars; i++ {
		sd := star_base[i]
		if sd.worm_here != FALSE {
			for j := 0; j < num_stars && sd.wormholeExit == nil; j++ {
				other := star_base[j]
				// NOTE: the C original compares other->z against sd->z
				// (not sd->worm_z); the bug is preserved here.
				if other.x == sd.worm_x && other.y == sd.worm_y && other.z == sd.z {
					sd.wormholeExit = other
					break
				}
			}
		}
	}
}

func unmarshalSystemsFile(root *cJSON, sa []*star_data_t, pa []*planet_data_t) {
	version := unmarshalVersion(cJSON_GetObjectItem(root, "version"))
	if version != 1 {
		fmt.Fprintf(os.Stderr, "error: systems: version: want 1, got %d\n", version)
		os.Exit(2)
	}
	unmarshalSystems(cJSON_GetObjectItem(root, "systems"), sa, pa)
	os.Stdout.Sync() // fflush(stdout)
}

func unmarshalTechnologies(root *cJSON, codes []string, levels []int, knowledge []int, xp []int, init_levels []int) {
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: technologies: property must not be null\n")
		os.Exit(2)
	} else if !cJSON_IsObject(root) {
		fmt.Fprintf(os.Stderr, "error: technologies: property must be an object\n")
		os.Exit(2)
	}
	for i := 0; i < 6; i++ {
		levels[i] = 0
		knowledge[i] = 0
		xp[i] = 0
		init_levels[i] = 0
	}
	for i := 0; i < 6; i++ {
		elem := cJSON_GetObjectItem(root, codes[i])
		if elem == nil {
			continue
		} else if !cJSON_IsObject(elem) {
			fmt.Fprintf(os.Stderr, "error: technologies: '%s' must be an object\n", codes[i])
			os.Exit(2)
		}
		levels[i] = jsonGetInt(elem, "level")
		knowledge[i] = jsonGetInt(elem, "knowledge")
		xp[i] = jsonGetInt(elem, "xp")
		init_levels[i] = jsonGetInt(elem, "init_level")
		// printf("unTech: code '%s' level %3d knowl %3d xp %3d init %3d\n", codes[i], levels[i], knowledge[i], xp[i], init_levels[i]);
	}
}

func unmarshalVersion(elem *cJSON) int {
	if elem == nil {
		return 0
	} else if !cJSON_IsNumber(elem) {
		fmt.Fprintf(os.Stderr, "error: version: element must be an integer\n")
		os.Exit(2)
	}
	return elem.valueint
}
