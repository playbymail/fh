package game

// Port of speciesio.c. Reads and writes the binary files "sp##.dat".
// Each species file contains one species record followed by its nampla
// records (namplaio.go) and then its ship records (shipio.go).
//
// Skipped (JSON exporter): speciesDataAsJson. The s-expression exporter
// speciesDataAsSExpr (ported in create.go) is called from saveSpeciesData
// below, so the "species%03d.txt" snapshot the C saveSpeciesData writes is
// produced by this port.

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// binary_species_data_t is 264 bytes:
//
//	offset   0: uint8 name[32]
//	offset  32: uint8 govt_name[32]
//	offset  64: uint8 govt_type[32]
//	offset  96: uint8 x, y, z, pn
//	offset 100: uint8 required_gas, required_gas_min, required_gas_max, reserved5
//	offset 104: uint8 neutral_gas[6]
//	offset 110: uint8 poison_gas[6]
//	offset 116: uint8 auto_orders
//	offset 117: uint8 reserved3
//	offset 118: int16 reserved4
//	offset 120: int16 tech_level[6]
//	offset 132: int16 init_tech_level[6]
//	offset 144: int16 tech_knowledge[6]
//	offset 156: int32 num_namplas
//	offset 160: int32 num_ships
//	offset 164: int32 tech_eps[6]
//	offset 188: int32 hp_original_base
//	offset 192: int32 econ_units
//	offset 196: int32 fleet_cost
//	offset 200: int32 fleet_percent_cost
//	offset 204: uint32 contact[NUM_CONTACT_WORDS] (4 words)
//	offset 220: uint32 ally[NUM_CONTACT_WORDS]
//	offset 236: uint32 enemy[NUM_CONTACT_WORDS]
//	offset 252: uint8 padding[12]
const binary_species_data_size = 264

// encodeSpeciesData translates one species record into its on-disk form.
func encodeSpeciesData(data []byte, sp *species_data_t) {
	for i := range data[:binary_species_data_size] {
		data[i] = 0
	}
	copyName(data[0:32], sp.name)
	copyName(data[32:64], sp.govt_name)
	copyName(data[64:96], sp.govt_type)
	data[96] = byte(sp.x)
	data[97] = byte(sp.y)
	data[98] = byte(sp.z)
	data[99] = byte(sp.pn)
	data[100] = byte(sp.required_gas)
	data[101] = byte(sp.required_gas_min)
	data[102] = byte(sp.required_gas_max)
	// reserved5 at 103 stays zero
	for g := 0; g < 6; g++ {
		data[104+g] = byte(sp.neutral_gas[g])
		data[110+g] = byte(sp.poison_gas[g])
	}
	data[116] = byte(sp.auto_orders)
	// reserved3 at 117 and reserved4 at 118 stay zero
	for j := 0; j < 6; j++ {
		binary.LittleEndian.PutUint16(data[120+2*j:], uint16(int16(sp.tech_level[j])))
		binary.LittleEndian.PutUint16(data[132+2*j:], uint16(int16(sp.init_tech_level[j])))
		binary.LittleEndian.PutUint16(data[144+2*j:], uint16(int16(sp.tech_knowledge[j])))
		binary.LittleEndian.PutUint32(data[164+4*j:], uint32(int32(sp.tech_eps[j])))
	}
	binary.LittleEndian.PutUint32(data[156:], uint32(int32(sp.num_namplas)))
	binary.LittleEndian.PutUint32(data[160:], uint32(int32(sp.num_ships)))
	binary.LittleEndian.PutUint32(data[188:], uint32(int32(sp.hp_original_base)))
	binary.LittleEndian.PutUint32(data[192:], uint32(int32(sp.econ_units)))
	binary.LittleEndian.PutUint32(data[196:], uint32(int32(sp.fleet_cost)))
	binary.LittleEndian.PutUint32(data[200:], uint32(int32(sp.fleet_percent_cost)))
	for j := 0; j < NUM_CONTACT_WORDS; j++ {
		binary.LittleEndian.PutUint32(data[204+4*j:], sp.contact[j])
		binary.LittleEndian.PutUint32(data[220+4*j:], sp.ally[j])
		binary.LittleEndian.PutUint32(data[236+4*j:], sp.enemy[j])
	}
	// padding at 252 stays zero
}

// decodeSpeciesData translates one on-disk record into a species record.
func decodeSpeciesData(sp *species_data_t, data []byte) {
	sp.name = nameString(data[0:32])
	sp.govt_name = nameString(data[32:64])
	sp.govt_type = nameString(data[64:96])
	sp.x = int(data[96])
	sp.y = int(data[97])
	sp.z = int(data[98])
	sp.pn = int(data[99])
	sp.required_gas = int(data[100])
	sp.required_gas_min = int(data[101])
	sp.required_gas_max = int(data[102])
	for g := 0; g < 6; g++ {
		sp.neutral_gas[g] = int(data[104+g])
		sp.poison_gas[g] = int(data[110+g])
	}
	sp.auto_orders = int(data[116])
	for j := 0; j < 6; j++ {
		sp.tech_level[j] = int(int16(binary.LittleEndian.Uint16(data[120+2*j:])))
		sp.init_tech_level[j] = int(int16(binary.LittleEndian.Uint16(data[132+2*j:])))
		sp.tech_knowledge[j] = int(int16(binary.LittleEndian.Uint16(data[144+2*j:])))
		sp.tech_eps[j] = int(int32(binary.LittleEndian.Uint32(data[164+4*j:])))
	}
	sp.num_namplas = int(int32(binary.LittleEndian.Uint32(data[156:])))
	sp.num_ships = int(int32(binary.LittleEndian.Uint32(data[160:])))
	sp.hp_original_base = int(int32(binary.LittleEndian.Uint32(data[188:])))
	sp.econ_units = int(int32(binary.LittleEndian.Uint32(data[192:])))
	sp.fleet_cost = int(int32(binary.LittleEndian.Uint32(data[196:])))
	sp.fleet_percent_cost = int(int32(binary.LittleEndian.Uint32(data[200:])))
	for j := 0; j < NUM_CONTACT_WORDS; j++ {
		sp.contact[j] = binary.LittleEndian.Uint32(data[204+4*j:])
		sp.ally[j] = binary.LittleEndian.Uint32(data[220+4*j:])
		sp.enemy[j] = binary.LittleEndian.Uint32(data[236+4*j:])
	}
}

// get_species_data will read in data files for all species
func get_species_data() {
	for species_index := 0; species_index < galaxy.num_species; species_index++ {
		// clear out any existing species data
		spec_data[species_index] = species_data_t{}
		namp_data[species_index] = nil
		ship_data[species_index] = nil
		num_new_namplas[species_index] = 0
		num_new_ships[species_index] = 0
		data_modified[species_index] = FALSE
		data_in_memory[species_index] = FALSE
	}

	for species_index := 0; species_index < galaxy.num_species; species_index++ {
		sp := &spec_data[species_index]

		// get the filename for the species
		filename := fmt.Sprintf("sp%02d.dat", species_index+1)

		// see if it exists
		if _, err := os.Stat(filename); err != nil {
			sp.pn = 0 /* Extinct! */
			continue
		}

		/* Open the species data file. */
		fp, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "get_species_data: %v\n", err)
			continue
		}

		/* Read in species data. */
		data := make([]byte, binary_species_data_size)
		if _, err := io.ReadFull(fp, data); err != nil {
			fmt.Fprintf(os.Stderr, "get_species_data: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nCannot read species record in file '%s'!\n\n", filename)
			os.Exit(2)
		}

		// translate data
		decodeSpeciesData(sp, data)

		/* load nampla data from file and create empty slots for future use */
		namp_data[species_index] = get_nampla_data(sp.num_namplas, extra_namplas, fp)

		/* load ship data from file and create empty slots for future use */
		ship_data[species_index] = get_ship_data(sp.num_ships, extra_ships, fp)

		data_in_memory[species_index] = TRUE
		num_new_namplas[species_index] = 0
		num_new_ships[species_index] = 0

		// mdhender: added fields to help clean up code
		sp.id = species_index + 1
		sp.index = species_index
		sp.home.nampla = namp_data[species_index][0]
		sp.home.planet = sp.home.nampla.planet
		sp.home.star = sp.home.nampla.star

		fp.Close()
	}
}

// save_species_data will write all data that has been modified
func save_species_data() {
	for species_index := 0; species_index < galaxy.num_species; species_index++ {
		if data_in_memory[species_index] != FALSE && data_modified[species_index] != FALSE {
			// get the filename for the species
			filename := fmt.Sprintf("sp%02d.dat", species_index+1)

			/* Open the species data file. */
			fp, err := os.Create(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "save_species_data: %v\n", err)
				fmt.Fprintf(os.Stderr, "\n\tCannot create new version of file '%s'!\n", filename)
				os.Exit(2)
			}
			// save the species, colonies, and ship data
			saveSpeciesData(&spec_data[species_index], namp_data[species_index], ship_data[species_index], fp)
			// be kind and signal that it's been saved
			data_modified[species_index] = FALSE
			// closing the file is always nice
			fp.Close()
		}
	}
}

func saveSpeciesData(sp *species_data_t, colonies []*nampla_data_t, ships []*ship_data_t, fp *os.File) {
	// use a buffer to translate the data
	spData := make([]byte, binary_species_data_size)
	encodeSpeciesData(spData, sp)

	// save the translated data
	if _, err := fp.Write(spData); err != nil {
		fmt.Fprintf(os.Stderr, "saveSpeciesData: %v\n", err)
		fmt.Fprintf(os.Stderr, "error: cannot write species record to file\n")
		os.Exit(2)
	}
	// save colonies data
	save_nampla_data(colonies, sp.num_namplas, fp)
	// save ships data
	save_ship_data(ships, sp.num_ships, fp)

	// The C saveSpeciesData (speciesio.c) also writes a "species%03d.txt"
	// s-expression snapshot for every species record it saves.
	filename := fmt.Sprintf("species%03d.txt", sp.id)
	if f, err := os.Create(filename); err == nil {
		speciesDataAsSExpr(sp, f)
		f.Close()
	}
}
