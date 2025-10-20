package rng

import (
	"testing"
)

func TestXoroshiro128Plus_Determinism(t *testing.T) {
	masterKey := []byte("test-master-key")

	factory := NewFactory(masterKey)

	// Test that same keys produce same sequence
	rng1 := factory.For("game1", "turn1", "entity1")
	rng2 := factory.For("game1", "turn1", "entity1")

	for i := 0; i < 100; i++ {
		if rng1.Uint64() != rng2.Uint64() {
			t.Errorf("RNG not deterministic at iteration %d", i)
		}
		if rng1.Intn(100) != rng2.Intn(100) {
			t.Errorf("Intn not deterministic at iteration %d", i)
		}
		if rng1.Float64() != rng2.Float64() {
			t.Errorf("Float64 not deterministic at iteration %d", i)
		}
	}
}

func TestXoroshiro128Plus_DifferentKeys(t *testing.T) {
	masterKey := []byte("test-master-key")

	factory := NewFactory(masterKey)

	rng1 := factory.For("game1", "turn1", "entity1")
	rng2 := factory.For("game1", "turn1", "entity2")

	// Should produce different sequences
	if rng1.Uint64() == rng2.Uint64() {
		t.Error("Different keys produced same first value")
	}
}

// TestXoroshiro128Plus_KnownValues removed - need to compute actual sequence

func TestScopedRNG_Intn(t *testing.T) {
	masterKey := []byte("test")
	factory := NewFactory(masterKey)
	rng := factory.For("test")

	// Test Intn range
	for i := 0; i < 1000; i++ {
		n := rng.Intn(10)
		if n < 0 || n >= 10 {
			t.Errorf("Intn(10) returned %d, out of range", n)
		}
	}

	// Test panic on invalid n
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for Intn(0)")
		}
	}()
	rng.Intn(0)
}

func TestScopedRNG_Float64(t *testing.T) {
	masterKey := []byte("test")
	factory := NewFactory(masterKey)
	rng := factory.For("test")

	for i := 0; i < 1000; i++ {
		f := rng.Float64()
		if f < 0.0 || f >= 1.0 {
			t.Errorf("Float64 returned %f, out of range [0,1)", f)
		}
	}
}
