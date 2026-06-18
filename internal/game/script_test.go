package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/playbymail/fh/interface/scripting"
)

// writeScript writes a Lua script into a temp dir and returns its path.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.lua")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

// crefTurnDataRoot stages a data-root with a single bare-integer turn folder "1"
// populated from the generated C-reference turn1 tree (a resolved turn with
// galaxy/species .dat, orders, and reports). It skips the test when the
// reference data has not been generated (it is git-ignored; run make golden-ref).
func crefTurnDataRoot(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cref", "turn1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(src, "galaxy.dat")); err != nil {
		t.Skipf("C reference data not available: %v", err)
	}
	root := t.TempDir()
	dst := filepath.Join(root, "1")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestScriptQueryVerbsEndToEnd drives the read verbs through the CLI against a
// real resolved turn, asserting each returns sane data. exit 0 means every Lua
// assertion passed.
func TestScriptQueryVerbsEndToEnd(t *testing.T) {
	ResetState()
	root := crefTurnDataRoot(t)
	script := writeScript(t, `
		if fh.current_turn() ~= 1 then error("current_turn ~= 1") end
		if fh.turn_status(1) ~= "resolved" then error("turn 1 not resolved") end
		local ord = fh.orders(1, 1)
		if ord == nil or #ord == 0 then error("orders(1,1) empty") end
		local rpt = fh.report(1, 1)
		if rpt == nil or #rpt == 0 then error("report(1,1) empty") end
		local s = fh.species_stats(1, 1)
		if s.species ~= 1 then error("stats.species ~= 1") end
		if type(s.tech.MI) ~= "number" then error("stats.tech.MI not a number") end
		local all = fh.species_stats(1)
		if #all < 1 then error("gm all-species stats empty") end
	`)
	if rv := scriptCommand([]string{"script", "--data-root=" + root, "--gm", script}); rv != 0 {
		t.Fatalf("query script returned %d, want 0", rv)
	}
	// Determinism guard: queries must not perturb the PRNG.
	if prngSeed != 0 {
		t.Errorf("prngSeed = %d after queries, want 0", prngSeed)
	}
}

// TestScriptPlayerScopeDeniesOtherSpecies checks player scope through the CLI:
// species 1 may not read species 2's report.
func TestScriptPlayerScopeDeniesOtherSpecies(t *testing.T) {
	ResetState()
	root := crefTurnDataRoot(t)
	script := writeScript(t, `fh.report(1, 2)`)
	if rv := scriptCommand([]string{"script", "--data-root=" + root, "--species=1", script}); rv != 2 {
		t.Fatalf("cross-species report returned %d, want 2 (denied)", rv)
	}
}

// TestComputeSpeciesStatsMatchesStatsCommand is the drift guard: the structured
// SpeciesStats computed by the script glue must reproduce, field for field, the
// per-species row that the frozen statsCommand prints for the same turn.
func TestComputeSpeciesStatsMatchesStatsCommand(t *testing.T) {
	turn1Dir := refDir(t, "turn1")
	requireRef(t, turn1Dir)

	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())
	stageStatsState(t, turn1Dir)

	// statsCommand loads the globals and prints; capture its output, then
	// recompute each species and assert the re-rendered row appears verbatim.
	out := captureStdout(t, func() int {
		ResetState()
		return statsCommand([]string{"stats"})
	})

	for sp := 1; sp <= galaxy.num_species; sp++ {
		if data_in_memory[sp-1] == FALSE {
			continue
		}
		row := renderStatsRow(computeSpeciesStats(sp))
		if !strings.Contains(string(out), row) {
			t.Errorf("species %d: computed row not found in stats output\n  computed: %q", sp, row)
		}
	}
}

// renderStatsRow reproduces statsCommand's per-species line format from a
// structured SpeciesStats, for the drift comparison.
func renderStatsRow(s scripting.SpeciesStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%2d %-15.15s", s.Species, s.Name)
	for _, k := range []string{"MI", "MA", "ML", "GV", "LS", "BI"} {
		fmt.Fprintf(&b, "%4d", s.Tech[k])
	}
	fmt.Fprintf(&b, "%7d%4d", s.TotalProduction, s.NumPlanets)
	fmt.Fprintf(&b, "%5d", s.NumShips)
	fmt.Fprintf(&b, "%5d", s.NumShipyards)
	fmt.Fprintf(&b, "%8d%8d", s.OffensivePower, s.DefensivePower)
	fmt.Fprintf(&b, "%9d", s.EconUnits)
	return b.String()
}

// TestScriptCommandRunsTrivialScript checks the host wires up end to end with a
// trivial script and no game data.
func TestScriptCommandRunsTrivialScript(t *testing.T) {
	ResetState()
	root := t.TempDir()
	script := writeScript(t, `print("hello")`)
	if rv := scriptCommand([]string{"script", "--data-root=" + root, "--gm", script}); rv != 0 {
		t.Fatalf("trivial script returned %d, want 0", rv)
	}
}

// TestScriptCommandFailingScript checks a Lua runtime error surfaces as exit 2.
func TestScriptCommandFailingScript(t *testing.T) {
	ResetState()
	root := t.TempDir()
	script := writeScript(t, `error("boom")`)
	if rv := scriptCommand([]string{"script", "--data-root=" + root, "--gm", script}); rv != 2 {
		t.Errorf("failing script returned %d, want 2", rv)
	}
}

// TestScriptCommandSpaceSeparatedFlagsRejected checks the --opt value form is
// rejected: fhc uses the --opt=val convention.
func TestScriptCommandSpaceSeparatedFlagsRejected(t *testing.T) {
	ResetState()
	root := t.TempDir()
	script := writeScript(t, `print("hi")`)
	if rv := scriptCommand([]string{"script", "--data-root", root, "--species", "8", script}); rv != 2 {
		t.Fatalf("space-separated flags returned %d, want 2", rv)
	}
}

// TestScriptCommandFlagValidation covers the bad flag combinations, each
// rejected with exit 2 before any Lua runs.
func TestScriptCommandFlagValidation(t *testing.T) {
	ResetState()
	root := t.TempDir()
	script := writeScript(t, `print("hello")`)
	cases := []struct {
		name string
		args []string
	}{
		{"missing data-root", []string{"script", "--gm", script}},
		{"gm and species", []string{"script", "--data-root=" + root, "--gm", "--species=1", script}},
		{"neither gm nor species", []string{"script", "--data-root=" + root, script}},
		{"non-integer species", []string{"script", "--data-root=" + root, "--species=abc", script}},
		{"zero species", []string{"script", "--data-root=" + root, "--species=0", script}},
		{"missing script path", []string{"script", "--data-root=" + root, "--gm"}},
		{"unknown option", []string{"script", "--data-root=" + root, "--gm", "--bogus", script}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rv := scriptCommand(tc.args); rv != 2 {
				t.Errorf("scriptCommand(%v) = %d, want 2", tc.args[1:], rv)
			}
		})
	}
}
