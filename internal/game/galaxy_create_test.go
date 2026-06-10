package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestCreateGalaxyMatchesC runs the ported createGalaxy with the same
// seed and parameters as the C reference run (fh create galaxy
// --species=9, FH_SEED=1924085713) and compares every output file byte
// for byte against the C engine's output in testdata/cref/galaxy.
func TestCreateGalaxyMatchesC(t *testing.T) {
	refDir, err := filepath.Abs("../../testdata/cref/galaxy")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(refDir, "galaxy.dat")); err != nil {
		t.Skipf("C reference data not available: %v", err)
	}

	ResetState()
	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())

	// fh create galaxy --species=9 computes stars and radius like the C
	// createGalaxyCommand: stars = species*90/15, radius from the
	// standard-density volume.
	desiredNumSpecies := 9
	desiredNumStars := (desiredNumSpecies * STANDARD_NUMBER_OF_STAR_SYSTEMS) / STANDARD_NUMBER_OF_SPECIES
	minVolume := desiredNumStars * STANDARD_GALACTIC_RADIUS * STANDARD_GALACTIC_RADIUS * STANDARD_GALACTIC_RADIUS / STANDARD_NUMBER_OF_STAR_SYSTEMS
	galacticRadius := MIN_RADIUS
	for galacticRadius*galacticRadius*galacticRadius < minVolume {
		galacticRadius++
	}

	if rc := createGalaxy(galacticRadius, desiredNumStars, desiredNumSpecies); rc != 0 {
		t.Fatalf("createGalaxy returned %d", rc)
	}

	for _, name := range []string{"galaxy.dat", "stars.dat", "planets.dat", "galaxy.txt", "stars.txt", "planets.txt"} {
		got, err := os.ReadFile(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		want, err := os.ReadFile(filepath.Join(refDir, name))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
				name, len(got), len(want), firstDiff(got, want))
		}
	}
}

func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
