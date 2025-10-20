package rng

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// xoroshiro128+ implementation for deterministic RNG.
// Based on https://prng.di.unimi.it/xoroshiro128plus.c

type xoroshiro128plus struct {
	s [2]uint64
}

func newXoroshiro128Plus(s0, s1 uint64) *xoroshiro128plus {
	if s0 == 0 && s1 == 0 {
		// Avoid all-zero state
		s0 = 1
	}
	return &xoroshiro128plus{s: [2]uint64{s0, s1}}
}

func (r *xoroshiro128plus) Next() uint64 {
	s0 := r.s[0]
	s1 := r.s[1]
	result := s0 + s1

	s1 ^= s0
	r.s[0] = rotl(s0, 24) ^ s1 ^ (s1 << 16)
	r.s[1] = rotl(s1, 37)

	return result
}

func rotl(x uint64, k int) uint64 {
	return (x << k) | (x >> (64 - k))
}

// Factory implements the RNG factory.
type factory struct {
	masterKey []byte
}

// newFactory creates a new RNG factory with the given master key.
func newFactory(masterKey []byte) Factory {
	return &factory{masterKey: masterKey}
}

// For derives a scoped RNG from stable keys.
func (f *factory) For(keys ...string) Scoped {
	input := ""
	for i, key := range keys {
		if i > 0 {
			input += "|"
		}
		input += key
	}

	mac := hmac.New(sha256.New, f.masterKey)
	mac.Write([]byte(input))
	sum := mac.Sum(nil)

	s0 := binary.LittleEndian.Uint64(sum[0:8])
	s1 := binary.LittleEndian.Uint64(sum[8:16])

	return &scopedRNG{rng: newXoroshiro128Plus(s0, s1)}
}

// scopedRNG wraps the xoroshiro RNG to implement Scoped.
type scopedRNG struct {
	rng *xoroshiro128plus
}

func (r *scopedRNG) Uint64() uint64 {
	return r.rng.Next()
}

func (r *scopedRNG) Float64() float64 {
	return float64(r.Uint64()>>11) * (1.0 / (1 << 53))
}

func (r *scopedRNG) Intn(n int) int {
	if n <= 0 {
		panic("invalid argument to Intn")
	}
	if n&(n-1) == 0 { // n is power of two
		return int(r.Uint64() & uint64(n-1))
	}
	max := uint64((1 << 63) - 1 - (1<<63)%uint64(n))
	v := r.Uint64()
	for v > max {
		v = r.Uint64()
	}
	return int(v % uint64(n))
}
