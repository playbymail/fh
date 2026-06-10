package game

// Port of galaxyio.c. Reads and writes the binary file "galaxy.dat".
//
// The on-disk record layouts in this file (and the other *io.go files)
// mirror the binary_*_data_t structs in ../Far-Horizons/src/data.h with
// native x86-64 alignment, little-endian. Encoders write explicit zero
// bytes for reserved fields and alignment padding so the output is
// byte-identical to the C engine's files.
//
// Skipped (JSON exporter): galaxyDataAsJson. The galaxyDataAsSexpr
// exporter is ported because galaxy.go already calls it.

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Shared helpers for the binary .dat codecs.

// nameString decodes a NUL-padded fixed byte array into a Go string,
// truncating at the first NUL (C string semantics).
func nameString(b []byte) string {
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// copyName encodes a Go string into a NUL-padded fixed byte array,
// following the C engine's zstrcpy semantics: the destination is
// zero-filled and the final byte is always NUL.
func copyName(dst []byte, name string) {
	for i := range dst {
		dst[i] = 0
	}
	n := len(name)
	if n > len(dst)-1 {
		n = len(dst) - 1
	}
	copy(dst, name[:n])
}

func galaxyDataAsSexpr(fp *os.File) {
	fmt.Fprintf(fp, "(galaxy (turn %13d)\n        (num_species %6d)\n        (d_num_species %4d)\n        (radius %11d))\n",
		galaxy.turn_number, galaxy.num_species, galaxy.d_num_species, galaxy.radius)
}

// binary_galaxy_data_t is 16 bytes:
//
//	offset  0: int32 d_num_species
//	offset  4: int32 num_species
//	offset  8: int32 radius
//	offset 12: int32 turn_number
const binary_galaxy_data_size = 16

func get_galaxy_data() {
	data, err := os.ReadFile("galaxy.dat")
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n\tCannot open file galaxy.dat!\n")
		os.Exit(255)
	}
	if len(data) < binary_galaxy_data_size {
		fmt.Fprintf(os.Stderr, "\n\tCannot read data in file 'galaxy.dat'!\n\n")
		os.Exit(255)
	}
	galaxy.d_num_species = int(int32(binary.LittleEndian.Uint32(data[0:])))
	galaxy.num_species = int(int32(binary.LittleEndian.Uint32(data[4:])))
	galaxy.radius = int(int32(binary.LittleEndian.Uint32(data[8:])))
	galaxy.turn_number = int(int32(binary.LittleEndian.Uint32(data[12:])))
}

func save_galaxy_data() {
	data := make([]byte, binary_galaxy_data_size)
	binary.LittleEndian.PutUint32(data[0:], uint32(int32(galaxy.d_num_species)))
	binary.LittleEndian.PutUint32(data[4:], uint32(int32(galaxy.num_species)))
	binary.LittleEndian.PutUint32(data[8:], uint32(int32(galaxy.radius)))
	binary.LittleEndian.PutUint32(data[12:], uint32(int32(galaxy.turn_number)))
	if err := os.WriteFile("galaxy.dat", data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "save_galaxy_data: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n\tCannot create new version of file 'galaxy.dat'!\n")
		os.Exit(255)
	}
}
