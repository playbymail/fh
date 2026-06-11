package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestListCommandMatchesC runs the `list` command over the validated
// post-turn-1 game state and compares its captured stdout byte-for-byte against
// the C references captured by testdata/cref/generate.sh. Each case exercises a
// different output branch of list.c.
func TestListCommandMatchesC(t *testing.T) {
	turn1Dir := refDir(t, "turn1")
	requireRef(t, turn1Dir)

	cases := []struct {
		ref  string
		args []string
	}{
		{"list_galaxy.log", []string{"list", "galaxy"}},
		{"list_galaxy_nopl.log", []string{"list", "galaxy", "--planets=false"}},
		{"list_galaxy_worm.log", []string{"list", "galaxy", "--wormholes=true"}},
		{"list_scanned_sp1.log", []string{"list", "scanned", "--species=1"}},
		{"list_scanned_sp2.log", []string{"list", "scanned", "--species=2"}},
	}

	for _, tc := range cases {
		t.Run(tc.ref, func(t *testing.T) {
			t.Setenv("FH_SEED", "1924085713")
			t.Chdir(t.TempDir())
			stageListState(t, turn1Dir)

			got := captureStdout(t, func() int {
				ResetState()
				return listCommand(tc.args)
			})

			want, err := os.ReadFile(filepath.Join(turn1Dir, tc.ref))
			if err != nil {
				t.Fatalf("read reference %s: %v", tc.ref, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("list %v output differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
					tc.args[1:], len(got), len(want), firstDiff(got, want))
			}
		})
	}
}

// stageListState lays down the binary game-state files `list` reads.
func stageListState(t *testing.T, turnDir string) {
	t.Helper()
	for _, name := range []string{
		"galaxy.dat", "stars.dat", "planets.dat",
		"sp01.dat", "sp02.dat", "sp03.dat", "sp04.dat",
	} {
		copyRefFile(t, filepath.Join(turnDir, name), name)
	}
}
