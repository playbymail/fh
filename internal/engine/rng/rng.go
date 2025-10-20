// Package rng implements deterministic RNG for the engine.
package rng

// Scoped provides deterministic random draws.
type Scoped interface {
	Uint64() uint64
	Float64() float64 // [0,1)
	Intn(n int) int
}

// Factory creates scoped RNGs from stable keys.
type Factory interface {
	For(keys ...string) Scoped
}

// NewFactory creates a new RNG factory with the given master key.
// Uses HMAC-SHA256 to derive seeds from keys, with xoroshiro128+ as the PRNG.
func NewFactory(masterKey []byte) Factory {
	return newFactory(masterKey)
}
