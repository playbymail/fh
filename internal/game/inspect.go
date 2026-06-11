package game

import (
	"fmt"
)

// Hardcoded C ABI sizes for the in-memory C structs and primitive types, as
// reported by `fh inspect` on the reference build (../Far-Horizons, v7.5.11,
// the platform that generates the golden data). These have no faithful Go
// equivalent: the Go ports of galaxy_data_t, star_data_t, etc. have a
// different memory layout, so their unsafe.Sizeof would not match the C
// engine. Per the maintainer's decision (option a), inspect reproduces the C
// output byte-for-byte by encoding the reference platform's C ABI here.
//
// These are parity-by-constant, not parity-by-computation: if the C engine is
// rebuilt on a platform with a different ABI (e.g. 32-bit long), these values —
// and the golden reference — would change. Revisit alongside the
// ../Far-Horizons target version bump.
const (
	cSizeofInt         = 4
	cSizeofLong        = 8
	cSizeofGalaxyData  = 16
	cSizeofStarData    = 168
	cSizeofPlanetData  = 96
	cSizeofSpeciesData = 376
	cSizeofNamplaData  = 296
	cSizeofShipData    = 260
	cSizeofUint16      = 2
	cSizeofUint32      = 4
	cSizeofUint64      = 8
)

// inspectCommand is a faithful port of the inline `inspect` branch in fh.c's
// dispatcher. It prints a fixed table of C sizeof() values and returns 0; it
// ignores its arguments, exactly as the C code does.
//
// The in-memory struct/primitive rows are the hardcoded reference-platform C
// ABI sizes (see the constants above). The binary_*_data_t rows are sourced
// from the Go on-disk record-size constants, which are verified to equal the C
// binary_*_data_t sizes by io_roundtrip_test.go — so those rows are
// parity-by-computation.
func inspectCommand(args []string) int {
	fmt.Printf("inspect: sizeof(int)                   == %5d\n", cSizeofInt)
	fmt.Printf("inspect: sizeof(long)                  == %5d\n", cSizeofLong)
	fmt.Printf("inspect: sizeof(galaxy_data_t)         == %5d\n", cSizeofGalaxyData)
	fmt.Printf("inspect: sizeof(star_data_t)           == %5d\n", cSizeofStarData)
	fmt.Printf("inspect: sizeof(planet_data_t)         == %5d\n", cSizeofPlanetData)
	fmt.Printf("inspect: sizeof(species_data_t)        == %5d\n", cSizeofSpeciesData)
	fmt.Printf("inspect: sizeof(nampla_data_t)         == %5d\n", cSizeofNamplaData)
	fmt.Printf("inspect: sizeof(ship_data_t)           == %5d\n", cSizeofShipData)
	fmt.Printf("inspect: sizeof(uint16_t)              == %5d\n", cSizeofUint16)
	fmt.Printf("inspect: sizeof(uint32_t)              == %5d\n", cSizeofUint32)
	fmt.Printf("inspect: sizeof(uint64_t)              == %5d\n", cSizeofUint64)
	fmt.Printf("inspect: sizeof(binary_galaxy_data_t)  == %5d\n", binary_galaxy_data_size)
	fmt.Printf("inspect: sizeof(binary_star_data_t)    == %5d\n", binary_star_data_size)
	fmt.Printf("inspect: sizeof(binary_planet_data_t)  == %5d\n", binary_planet_data_size)
	fmt.Printf("inspect: sizeof(binary_species_data_t) == %5d\n", binary_species_data_size)
	fmt.Printf("inspect: sizeof(binary_nampla_data_t)  == %5d\n", binary_nampla_data_size)
	fmt.Printf("inspect: sizeof(binary_ship_data_t)    == %5d\n", binary_ship_data_size)
	return 0
}
