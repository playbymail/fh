package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCreateSpeciesMatchesC reproduces the C reference setup run
// (FH_SEED=1924085713):
//
//	fh create galaxy --species=9
//	fh create home-system-templates
//	fh create species --config=species.cfg --radius=6
//
// and compares the species data files and the modified galaxy, star, and
// planet files byte for byte against testdata/cref/setup. Each C command
// runs as its own process with a freshly seeded PRNG, so the test resets
// the package state between the commands.
func TestCreateSpeciesMatchesC(t *testing.T) {
	refDir, err := filepath.Abs("../../testdata/cref/setup")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(refDir, "sp01.dat")); err != nil {
		t.Skipf("C reference data not available: %v", err)
	}

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())

	// fh create galaxy --species=9 computes stars and radius like the C
	// createGalaxyCommand: stars = species*90/15, radius from the
	// standard-density volume.
	ResetState()
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

	// fh create home-system-templates
	ResetState()
	if rc := createHomeSystemTemplates(); rc != 0 {
		t.Fatalf("createHomeSystemTemplates returned %d", rc)
	}

	// fh create species --config=species.cfg --radius=6
	ResetState()
	cfg, err := os.ReadFile(filepath.Join(refDir, "species.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("species.cfg", cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := createSpeciesCommand([]string{"species", "--config=species.cfg", "--radius=6"}); rc != 0 {
		t.Fatalf("createSpeciesCommand returned %d", rc)
	}

	names := []string{
		"galaxy.dat", "stars.dat", "planets.dat",
		"galaxy.hs.txt", "stars.hs.txt", "planets.hs.txt",
	}
	for n := 1; n <= 4; n++ {
		names = append(names,
			fmt.Sprintf("sp%02d.dat", n),
			fmt.Sprintf("sp%02d.log", n),
			fmt.Sprintf("species%03d.txt", n))
	}
	for _, name := range names {
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
