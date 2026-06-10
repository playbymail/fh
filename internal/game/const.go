package game

// Constants from const.h.
const (
	TRUE  = 1
	FALSE = 0

	STANDARD_NUMBER_OF_SPECIES      = 15 /* A standard game has 15 species. */
	STANDARD_NUMBER_OF_STAR_SYSTEMS = 90 /* A standard game has 90 star systems. */
	STANDARD_GALACTIC_RADIUS        = 20 /* A standard game has a galaxy with a radius of 20 parsecs. */

	/* Minimum and maximum values for a galaxy. */
	MIN_SPECIES  = 1
	MAX_SPECIES  = 100
	MIN_STARS    = 12
	MAX_STARS    = 1000
	MIN_RADIUS   = 6
	MAX_RADIUS   = 50
	MAX_DIAMETER = 2 * MAX_RADIUS
	MAX_PLANETS  = 9 * MAX_STARS
	MAX_OBS_LOCS = 5000

	HP_AVAILABLE_POP = 1500

	/* Assume at least 32 bits per long word. */
	NUM_CONTACT_WORDS = ((MAX_SPECIES - 1) / 32) + 1
)

// Item IDs from item.h.
const (
	RM        = 0  /* Raw Material Units. */
	PD        = 1  /* Planetary Defense Units. */
	SU        = 2  /* Starbase Units. */
	DR        = 3  /* Damage Repair Units. */
	CU        = 4  /* Colonist Units. */
	IU        = 5  /* Colonial Mining Units. */
	AU        = 6  /* Colonial Manufacturing Units. */
	FS        = 7  /* Fail-Safe Jump Units. */
	JP        = 8  /* Jump Portal Units. */
	FM        = 9  /* Forced Misjump Units. */
	FJ        = 10 /* Forced Jump Units. */
	GT        = 11 /* Gravitic Telescope Units. */
	FD        = 12 /* Field Distortion Units. */
	TP        = 13 /* Terraforming Plants. */
	GW        = 14 /* Germ Warfare Bombs. */
	SG1       = 15 /* Mark-1 Auxiliary Shield Generators. */
	SG2       = 16 /* Mark-2. */
	SG3       = 17 /* Mark-3. */
	SG4       = 18 /* Mark-4. */
	SG5       = 19 /* Mark-5. */
	SG6       = 20 /* Mark-6. */
	SG7       = 21 /* Mark-7. */
	SG8       = 22 /* Mark-8. */
	SG9       = 23 /* Mark-9. */
	GU1       = 24 /* Mark-1 Auxiliary Gun Units. */
	GU2       = 25 /* Mark-2. */
	GU3       = 26 /* Mark-3. */
	GU4       = 27 /* Mark-4. */
	GU5       = 28 /* Mark-5. */
	GU6       = 29 /* Mark-6. */
	GU7       = 30 /* Mark-7. */
	GU8       = 31 /* Mark-8. */
	GU9       = 32 /* Mark-9. */
	X1        = 33 /* Unassigned. */
	X2        = 34 /* Unassigned. */
	X3        = 35 /* Unassigned. */
	X4        = 36 /* Unassigned. */
	X5        = 37 /* Unassigned. */
	MAX_ITEMS = 38
)

// Ship classes from ship.h.
const (
	PB               = 0  /* Picketboat. */
	CT               = 1  /* Corvette. */
	ES               = 2  /* Escort. */
	FF               = 3  /* Frigate. (was FG, was 4) */
	DD               = 4  /* Destroyer. (was 3) */
	CL               = 5  /* Light Cruiser. */
	CS               = 6  /* Strike Cruiser. */
	CA               = 7  /* Heavy Cruiser. */
	CC               = 8  /* Command Cruiser. */
	BC               = 9  /* Battlecruiser. */
	BS               = 10 /* Battleship. */
	DN               = 11 /* Dreadnought. */
	SD               = 12 /* Super Dreadnought. */
	BM               = 13 /* Battlemoon. */
	BW               = 14 /* Battleworld. */
	BR               = 15 /* Battlestar. */
	BA               = 16 /* Starbase. */
	TR               = 17 /* Transport. */
	NUM_SHIP_CLASSES = 18

	NUM_EXTRA_SHIPS = 100
)

// Ship types from ship.h.
const (
	FTL       = 0
	SUB_LIGHT = 1
	STARBASE  = 2
)

// Ship status codes from ship.h.
const (
	UNDER_CONSTRUCTION = 0
	ON_SURFACE         = 1
	IN_ORBIT           = 2
	IN_DEEP_SPACE      = 3
	JUMPED_IN_COMBAT   = 4
	FORCED_JUMP        = 5
)

// Tech level IDs from species.h.
const (
	MI = 0 /* Mining tech level. */
	MA = 1 /* Manufacturing tech level. */
	ML = 2 /* Military tech level. */
	GV = 3 /* Gravitics tech level. */
	LS = 4 /* Life Support tech level. */
	BI = 5 /* Biology tech level. */
)

// Named planet status codes from nampla.h. These are logically ORed together.
const (
	HOME_PLANET      = 1
	COLONY           = 2
	POPULATED        = 8
	MINING_COLONY    = 16
	RESORT_COLONY    = 32
	DISBANDED_COLONY = 64

	NUM_EXTRA_NAMPLAS = 50
)

// Star types from star.h.
const (
	DWARF         = 1
	DEGENERATE    = 2
	MAIN_SEQUENCE = 3
	GIANT         = 4

	NUM_EXTRA_STARS = 20
)

// Star colors from star.h.
const (
	BLUE         = 1
	BLUE_WHITE   = 2
	WHITE        = 3
	YELLOW_WHITE = 4
	YELLOW       = 5
	ORANGE       = 6
	RED          = 7
)

// Gases in planetary atmospheres from planet.h.
const (
	H2  = 1  /* Hydrogen */
	CH4 = 2  /* Methane */
	HE  = 3  /* Helium */
	NH3 = 4  /* Ammonia */
	N2  = 5  /* Nitrogen */
	CO2 = 6  /* Carbon Dioxide */
	O2  = 7  /* Oxygen */
	HCL = 8  /* Hydrogen Chloride */
	CL2 = 9  /* Chlorine */
	F2  = 10 /* Fluorine */
	H2O = 11 /* Steam */
	SO2 = 12 /* Sulfur Dioxide */
	H2S = 13 /* Hydrogen Sulfide */

	NUM_EXTRA_PLANETS = 100
)

// Command codes from command.h.
const (
	UNDEFINED    = 0
	ALLY         = 1
	AMBUSH       = 2
	ATTACK       = 3
	AUTO         = 4
	BASE         = 5
	BATTLE       = 6
	BUILD        = 7
	CONTINUE     = 8
	DEEP         = 9
	DESTROY      = 10
	DEVELOP      = 11
	DISBAND      = 12
	END          = 13
	ENEMY        = 14
	ENGAGE       = 15
	ESTIMATE     = 16
	HAVEN        = 17
	HIDE         = 18
	HIJACK       = 19
	IBUILD       = 20
	ICONTINUE    = 21
	INSTALL      = 22
	INTERCEPT    = 23
	JUMP         = 24
	LAND         = 25
	MESSAGE      = 26
	MOVE         = 27
	NAME         = 28
	NEUTRAL      = 29
	ORBIT        = 30
	PJUMP        = 31
	PRODUCTION   = 32
	RECYCLE      = 33
	REPAIR       = 34
	RESEARCH     = 35
	SCAN         = 36
	SEND         = 37
	SHIPYARD     = 38
	START        = 39
	SUMMARY      = 40
	SURRENDER    = 41
	TARGET       = 42
	TEACH        = 43
	TECH         = 44
	TELESCOPE    = 45
	TERRAFORM    = 46
	TRANSFER     = 47
	UNLOAD       = 48
	UPGRADE      = 49
	VISITED      = 50
	WITHDRAW     = 51
	WORMHOLE     = 52
	ZZZ          = 53
	NUM_COMMANDS = ZZZ + 1
)

// Constants needed for parsing, from command.h.
const (
	UNKNOWN    = 0
	TECH_ID    = 1
	ITEM_CLASS = 2
	SHIP_CLASS = 3
	PLANET_ID  = 4
	SPECIES_ID = 5
)

// Interspecies transaction constants from transaction.h.
const (
	MAX_TRANSACTIONS = 1000

	EU_TRANSFER               = 1
	MESSAGE_TO_SPECIES        = 2
	BESIEGE_PLANET            = 3
	SIEGE_EU_TRANSFER         = 4
	TECH_TRANSFER             = 5
	DETECTION_DURING_SIEGE    = 6
	SHIP_MISHAP               = 7
	ASSIMILATION              = 8
	INTERSPECIES_CONSTRUCTION = 9
	TELESCOPE_DETECTION       = 10
	ALIEN_JUMP_PORTAL_USAGE   = 11
	KNOWLEDGE_TRANSFER        = 12
	LANDING_REQUEST           = 13
	LOOTING_EU_TRANSFER       = 14
	ALLIES_ORDER              = 15
)

// Location constants from location.h.
const (
	MAX_LOCATIONS = 10000
)
