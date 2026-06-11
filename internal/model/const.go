// Package model defines the idiomatic Go domain model for the fh engine.
//
// Unlike internal/game (the byte-faithful C port), this package uses idiomatic
// Go naming and holds no package-level mutable game state. The constants and
// lookup tables below mirror the values the C engine uses when it renders
// player reports, so the fh renderer can reproduce them byte-for-byte.
package model

// MaxItems is the number of distinct item types a planet or ship can carry.
const MaxItems = 38

// Item indices (subset used by the renderer; mirrors item.h).
const (
	RM = 0  // Raw Material Units.
	PD = 1  // Planetary Defense Units.
	CU = 4  // Colonist Units.
	IU = 5  // Colonial Mining Units.
	AU = 6  // Colonial Manufacturing Units.
	FD = 12 // Field Distortion Units.
)

// Tech level indices (mirrors species.h).
const (
	MI = 0 // Mining.
	MA = 1 // Manufacturing.
	ML = 2 // Military.
	GV = 3 // Gravitics.
	LS = 4 // Life Support.
	BI = 5 // Biology.
)

// Named-planet status bits (mirrors nampla.h). Logically ORed together.
const (
	HomePlanet      = 1
	Colony          = 2
	Populated       = 8
	MiningColony    = 16
	ResortColony    = 32
	DisbandedColony = 64
)

// Ship classes (subset used by the renderer; mirrors ship.h).
const (
	BA = 16 // Starbase.
	TR = 17 // Transport.
)

// Ship types (mirrors ship.h).
const (
	FTL      = 0
	SubLight = 1
	Starbase = 2
)

// Ship status codes (mirrors ship.h).
const (
	UnderConstruction = 0
	OnSurface         = 1
	InOrbit           = 2
	InDeepSpace       = 3
	JumpedInCombat    = 4
	ForcedJump        = 5
)

// MaxSpecies and the derived contact-bitfield word count mirror const.h.
const (
	MaxSpecies      = 100
	NumContactWords = ((MaxSpecies - 1) / 32) + 1
)

// ItemAbbr holds the two/three-letter code for each item type.
var ItemAbbr = [MaxItems]string{
	"RM", "PD", "SU", "DR", "CU", "IU", "AU", "FS",
	"JP", "FM", "FJ", "GT", "FD", "TP", "GW", "SG1",
	"SG2", "SG3", "SG4", "SG5", "SG6", "SG7", "SG8", "SG9",
	"GU1", "GU2", "GU3", "GU4", "GU5", "GU6", "GU7", "GU8",
	"GU9", "X1", "X2", "X3", "X4", "X5",
}

// ItemCarryCapacity is the cargo capacity each item consumes.
var ItemCarryCapacity = [MaxItems]int{
	1, 3, 20, 1, 1, 1, 1, 1,
	10, 5, 5, 20, 1, 100, 100, 5,
	10, 15, 20, 25, 30, 35, 40, 45,
	5, 10, 15, 20, 25, 30, 35, 40,
	45, 9999, 9999, 9999, 9999, 9999,
}

// ItemName holds the singular display name for each item type.
var ItemName = [MaxItems]string{
	"Raw Material Unit",
	"Planetary Defense Unit",
	"Starbase Unit",
	"Damage Repair Unit",
	"Colonist Unit",
	"Colonial Mining Unit",
	"Colonial Manufacturing Unit",
	"Fail-Safe Jump Unit",
	"Jump Portal Unit",
	"Forced Misjump Unit",
	"Forced Jump Unit",
	"Gravitic Telescope Unit",
	"Field Distortion Unit",
	"Terraforming Plant",
	"Germ Warfare Bomb",
	"Mark-1 Shield Generator",
	"Mark-2 Shield Generator",
	"Mark-3 Shield Generator",
	"Mark-4 Shield Generator",
	"Mark-5 Shield Generator",
	"Mark-6 Shield Generator",
	"Mark-7 Shield Generator",
	"Mark-8 Shield Generator",
	"Mark-9 Shield Generator",
	"Mark-1 Gun Unit",
	"Mark-2 Gun Unit",
	"Mark-3 Gun Unit",
	"Mark-4 Gun Unit",
	"Mark-5 Gun Unit",
	"Mark-6 Gun Unit",
	"Mark-7 Gun Unit",
	"Mark-8 Gun Unit",
	"Mark-9 Gun Unit",
	"X1 Unit",
	"X2 Unit",
	"X3 Unit",
	"X4 Unit",
	"X5 Unit",
}

// ShipAbbr holds the class code for each ship class index.
var ShipAbbr = [18]string{
	"PB", "CT", "ES", "FF", "DD", "CL", "CS",
	"CA", "CC", "BC", "BS", "DN", "SD", "BM",
	"BW", "BR", "BA", "TR",
}

// ShipTonnage is the tonnage (in 10k-ton units) for each ship class index.
var ShipTonnage = [18]int{
	1, 2, 5, 10, 15, 20, 25,
	30, 35, 40, 45, 50, 55, 60,
	65, 70, 1, 1,
}

// ShipType maps a ship's drive type to the suffix used in its name.
var ShipType = [3]string{"", "S", "S"}

// TechName holds the full display name for each tech discipline.
var TechName = [6]string{
	"Mining",
	"Manufacturing",
	"Military",
	"Gravitics",
	"Life Support",
	"Biology",
}

// GasString maps a gas code (0..13) to its display symbol; index 0 is blank.
var GasString = [14]string{
	"   ", "H2", "CH4", "He", "NH3", "N2", "CO2",
	"O2", "HCl", "Cl2", "F2", "H2O", "SO2", "H2S",
}
