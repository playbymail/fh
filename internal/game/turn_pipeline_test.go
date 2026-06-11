package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// pipelineStep is one command in a turn pipeline. The C engine runs each as
// its own freshly-seeded process, so runSteps calls ResetState() before each
// one; the lazy PRNG re-seeds from FH_SEED on the next rnd() call, and the
// commands chain through the filesystem (.dat / .ord files) rather than
// shared memory — exactly how the C engine works.
type pipelineStep struct {
	name string
	run  func() int
}

// turnTail is the common per-turn phase sequence shared by every turn:
// combat through report. (locations and order handling differ between the
// first turn and later turns and are prepended by the caller.)
func turnTail() []pipelineStep {
	return []pipelineStep{
		{"combat", func() int { return combatCommand([]string{"combat"}) }},
		{"pre-departure", func() int { return preDepartureCommand([]string{"pre-departure"}) }},
		{"jump", func() int { return jumpCommand([]string{"jump"}) }},
		{"production", func() int { return productionCommand([]string{"production"}) }},
		{"post-arrival", func() int { return postArrivalCommand([]string{"post-arrival"}) }},
		{"finish", func() int { return finishCommand([]string{"finish"}) }},
		{"report", func() int { return reportCommand([]string{"report"}) }},
	}
}

// TestTurnPipelineMatchesC runs the full turn-1 pipeline — locations,
// create orders, combat, pre-departure, jump, production, post-arrival,
// finish, report — starting from the post-setup state and compares the disk
// artifacts byte-for-byte against the C reference in testdata/cref/turn1
// (produced by testdata/cref/generate.sh).
//
// Only disk artifacts are compared. The combat.log/jump.log-style files in
// the reference set are captured stdout from generate.sh and are out of
// scope here; the meaningful byte-faithful outputs are the binary .dat
// state, the generated .ord orders, and the per-species .rpt reports.
func TestTurnPipelineMatchesC(t *testing.T) {
	setupDir, turn1Dir := refDir(t, "setup"), refDir(t, "turn1")
	requireRef(t, turn1Dir)

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())
	stagePostSetupState(t, setupDir, turn1Dir)
	silenceStdio(t)

	// Turn 1, mirroring testdata/cref/generate.sh.
	steps := append([]pipelineStep{
		{"locations", func() int { return locationCommand([]string{"locations"}) }},
		{"create orders", func() int { return createOrdersCommand([]string{"orders"}) }},
	}, turnTail()...)
	runSteps(t, steps)

	compareOutputs(t, turn1Dir, turnArtifacts(1), nil)
}

// TestTurnTwoPipelineMatchesC continues from the turn-1 state into a second
// turn and compares the turn-2 artifacts against testdata/cref/turn2. It runs
// turn 1 first (in the same working directory) so every intermediate file —
// accumulated species logs, locations, transactions — is produced naturally,
// exactly as the C engine does in generate.sh's accumulating run directory.
//
// Turn 2 exercises code paths turn 1 does not: the `turn_number != 1`
// branches in finish/report and the "EVENT LOG FOR TURN" report handling.
func TestTurnTwoPipelineMatchesC(t *testing.T) {
	setupDir, turn1Dir, turn2Dir := refDir(t, "setup"), refDir(t, "turn1"), refDir(t, "turn2")
	requireRef(t, turn2Dir)

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())
	stagePostSetupState(t, setupDir, turn1Dir)
	silenceStdio(t)

	// Turn 1 (produces the starting state for turn 2).
	turn1 := append([]pipelineStep{
		{"locations", func() int { return locationCommand([]string{"locations"}) }},
		{"create orders", func() int { return createOrdersCommand([]string{"orders"}) }},
	}, turnTail()...)
	runSteps(t, turn1)

	// The phases do not consume sp0X.ord, so remove the turn-1 orders to
	// force `create orders` to regenerate fresh defaults from the turn-1
	// state (mirrors the `rm` in generate.sh).
	for n := 1; n <= 4; n++ {
		if err := os.Remove(fmt.Sprintf("sp%02d.ord", n)); err != nil {
			t.Fatalf("remove turn-1 orders: %v", err)
		}
	}

	// Turn 2.
	turn2 := append([]pipelineStep{
		{"create orders", func() int { return createOrdersCommand([]string{"orders"}) }},
		{"locations", func() int { return locationCommand([]string{"locations"}) }},
	}, turnTail()...)
	runSteps(t, turn2)

	compareOutputs(t, turn2Dir, turnArtifacts(2), nil)
}

// turnArtifacts is the list of disk artifacts a turn pipeline produces, for
// the given turn number (reports are named sp0X.rpt.tN).
func turnArtifacts(turn int) []string {
	out := []string{"galaxy.dat", "planets.dat", "locations.dat"}
	for n := 1; n <= 4; n++ {
		out = append(out, fmt.Sprintf("sp%02d.dat", n), fmt.Sprintf("sp%02d.ord", n))
	}
	for n := 1; n <= 4; n++ {
		out = append(out, fmt.Sprintf("sp%02d.rpt.t%d", n, turn))
	}
	return out
}

// stagePostSetupState lays down the post-setup game state as a pipeline's
// starting point: the binary .dat files plus the sp0X.log species-creation
// logs (the report phase prepends each species' log, which holds the
// home-system scan, to its turn report) and the GM noorders template.
func stagePostSetupState(t *testing.T, setupDir, turnDir string) {
	t.Helper()
	for _, name := range []string{
		"galaxy.dat", "stars.dat", "planets.dat",
		"sp01.dat", "sp02.dat", "sp03.dat", "sp04.dat",
		"sp01.log", "sp02.log", "sp03.log", "sp04.log",
	} {
		copyRefFile(t, filepath.Join(setupDir, name), name)
	}
	copyRefFile(t, filepath.Join(turnDir, "noorders.txt"), "noorders.txt")
}

// runSteps runs each command step with a fresh ResetState, failing the test
// on the first non-zero return.
func runSteps(t *testing.T, steps []pipelineStep) {
	t.Helper()
	for _, step := range steps {
		ResetState()
		if rc := step.run(); rc != 0 {
			t.Fatalf("pipeline step %q returned %d", step.name, rc)
		}
	}
}

// compareOutputs compares each named file in the working directory against
// the same-named file in the reference directory, byte for byte. Files listed
// in known (name -> reason) are expected to diverge: a mismatch is logged
// rather than failed, and an unexpected match is flagged so the entry can be
// removed once the underlying bug is fixed.
func compareOutputs(t *testing.T, refDir string, names []string, known map[string]string) {
	t.Helper()
	for _, name := range names {
		reason, isKnown := known[name]
		if !isKnown {
			compareRefFile(t, name, filepath.Join(refDir, name))
			continue
		}
		got, err := os.ReadFile(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		want, err := os.ReadFile(filepath.Join(refDir, name))
		if err != nil {
			t.Errorf("%s: reference: %v", name, err)
			continue
		}
		if bytes.Equal(got, want) {
			t.Errorf("%s now matches the C reference — remove it from knownTurn2Divergence (%s)", name, reason)
			continue
		}
		t.Logf("known divergence: %s — %s (got %d bytes, want %d, first diff at byte %d)",
			name, reason, len(got), len(want), firstDiff(got, want))
	}
}

// refDir returns the absolute path to a testdata/cref subdirectory.
func refDir(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("../../testdata/cref", name))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// requireRef skips the test when the C reference data has not been generated
// (testdata/cref is git-ignored; run testdata/cref/generate.sh to create it).
func requireRef(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, "galaxy.dat")); err != nil {
		t.Skipf("C reference data not available in %s: %v", dir, err)
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
// rest of the test, restoring them via t.Cleanup.
func silenceStdio(t *testing.T) {
	t.Helper()
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = devnull, devnull
	t.Cleanup(func() {
		os.Stdout, os.Stderr = origOut, origErr
		_ = devnull.Close()
	})
}
