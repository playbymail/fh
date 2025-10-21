package rng

// AlgorithmM implements the Scoped interface using Algorithm M from Far Horizons.
type AlgorithmM struct {
	seed uint64
}

// NewAlgorithmM creates a new AlgorithmM RNG with the given seed.
func NewAlgorithmM(seed uint64) Scoped {
	return &AlgorithmM{seed: seed}
}

// Uint64 returns the next random uint64.
func (a *AlgorithmM) Uint64() uint64 {
	// Algorithm M: combination of congruential and shift-register methods
	// From prng.c
	cong := a.seed + (a.seed << 5) + (a.seed << 14) // *16417
	shift := (a.seed >> 15) ^ a.seed
	shift ^= (shift << 17)
	a.seed = cong ^ shift
	return a.seed
}

// Float64 returns a random float64 in [0,1).
func (a *AlgorithmM) Float64() float64 {
	return float64(a.Uint64()) / (1 << 64)
}

// Intn returns a random int in [0,n).
func (a *AlgorithmM) Intn(n int) int {
	if n <= 0 {
		panic("invalid argument to Intn")
	}
	// Mimic prng: ((seed & 0xFFFF) * n) >> 16
	val := a.Uint64()
	lower := val & 0xFFFF
	return int((lower * uint64(n)) >> 16)
}

// GenerateGoldenUint64 generates count uint64 numbers using the given seed.
func GenerateGoldenUint64(seed uint64, count int) []uint64 {
	rng := NewAlgorithmM(seed)
	var numbers []uint64
	for i := 0; i < count; i++ {
		numbers = append(numbers, rng.Uint64())
	}
	return numbers
}

// GenerateGoldenIntn generates count int numbers in [0,n) using the given seed.
func GenerateGoldenIntn(seed uint64, n, count int) []int {
	rng := NewAlgorithmM(seed)
	var numbers []int
	for i := 0; i < count; i++ {
		numbers = append(numbers, rng.Intn(n))
	}
	return numbers
}
