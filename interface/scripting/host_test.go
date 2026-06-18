package scripting

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// fakeGame is an in-memory Game for exercising the host's sandbox and scope
// policy without a real engine. Turns 1 (resolved) and 2 (pending) exist for
// species 1 and 8.
type fakeGame struct{}

func (fakeGame) CurrentTurn() (int, error) { return 2, nil }

func (fakeGame) TurnStatus(turn int) (string, error) {
	switch turn {
	case 1:
		return TurnResolved, nil
	case 2:
		return TurnPending, nil
	default:
		return "", fmt.Errorf("turn %d does not exist", turn)
	}
}

func (fakeGame) SpeciesIDs() ([]int, error) { return []int{1, 8}, nil }

func (fakeGame) Orders(turn, species int) (string, bool, error) {
	if turn == 1 {
		return fmt.Sprintf("orders for sp%02d turn %d", species, turn), true, nil
	}
	return "", false, nil
}

func (fakeGame) Report(turn, species int) (string, error) {
	if turn != 1 {
		return "", fmt.Errorf("turn %d is not resolved", turn)
	}
	return fmt.Sprintf("report for sp%02d turn %d", species, turn), nil
}

func (fakeGame) SpeciesStats(turn, species int) (SpeciesStats, error) {
	return SpeciesStats{
		Species:         species,
		Name:            fmt.Sprintf("Species %02d", species),
		Tech:            map[string]int{"MI": 10, "MA": 10, "ML": 1, "GV": 1, "LS": 1, "BI": 10},
		TotalProduction: 100 * species,
		NumPlanets:      1,
		EconUnits:       species,
	}, nil
}

// writeScript writes a Lua script into a temp dir and returns its path.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.lua")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

// runGM / runPlayer run a script body under the respective scope and return any
// error from the host (a Lua error surfaces here).
func runGM(t *testing.T, body string) error {
	t.Helper()
	return NewHost(fakeGame{}, ScopeGM, 0).Run(writeScript(t, body))
}

func runPlayer(t *testing.T, id int, body string) error {
	t.Helper()
	return NewHost(fakeGame{}, ScopePlayer, id).Run(writeScript(t, body))
}

// TestQueryVerbs checks the read verbs return the expected values under GM scope.
func TestQueryVerbs(t *testing.T) {
	body := `
		if fh.current_turn() ~= 2 then error("current_turn") end
		if fh.turn_status(1) ~= "resolved" then error("turn_status(1)") end
		if fh.turn_status(2) ~= "pending" then error("turn_status(2)") end
		if fh.orders(1, 8) ~= "orders for sp08 turn 1" then error("orders(1,8)") end
		if fh.orders(2, 8) ~= nil then error("orders(2,8) should be nil") end
		if fh.report(1, 8) ~= "report for sp08 turn 1" then error("report(1,8)") end
		local s = fh.species_stats(1, 8)
		if s.species ~= 8 then error("stats.species") end
		if s.total_production ~= 800 then error("stats.total_production") end
		if s.tech.MI ~= 10 then error("stats.tech.MI") end
	`
	if err := runGM(t, body); err != nil {
		t.Fatalf("gm query script: %v", err)
	}
}

// TestReportUnresolvedRaises checks fh.report on a pending turn is an error.
func TestReportUnresolvedRaises(t *testing.T) {
	if err := runGM(t, `fh.report(2, 8)`); err == nil {
		t.Fatal("report on pending turn: want error")
	}
}

// TestTurnStatusMissingRaises checks an absent turn surfaces as an error.
func TestTurnStatusMissingRaises(t *testing.T) {
	if err := runGM(t, `fh.turn_status(99)`); err == nil {
		t.Fatal("turn_status on missing turn: want error")
	}
}

// TestPlayerScopeOwnSpecies checks a player may read its own species, with the
// id implied or stated explicitly.
func TestPlayerScopeOwnSpecies(t *testing.T) {
	body := `
		if fh.orders(1) ~= "orders for sp08 turn 1" then error("implied id") end
		if fh.orders(1, 8) ~= "orders for sp08 turn 1" then error("explicit id") end
		local s = fh.species_stats(1)
		if s.species ~= 8 then error("species_stats implied") end
	`
	if err := runPlayer(t, 8, body); err != nil {
		t.Fatalf("player own-species script: %v", err)
	}
}

// TestPlayerScopeDeniesOtherSpecies checks a player cannot read another species.
func TestPlayerScopeDeniesOtherSpecies(t *testing.T) {
	for _, verb := range []string{`fh.orders(1, 1)`, `fh.report(1, 1)`, `fh.species_stats(1, 1)`} {
		if err := runPlayer(t, 8, verb); err == nil {
			t.Errorf("%s under player scope 8: want denial", verb)
		}
	}
}

// TestGMRequiresSpeciesForPerSpeciesVerbs checks GM scope demands a species id
// for the per-species verbs (orders/report), since there is no implied one.
func TestGMRequiresSpeciesForPerSpeciesVerbs(t *testing.T) {
	for _, verb := range []string{`fh.orders(1)`, `fh.report(1)`} {
		if err := runGM(t, verb); err == nil {
			t.Errorf("%s under gm scope: want error (species required)", verb)
		}
	}
}

// TestGMSpeciesStatsAllSpecies checks GM species_stats with no id returns an
// array covering every species in the roster, ascending.
func TestGMSpeciesStatsAllSpecies(t *testing.T) {
	body := `
		local all = fh.species_stats(1)
		if #all ~= 2 then error("want 2 species, got " .. #all) end
		if all[1].species ~= 1 then error("all[1].species ~= 1") end
		if all[2].species ~= 8 then error("all[2].species ~= 8") end
	`
	if err := runGM(t, body); err != nil {
		t.Fatalf("gm all-species stats: %v", err)
	}
}

// TestSandboxDangerousGlobalsAbsent checks the filesystem/OS/introspection
// libraries and code-loading globals are nil in the sandbox.
func TestSandboxDangerousGlobalsAbsent(t *testing.T) {
	cases := map[string]string{
		"os":              `if os ~= nil then error("os leaked") end`,
		"io":              `if io ~= nil then error("io leaked") end`,
		"debug":           `if debug ~= nil then error("debug leaked") end`,
		"require":         `if require ~= nil then error("require leaked") end`,
		"load":            `if load ~= nil then error("load leaked") end`,
		"loadfile":        `if loadfile ~= nil then error("loadfile leaked") end`,
		"dofile":          `if dofile ~= nil then error("dofile leaked") end`,
		"math.random":     `if math.random ~= nil then error("math.random leaked") end`,
		"math.randomseed": `if math.randomseed ~= nil then error("math.randomseed leaked") end`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if err := runGM(t, body); err != nil {
				t.Errorf("sandbox %s: %v", name, err)
			}
		})
	}
}

// TestSandboxAllowedLibsWork is the positive control: the pure libraries we keep
// (base/string/table/math) remain callable.
func TestSandboxAllowedLibsWork(t *testing.T) {
	body := `
		print("hello")
		if string.upper("ab") ~= "AB" then error("string.upper broken") end
		if math.floor(1.9) ~= 1 then error("math.floor broken") end
	`
	if err := runGM(t, body); err != nil {
		t.Errorf("allowed-libs script: %v", err)
	}
}
