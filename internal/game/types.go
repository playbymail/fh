package game

// Core entity types from engine.h. Types use the C typedef names (the
// "_t" suffix) because the bare struct names collide with C global
// variable names (for example the ship_data global array). Field names
// keep the C spelling so the port can be verified against the original
// source, with one exception: "type" is a reserved word in Go, so
//
//	star_data.type  -> star_type
//	ship_data.type  -> ship_type
//	trans_data.type -> trans_type
//
// Fixed-size C char arrays become Go strings; the binary I/O layer
// handles conversion.

type galaxy_data_t struct {
	d_num_species int /* Design number of species in galaxy. */
	num_species   int /* Actual number of species allocated. */
	radius        int /* Galactic radius in parsecs. */
	turn_number   int /* Current turn number. */
}

type star_data_t struct {
	id           int /* Unique identifier for this system. */
	index        int /* Index of this system in star_base array. */
	x, y, z      int /* Coordinates. */
	star_type    int /* Dwarf, degenerate, main sequence or giant. ("type" in C.) */
	color        int /* Star color. Blue, blue-white, etc. */
	size         int /* Star size, from 0 thru 9 inclusive. */
	num_planets  int /* Number of usable planets in star system. */
	home_system  int /* TRUE if this is a good potential home system. */
	worm_here    int /* TRUE if wormhole entry/exit. */
	worm_x       int /* Coordinates of wormhole's exit. */
	worm_y       int
	worm_z       int
	wormholeExit *star_data_t
	planet_index int                       /* Index (starting at zero) into the file "planets.dat" of the first planet in the star system. */
	message      int                       /* Message associated with this star system, if any. */
	visited_by   [NUM_CONTACT_WORDS]uint32 /* A bit is set if corresponding species has been here. */
	planets      [10]*planet_data_t
}

type planet_data_t struct {
	id                int          /* Unique identifier for this planet. */
	index             int          /* Index of this planet into the planet_base array. */
	temperature_class int          /* Temperature class, 1-30. */
	pressure_class    int          /* Pressure class, 0-29. */
	special           int          /* 0 = not special, 1 = ideal home planet, 2 = ideal colony planet, 3 = radioactive hellhole. */
	gas               [4]int       /* Gas in atmosphere. Zero if none. */
	gas_percent       [4]int       /* Percentage of gas in atmosphere. */
	diameter          int          /* Diameter in thousands of kilometers. */
	gravity           int          /* Surface gravity. Multiple of Earth gravity times 100. */
	mining_difficulty int          /* Mining difficulty times 100. */
	econ_efficiency   int          /* Economic efficiency. Always 100 for a home planet. */
	md_increase       int          /* Increase in mining difficulty. */
	message           int          /* Message associated with this planet, if any. */
	isValid           int          /* FALSE if the record is invalid. */
	star              *star_data_t /* Star the planet is orbiting. */
	orbit             int          /* Orbit of planet in the system. */
}

type nampla_data_t struct {
	id             int            /* Unique identifier for this named planet. */
	name           string         /* Name of planet. */
	x, y, z, pn    int            /* Coordinates. */
	status         int            /* Status of planet. */
	hiding         int            /* HIDE order given. */
	hidden         int            /* Colony is hidden. */
	planet_index   int            /* Index (starting at zero) into the file "planets.dat" of this planet. */
	siege_eff      int            /* Siege effectiveness - a percentage between 0 and 99. */
	shipyards      int            /* Number of shipyards on planet. */
	IUs_needed     int            /* Incoming ship with only CUs on board. */
	AUs_needed     int            /* Incoming ship with only CUs on board. */
	auto_IUs       int            /* Number of IUs to be automatically installed. */
	auto_AUs       int            /* Number of AUs to be automatically installed. */
	IUs_to_install int            /* Colonial mining units to be installed. */
	AUs_to_install int            /* Colonial manufacturing units to be installed. */
	mi_base        int            /* Mining base times 10. */
	ma_base        int            /* Manufacturing base times 10. */
	pop_units      int            /* Number of available population units. */
	item_quantity  [MAX_ITEMS]int /* Quantity of each item available. */
	use_on_ambush  int            /* Amount to use on ambush. */
	message        int            /* Message associated with this planet, if any. */
	special        int            /* Different for each application. */
	star           *star_data_t   /* System the colony is in. */
	planet         *planet_data_t /* Planet the colony is on. */
}

type ship_data_t struct {
	id                   int            /* Unique identifier for this ship. */
	name                 string         /* Name of ship. */
	x, y, z, pn          int            /* Current coordinates. */
	status               int            /* Current status of ship. */
	ship_type            int            /* Ship type. ("type" in C.) */
	dest_x               int            /* Destination if ship was forced to jump from combat. */
	dest_y               int            /* Ditto. */
	dest_z               int            /* Ditto. Also used by TELESCOPE command. */
	just_jumped          int            /* Set if ship jumped this turn. */
	arrived_via_wormhole int            /* Ship arrived via wormhole in the PREVIOUS turn. */
	class                int            /* Ship class. */
	tonnage              int            /* Ship tonnage divided by 10,000. */
	item_quantity        [MAX_ITEMS]int /* Quantity of each item carried. */
	age                  int            /* Ship age. */
	remaining_cost       int            /* The cost needed to complete the ship if still under construction. */
	loading_point        int            /* Nampla index for planet where ship was last loaded with CUs. Zero = none. Use 9999 for home planet. */
	unloading_point      int            /* Nampla index for planet that ship should be given orders to jump to where it will unload. Zero = none. Use 9999 for home planet. */
	special              int            /* Different for each application. */
}

type species_home_t struct {
	star   *star_data_t   /* Star containing the planet containing the colony. */
	planet *planet_data_t /* Planet containing the colony. */
	nampla *nampla_data_t /* Nampla defining the colony. */
}

type species_data_t struct {
	id                 int                       /* Unique identifier for this species. */
	index              int                       /* Index of this species in spec_data array. */
	name               string                    /* Name of species. */
	govt_name          string                    /* Name of government. */
	govt_type          string                    /* Type of government. */
	x, y, z, pn        int                       /* Coordinates of home planet. */
	required_gas       int                       /* Gas required by species. */
	required_gas_min   int                       /* Minimum needed percentage. */
	required_gas_max   int                       /* Maximum allowed percentage. */
	neutral_gas        [6]int                    /* Gases neutral to species. */
	poison_gas         [6]int                    /* Gases poisonous to species. */
	auto_orders        int                       /* AUTO command was issued. */
	tech_level         [6]int                    /* Actual tech levels. */
	init_tech_level    [6]int                    /* Tech levels at start of turn. */
	tech_knowledge     [6]int                    /* Unapplied tech level knowledge. */
	num_namplas        int                       /* Number of named planets, including home planet and colonies. */
	num_ships          int                       /* Number of ships. */
	tech_eps           [6]int                    /* Experience points for tech levels. */
	hp_original_base   int                       /* If non-zero, home planet was bombed either by bombardment or germ warfare and has not yet fully recovered. Value is total economic base before bombing. */
	econ_units         int                       /* Number of economic units. */
	fleet_cost         int                       /* Total fleet maintenance cost. */
	fleet_percent_cost int                       /* Fleet maintenance cost as a percentage times one hundred. */
	contact            [NUM_CONTACT_WORDS]uint32 /* A bit is set if corresponding species has been met. */
	ally               [NUM_CONTACT_WORDS]uint32 /* A bit is set if corresponding species is considered an ally. */
	enemy              [NUM_CONTACT_WORDS]uint32 /* A bit is set if corresponding species is considered an enemy. */
	home               species_home_t
}

// sp_loc_data_t is from location.h.
type sp_loc_data_t struct {
	s       int /* Species number. */
	x, y, z int
}

// trans_data_t is from transaction.h.
type trans_data_t struct {
	trans_type int /* Transaction type. ("type" in C.) */
	donor      int
	recipient  int
	value      int /* Value of transaction. */
	x, y, z    int
	pn         int /* Location associated with transaction. */
	number1    int /* Other items associated with transaction. */
	name1      string
	number2    int
	name2      string
	number3    int
	name3      string
}
