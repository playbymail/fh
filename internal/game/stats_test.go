package game

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestStatsMatchesC runs the `stats` command over the validated post-turn-1
// game state and compares its captured stdout byte-for-byte against the C
// reference in testdata/cref/turn1/stats.log (the captured C `stats` stdout
// produced by testdata/cref/generate.sh).
//
// `stats` reads galaxy.dat, planets.dat, and the species sp0X.dat files and
// writes its entire report to stdout (the C engine uses printf throughout), so
// the only artifact to compare is that captured stdout. We run it against the
// turn1 reference state directly (the snapshot generate.sh took right before it
// invoked `stats`).
func TestStatsMatchesC(t *testing.T) {
	turn1Dir := refDir(t, "turn1")
	requireRef(t, turn1Dir)

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())
	stageStatsState(t, turn1Dir)

	got := captureStdout(t, func() int {
		ResetState()
		return statsCommand([]string{"stats"})
	})

	want, err := os.ReadFile(filepath.Join(turn1Dir, "stats.log"))
	if err != nil {
		t.Fatalf("read reference stats.log: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("stats output differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
			len(got), len(want), firstDiff(got, want))
	}
}

// stageStatsState lays down the game-state files `stats` reads: the shared
// galaxy/planets/stars binary records and each species' sp0X.dat record, copied
// from the turn1 reference snapshot.
func stageStatsState(t *testing.T, turnDir string) {
	t.Helper()
	for _, name := range []string{
		"galaxy.dat", "stars.dat", "planets.dat",
		"sp01.dat", "sp02.dat", "sp03.dat", "sp04.dat",
	} {
		copyRefFile(t, filepath.Join(turnDir, name), name)
	}
}

// captureStdout redirects os.Stdout to a pipe while fn runs, returning
// everything fn wrote to stdout. Restores os.Stdout before returning.
func captureStdout(t *testing.T, fn func() int) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w

	done := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- data
	}()

	rc := fn()

	os.Stdout = orig
	_ = w.Close()
	out := <-done
	_ = r.Close()

	if rc != 0 {
		t.Fatalf("statsCommand returned %d", rc)
	}
	return out
}
