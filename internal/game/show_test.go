package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestShowCommandMatchesC runs the `show` command over the validated
// post-turn-1 game state and compares its captured stdout byte-for-byte against
// the C references captured by testdata/cref/generate.sh. Each case exercises a
// different output branch of show.c (game-value queries, the ASCII map, the
// paged map, and the help text).
func TestShowCommandMatchesC(t *testing.T) {
	turn1Dir := refDir(t, "turn1")
	requireRef(t, turn1Dir)

	cases := []struct {
		ref  string
		args []string
	}{
		{"show_values.log", []string{"show", "num_stars", "num_species", "num_planets", "num_natural_wormholes", "radius", "d_num_species", "turn_number"}},
		{"show_galaxy_ascii.log", []string{"show", "galaxy", "--ascii"}},
		{"show_galaxy.log", []string{"show", "galaxy"}},
		{"show_help.log", []string{"show", "help"}},
	}

	for _, tc := range cases {
		t.Run(tc.ref, func(t *testing.T) {
			t.Setenv("FH_SEED", "1924085713")
			t.Chdir(t.TempDir())
			stageShowState(t, turn1Dir)

			got := captureStdout(t, func() int {
				ResetState()
				return showCommand(tc.args)
			})

			want, err := os.ReadFile(filepath.Join(turn1Dir, tc.ref))
			if err != nil {
				t.Fatalf("read reference %s: %v", tc.ref, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("show %v output differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
					tc.args[1:], len(got), len(want), firstDiff(got, want))
			}
		})
	}
}

// TestShowGalaxyMapFileMatchesC verifies that `show galaxy` writes the
// galaxy.map file byte-for-byte identically to the C engine (the C reference
// galaxy.map is preserved as show_galaxy.map by generate.sh).
func TestShowGalaxyMapFileMatchesC(t *testing.T) {
	turn1Dir := refDir(t, "turn1")
	requireRef(t, turn1Dir)

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())
	stageShowState(t, turn1Dir)

	_ = captureStdout(t, func() int {
		ResetState()
		return showCommand([]string{"show", "galaxy"})
	})

	got, err := os.ReadFile("galaxy.map")
	if err != nil {
		t.Fatalf("read produced galaxy.map: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(turn1Dir, "show_galaxy.map"))
	if err != nil {
		t.Fatalf("read reference show_galaxy.map: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("galaxy.map differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
			len(got), len(want), firstDiff(got, want))
	}
}

// stageShowState lays down the binary game-state files `show` reads.
func stageShowState(t *testing.T, turnDir string) {
	t.Helper()
	for _, name := range []string{
		"galaxy.dat", "stars.dat", "planets.dat",
		"sp01.dat", "sp02.dat", "sp03.dat", "sp04.dat",
	} {
		copyRefFile(t, filepath.Join(turnDir, name), name)
	}
}
