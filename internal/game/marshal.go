package game

// Port of marshal.c — converts engine state to JSON for the export
// command. The C builds a cJSON tree and prints it with cJSON_Print;
// this port reproduces that with a minimal cJSON stand-in (defined at
// the bottom of this file along with the subset of cjson/helpers.c
// that the export/import modules use). The printer is byte-identical
// to cJSON_Print's formatted output: objects open with "{\n", members
// are indented with tabs and separated from their keys by ":\t",
// arrays are printed inline with ", " between elements.
//
// The global_* structs from engine.h ("used to convert to and from the
// JSON data files") are defined here as Go equivalents for parity with
// the C engine; the current marshal.c/unmarshal.c do not reference
// them, so nothing in this wave populates them.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"strconv"

	"encoding/json"
)

// ---------------------------------------------------------------------------
// global_* structs from engine.h. C char arrays become Go strings;
// NULL-terminated pointer arrays become slices.

type global_cluster_t struct {
	radius        int
	d_num_species int
	num_systems   int
	systems       []*global_system_t
}

type global_coords_t struct {
	x, y, z int
}

type global_location_t struct {
	x, y, z    int
	orbit      int // the first orbit is 1
	colony     string
	deep_space int
	in_orbit   int
	on_surface int
	system     *global_system_t
	systemId   int
	planet     *global_planet_t
	planetId   int
}

type global_colony_t struct {
	id            int
	name          string
	homeworld     int
	location      global_location_t
	develop       [3]*global_develop_t
	hiding        int
	hidden        int
	inventory     []*global_item_t
	ma_base       int
	message       int
	mi_base       int
	pop_units     int
	shipyards     int
	siege_eff     int
	special       int
	status        int
	use_on_ambush int
	_nampla       *nampla_data_t
}

type global_data_t struct {
	turn        int
	cluster     *global_cluster_t
	num_species int
	species     []*global_species_t
}

type global_develop_t struct {
	code             string
	auto_install     int
	units_needed     int
	units_to_install int
}

type global_gas_t struct {
	code      string
	atmos_pct int
	min_pct   int
	max_pct   int
	required  int
}

type global_item_t struct {
	code     string
	quantity int
}

type global_planet_t struct {
	id                  int
	orbit               int
	diameter            int
	econ_efficiency     int
	gases               [5]*global_gas_t
	gravity             int
	idealHomePlanet     int
	idealColonyPlanet   int
	md_increase         int
	message             int
	mining_difficulty   int
	pressure_class      int
	radioactiveHellHole int
	temperature_class   int
	_planet             *planet_data_t
}

type global_ship_t struct {
	id                   int
	name                 string
	age                  int
	arrived_via_wormhole int
	inventory            []*global_item_t
	location             global_location_t
	destination          global_location_t
	just_jumped          int
	loading_point        string
	remaining_cost       int
	special              int
	status               int
	tonnage              int // valid only for starbases
	unloading_point      string
	_ship                *ship_data_t
}

type global_skill_t struct {
	code            string
	name            string
	init_level      int
	current_level   int
	knowledge_level int
	xps             int
}

type global_species_t struct {
	id               int
	name             string
	govt_name        string
	govt_type        string
	skills           [7]*global_skill_t
	auto_orders      int
	econ_units       int
	hp_original_base int
	required_gases   [2]*global_gas_t
	neutral_gases    [7]*global_gas_t
	poison_gases     [7]*global_gas_t
	colonies         []*global_colony_t
	ships            []*global_ship_t
	contacts         [MAX_SPECIES + 1]int
	allies           [MAX_SPECIES + 1]int
	enemies          [MAX_SPECIES + 1]int
	_species         *species_data_t
}

type global_system_t struct {
	id           int
	coords       global_coords_t
	color        int
	home_system  int
	message      int
	size         int
	system_type  int // "type" in C
	wormholeExit int
	num_planets  int
	planets      []*global_planet_t
	visited_by   [MAX_SPECIES + 1]int
	_star        *star_data_t
}

// ---------------------------------------------------------------------------
// marshal.c

func marshalAtmosphericGas(code int, pct int) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: gas: unable to allocate memory\n")
		os.Exit(2)
	}
	jsonAddIntToObj(obj, "gas", "code", code)
	jsonAddIntToObj(obj, "gas", "percent", pct)
	return obj
}

func marshalAtmosphericGases(code []int, pct []int) *cJSON {
	array := cJSON_CreateArray()
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: atmosphere: unable to allocate memory\n")
		os.Exit(2)
	}
	for i := 0; i < 4; i++ {
		if !cJSON_AddItemToArray(array, marshalAtmosphericGas(code[i], pct[i])) {
			fmt.Fprintf(os.Stderr, "error: atmosphere: unable to extend array\n")
			os.Exit(2)
		}
	}
	return array
}

func marshalConsUnits(num_needed int, num_auto int, num_install int) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: cons_units: unable to allocate memory\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "needed", float64(num_needed)) == nil {
		fmt.Fprintf(os.Stderr, "error: cons_units: unable to add property 'needed'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "auto", float64(num_auto)) == nil {
		fmt.Fprintf(os.Stderr, "error: cons_units: unable to add property 'needed'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "install", float64(num_install)) == nil {
		fmt.Fprintf(os.Stderr, "error: cons_units: unable to add property 'install'\n")
		os.Exit(2)
	}
	return obj
}

func marshalCoords(x int, y int, z int) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: coords: unable to allocate object\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "x", float64(x)) == nil {
		fmt.Fprintf(os.Stderr, "error: coords: unable to add property 'x'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "y", float64(y)) == nil {
		fmt.Fprintf(os.Stderr, "error: coords: unable to add property 'y'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "z", float64(z)) == nil {
		fmt.Fprintf(os.Stderr, "error: coords: unable to add property 'z'\n")
		os.Exit(2)
	}
	return obj
}

func marshalCoordsWithOrbit(x int, y int, z int, orbit int) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: coords_with_orbit: unable to allocate object\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "x", float64(x)) == nil {
		fmt.Fprintf(os.Stderr, "error: coords_with_orbit: unable to add property 'x'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "y", float64(y)) == nil {
		fmt.Fprintf(os.Stderr, "error: coords_with_orbit: unable to add property 'y'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "z", float64(z)) == nil {
		fmt.Fprintf(os.Stderr, "error: coords_with_orbit: unable to add property 'z'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "orbit", float64(orbit)) == nil {
		fmt.Fprintf(os.Stderr, "error: coords_with_orbit: unable to add property 'orbit'\n")
		os.Exit(2)
	}
	return obj
}

func marshalGalaxy(g *galaxy_data_t) *cJSON {
	root := cJSON_CreateObject()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: galaxy: property 'galaxy': unable to allocate\n")
		os.Exit(2)
	}
	_ = g // the C original ignores its parameter and reads the globals
	jsonAddIntToObj(root, "galaxy", "turn_number", galaxy.turn_number)
	jsonAddIntToObj(root, "galaxy", "num_species", galaxy.num_species)
	jsonAddIntToObj(root, "galaxy", "d_num_species", galaxy.d_num_species)
	jsonAddIntToObj(root, "galaxy", "radius", galaxy.radius)
	return root
}

func marshalGalaxyFile() *cJSON {
	root := cJSON_CreateObject()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: galaxy: unable to allocate root\n")
		os.Exit(2)
	}
	cJSON_AddItemToObject(root, "version", marshalVersion(1))
	cJSON_AddItemToObject(root, "galaxy", marshalGalaxy(&galaxy))
	return root
}

func marshalGases(code []int) *cJSON {
	obj := cJSON_CreateArray()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: gases: unable to allocate memory\n")
		os.Exit(2)
	}
	for i := 0; i < 6; i++ {
		if !cJSON_AddItemToArray(obj, cJSON_CreateNumber(float64(code[i]))) {
			fmt.Fprintf(os.Stderr, "error: gases: unable to extend array\n")
			os.Exit(2)
		}
	}
	return obj
}

func marshalGovernment(name string, govtType string) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: government: unable to allocate memory\n")
		os.Exit(2)
	}
	jsonAddStringToObj(obj, "government", "name", name)
	jsonAddStringToObj(obj, "government", "type", govtType)
	return obj
}

func marshalItem(code int, qty int) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: item: unable to allocate memory\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "code", float64(code)) == nil {
		fmt.Fprintf(os.Stderr, "error: item: unable to add property 'code'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "qty", float64(qty)) == nil {
		fmt.Fprintf(os.Stderr, "error: item: unable to add property 'qty'\n")
		os.Exit(2)
	}
	return obj
}

func marshalItems(items []int) *cJSON {
	array := cJSON_CreateArray()
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: items: unable to allocate array\n")
	}
	for i := 0; i < MAX_ITEMS; i++ {
		if items[i] > 0 {
			cJSON_AddItemToArray(array, marshalItem(i, items[i]))
		}
	}
	return array
}

func marshalMiningDifficulty(base int, increase int) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: mining_difficulty: unable to allocate object\n")
		os.Exit(2)
	}
	jsonAddIntToObj(obj, "mining_difficulty", "base", base)
	jsonAddIntToObj(obj, "mining_difficulty", "increase", increase)
	return obj
}

func marshalNamedPlanet(npd *nampla_data_t) *cJSON {
	objName := "named_planet"
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: named_planet: unable to allocate object\n")
		os.Exit(2)
	}
	jsonAddStringToObj(obj, objName, "name", npd.name)
	cJSON_AddItemToObject(obj, "location", marshalCoordsWithOrbit(npd.x, npd.y, npd.z, npd.pn))
	jsonAddIntToObj(obj, objName, "status", npd.status)
	jsonAddIntToObj(obj, objName, "hiding", npd.hiding)
	jsonAddIntToObj(obj, objName, "hidden", npd.hidden)
	jsonAddIntToObj(obj, objName, "siege_eff", npd.siege_eff)
	jsonAddIntToObj(obj, objName, "shipyards", npd.shipyards)
	cJSON_AddItemToObject(obj, "ius", marshalConsUnits(npd.IUs_needed, npd.auto_IUs, npd.IUs_to_install))
	cJSON_AddItemToObject(obj, "aus", marshalConsUnits(npd.AUs_needed, npd.auto_AUs, npd.AUs_to_install))
	jsonAddIntToObj(obj, objName, "mi_base", npd.mi_base)
	jsonAddIntToObj(obj, objName, "ma_base", npd.ma_base)
	jsonAddIntToObj(obj, objName, "pop_units", npd.pop_units)
	cJSON_AddItemToObject(obj, "items", marshalItems(npd.item_quantity[:]))
	jsonAddIntToObj(obj, objName, "use_on_ambush", npd.use_on_ambush)
	jsonAddIntToObj(obj, objName, "message", npd.message)
	jsonAddIntToObj(obj, objName, "special", npd.special)
	return obj
}

func marshalNamedPlanets(npa []*nampla_data_t, num int) *cJSON {
	array := cJSON_CreateArray()
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: named_planets: unable to allocate memory\n")
		os.Exit(2)
	}
	for i := 0; i < num; i++ {
		if !cJSON_AddItemToArray(array, marshalNamedPlanet(npa[i])) {
			fmt.Fprintf(os.Stderr, "error: species: unable to extend array 'named_planets'")
			os.Exit(2)
		}
	}
	return array
}

func marshalPlanet(pd *planet_data_t) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: planet: unable to allocate object\n")
		os.Exit(2)
	}
	jsonAddIntToObj(obj, "planet", "temperature_class", pd.temperature_class)
	jsonAddIntToObj(obj, "planet", "pressure_class", pd.pressure_class)
	jsonAddIntToObj(obj, "planet", "special", pd.special)
	cJSON_AddItemToObject(obj, "atmosphere", marshalAtmosphericGases(pd.gas[:], pd.gas_percent[:]))
	jsonAddIntToObj(obj, "planet", "diameter", pd.diameter)
	jsonAddIntToObj(obj, "planet", "gravity", pd.gravity)
	cJSON_AddItemToObject(obj, "mining_difficulty", marshalMiningDifficulty(pd.mining_difficulty, pd.md_increase))
	jsonAddIntToObj(obj, "planet", "econ_efficiency", pd.econ_efficiency)
	jsonAddIntToObj(obj, "planet", "message", pd.message)
	return obj
}

func marshalPlanets(planets []*planet_data_t, num int) *cJSON {
	array := cJSON_CreateArray()
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: planets: unable to allocate array\n")
		os.Exit(2)
	}
	for p := 0; p < num; p++ {
		pd := planets[p]
		jsonAddItemToArray(array, "planets", marshalPlanet(pd))
	}
	return array
}

func marshalRequiredAtmosphericGas(code int, min_pct int, max_pct int) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: required_atmospheric_gas: unable to allocate object\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "code", float64(code)) == nil {
		fmt.Fprintf(os.Stderr, "error: required_atmospheric_gas: unable to add property 'code'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "min_pct", float64(min_pct)) == nil {
		fmt.Fprintf(os.Stderr, "error: required_atmospheric_gas: unable to add property 'min_pct'\n")
		os.Exit(2)
	}
	if cJSON_AddNumberToObject(obj, "max_pct", float64(max_pct)) == nil {
		fmt.Fprintf(os.Stderr, "error: required_atmospheric_gas: unable to add property 'max_pct'\n")
		os.Exit(2)
	}
	return obj
}

func marshalShip(sd *ship_data_t) *cJSON {
	objName := "ship"
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: ship: unable to allocate object\n")
		os.Exit(2)
	}
	jsonAddStringToObj(obj, objName, "name", sd.name)
	cJSON_AddItemToObject(obj, "location", marshalCoordsWithOrbit(sd.x, sd.y, sd.z, sd.pn))
	jsonAddIntToObj(obj, objName, "status", sd.status)
	jsonAddIntToObj(obj, objName, "type", sd.ship_type)
	cJSON_AddItemToObject(obj, "dest", marshalCoords(sd.dest_x, sd.dest_y, sd.dest_z))
	jsonAddIntToObj(obj, objName, "just_jumped", sd.just_jumped)
	jsonAddBoolToObj(obj, objName, "arrived_via_wormhole", sd.arrived_via_wormhole)
	jsonAddIntToObj(obj, objName, "class", sd.class)
	jsonAddIntToObj(obj, objName, "tonnage", sd.tonnage)
	cJSON_AddItemToObject(obj, "cargo", marshalItems(sd.item_quantity[:]))
	jsonAddIntToObj(obj, objName, "age", sd.age)
	jsonAddIntToObj(obj, objName, "remaining_cost", sd.remaining_cost)
	jsonAddIntToObj(obj, objName, "loading_point", sd.loading_point)
	jsonAddIntToObj(obj, objName, "unloading_point", sd.unloading_point)
	jsonAddIntToObj(obj, objName, "special", sd.special)
	return obj
}

func marshalShips(sa []*ship_data_t, num int) *cJSON {
	array := cJSON_CreateArray()
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: ships: unable to allocate memory\n")
		os.Exit(2)
	}
	for i := 0; i < num; i++ {
		if !cJSON_AddItemToArray(array, marshalShip(sa[i])) {
			fmt.Fprintf(os.Stderr, "error: ships: unable to extend array")
			os.Exit(2)
		}
	}
	return array
}

func marshalSpecies(sp *species_data_t) *cJSON {
	root := cJSON_CreateObject()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: species: unable to allocate property 'species'\n")
		os.Exit(2)
	}
	objName := "species"
	jsonAddIntToObj(root, objName, "id", sp.id)
	jsonAddStringToObj(root, objName, "name", sp.name)
	cJSON_AddItemToObject(root, "government", marshalGovernment(sp.govt_name, sp.govt_type))
	cJSON_AddItemToObject(root, "home_world", marshalCoordsWithOrbit(sp.x, sp.y, sp.z, sp.pn))
	cJSON_AddItemToObject(root, "atmosphere", marshalSpeciesAtmosphere(
		marshalRequiredAtmosphericGas(sp.required_gas, sp.required_gas_min, sp.required_gas_max),
		marshalGases(sp.neutral_gas[:]),
		marshalGases(sp.poison_gas[:])))
	jsonAddBoolToObj(root, objName, "auto_orders", sp.auto_orders)
	cJSON_AddItemToObject(root, "tech",
		marshalTechnologies(tech_level_names[:], sp.tech_level[:], sp.tech_knowledge[:], sp.tech_eps[:], sp.init_tech_level[:]))
	jsonAddIntToObj(root, objName, "hp_original_base", sp.hp_original_base)
	jsonAddIntToObj(root, objName, "econ_units", sp.econ_units)
	jsonAddIntToObj(root, objName, "fleet_cost", sp.fleet_cost)
	jsonAddIntToObj(root, objName, "fleet_percent_cost", sp.fleet_percent_cost)
	cJSON_AddItemToObject(root, "contacts", marshalSpeciesBitfield(sp.contact[:]))
	cJSON_AddItemToObject(root, "allies", marshalSpeciesBitfield(sp.ally[:]))
	cJSON_AddItemToObject(root, "enemies", marshalSpeciesBitfield(sp.enemy[:]))
	return root
}

func marshalSpeciesAtmosphere(required *cJSON, neutral *cJSON, poison *cJSON) *cJSON {
	root := cJSON_CreateObject()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: species: unable to allocate atmosphere\n")
		os.Exit(2)
	}
	cJSON_AddItemToObject(root, "required", required)
	cJSON_AddItemToObject(root, "neutral", neutral)
	cJSON_AddItemToObject(root, "poison", poison)
	return root
}

func marshalSpeciesBitfield(bits []uint32) *cJSON {
	array := cJSON_CreateArray()
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: species_bits: unable to allocate memory\n")
		os.Exit(2)
	}
	for alien := 1; alien <= MAX_SPECIES; alien++ { // alien is 1..MAX_SPECIES
		word := (alien - 1) / 32
		bit := (alien - 1) % 32
		mask := uint32(1) << bit
		if (bits[word] & mask) == 0 {
			// not a hit for this alien
			continue
		}
		if !cJSON_AddItemToArray(array, cJSON_CreateNumber(float64(alien))) {
			fmt.Fprintf(os.Stderr, "error: species_bits: unable to extend array\n")
			os.Exit(2)
		}
	}
	return array
}

func marshalSpeciesFile(sp *species_data_t, npa []*nampla_data_t, sa []*ship_data_t) *cJSON {
	root := cJSON_CreateObject()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: species: unable to allocate root\n")
		os.Exit(2)
	}
	cJSON_AddItemToObject(root, "version", marshalVersion(1))
	cJSON_AddItemToObject(root, "species", marshalSpecies(sp))
	cJSON_AddItemToObject(root, "named_planets", marshalNamedPlanets(npa, sp.num_namplas))
	cJSON_AddItemToObject(root, "ships", marshalShips(sa, sp.num_ships))
	return root
}

func marshalSystem(sd *star_data_t) *cJSON {
	obj := cJSON_CreateObject()
	if obj == nil {
		fmt.Fprintf(os.Stderr, "error: system: unable to allocate object\n")
		os.Exit(2)
	}
	jsonAddIntToObj(obj, "system", "x", sd.x)
	jsonAddIntToObj(obj, "system", "y", sd.y)
	jsonAddIntToObj(obj, "system", "z", sd.z)
	jsonAddStringToObj(obj, "system", "type", string(rune(star_type(sd.star_type))))
	jsonAddStringToObj(obj, "system", "color", string(rune(star_color(sd.color))))
	jsonAddIntToObj(obj, "system", "size", sd.size)
	jsonAddBoolToObj(obj, "system", "home_system", sd.home_system)
	jsonAddBoolToObj(obj, "system", "worm_here", sd.worm_here)
	jsonAddIntToObj(obj, "system", "worm_x", sd.worm_x)
	jsonAddIntToObj(obj, "system", "worm_y", sd.worm_y)
	jsonAddIntToObj(obj, "system", "worm_z", sd.worm_z)
	cJSON_AddItemToObject(obj, "visited_by", marshalSpeciesBitfield(sd.visited_by[:]))
	jsonAddIntToObj(obj, "system", "message", sd.message)
	jsonAddItemToObj(obj, "system", "planets", marshalPlanets(planet_base[sd.planet_index:], sd.num_planets))
	return obj
}

func marshalSystems(sa []*star_data_t, num int) *cJSON {
	array := cJSON_CreateArray()
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: systems: property 'sysmtems': unable to allocate\n")
		os.Exit(2)
	}
	for i := 0; i < num; i++ {
		if !cJSON_AddItemToArray(array, marshalSystem(sa[i])) {
			fmt.Fprintf(os.Stderr, "error: systems: unable to extend array 'systems'")
			os.Exit(2)
		}
	}
	return array
}

func marshalSystemsFile() *cJSON {
	root := cJSON_CreateObject()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: systems: unable to allocate root\n")
		os.Exit(2)
	}
	cJSON_AddItemToObject(root, "version", marshalVersion(1))
	cJSON_AddItemToObject(root, "systems", marshalSystems(star_base, num_stars))
	return root
}

func marshalTechnologies(codes []string, levels []int, knowledge []int, xp []int, init_levels []int) *cJSON {
	root := cJSON_CreateObject()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: technologies: unable to allocate memory\n")
		os.Exit(2)
	}
	for i := 0; i < 6; i++ {
		cJSON_AddItemToObject(root, codes[i], marshalTechnology(levels[i], knowledge[i], xp[i], init_levels[i]))
	}
	return root
}

func marshalTechnology(level int, knowledge int, xp int, init_level int) *cJSON {
	root := cJSON_CreateObject()
	if root == nil {
		fmt.Fprintf(os.Stderr, "error: technology: unable to allocate memory\n")
		os.Exit(2)
	}
	cJSON_AddNumberToObject(root, "level", float64(level))
	cJSON_AddNumberToObject(root, "knowledge", float64(knowledge))
	cJSON_AddNumberToObject(root, "xp", float64(xp))
	cJSON_AddNumberToObject(root, "init_level", float64(init_level))
	return root
}

func marshalVersion(version int) *cJSON {
	elem := cJSON_CreateNumber(float64(version))
	if elem == nil {
		fmt.Fprintf(os.Stderr, "error: version: unable to allocate memory\n")
		os.Exit(2)
	}
	return elem
}

// ---------------------------------------------------------------------------
// Minimal cJSON stand-in for the marshal/unmarshal modules. PORTING.md
// says the cjson/ internals are not ported wholesale; this is just
// enough of the API (with the C names) for marshal.go, unmarshal.go,
// export.go and import.go. The printer reproduces cJSON_Print's
// formatted output byte for byte.

const (
	cJSON_False = iota
	cJSON_True
	cJSON_NULL
	cJSON_Number
	cJSON_String
	cJSON_Array
	cJSON_Object
)

type cJSON struct {
	kind        int    // "type" in C
	key         string // "string" in C: member name when child of an object
	valuestring string
	valueint    int
	valuedouble float64
	child       []*cJSON // linked list (child/next) in C
}

// children iterates like cJSON_ArrayForEach, tolerating a nil node.
func (j *cJSON) children() []*cJSON {
	if j == nil {
		return nil
	}
	return j.child
}

func cJSON_CreateObject() *cJSON { return &cJSON{kind: cJSON_Object} }

func cJSON_CreateArray() *cJSON { return &cJSON{kind: cJSON_Array} }

// cJSON_CreateNumber saturates valueint exactly as the bundled cJSON does.
func cJSON_CreateNumber(num float64) *cJSON {
	item := &cJSON{kind: cJSON_Number, valuedouble: num}
	if num >= float64(math.MaxInt32) {
		item.valueint = math.MaxInt32
	} else if num <= float64(math.MinInt32) {
		item.valueint = math.MinInt32
	} else {
		item.valueint = int(num)
	}
	return item
}

func cJSON_CreateString(s string) *cJSON {
	return &cJSON{kind: cJSON_String, valuestring: s}
}

func cJSON_CreateBool(boolean int) *cJSON {
	if boolean != 0 {
		return &cJSON{kind: cJSON_True, valueint: 1}
	}
	return &cJSON{kind: cJSON_False}
}

func cJSON_AddItemToObject(obj *cJSON, name string, item *cJSON) bool {
	if obj == nil || item == nil {
		return false
	}
	item.key = name
	obj.child = append(obj.child, item)
	return true
}

func cJSON_AddItemToArray(array *cJSON, item *cJSON) bool {
	if array == nil || item == nil {
		return false
	}
	array.child = append(array.child, item)
	return true
}

func cJSON_AddNumberToObject(obj *cJSON, name string, num float64) *cJSON {
	item := cJSON_CreateNumber(num)
	if !cJSON_AddItemToObject(obj, name, item) {
		return nil
	}
	return item
}

func cJSON_AddBoolToObject(obj *cJSON, name string, boolean int) *cJSON {
	item := cJSON_CreateBool(boolean)
	if !cJSON_AddItemToObject(obj, name, item) {
		return nil
	}
	return item
}

func cJSON_AddStringToObject(obj *cJSON, name string, s string) *cJSON {
	item := cJSON_CreateString(s)
	if !cJSON_AddItemToObject(obj, name, item) {
		return nil
	}
	return item
}

// cJSON_GetObjectItem matches the first member whose name compares equal
// case-insensitively (byte-wise tolower), like the C original.
func cJSON_GetObjectItem(obj *cJSON, name string) *cJSON {
	if obj == nil {
		return nil
	}
	for _, c := range obj.child {
		if len(c.key) == len(name) {
			match := true
			for i := 0; i < len(name); i++ {
				if jsonToLower(c.key[i]) != jsonToLower(name[i]) {
					match = false
					break
				}
			}
			if match {
				return c
			}
		}
	}
	return nil
}

func cJSON_GetObjectItemCaseSensitive(obj *cJSON, name string) *cJSON {
	if obj == nil {
		return nil
	}
	for _, c := range obj.child {
		if c.key == name {
			return c
		}
	}
	return nil
}

func cJSON_GetArraySize(array *cJSON) int {
	if array == nil {
		return 0
	}
	return len(array.child)
}

func cJSON_IsObject(item *cJSON) bool { return item != nil && item.kind == cJSON_Object }
func cJSON_IsArray(item *cJSON) bool  { return item != nil && item.kind == cJSON_Array }
func cJSON_IsNumber(item *cJSON) bool { return item != nil && item.kind == cJSON_Number }
func cJSON_IsString(item *cJSON) bool { return item != nil && item.kind == cJSON_String }
func cJSON_IsBool(item *cJSON) bool {
	return item != nil && (item.kind == cJSON_True || item.kind == cJSON_False)
}

// cJSON_Delete is a no-op; the garbage collector reclaims the tree.
func cJSON_Delete(item *cJSON) {}

func jsonToLower(c byte) byte {
	if 'A' <= c && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

// cJSON_Print renders the tree using cJSON's formatted layout: objects
// are multi-line with tab indentation and ":\t" after each key; arrays
// are single-line (relative to their own brackets) with ", " separators.
func cJSON_Print(item *cJSON) string {
	var buf bytes.Buffer
	jsonPrintValue(&buf, item, 0)
	return buf.String()
}

func jsonPrintValue(buf *bytes.Buffer, item *cJSON, depth int) {
	switch item.kind {
	case cJSON_False:
		buf.WriteString("false")
	case cJSON_True:
		buf.WriteString("true")
	case cJSON_NULL:
		buf.WriteString("null")
	case cJSON_Number:
		jsonPrintNumber(buf, item)
	case cJSON_String:
		jsonPrintString(buf, item.valuestring)
	case cJSON_Array:
		buf.WriteByte('[')
		for i, c := range item.child {
			jsonPrintValue(buf, c, depth+1)
			if i < len(item.child)-1 {
				buf.WriteString(", ")
			}
		}
		buf.WriteByte(']')
	case cJSON_Object:
		buf.WriteString("{\n")
		for i, c := range item.child {
			for t := 0; t < depth+1; t++ {
				buf.WriteByte('\t')
			}
			jsonPrintString(buf, c.key)
			buf.WriteString(":\t")
			jsonPrintValue(buf, c, depth+1)
			if i < len(item.child)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		for t := 0; t < depth; t++ {
			buf.WriteByte('\t')
		}
		buf.WriteByte('}')
	}
}

// jsonPrintNumber reproduces cJSON's print_number. Every number the
// exporter emits is integral, so the "%d" branch is the one that runs;
// the floating-point fallback approximates C's "%1.15g".
func jsonPrintNumber(buf *bytes.Buffer, item *cJSON) {
	d := item.valuedouble
	if math.IsNaN(d) || math.IsInf(d, 0) {
		buf.WriteString("null")
	} else if d == float64(item.valueint) {
		fmt.Fprintf(buf, "%d", item.valueint)
	} else {
		s := strconv.FormatFloat(d, 'g', 15, 64)
		if test, err := strconv.ParseFloat(s, 64); err != nil || test != d {
			s = strconv.FormatFloat(d, 'g', 17, 64)
		}
		buf.WriteString(s)
	}
}

// jsonPrintString reproduces cJSON's print_string_ptr escaping.
func jsonPrintString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			buf.WriteString("\\\"")
		case '\\':
			buf.WriteString("\\\\")
		case '\b':
			buf.WriteString("\\b")
		case '\f':
			buf.WriteString("\\f")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		default:
			if c < 32 {
				fmt.Fprintf(buf, "\\u%04x", c)
			} else {
				buf.WriteByte(c)
			}
		}
	}
	buf.WriteByte('"')
}

// ---------------------------------------------------------------------------
// Subset of cjson/helpers.c used by the export/import modules. The
// allocation-failure branches of the C originals cannot occur in Go and
// are dropped.

func jsonAddBoolToObj(obj *cJSON, objName string, propName string, value int) {
	if cJSON_AddBoolToObject(obj, propName, value) == nil {
		fmt.Fprintf(os.Stderr, "%s: unable to add property '%s'\n", objName, propName)
		os.Exit(2)
	}
}

func jsonAddIntToObj(obj *cJSON, objName string, propName string, value int) {
	if cJSON_AddNumberToObject(obj, propName, float64(value)) == nil {
		fmt.Fprintf(os.Stderr, "%s: unable to add property '%s'\n", objName, propName)
		os.Exit(2)
	}
}

func jsonAddItemToArray(array *cJSON, objName string, value *cJSON) {
	if !cJSON_AddItemToArray(array, value) {
		fmt.Fprintf(os.Stderr, "%s: unable to extend array\n", objName)
		os.Exit(2)
	}
}

func jsonAddItemToObj(obj *cJSON, objName string, propName string, value *cJSON) {
	if !cJSON_AddItemToObject(obj, propName, value) {
		fmt.Fprintf(os.Stderr, "%s: unable to add property '%s'\n", objName, propName)
		os.Exit(2)
	}
}

func jsonAddStringToObj(obj *cJSON, objName string, propName string, value string) {
	if cJSON_AddStringToObject(obj, propName, value) == nil {
		fmt.Fprintf(os.Stderr, "%s: unable to add property '%s'\n", objName, propName)
		os.Exit(2)
	}
}

func jsonGetBool(obj *cJSON, property string) int {
	item := cJSON_GetObjectItemCaseSensitive(obj, property)
	if item == nil {
		fmt.Fprintf(os.Stderr, "property: %s: missing\n", property)
		os.Exit(2)
	} else if !cJSON_IsBool(item) {
		fmt.Fprintf(os.Stderr, "property: %s: not a boolean\n", property)
		os.Exit(2)
	}
	if item.valueint == 0 {
		return 0
	}
	return 1
}

func jsonGetInt(obj *cJSON, property string) int {
	item := cJSON_GetObjectItemCaseSensitive(obj, property)
	if item == nil {
		fmt.Fprintf(os.Stderr, "property: %s: missing\n", property)
		os.Exit(2)
	} else if !cJSON_IsNumber(item) {
		fmt.Fprintf(os.Stderr, "property: %s: not an integer\n", property)
		os.Exit(2)
	}
	return item.valueint
}

// jsonGetString copies at most size-1 bytes of the property's value;
// size mirrors the sizeof() the C call sites pass for their fixed
// buffers, and longer values are fatal exactly as in C.
func jsonGetString(obj *cJSON, property string, size int) string {
	item := cJSON_GetObjectItemCaseSensitive(obj, property)
	if item == nil {
		fmt.Fprintf(os.Stderr, "property: %s: must not be null\n", property)
		os.Exit(2)
	} else if !cJSON_IsString(item) {
		fmt.Fprintf(os.Stderr, "property: %s: not a string\n", property)
		os.Exit(2)
	} else if len(item.valuestring)+1 > size {
		fmt.Fprintf(os.Stderr, "jsonGetString: strlen %d exceeds limit %d\n", len(item.valuestring)+1, size)
		os.Exit(2)
	}
	return item.valuestring
}

// jsonPerror mimics perror(prefix): "prefix: strerror(errno)".
func jsonPerror(prefix string, err error) {
	msg := err.Error()
	var pe *fs.PathError
	if errors.As(err, &pe) {
		msg = pe.Err.Error()
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", prefix, msg)
}

// jsonParseFile returns nil if the file cannot be opened (like fopen);
// on a parse error it reports the location and exits, like the C helper.
func jsonParseFile(name string) *cJSON {
	fp, err := os.Open(name)
	if err != nil {
		return nil
	}
	buffer, err := io.ReadAll(fp)
	fp.Close()
	if err != nil {
		jsonPerror("json: parseFile: reading entire input", err)
		os.Exit(2)
	}

	// skip a UTF-8 byte order mark, as cJSON does
	buffer = bytes.TrimPrefix(buffer, []byte{0xEF, 0xBB, 0xBF})

	dec := json.NewDecoder(bytes.NewReader(buffer))
	dec.UseNumber()
	root, perr := jsonParseValue(dec)
	if perr != nil {
		offset := dec.InputOffset()
		var se *json.SyntaxError
		if errors.As(perr, &se) {
			offset = se.Offset
		}
		line, col, text := jsonErrorContext(buffer, offset)
		fmt.Fprintf(os.Stderr, "%s:%d:%d: error parsing just before\n\t%s\n", name, line, col, text)
		os.Exit(2)
	}
	return root
}

// jsonParseValue builds a cJSON node from the decoder's token stream so
// that member order and duplicate keys behave exactly like cJSON.
func jsonParseValue(dec *json.Decoder) (*cJSON, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return jsonParseToken(dec, tok)
}

func jsonParseToken(dec *json.Decoder, tok json.Token) (*cJSON, error) {
	switch t := tok.(type) {
	case json.Delim:
		if t == '[' {
			node := cJSON_CreateArray()
			for dec.More() {
				child, err := jsonParseValue(dec)
				if err != nil {
					return nil, err
				}
				node.child = append(node.child, child)
			}
			if _, err := dec.Token(); err != nil { // consume ']'
				return nil, err
			}
			return node, nil
		}
		if t == '{' {
			node := cJSON_CreateObject()
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := keyTok.(string)
				child, err := jsonParseValue(dec)
				if err != nil {
					return nil, err
				}
				child.key = key
				node.child = append(node.child, child)
			}
			if _, err := dec.Token(); err != nil { // consume '}'
				return nil, err
			}
			return node, nil
		}
		return nil, fmt.Errorf("unexpected delimiter %v", t)
	case string:
		return cJSON_CreateString(t), nil
	case json.Number:
		f, _ := t.Float64()
		return cJSON_CreateNumber(f), nil
	case bool:
		if t {
			return cJSON_CreateBool(TRUE), nil
		}
		return cJSON_CreateBool(FALSE), nil
	default: // nil (JSON null)
		return &cJSON{kind: cJSON_NULL}, nil
	}
}

// jsonErrorContext converts a byte offset into the line/column and
// nearby text reported by cJSON_GetError.
func jsonErrorContext(data []byte, offset int64) (int, int, string) {
	line, col := 1, 1
	pos := 0
	for pos < len(data) && int64(pos) < offset && data[pos] != 0 {
		if data[pos] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		pos++
	}
	end := pos + 255
	if end > len(data) {
		end = len(data)
	}
	text := data[pos:end]
	if i := bytes.IndexByte(text, 0); i >= 0 {
		text = text[:i]
	}
	return line, col, string(text)
}

// jsonWriteFile converts the tree to text and writes it, with a trailing
// newline, to the named file.
func jsonWriteFile(root *cJSON, kind string, name string) {
	// convert json to text
	text := cJSON_Print(root)
	// save it to the file and close it
	fp, err := os.Create(name)
	if err != nil {
		jsonPerror("fh: export: json:", err)
		fmt.Fprintf(os.Stderr, "error: %s: can not create file!\n", name)
		os.Exit(2)
	}
	fmt.Fprintf(fp, "%s\n", text)
	fp.Close()
}
