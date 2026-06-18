package gamescript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/playbymail/fh/interface/scripting"
)

// fhcBinary locates the prebuilt fhc engine the adapter shells out to, skipping
// the test if it has not been built. The adapter drives the real engine exactly
// as the shell scripts do, so this is the oracle-equivalent integration check.
func fhcBinary(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "dist", "local", "fhc"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fhc binary not built at %s — run: go build -o dist/local/fhc ./cmd/fhc", path)
	}
	return path
}

// TestAdapterFullLifecycle drives genesis -> freeze-and-forward -> run-turn
// through the real fhc engine and checks the lifecycle state machine plus the
// player reports at each step.
func TestAdapterFullLifecycle(t *testing.T) {
	eng := New(fhcBinary(t))
	root := t.TempDir()

	if err := scripting.Genesis(eng, root, 12345, 3); err != nil {
		t.Fatalf("Genesis: %v", err)
	}
	if st, err := scripting.TurnStateOf(eng, root, 0); err != nil || st != scripting.TurnResolved {
		t.Fatalf("turn 0 = %s (err %v), want resolved", st, err)
	}

	next, err := scripting.FreezeAndForward(eng, root)
	if err != nil || next != 1 {
		t.Fatalf("FreezeAndForward = %d, %v; want 1, nil", next, err)
	}
	if st, _ := scripting.TurnStateOf(eng, root, 1); st != scripting.TurnPending {
		t.Fatalf("turn 1 after forward = %s, want pending", st)
	}

	resolved, err := scripting.RunTurn(eng, root)
	if err != nil || resolved != 1 {
		t.Fatalf("RunTurn = %d, %v; want 1, nil", resolved, err)
	}
	if st, _ := scripting.TurnStateOf(eng, root, 1); st != scripting.TurnResolved {
		t.Fatalf("turn 1 after run = %s, want resolved", st)
	}

	// report writes one spNN.rpt.t1 per species.
	reports, _ := filepath.Glob(filepath.Join(root, "1", "sp*.rpt.t1"))
	if len(reports) != 3 {
		t.Fatalf("got %d turn-1 reports, want 3", len(reports))
	}

	// Idempotence + ordering guards through the real engine.
	if _, err := scripting.RunTurn(eng, root); err == nil {
		t.Fatal("RunTurn of resolved turn 1: want refusal")
	}
}

// TestAdapterGenesisDeterministic confirms two genesis runs at the same seed
// produce byte-identical galaxy state, matching the shell tool's determinism.
func TestAdapterGenesisDeterministic(t *testing.T) {
	eng := New(fhcBinary(t))
	a, b := t.TempDir(), t.TempDir()
	if err := scripting.Genesis(eng, a, 12345, 3); err != nil {
		t.Fatalf("genesis a: %v", err)
	}
	if err := scripting.Genesis(eng, b, 12345, 3); err != nil {
		t.Fatalf("genesis b: %v", err)
	}
	ga, err := os.ReadFile(filepath.Join(a, "0", "galaxy.dat"))
	if err != nil {
		t.Fatal(err)
	}
	gb, err := os.ReadFile(filepath.Join(b, "0", "galaxy.dat"))
	if err != nil {
		t.Fatal(err)
	}
	if string(ga) != string(gb) {
		t.Fatal("galaxy.dat differs between same-seed genesis runs")
	}
}
