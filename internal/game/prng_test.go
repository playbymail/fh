package game

import (
	"bytes"
	"os"
	"testing"
)

// TestLogRandomCommandMatchesC verifies the ported PRNG produces the same
// sequence as the C engine. The golden file was captured from the C
// binary: Far-Horizons/build/fh logrnd.
func TestLogRandomCommandMatchesC(t *testing.T) {
	golden, err := os.ReadFile("testdata/logrnd_golden.txt")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	ResetState()
	var buf bytes.Buffer
	logRandomCommand(&buf)

	if !bytes.Equal(buf.Bytes(), golden) {
		t.Errorf("PRNG sequence diverges from C engine\ngot:\n%s\nwant:\n%s", buf.String(), golden)
	}
}

// TestRndSeedFromEnv verifies FH_SEED seeding matches the C behavior of
// concatenating only the digits in the value.
func TestRndSeedFromEnv(t *testing.T) {
	ResetState()
	t.Setenv("FH_SEED", "abc123x45")
	rnd(10)
	if got := prngGetSeed(); got == 0 {
		t.Fatal("seed not initialized")
	}
	ResetState()
	prngSetSeed(12345)
	want := rnd(1000000)
	ResetState()
	t.Setenv("FH_SEED", "12345")
	if got := rnd(1000000); got != want {
		t.Errorf("rnd with FH_SEED=12345 = %d, want %d", got, want)
	}
}
