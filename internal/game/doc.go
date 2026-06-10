// Package game is a faithful port of the Far Horizons game engine from C
// (github.com/playbymail/Far-Horizons, src/*.c) to Go.
//
// Porting conventions:
//
//   - Function, variable, and type names keep their original C spelling
//     (snake_case and all) so the Go code can be diffed against the C
//     source while validating the port. Idiomatic renames can happen once
//     the port is verified against the C engine.
//   - The C engine is single threaded and uses module-level globals; this
//     package mirrors that with package-level variables (see vars.go).
//     ResetState restores them to their zero values for tests.
//   - C pointer arithmetic over contiguous arrays (for example
//     "nampla = nampla_base - 1; nampla++") is translated to slice
//     indexing.
//   - All randomness flows through rnd (prng.go), which reproduces the
//     C engine's "Algorithm M" generator exactly, including seeding from
//     the FH_SEED environment variable, so a fixed seed produces the same
//     galaxy and turn results as the C engine.
//   - Binary data files (galaxy.dat, stars.dat, planets.dat, sp##.dat,
//     locations.dat, interspecies.dat) keep the exact on-disk layout of
//     the C structs in data.h (little endian) so games can move between
//     the C and Go engines.
package game
