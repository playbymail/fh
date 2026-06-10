package game

import "os"

// Port of prng.c — the C engine's "Algorithm M" pseudo-random number
// generator, a combination of the congruential and shift-register
// methods. The sequence must match the C engine exactly for a given
// seed so that galaxy creation and turn processing are reproducible
// across both engines.

var prngSeed uint64

// rnd returns a random int between 1 and max, inclusive.
func rnd(max int) int {
	if prngSeed == 0 {
		if envSeed := os.Getenv("FH_SEED"); envSeed != "" {
			for _, ch := range envSeed {
				if '0' <= ch && ch <= '9' {
					prngSeed = prngSeed*10 + uint64(ch-'0')
				}
			}
		}
		if prngSeed == 0 {
			prngSeed = defaultHistoricalSeedValue
		}
	}

	/* For congruential method, multiply previous value by the prime number 16417. */
	congResult := prngSeed + (prngSeed << 5) + (prngSeed << 14) /* Effectively multiply by 16417. */

	/* For shift-register method, use shift-right 15 and shift-left 17 with no-carry addition (i.e., exclusive-or). */
	shiftResult := (prngSeed >> 15) ^ prngSeed
	shiftResult ^= shiftResult << 17

	prngSeed = congResult ^ shiftResult

	return int(((prngSeed&0x0000FFFF)*uint64(max))>>16) + 1
}

// prngGetSeed returns the current seed value.
func prngGetSeed() uint64 {
	return prngSeed
}

// prngSetSeed sets the seed for the generator. A value of zero causes the
// next call to rnd to re-seed from the FH_SEED environment variable or
// the historical default.
func prngSetSeed(seed uint64) {
	prngSeed = seed
}
