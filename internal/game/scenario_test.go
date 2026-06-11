package game

import (
	"fmt"
	"path/filepath"
	"testing"
)

// Scenario golden tests drive the turn pipeline with committed, hand-written
// order files (testdata/scenarios/<name>/...) instead of the default orders
// `create orders` generates. They exercise order-processor paths the
// default-order pipeline tests never reach — do_BUILD_command/new_ship,
// jumps, cargo transfers, and combat resolution — and diff every turn
// artifact byte-for-byte against the C-engine reference in
// testdata/cref/<name>/ (produced by the run_scenario blocks in
// testdata/cref/generate.sh).

// setupDats are the post-setup files staged as a scenario's starting point:
// the shared binary game state, each species' record, and the sp0X.log
// species-creation logs (the report phase prepends each species' log, which
// holds its home-system scan, to its turn report).
var setupDats = []string{
	"galaxy.dat", "stars.dat", "planets.dat",
	"sp01.dat", "sp02.dat", "sp03.dat", "sp04.dat",
	"sp01.log", "sp02.log", "sp03.log", "sp04.log",
}

// stageSetupState lays down the post-setup game state from setupDir into the
// current working directory.
func stageSetupState(t *testing.T, setupDir string) {
	t.Helper()
	for _, f := range setupDats {
		copyRefFile(t, filepath.Join(setupDir, f), f)
	}
}

// stageOrders copies the four sp0X.ord order files from ordersDir into the
// current working directory (in place of `create orders`).
func stageOrders(t *testing.T, ordersDir string) {
	t.Helper()
	for n := 1; n <= 4; n++ {
		fn := fmt.Sprintf("sp%02d.ord", n)
		copyRefFile(t, filepath.Join(ordersDir, fn), fn)
	}
}

// runScenario stages the post-setup state and a single turn of committed hand
// orders (testdata/scenarios/<name>/sp0X.ord), runs locations followed by the
// combat..report turn tail (mirroring run_scenario in generate.sh), then diffs
// turnArtifacts(1) against testdata/cref/<name>/ byte-for-byte.
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
	stageSetupState(t, setupDir)
	stageOrders(t, ordersDir)
	silenceStdio(t)

	steps := append([]pipelineStep{
		{"locations", func() int { return locationCommand([]string{"locations"}) }},
	}, turnTail()...)
	runSteps(t, steps)

	compareOutputs(t, scenarioDir, turnArtifacts(1), nil)
}

// runScenarioMulti stages the post-setup state and runs an N-turn hand-order
// scenario (mirroring run_scenario_multi in generate.sh): each turn k runs
// locations, applies the committed orders from testdata/scenarios/<name>/tk/,
// and runs the combat..report tail. The lastTurn artifacts are diffed against
// testdata/cref/<name>/ byte-for-byte. Used for scenarios that need an earlier
// turn to establish state the later turn acts on (e.g. build a ship in t1,
// jump it in t2).
func runScenarioMulti(t *testing.T, name string, nturns int) {
	t.Helper()
	setupDir := refDir(t, "setup")
	scenarioDir := refDir(t, name)
	requireRef(t, scenarioDir)
	base, err := filepath.Abs(filepath.Join("../../testdata/scenarios", name))
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())
	stageSetupState(t, setupDir)
	silenceStdio(t)

	for k := 1; k <= nturns; k++ {
		ResetState()
		if rc := locationCommand([]string{"locations"}); rc != 0 {
			t.Fatalf("turn %d: locations returned %d", k, rc)
		}
		stageOrders(t, filepath.Join(base, fmt.Sprintf("t%d", k)))
		runSteps(t, turnTail())
	}

	compareOutputs(t, scenarioDir, turnArtifacts(nturns), nil)
}

// TestBuildScenarioMatchesC drives the production phase with hand-written BUILD
// orders: each species builds colonial installations (IU/AU), one species also
// builds planetary defenses (PD), and each builds a ship of a different class
// (CT/ES/TR1/PB) — exercising do_BUILD_command and the new_ship slot
// allocation that the default-order pipeline never triggers.
func TestBuildScenarioMatchesC(t *testing.T) { runScenario(t, "build") }

// TestJumpScenarioMatchesC drives the jump pipeline over two turns: turn 1
// builds a fully-paid FTL corvette (CT) per species, turn 2 issues a JUMP
// order relocating each corvette to a nearby star — exercising
// do_JUMP_command, mishap-chance computation, and arrival/location handling.
// (Ships start under construction and jump runs before production, so a
// freshly built ship cannot jump the same turn it is built.)
func TestJumpScenarioMatchesC(t *testing.T) { runScenarioMulti(t, "jump", 2) }
