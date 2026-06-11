package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestTurnPipelineMatchesC runs the full turn-1 pipeline — locations,
// create orders, combat, pre-departure, jump, production, post-arrival,
// finish, report — starting from the post-setup state and compares the
// disk artifacts byte-for-byte against the C reference in
// testdata/cref/turn1 (produced by testdata/cref/generate.sh).
//
// The C engine runs each command as its own freshly-seeded process, so
// the test calls ResetState() before every command; the lazy PRNG
// re-seeds from FH_SEED on the next rnd() call. Commands chain through
// the filesystem (.dat / .ord files), not shared memory, so the pipeline
// works the same way the C engine does.
//
// Only disk artifacts are compared. The combat.log/jump.log-style files in
// the reference set are captured stdout from generate.sh and are out of
// scope here; the meaningful byte-faithful outputs are the binary .dat
// state, the generated .ord orders, and the per-species .rpt reports.
func TestTurnPipelineMatchesC(t *testing.T) {
	setupDir, err := filepath.Abs("../../testdata/cref/setup")
	if err != nil {
		t.Fatal(err)
	}
	turnDir, err := filepath.Abs("../../testdata/cref/turn1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(turnDir, "galaxy.dat")); err != nil {
		t.Skipf("C reference data not available: %v", err)
	}

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())

	// Stage the post-setup state as the pipeline's starting point. The
	// sp0X.log species-creation logs are required too: the report phase
	// prepends each species' log (which holds the home-system scan) to its
	// turn report, exactly as the C engine does in its accumulating run dir.
	for _, name := range []string{
		"galaxy.dat", "stars.dat", "planets.dat",
		"sp01.dat", "sp02.dat", "sp03.dat", "sp04.dat",
		"sp01.log", "sp02.log", "sp03.log", "sp04.log",
	} {
		copyRefFile(t, filepath.Join(setupDir, name), name)
	}
	// The GM noorders template consumed by `create orders`.
	copyRefFile(t, filepath.Join(turnDir, "noorders.txt"), "noorders.txt")

	// The engine is chatty on stdout/stderr; silence it for the run.
	restore := silenceStdio(t)

	// Run the turn-1 pipeline, mirroring testdata/cref/generate.sh. Each
	// step resets package state to emulate a fresh C process.
	steps := []struct {
		name string
		run  func() int
	}{
		{"locations", func() int { return locationCommand([]string{"locations"}) }},
		{"create orders", func() int { return createOrdersCommand([]string{"orders"}) }},
		{"combat", func() int { return combatCommand([]string{"combat"}) }},
		{"pre-departure", func() int { return preDepartureCommand([]string{"pre-departure"}) }},
		{"jump", func() int { return jumpCommand([]string{"jump"}) }},
		{"production", func() int { return productionCommand([]string{"production"}) }},
		{"post-arrival", func() int { return postArrivalCommand([]string{"post-arrival"}) }},
		{"finish", func() int { return finishCommand([]string{"finish"}) }},
		{"report", func() int { return reportCommand([]string{"report"}) }},
	}
	for _, step := range steps {
		ResetState()
		if rc := step.run(); rc != 0 {
			restore()
			t.Fatalf("pipeline step %q returned %d", step.name, rc)
		}
	}
	restore()

	// Compare every disk artifact the pipeline produces.
	outputs := []string{
		"galaxy.dat", "planets.dat", "locations.dat",
		"sp01.dat", "sp02.dat", "sp03.dat", "sp04.dat",
		"sp01.ord", "sp02.ord", "sp03.ord", "sp04.ord",
		"sp01.rpt.t1", "sp02.rpt.t1", "sp03.rpt.t1", "sp04.rpt.t1",
	}
	for _, name := range outputs {
		compareRefFile(t, name, filepath.Join(turnDir, name))
	}
}

// copyRefFile copies a reference file into the current working directory.
func copyRefFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("stage %s: %v", dst, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("stage %s: %v", dst, err)
	}
}

// compareRefFile compares a file produced in the working directory against
// the C reference, reporting the first divergent byte on mismatch.
func compareRefFile(t *testing.T, name, ref string) {
	t.Helper()
	got, err := os.ReadFile(name)
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	want, err := os.ReadFile(ref)
	if err != nil {
		t.Errorf("%s: reference: %v", name, err)
		return
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
			name, len(got), len(want), firstDiff(got, want))
	}
}

// silenceStdio redirects os.Stdout and os.Stderr to the null device for the
// duration of a noisy run, returning a function that restores them.
func silenceStdio(t *testing.T) func() {
	t.Helper()
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = devnull, devnull
	return func() {
		os.Stdout, os.Stderr = origOut, origErr
		_ = devnull.Close()
	}
}
