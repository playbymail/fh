// Package fhjson reads and writes a model.World as the Far Horizons JSON
// export: galaxy.json, systems.json, species.%03d.json, the per-species event
// logs (sp%02d.log), and the locations index (locations.dat). This is the same
// well-defined layout the byte-faithful fhc engine produces, re-implemented
// here from model.World so neither the fh engine nor the JSON store depends on
// internal/game (the C port with its package globals).
//
// It is shared by the engine's fhc-import path (LoadWorldFromDir) and by the
// JSON store backend (internal/data/store).
package fhjson

import "github.com/playbymail/fh/internal/model"

// galaxyFile mirrors galaxy.json.
type galaxyFile struct {
	Version int        `json:"version"`
	Galaxy  galaxyJSON `json:"galaxy"`
}

type galaxyJSON struct {
	TurnNumber  int `json:"turn_number"`
	NumSpecies  int `json:"num_species"`
	DNumSpecies int `json:"d_num_species"`
	Radius      int `json:"radius"`
}

// systemsFile mirrors systems.json.
type systemsFile struct {
	Systems []systemJSON `json:"systems"`
}

type systemJSON struct {
	X          int          `json:"x"`
	Y          int          `json:"y"`
	Z          int          `json:"z"`
	Type       string       `json:"type"`  // single star-type char
	Color      string       `json:"color"` // single star-color char
	Size       int          `json:"size"`
	HomeSystem bool         `json:"home_system"`
	WormHere   bool         `json:"worm_here"`
	WormX      int          `json:"worm_x"`
	WormY      int          `json:"worm_y"`
	WormZ      int          `json:"worm_z"`
	VisitedBy  []int        `json:"visited_by"` // 1-based species numbers
	Message    int          `json:"message"`
	Planets    []planetJSON `json:"planets"`
}

type planetJSON struct {
	TemperatureClass int       `json:"temperature_class"`
	PressureClass    int       `json:"pressure_class"`
	Special          int       `json:"special"`
	Atmosphere       []gasJSON `json:"atmosphere"`
	Diameter         int       `json:"diameter"`
	Gravity          int       `json:"gravity"`
	MiningDifficulty struct {
		Base     int `json:"base"`
		Increase int `json:"increase"`
	} `json:"mining_difficulty"`
	EconEfficiency int `json:"econ_efficiency"`
	Message        int `json:"message"`
}

type gasJSON struct {
	Code    int `json:"code"`
	Percent int `json:"percent"`
}

// speciesFile mirrors species.%03d.json.
type speciesFile struct {
	Species      speciesJSON  `json:"species"`
	NamedPlanets []namplaJSON `json:"named_planets"`
	Ships        []shipJSON   `json:"ships"`
}

type speciesJSON struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Government struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"government"`
	HomeWorld struct {
		X     int `json:"x"`
		Y     int `json:"y"`
		Z     int `json:"z"`
		Orbit int `json:"orbit"`
	} `json:"home_world"`
	Atmosphere struct {
		Required struct {
			Code   int `json:"code"`
			MinPct int `json:"min_pct"`
			MaxPct int `json:"max_pct"`
		} `json:"required"`
		Neutral []int `json:"neutral"`
		Poison  []int `json:"poison"`
	} `json:"atmosphere"`
	AutoOrders       bool                `json:"auto_orders"`
	Tech             map[string]techJSON `json:"tech"`
	HpOriginalBase   int                 `json:"hp_original_base"`
	EconUnits        int                 `json:"econ_units"`
	FleetCost        int                 `json:"fleet_cost"`
	FleetPercentCost int                 `json:"fleet_percent_cost"`
	Contacts         []int               `json:"contacts"`
	Allies           []int               `json:"allies"`
	Enemies          []int               `json:"enemies"`
}

type techJSON struct {
	Level     int `json:"level"`
	Knowledge int `json:"knowledge"`
	XP        int `json:"xp"`
	InitLevel int `json:"init_level"`
}

type coordOrbitJSON struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Z     int `json:"z"`
	Orbit int `json:"orbit"`
}

type coordJSON struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

type itemJSON struct {
	Code int `json:"code"`
	Qty  int `json:"qty"`
}

type consJSON struct {
	Needed  int `json:"needed"`
	Auto    int `json:"auto"`
	Install int `json:"install"`
}

type namplaJSON struct {
	Name        string         `json:"name"`
	Location    coordOrbitJSON `json:"location"`
	Status      int            `json:"status"`
	Hiding      int            `json:"hiding"`
	Hidden      int            `json:"hidden"`
	SiegeEff    int            `json:"siege_eff"`
	Shipyards   int            `json:"shipyards"`
	IUs         consJSON       `json:"ius"`
	AUs         consJSON       `json:"aus"`
	MiBase      int            `json:"mi_base"`
	MaBase      int            `json:"ma_base"`
	PopUnits    int            `json:"pop_units"`
	Items       []itemJSON     `json:"items"`
	UseOnAmbush int            `json:"use_on_ambush"`
	Message     int            `json:"message"`
	Special     int            `json:"special"`
}

type shipJSON struct {
	Name               string         `json:"name"`
	Location           coordOrbitJSON `json:"location"`
	Status             int            `json:"status"`
	Type               int            `json:"type"`
	Dest               coordJSON      `json:"dest"`
	JustJumped         bool           `json:"just_jumped"`
	ArrivedViaWormhole bool           `json:"arrived_via_wormhole"`
	Class              int            `json:"class"`
	Tonnage            int            `json:"tonnage"`
	Cargo              []itemJSON     `json:"cargo"`
	Age                int            `json:"age"`
	RemainingCost      int            `json:"remaining_cost"`
	LoadingPoint       int            `json:"loading_point"`
	UnloadingPoint     int            `json:"unloading_point"`
	Special            int            `json:"special"`
}

// techKeys maps fhc's tech-discipline keys to model tech indices.
var techKeys = map[string]int{
	"MI": model.MI, "MA": model.MA, "ML": model.ML,
	"GV": model.GV, "LS": model.LS, "BI": model.BI,
}

// techKeyByIndex is the inverse of techKeys, used when writing the tech object.
var techKeyByIndex = [6]string{
	model.MI: "MI", model.MA: "MA", model.ML: "ML",
	model.GV: "GV", model.LS: "LS", model.BI: "BI",
}

// Star type/color characters, mirroring internal/game's type_char and
// color_char tables (tables.go). type_char has a duplicate space at indices 0
// and 3, so the char encoding is not perfectly invertible for star type 3;
// type and color are informational for the renderer, so this does not affect
// report output.
var typeChars = []byte(" dD g")     // index 0..4
var colorChars = []byte(" OBAFGKM") // index 0..7

func typeToChar(c int) string {
	if c >= 0 && c < len(typeChars) {
		return string(rune(typeChars[c]))
	}
	return "?"
}

func colorToChar(c int) string {
	if c >= 0 && c < len(colorChars) {
		return string(rune(colorChars[c]))
	}
	return "?"
}

func charToType(s string) int  { return charIndex(typeChars, s) }
func charToColor(s string) int { return charIndex(colorChars, s) }

func charIndex(table []byte, s string) int {
	if s == "" {
		return 0
	}
	b := s[0]
	for i, c := range table {
		if c == b {
			return i
		}
	}
	return 0
}

// bitfieldToList expands a contact/ally/enemy/visited_by bitfield into the
// sorted list of 1-based species numbers whose bit is set.
func bitfieldToList(bits [model.NumContactWords]uint32) []int {
	out := make([]int, 0)
	for sp := 1; sp <= model.MaxSpecies; sp++ {
		if model.ContactBit(bits, sp-1) {
			out = append(out, sp)
		}
	}
	return out
}
