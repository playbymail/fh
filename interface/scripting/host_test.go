package scripting

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

// stageDataRoot stages a fake data-root: turn dirs 1/2/3, each with species
// subdirs 1 and 8 (directories only — the scan reads no game state), plus decoy
// non-integer / plain entries to confirm they are ignored.
func stageDataRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, turn := range []int{1, 2, 3} {
		for _, sp := range []int{1, 8} {
			if err := os.MkdirAll(filepath.Join(root, strconv.Itoa(turn), strconv.Itoa(sp)), 0o755); err != nil {
				t.Fatalf("stage data-root: %v", err)
			}
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "notaturn"), 0o755); err != nil {
		t.Fatalf("stage decoy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "galaxy.dat"), []byte("x"), 0o644); err != nil {
		t.Fatalf("stage decoy file: %v", err)
	}
	return root
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

// TestScanGMFullRoster checks the pure scan() helper: integer turn dirs ascending
// and the ascending union of species across turns; decoys ignored.
func TestScanGMFullRoster(t *testing.T) {
	root := stageDataRoot(t)
	h := NewHost(fakeEngine{}, root, ScopeGM, 0)

	turns, species, err := h.scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if want := []int{1, 2, 3}; !reflect.DeepEqual(turns, want) {
		t.Errorf("turns = %v, want %v", turns, want)
	}
	if want := []int{1, 8}; !reflect.DeepEqual(species, want) {
		t.Errorf("species = %v, want %v", species, want)
	}
}

// TestScanUnionAcrossTurns checks a species present in only one turn dir still
// appears in the roster, ascending.
func TestScanUnionAcrossTurns(t *testing.T) {
	root := stageDataRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "2", "5"), 0o755); err != nil {
		t.Fatalf("stage extra species: %v", err)
	}
	h := NewHost(fakeEngine{}, root, ScopeGM, 0)
	_, species, err := h.scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if want := []int{1, 5, 8}; !reflect.DeepEqual(species, want) {
		t.Errorf("species = %v, want %v (union)", species, want)
	}
}

// TestScanMissingDataRoot checks a missing data root is an error.
func TestScanMissingDataRoot(t *testing.T) {
	h := NewHost(fakeEngine{}, filepath.Join(t.TempDir(), "nope"), ScopeGM, 0)
	if _, _, err := h.scan(); err == nil {
		t.Fatal("scan of missing data-root: want error")
	}
}

// TestGMHandleScopeGating checks fh.gm() is a usable handle under GM scope and
// nil under player scope (so a player script cannot reach the mutating verbs).
func TestGMHandleScopeGating(t *testing.T) {
	root := stageDataRoot(t)

	gmScript := writeScript(t, `
		local gm = fh.gm()
		if gm == nil then error("gm handle should be non-nil under GM scope") end
		if type(gm.run_turn) ~= "function" then error("gm.run_turn missing") end
	`)
	if err := NewHost(fakeEngine{}, root, ScopeGM, 0).Run(gmScript); err != nil {
		t.Fatalf("gm-scope script: %v", err)
	}

	playerScript := writeScript(t, `
		if fh.gm() ~= nil then error("gm handle must be nil under player scope") end
	`)
	if err := NewHost(fakeEngine{}, root, ScopePlayer, 8).Run(playerScript); err != nil {
		t.Fatalf("player-scope script: %v", err)
	}
}

// TestGMLifecycleVerbs drives the GM lifecycle verbs from Lua against the fake
// engine: genesis -> freeze_and_forward -> run_turn, checking the returned turn
// numbers. A fresh data-root (no genesis turn yet) is used.
func TestGMLifecycleVerbs(t *testing.T) {
	root := t.TempDir()
	script := writeScript(t, `
		local gm = fh.gm()
		gm:genesis{ seed = 12345, species = 3 }
		local n = gm:freeze_and_forward()
		if n ~= 1 then error("freeze_and_forward returned " .. n .. ", want 1") end
		local r = gm:run_turn()
		if r ~= 1 then error("run_turn returned " .. r .. ", want 1") end
	`)
	if err := NewHost(fakeEngine{}, root, ScopeGM, 0).Run(script); err != nil {
		t.Fatalf("gm lifecycle script: %v", err)
	}
}

// TestGMVerbErrorsPropagate checks a guard violation in a GM verb surfaces as a
// Lua error (forwarding a pending turn).
func TestGMVerbErrorsPropagate(t *testing.T) {
	root := t.TempDir()
	script := writeScript(t, `
		local gm = fh.gm()
		gm:genesis{ seed = 1, species = 1 }
		gm:freeze_and_forward()   -- turn 1, pending
		gm:freeze_and_forward()   -- pending -> must error
	`)
	if err := NewHost(fakeEngine{}, root, ScopeGM, 0).Run(script); err == nil {
		t.Fatal("forwarding a pending turn from Lua: want error")
	}
}
