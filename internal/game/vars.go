package game

import "os"

// Package-level game state mirroring the C engine's globals. The original
// declares these across the *vars.c files and a few of the *io.c files;
// they are centralized here so module ports never redeclare them.
// C arrays accessed through pointer arithmetic (star_base, planet_base,
// nampla_base, ship_base) become slices of pointers.

// galaxyio.c
var galaxy galaxy_data_t

// stario.c
var num_stars int
var star_base []*star_data_t
var star_data_modified int
var num_natural_wormholes int

// starvars.c
var star *star_data_t

// planetio.c
var num_planets int
var planet_base []*planet_data_t
var planet_data_modified int

// planetvars.c
var home_planet *planet_data_t
var planet *planet_data_t

// speciesio.c
var data_in_memory [MAX_SPECIES]int
var data_modified [MAX_SPECIES]int
var spec_data [MAX_SPECIES]species_data_t

// speciesvars.c
var sp_tech_level [6]int
var species *species_data_t
var species_index int  // zero-based index, mostly for accessing arrays
var species_number int // one-based index, for reports and file names

// namplavars.c
var nampla *nampla_data_t
var nampla_base []*nampla_data_t
var namp_data [MAX_SPECIES][]*nampla_data_t
var nampla_index int
var next_nampla *nampla_data_t
var next_nampla_index int
var extra_namplas = NUM_EXTRA_NAMPLAS
var num_new_namplas [MAX_SPECIES]int

// shipvars.c
var full_ship_id string
var ignore_field_distorters = FALSE
var ship *ship_data_t
var ship_base []*ship_data_t
var ship_data [MAX_SPECIES][]*ship_data_t
var ship_index int
var truncate_name = FALSE
var extra_ships = NUM_EXTRA_SHIPS
var num_new_ships [MAX_SPECIES]int

// enginevars.c
var correct_spelling_required = FALSE

const defaultHistoricalSeedValue uint64 = 1924085713

var first_pass = FALSE
var post_arrival_phase = FALSE
var prompt_gm int
var test_mode int
var verbose_mode int
var upper_name string

// commandvars.c
var abbr_type int
var abbr_index int
var alien_portal *ship_data_t
var end_of_file = FALSE
var g_spec_number int
var g_spec_name string
var input_abbr string
var input_file *cfile
var input_line string
var input_line_pointer int // index into input_line (C uses a char pointer)
var just_opened_file int
var original_line string
var original_name string
var other_species *species_data_t
var other_species_number int
var sub_light int
var tonnage int
var value int // C declares this long

// jumpvars.c
var using_alien_portal int
var jump_portal_age int
var jump_portal_gv int
var jump_portal_name string
var jump_portal_units int

// locationvars.c — yes, the C engine really does use bare globals named
// x, y, z, and pn for the current location being processed.
var x int
var y int
var z int
var pn int

// locationio.c
var loc [MAX_LOCATIONS]sp_loc_data_t
var num_locs int

// logvars.c
var header_printed int
var log_file *os.File
var log_stdout = TRUE
var log_summary = FALSE
var log_to_file = TRUE
var logging_disabled = FALSE
var summary_file *os.File

// ordersvars.c
var orders_file *os.File

// productionvars.c
var doing_production int
var last_planet_produced = FALSE
var production_done [1000]byte
var shipyard_built int
var shipyard_capacity int

// transactionio.c
var num_transactions int
var transaction [MAX_TRANSACTIONS]trans_data_t

// ResetState restores every package-level variable to its initial value,
// matching a fresh start of a C engine process. Tests rely on this.
func ResetState() {
	galaxy = galaxy_data_t{}

	num_stars = 0
	star_base = nil
	star_data_modified = 0
	num_natural_wormholes = 0
	star = nil

	num_planets = 0
	planet_base = nil
	planet_data_modified = 0
	home_planet = nil
	planet = nil

	data_in_memory = [MAX_SPECIES]int{}
	data_modified = [MAX_SPECIES]int{}
	spec_data = [MAX_SPECIES]species_data_t{}
	sp_tech_level = [6]int{}
	species = nil
	species_index = 0
	species_number = 0

	nampla = nil
	nampla_base = nil
	namp_data = [MAX_SPECIES][]*nampla_data_t{}
	nampla_index = 0
	next_nampla = nil
	next_nampla_index = 0
	extra_namplas = NUM_EXTRA_NAMPLAS
	num_new_namplas = [MAX_SPECIES]int{}

	full_ship_id = ""
	ignore_field_distorters = FALSE
	ship = nil
	ship_base = nil
	ship_data = [MAX_SPECIES][]*ship_data_t{}
	ship_index = 0
	truncate_name = FALSE
	extra_ships = NUM_EXTRA_SHIPS
	num_new_ships = [MAX_SPECIES]int{}

	correct_spelling_required = FALSE
	first_pass = FALSE
	post_arrival_phase = FALSE
	prompt_gm = 0
	test_mode = 0
	verbose_mode = 0
	upper_name = ""

	abbr_type = 0
	abbr_index = 0
	alien_portal = nil
	end_of_file = FALSE
	g_spec_number = 0
	g_spec_name = ""
	input_abbr = ""
	input_file = nil
	input_line = ""
	input_line_pointer = 0
	just_opened_file = 0
	original_line = ""
	original_name = ""
	other_species = nil
	other_species_number = 0
	sub_light = 0
	tonnage = 0
	value = 0

	using_alien_portal = 0
	jump_portal_age = 0
	jump_portal_gv = 0
	jump_portal_name = ""
	jump_portal_units = 0

	x, y, z, pn = 0, 0, 0, 0

	loc = [MAX_LOCATIONS]sp_loc_data_t{}
	num_locs = 0

	header_printed = 0
	log_file = nil
	log_stdout = TRUE
	log_summary = FALSE
	log_to_file = TRUE
	logging_disabled = FALSE
	summary_file = nil

	orders_file = nil

	doing_production = 0
	last_planet_produced = FALSE
	production_done = [1000]byte{}
	shipyard_built = 0
	shipyard_capacity = 0

	num_transactions = 0
	transaction = [MAX_TRANSACTIONS]trans_data_t{}

	resetLogState()
	resetMoneyState()
	resetCombatState()
	resetInterceptState()
	resetPlanetState()

	prngSeed = 0
}
