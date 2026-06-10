package game

// Round-trip tests for the binary .dat codecs (galaxyio.go, stario.go,
// planetio.go, speciesio.go, namplaio.go, shipio.go, locationio.go,
// transactionio.go).
//
// NOTE: tests in this package mutate package-level globals and must NOT
// run in parallel (no t.Parallel()). Tests that touch globals call
// ResetState() first.

import (
	"os"
	"reflect"
	"testing"
)

// TestRoundTripRecordSizes verifies the encoded record sizes match the
// sizes of the binary_*_data_t structs as compiled by the C engine on
// x86-64 (measured with sizeof).
func TestRoundTripRecordSizes(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"binary_galaxy_data_t", binary_galaxy_data_size, 16},
		{"binary_star_data_t", binary_star_data_size, 52},
		{"binary_planet_data_t", binary_planet_data_size, 40},
		{"binary_nampla_data_t", binary_nampla_data_size, 288},
		{"binary_ship_data_t", binary_ship_data_size, 172},
		{"binary_species_data_t", binary_species_data_size, 264},
		{"locationio binary record", binary_location_data_size, 4},
		{"transactionio binary record", binary_transaction_data_size, 148},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: record size %d, want %d", tc.name, tc.got, tc.want)
		}
	}
}

func TestRoundTripStar(t *testing.T) {
	want := star_data_t{
		x: 11, y: 22, z: 33,
		star_type:   MAIN_SEQUENCE,
		color:       ORANGE,
		size:        7,
		num_planets: 9,
		home_system: TRUE,
		worm_here:   TRUE,
		worm_x:      44, worm_y: 55, worm_z: 66,
		planet_index: 1234,
		message:      987654,
		visited_by:   [NUM_CONTACT_WORDS]uint32{0xdeadbeef, 0x01020304, 0xffffffff, 0x80000001},
	}
	buf := make([]byte, binary_star_data_size)
	encodeStarData(buf, &want)
	var got star_data_t
	decodeStarData(&got, buf)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("star round trip:\n got %+v\nwant %+v", got, want)
	}
}

func TestRoundTripPlanet(t *testing.T) {
	want := planet_data_t{
		temperature_class: 23,
		pressure_class:    11,
		special:           2,
		gas:               [4]int{O2, N2, CO2, HE},
		gas_percent:       [4]int{21, 70, 5, 4},
		diameter:          12,
		gravity:           98,
		mining_difficulty: 124,
		econ_efficiency:   100,
		md_increase:       3,
		message:           -42,
	}
	buf := make([]byte, binary_planet_data_size)
	encodePlanetData(buf, &want)
	var got planet_data_t
	decodePlanetData(&got, buf)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("planet round trip:\n got %+v\nwant %+v", got, want)
	}
}

func TestRoundTripNampla(t *testing.T) {
	want := nampla_data_t{
		name: "Earth Prime Colony",
		x:    12, y: 34, z: 56, pn: 3,
		status:         HOME_PLANET | POPULATED,
		hiding:         TRUE,
		hidden:         FALSE,
		planet_index:   777,
		siege_eff:      45,
		shipyards:      6,
		IUs_needed:     100,
		AUs_needed:     200,
		auto_IUs:       300,
		auto_AUs:       400,
		IUs_to_install: 500,
		AUs_to_install: 600,
		mi_base:        7000,
		ma_base:        8000,
		pop_units:      9000,
		use_on_ambush:  111,
		message:        222,
		special:        -333,
	}
	for j := 0; j < MAX_ITEMS; j++ {
		want.item_quantity[j] = j*100 + 1
	}
	buf := make([]byte, binary_nampla_data_size)
	encodeNamplaData(buf, &want)
	var got nampla_data_t
	decodeNamplaData(&got, buf)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nampla round trip:\n got %+v\nwant %+v", got, want)
	}
}

func TestRoundTripShip(t *testing.T) {
	want := ship_data_t{
		name: "ISS Distinctive",
		x:    9, y: 8, z: 7, pn: 2,
		status:    IN_ORBIT,
		ship_type: FTL,
		dest_x:    101, dest_y: 102, dest_z: 103,
		just_jumped:          TRUE,
		arrived_via_wormhole: TRUE,
		class:                TR,
		tonnage:              25,
		age:                  -12,
		remaining_cost:       4321,
		loading_point:        9999,
		unloading_point:      17,
		special:              -99999,
	}
	for j := 0; j < MAX_ITEMS; j++ {
		want.item_quantity[j] = j*3 + 1
	}
	buf := make([]byte, binary_ship_data_size)
	encodeShipData(buf, &want)
	var got ship_data_t
	decodeShipData(&got, buf)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ship round trip:\n got %+v\nwant %+v", got, want)
	}
}

func TestRoundTripSpecies(t *testing.T) {
	want := species_data_t{
		name:      "Distinctive Species",
		govt_name: "The Government",
		govt_type: "Democracy",
		x:         60, y: 61, z: 62, pn: 4,
		required_gas:       O2,
		required_gas_min:   12,
		required_gas_max:   60,
		neutral_gas:        [6]int{HE, N2, CO2, H2O, F2, NH3},
		poison_gas:         [6]int{H2S, SO2, CL2, HCL, CH4, H2},
		auto_orders:        TRUE,
		tech_level:         [6]int{10, 11, 12, 13, 14, 15},
		init_tech_level:    [6]int{9, 10, 11, 12, 13, 14},
		tech_knowledge:     [6]int{1, 2, 3, 4, 5, 6},
		num_namplas:        42,
		num_ships:          137,
		tech_eps:           [6]int{100, 200, 300, 400, 500, 600},
		hp_original_base:   4000,
		econ_units:         123456,
		fleet_cost:         7890,
		fleet_percent_cost: 250,
		contact:            [NUM_CONTACT_WORDS]uint32{1, 2, 3, 4},
		ally:               [NUM_CONTACT_WORDS]uint32{5, 6, 7, 8},
		enemy:              [NUM_CONTACT_WORDS]uint32{0xaaaaaaaa, 0x55555555, 0x12345678, 0x9abcdef0},
	}
	buf := make([]byte, binary_species_data_size)
	encodeSpeciesData(buf, &want)
	var got species_data_t
	decodeSpeciesData(&got, buf)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("species round trip:\n got %+v\nwant %+v", got, want)
	}
}

// chtmpdir switches the working directory to a temp dir for the test and
// restores it afterwards.
func chtmpdir(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRoundTripGalaxyFile(t *testing.T) {
	ResetState()
	chtmpdir(t)

	want := galaxy_data_t{
		d_num_species: 15,
		num_species:   12,
		radius:        20,
		turn_number:   37,
	}
	galaxy = want
	save_galaxy_data()

	fi, err := os.Stat("galaxy.dat")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() != 16 {
		t.Errorf("galaxy.dat size %d, want 16", fi.Size())
	}

	galaxy = galaxy_data_t{}
	get_galaxy_data()
	if !reflect.DeepEqual(galaxy, want) {
		t.Errorf("galaxy round trip:\n got %+v\nwant %+v", galaxy, want)
	}
}

func TestRoundTripLocationFile(t *testing.T) {
	ResetState()
	chtmpdir(t)

	num_locs = 3
	loc[0] = sp_loc_data_t{s: 1, x: 10, y: 11, z: 12}
	loc[1] = sp_loc_data_t{s: 2, x: 20, y: 21, z: 22}
	loc[2] = sp_loc_data_t{s: 99, x: 255, y: 0, z: 128}
	want := [3]sp_loc_data_t{loc[0], loc[1], loc[2]}
	save_location_data()

	fi, err := os.Stat("locations.dat")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() != int64(3*binary_location_data_size) {
		t.Errorf("locations.dat size %d, want %d", fi.Size(), 3*binary_location_data_size)
	}

	num_locs = 0
	loc = [MAX_LOCATIONS]sp_loc_data_t{}
	get_location_data()
	if num_locs != 3 {
		t.Fatalf("num_locs = %d, want 3", num_locs)
	}
	for i := range want {
		if loc[i] != want[i] {
			t.Errorf("loc[%d] = %+v, want %+v", i, loc[i], want[i])
		}
	}
}

func TestRoundTripTransactionFile(t *testing.T) {
	ResetState()
	chtmpdir(t)

	num_transactions = 2
	transaction[0] = trans_data_t{
		trans_type: EU_TRANSFER,
		donor:      3,
		recipient:  7,
		value:      123456,
		x:          1, y: 2, z: 3, pn: 4,
		number1: 11, name1: "First Name",
		number2: -22, name2: "Second Name",
		number3: 33, name3: "Third Name",
	}
	transaction[1] = trans_data_t{
		trans_type: MESSAGE_TO_SPECIES,
		donor:      9,
		recipient:  10,
		value:      -1,
		x:          200, y: 201, z: 202, pn: 0,
		number1: 0, name1: "",
		number2: 2147483647, name2: "Wxyz",
		number3: -2147483648, name3: "End",
	}
	want := [2]trans_data_t{transaction[0], transaction[1]}
	save_transaction_data()

	fi, err := os.Stat("interspecies.dat")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() != int64(2*binary_transaction_data_size) {
		t.Errorf("interspecies.dat size %d, want %d", fi.Size(), 2*binary_transaction_data_size)
	}

	num_transactions = 0
	transaction = [MAX_TRANSACTIONS]trans_data_t{}
	get_transaction_data()
	if num_transactions != 2 {
		t.Fatalf("num_transactions = %d, want 2", num_transactions)
	}
	for i := range want {
		if !reflect.DeepEqual(transaction[i], want[i]) {
			t.Errorf("transaction[%d]:\n got %+v\nwant %+v", i, transaction[i], want[i])
		}
	}
}

// TestRoundTripStarsFile exercises the stars.dat reader/writer including
// the wormhole linking done by get_star_data.
func TestRoundTripStarsFile(t *testing.T) {
	ResetState()
	chtmpdir(t)

	num_stars = 2
	a := &star_data_t{
		x: 1, y: 2, z: 3,
		star_type:   DWARF,
		color:       RED,
		size:        4,
		num_planets: 0,
		worm_here:   TRUE,
		worm_x:      5, worm_y: 6, worm_z: 7,
		planet_index: 0,
		message:      9,
		visited_by:   [NUM_CONTACT_WORDS]uint32{1, 0, 0, 0x80000000},
	}
	b := &star_data_t{
		x: 5, y: 6, z: 7,
		star_type:   GIANT,
		color:       BLUE,
		size:        9,
		num_planets: 0,
		worm_here:   TRUE,
		worm_x:      1, worm_y: 2, worm_z: 3,
	}
	star_base = []*star_data_t{a, b}
	save_star_data()

	fi, err := os.Stat("stars.dat")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() != int64(4+2*binary_star_data_size) {
		t.Errorf("stars.dat size %d, want %d", fi.Size(), 4+2*binary_star_data_size)
	}

	num_stars = 0
	star_base = nil
	get_star_data()
	if num_stars != 2 {
		t.Fatalf("num_stars = %d, want 2", num_stars)
	}
	got := star_base[0]
	if got.x != a.x || got.y != a.y || got.z != a.z || got.star_type != a.star_type ||
		got.color != a.color || got.size != a.size || got.worm_here != a.worm_here ||
		got.worm_x != a.worm_x || got.worm_y != a.worm_y || got.worm_z != a.worm_z ||
		got.planet_index != a.planet_index || got.message != a.message ||
		got.visited_by != a.visited_by {
		t.Errorf("star[0] round trip:\n got %+v\nwant %+v", got, a)
	}
	if got.id != 1 || got.index != 0 {
		t.Errorf("star[0] id/index = %d/%d, want 1/0", got.id, got.index)
	}
	if num_natural_wormholes != 1 {
		t.Errorf("num_natural_wormholes = %d, want 1", num_natural_wormholes)
	}
	if star_base[0].wormholeExit != star_base[1] || star_base[1].wormholeExit != star_base[0] {
		t.Errorf("wormhole exits not linked")
	}
}

// TestRoundTripPlanetsFile exercises the planets.dat reader/writer and
// the getPlanetData/savePlanetData helpers.
func TestRoundTripPlanetsFile(t *testing.T) {
	ResetState()
	chtmpdir(t)

	want := planet_data_t{
		temperature_class: 5,
		pressure_class:    6,
		special:           1,
		gas:               [4]int{N2, O2, 0, 0},
		gas_percent:       [4]int{78, 21, 0, 0},
		diameter:          13,
		gravity:           101,
		mining_difficulty: 85,
		econ_efficiency:   100,
		md_increase:       2,
		message:           7,
	}
	pb := []*planet_data_t{{}, {}}
	*pb[0] = want
	*pb[1] = planet_data_t{temperature_class: 30, message: -1}
	savePlanetData(pb, 2, "test_planets.dat")

	got := getPlanetData(3, "test_planets.dat")
	// 2 real records + 3 extra + 1 sentinel, matching the C allocation
	if len(got) != 2+3+1 {
		t.Fatalf("getPlanetData returned %d records, want %d", len(got), 2+3+1)
	}
	if !reflect.DeepEqual(*got[0], want) {
		t.Errorf("planet[0] round trip:\n got %+v\nwant %+v", *got[0], want)
	}
	for i := 2; i < len(got); i++ {
		if !reflect.DeepEqual(*got[i], planet_data_t{}) {
			t.Errorf("extra planet record %d not zero: %+v", i, *got[i])
		}
	}

	// also round trip through the global save_planet_data/get_planet_data
	num_planets = 2
	planet_base = pb
	save_planet_data()
	num_planets = 0
	planet_base = nil
	get_planet_data()
	if num_planets != 2 {
		t.Fatalf("num_planets = %d, want 2", num_planets)
	}
	gp := *planet_base[0]
	gp.id, gp.index = 0, 0 // assigned by get_planet_data
	if !reflect.DeepEqual(gp, want) {
		t.Errorf("planet round trip via planets.dat:\n got %+v\nwant %+v", gp, want)
	}
}
