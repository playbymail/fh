package game

// Port of planetio.c. Reads and writes the binary file "planets.dat".
//
// Skipped (JSON exporter): planetDataAsJson. The planetDataAsSExpr
// exporter is ported because galaxy.go and planet.go already call it.

import (
	"encoding/binary"
	"fmt"
	"os"
)

// binary_planet_data_t is 40 bytes:
//
//	offset  0: uint8 temperature_class
//	offset  1: uint8 pressure_class
//	offset  2: uint8 special
//	offset  3: uint8 reserved1
//	offset  4: uint8 gas[4]
//	offset  8: uint8 gas_percent[4]
//	offset 12: int16 reserved2
//	offset 14: int16 diameter
//	offset 16: int16 gravity
//	offset 18: int16 mining_difficulty
//	offset 20: int16 econ_efficiency
//	offset 22: int16 md_increase
//	offset 24: int32 message
//	offset 28: int32 reserved3
//	offset 32: int32 reserved4
//	offset 36: int32 reserved5
const binary_planet_data_size = 40

// encodePlanetData translates one planet record into its on-disk form.
func encodePlanetData(data []byte, p *planet_data_t) {
	for i := range data[:binary_planet_data_size] {
		data[i] = 0
	}
	data[0] = byte(p.temperature_class)
	data[1] = byte(p.pressure_class)
	data[2] = byte(p.special)
	// reserved1 at 3 stays zero
	for g := 0; g < 4; g++ {
		data[4+g] = byte(p.gas[g])
		data[8+g] = byte(p.gas_percent[g])
	}
	// reserved2 at 12 stays zero
	binary.LittleEndian.PutUint16(data[14:], uint16(int16(p.diameter)))
	binary.LittleEndian.PutUint16(data[16:], uint16(int16(p.gravity)))
	binary.LittleEndian.PutUint16(data[18:], uint16(int16(p.mining_difficulty)))
	binary.LittleEndian.PutUint16(data[20:], uint16(int16(p.econ_efficiency)))
	binary.LittleEndian.PutUint16(data[22:], uint16(int16(p.md_increase)))
	binary.LittleEndian.PutUint32(data[24:], uint32(int32(p.message)))
	// reserved3 at 28, reserved4 at 32, reserved5 at 36 stay zero
}

// decodePlanetData translates one on-disk record into a planet record.
func decodePlanetData(p *planet_data_t, data []byte) {
	p.temperature_class = int(data[0])
	p.pressure_class = int(data[1])
	p.special = int(data[2])
	for g := 0; g < 4; g++ {
		p.gas[g] = int(data[4+g])
		p.gas_percent[g] = int(data[8+g])
	}
	p.diameter = int(int16(binary.LittleEndian.Uint16(data[14:])))
	p.gravity = int(int16(binary.LittleEndian.Uint16(data[16:])))
	p.mining_difficulty = int(int16(binary.LittleEndian.Uint16(data[18:])))
	p.econ_efficiency = int(int16(binary.LittleEndian.Uint16(data[20:])))
	p.md_increase = int(int16(binary.LittleEndian.Uint16(data[22:])))
	p.message = int(int32(binary.LittleEndian.Uint32(data[24:])))
}

func get_planet_data() {
	/* Open planet file. */
	data, err := os.ReadFile("planets.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get_planet_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot open file planets.dat!\n")
		os.Exit(255)
	}
	/* Read header data. */
	if len(data) < 4 {
		fmt.Fprintf(os.Stderr, "\n\tCannot read num_planets in file 'planets.dat'!\n\n")
		os.Exit(255)
	}
	numPlanets := int(int32(binary.LittleEndian.Uint32(data[0:])))
	/* Read it all into memory. */
	if len(data) < 4+numPlanets*binary_planet_data_size {
		fmt.Fprintf(os.Stderr, "\nCannot read planet file into memory!\n\n")
		os.Exit(255)
	}

	num_planets = numPlanets
	planet_base = make([]*planet_data_t, num_planets)

	for i := 0; i < num_planets; i++ {
		p := &planet_data_t{}
		planet_base[i] = p
		decodePlanetData(p, data[4+i*binary_planet_data_size:])

		// mdhender: added fields to help clean up code
		p.id = i + 1
		p.index = i
	}

	// mdhender: added fields to help clean up code
	for sn := 0; sn < num_stars; sn++ {
		star := star_base[sn]
		for pn := 0; pn < star.num_planets; pn++ {
			p := planet_base[star.planet_index+pn]
			p.star = star
			p.orbit = pn + 1
		}
	}

	planet_data_modified = FALSE
}

// getPlanetData returns the planet data
func getPlanetData(extraRecords int, filename string) []*planet_data_t {
	// open binary input file
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getPlanetData: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot open file '%s'!\n", filename)
		os.Exit(255)
	}

	// read header data, which is just the number of records in the file
	if len(data) < 4 {
		fmt.Fprintf(os.Stderr, "\n\tCannot read num_planets in file '%s'!\n\n", filename)
		os.Exit(255)
	}
	numRecords := int(int32(binary.LittleEndian.Uint32(data[0:])))

	// read all records into memory
	if len(data) < 4+numRecords*binary_planet_data_size {
		fmt.Fprintf(os.Stderr, "\nCannot read planet file '%s' into memory!\n\n", filename)
		os.Exit(255)
	}

	// allocate memory for the translated records plus extra records plus the sentinel record
	if extraRecords < 0 {
		extraRecords = 0
	}
	planetBase := make([]*planet_data_t, numRecords+extraRecords+1)
	for i := range planetBase {
		planetBase[i] = &planet_data_t{}
	}

	// translate from the raw input record into the application record
	for i := 0; i < numRecords; i++ {
		p := planetBase[i]
		decodePlanetData(p, data[4+i*binary_planet_data_size:])
		p.isValid = FALSE
	}

	return planetBase
}

// planetDataAsSExpr writes the current planet_base array to a text file as an s-expression.
func planetDataAsSExpr(planetBase []*planet_data_t, numPlanets int, fp *os.File) {
	fmt.Fprintf(fp, "(planets")
	for i := 0; i < numPlanets; i++ {
		p := planetBase[i]
		fmt.Fprintf(fp,
			"\n  (planet (id %5d) (diameter %3d) (gravity %2d.%02d) (temperature_class %3d) (pressure_class %3d) (special %2d) (gases (%2d %3d) (%2d %3d) (%2d %3d) (%2d %3d)) (mining_difficulty %3d.%02d %3d) (econ_efficiency %3d) (message %d))",
			i+1,
			p.diameter,
			p.gravity/100, p.gravity%100,
			p.temperature_class, p.pressure_class, p.special,
			p.gas[0], p.gas_percent[0],
			p.gas[1], p.gas_percent[1],
			p.gas[2], p.gas_percent[2],
			p.gas[3], p.gas_percent[3],
			p.mining_difficulty/100, p.mining_difficulty%100,
			p.md_increase, p.econ_efficiency, p.message)
	}
	fmt.Fprintf(fp, ")\n")
}

func save_planet_data() {
	numPlanets := num_planets
	buf := make([]byte, 4+numPlanets*binary_planet_data_size)
	/* Write header data. */
	binary.LittleEndian.PutUint32(buf[0:], uint32(int32(numPlanets)))
	for i := 0; i < num_planets; i++ {
		encodePlanetData(buf[4+i*binary_planet_data_size:], planet_base[i])
	}

	/* Open planet file for writing and write planet data to disk. */
	if err := os.WriteFile("planets.dat", buf, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "save_planet_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create file 'planets.dat'!\n")
		os.Exit(255)
	}

	planet_data_modified = FALSE
}

func savePlanetData(planetBase []*planet_data_t, numPlanets int, filename string) {
	numRecords := numPlanets
	buf := make([]byte, 4+numRecords*binary_planet_data_size)
	/* Write header data. */
	binary.LittleEndian.PutUint32(buf[0:], uint32(int32(numRecords)))
	for i := 0; i < numPlanets; i++ {
		encodePlanetData(buf[4+i*binary_planet_data_size:], planetBase[i])
	}

	/* Open planet file for writing and write planet data to disk. */
	if err := os.WriteFile(filename, buf, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "savePlanetData: %v\n", err)
		fmt.Fprintf(os.Stderr, "error: cannot create file '%s'!\n", filename)
		os.Exit(2)
	}
}
