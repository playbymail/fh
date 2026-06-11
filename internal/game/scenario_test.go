package game

import (
	"fmt"
	"path/filepath"
	"testing"
)

// Scenario golden tests drive the turn pipeline with committed, hand-written
// order files (testdata/scenarios/<name>/sp0X.ord) instead of the default
// orders `create orders` generates. They exercise order-processor paths the
// default-order pipeline tests never reach — do_BUILD_command/new_ship,
// jumps, cargo transfers, and combat resolution — and diff every turn
// artifact byte-for-byte against the C-engine reference in
// testdata/cref/<name>/ (produced by the run_scenario block in
// testdata/cref/generate.sh).

// stageScenario lays down the post-setup game state (the shared binary .dat
// files, each species' record, and the sp0X.log species-creation logs the
// report phase prepends to its turn report) plus the committed hand-written
// orders for the named scenario. Unlike stagePostSetupState it copies the
// orders from testdata/scenarios/<name> in place of `create orders` and does
// not stage the noorders.txt GM template (no create-orders phase runs).
func stageScenario(t *testing.T, setupDir, ordersDir string) {
	t.Helper()
	for _, f := range []string{
		"galaxy.dat", "stars.dat", "planets.dat",
		"sp01.dat", "sp02.dat", "sp03.dat", "sp04.dat",
		"sp01.log", "sp02.log", "sp03.log", "sp04.log",
	} {
		copyRefFile(t, filepath.Join(setupDir, f), f)
	}
	for n := 1; n <= 4; n++ {
		fn := fmt.Sprintf("sp%02d.ord", n)
		copyRefFile(t, filepath.Join(ordersDir, fn), fn)
	}
}

// runScenario stages the post-setup state and hand orders for the named
// scenario, runs locations followed by the combat..report turn tail (mirroring
// the run_scenario block in generate.sh), then diffs turnArtifacts(1) against
// testdata/cref/<name>/ byte-for-byte. Scenarios are first-turn runs, so the
// reports are sp0X.rpt.t1 and the artifacts use turn 1.
func runScenario(t *testing.T, name string) {
	t.Helper()
	setupDir := refDir(t, "setup")
	scenarioDir := refDir(t, name)
	requireRef(t, scenarioDir)
	ordersDir, err := filepath.Abs(filepath.Join("../../testdata/scenarios", name))
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())
	stageScenario(t, setupDir, ordersDir)
	silenceStdio(t)

	// locations runs first (no create orders — the hand orders are already
	// staged), then the shared combat..report tail.
	steps := append([]pipelineStep{
		{"locations", func() int { return locationCommand([]string{"locations"}) }},
	}, turnTail()...)
	runSteps(t, steps)

	compareOutputs(t, scenarioDir, turnArtifacts(1), nil)
}

// TestBuildScenarioMatchesC drives the production phase with hand-written BUILD
// orders: each species builds colonial installations (IU/AU), one species also
// builds planetary defenses (PD), and each builds a ship of a different class
// (CT/ES/TR1/PB) — exercising do_BUILD_command and the new_ship slot
// allocation that the default-order pipeline never triggers.
func TestBuildScenarioMatchesC(t *testing.T) { runScenario(t, "build") }
