package game

// Port of stario.c. Reads and writes the binary file "stars.dat".
//
// The starDataAsSExpr exporter is ported because galaxy.go already
// calls it.

import (
	"encoding/binary"
	"fmt"
	"os"
)

// binary_star_data_t is 52 bytes:
//
//	offset  0: uint8 x, y, z, type, color, size, num_planets,
//	           home_system, worm_here, worm_x, worm_y, worm_z
//	offset 12: int16 reserved1
//	offset 14: int16 reserved2
//	offset 16: int16 planet_index
//	offset 18: 2 bytes alignment padding
//	offset 20: int32 message
//	offset 24: uint32 visited_by[NUM_CONTACT_WORDS] (4 words)
//	offset 40: int32 reserved3
//	offset 44: int32 reserved4
//	offset 48: int32 reserved5
const binary_star_data_size = 52

// encodeStarData translates one star record into its on-disk form.
func encodeStarData(data []byte, s *star_data_t) {
	for i := range data[:binary_star_data_size] {
		data[i] = 0
	}
	data[0] = byte(s.x)
	data[1] = byte(s.y)
	data[2] = byte(s.z)
	data[3] = byte(s.star_type)
	data[4] = byte(s.color)
	data[5] = byte(s.size)
	data[6] = byte(s.num_planets)
	data[7] = byte(s.home_system)
	data[8] = byte(s.worm_here)
	data[9] = byte(s.worm_x)
	data[10] = byte(s.worm_y)
	data[11] = byte(s.worm_z)
	// reserved1 at 12, reserved2 at 14 stay zero
	binary.LittleEndian.PutUint16(data[16:], uint16(int16(s.planet_index)))
	// alignment padding at 18
	binary.LittleEndian.PutUint32(data[20:], uint32(int32(s.message)))
	for j := 0; j < NUM_CONTACT_WORDS; j++ {
		binary.LittleEndian.PutUint32(data[24+4*j:], s.visited_by[j])
	}
	// reserved3 at 40, reserved4 at 44, reserved5 at 48 stay zero
}

// decodeStarData translates one on-disk record into a star record.
func decodeStarData(s *star_data_t, data []byte) {
	s.x = int(data[0])
	s.y = int(data[1])
	s.z = int(data[2])
	s.star_type = int(data[3])
	s.color = int(data[4])
	s.size = int(data[5])
	s.num_planets = int(data[6])
	s.home_system = int(data[7])
	s.worm_here = int(data[8])
	s.worm_x = int(data[9])
	s.worm_y = int(data[10])
	s.worm_z = int(data[11])
	s.planet_index = int(int16(binary.LittleEndian.Uint16(data[16:])))
	s.message = int(int32(binary.LittleEndian.Uint32(data[20:])))
	for j := 0; j < NUM_CONTACT_WORDS; j++ {
		s.visited_by[j] = binary.LittleEndian.Uint32(data[24+4*j:])
	}
}

func get_star_data() {
	/* Open star file. */
	data, err := os.ReadFile("stars.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get_star_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot open file stars.dat!\n")
		os.Exit(255)
	}
	/* Read header data. */
	if len(data) < 4 {
		fmt.Fprintf(os.Stderr, "\n\tCannot read num_stars in file 'stars.dat'!\n\n")
		os.Exit(255)
	}
	numStars := int(int32(binary.LittleEndian.Uint32(data[0:])))

	num_stars = numStars
	star_base = make([]*star_data_t, num_stars)

	/* Read it all into memory. */
	if len(data) < 4+numStars*binary_star_data_size {
		fmt.Fprintf(os.Stderr, "\nCannot read star file into memory!\n\n")
		os.Exit(255)
	}

	// translate the data
	for i := 0; i < num_stars; i++ {
		s := &star_data_t{}
		star_base[i] = s
		decodeStarData(s, data[4+i*binary_star_data_size:])
		// mdhender: added to help clean up code
		s.id = i + 1
		s.index = i
		s.wormholeExit = nil
	}

	// link wormholes
	num_natural_wormholes = 0
	for i := 0; i < num_stars; i++ {
		s := star_base[i]
		if s.worm_here != FALSE && s.wormholeExit == nil {
			num_natural_wormholes++
			for w := 0; w < num_stars; w++ {
				p := star_base[w]
				if p.x == s.worm_x && p.y == s.worm_y && p.z == s.worm_z {
					s.wormholeExit = p
					p.wormholeExit = s
					break
				}
			}
		}
	}

	star_data_modified = FALSE
}

func save_star_data() {
	// open star file for writing
	fp, err := os.Create("stars.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "save_star_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create file 'stars.dat'!\n")
		os.Exit(255)
	}
	saveStarData(star_base, num_stars, fp)
	fp.Close()

	star_data_modified = FALSE
}

// caller should update `star_data_modified = FALSE` if they care to.
func saveStarData(starBase []*star_data_t, numStars int, fp *os.File) {
	if fp == nil {
		fmt.Fprintf(os.Stderr, "error: saveStarData: internal error: passed null file pointer\n")
		os.Exit(2)
	}
	buf := make([]byte, 4+numStars*binary_star_data_size)

	// write header data
	binary.LittleEndian.PutUint32(buf[0:], uint32(int32(numStars)))

	// translate the data
	for i := 0; i < numStars; i++ {
		encodeStarData(buf[4+i*binary_star_data_size:], starBase[i])
	}

	// write header and records
	if _, err := fp.Write(buf); err != nil {
		fmt.Fprintf(os.Stderr, "saveStarData: %v\n", err)
		fmt.Fprintf(os.Stderr, "error: cannot write stars data to file\n")
		os.Exit(2)
	}
}

func starDataAsSExpr(starBase []*star_data_t, numStars int, fp *os.File) {
	fmt.Fprintf(fp, "(stars")
	for i := 0; i < numStars; i++ {
		s := starBase[i]
		fmt.Fprintf(fp, "\n  (star (id %4d) (x %3d) (y %3d) (z %3d) (type '%c') (color '%c') (size '%c')",
			i+1, s.x, s.y, s.z,
			star_type(s.star_type), star_color(s.color), star_size(s.size))
		fmt.Fprintf(fp, "\n        (planets")
		for p := 0; p < s.num_planets; p++ {
			fmt.Fprintf(fp, " %4d", s.planet_index+p+1)
		}
		fmt.Fprintf(fp, ") (home_system %s)", sexprBool(s.home_system))
		fmt.Fprintf(fp, "\n        (wormhole (here %-5s) (exit_x %3d) (exit_y %3d) (exit_z %3d))",
			sexprBool(s.worm_here), s.worm_x, s.worm_y, s.worm_z)
		fmt.Fprintf(fp, "\n        (visited_by")
		for spidx := 0; spidx < galaxy.num_species; spidx++ {
			// write the species only if it has visited this system
			if (s.visited_by[spidx/32] & (1 << (spidx % 32))) != 0 {
				fmt.Fprintf(fp, " %3d", spidx+1)
			}
		}
		fmt.Fprintf(fp, ")")
		fmt.Fprintf(fp, "\n        (message %d))", s.message)
	}
	fmt.Fprintf(fp, ")\n")
}

// sexprBool renders a C int flag the way the C exporters do.
func sexprBool(flag int) string {
	if flag != FALSE {
		return "true"
	}
	return "false"
}
