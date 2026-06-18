package scripting

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fakeEngine is an in-memory stand-in for a real engine: it records a turn's
// number in a "turn.txt" marker (the analog of fhc's galaxy.dat), so a byte copy
// of a turn folder carries the number forward exactly as the real engine's .dat
// does. This lets the engine-agnostic lifecycle be tested without fhc.
type fakeEngine struct{}

func (fakeEngine) TurnNumber(turnDir string) (int, bool, error) {
	b, err := os.ReadFile(filepath.Join(turnDir, "turn.txt"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	return n, true, err
}

func (fakeEngine) Genesis(turnDir string, seed uint64, numSpecies int) error {
	if err := os.MkdirAll(turnDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(turnDir, "turn.txt"), []byte("0"), 0o644)
}

func (f fakeEngine) RunTurn(turnDir string) error {
	n, _, err := f.TurnNumber(turnDir)
	if err != nil {
		return err
	}
	// Simulate the pipeline's finish step: turn_number++.
	return os.WriteFile(filepath.Join(turnDir, "turn.txt"), []byte(strconv.Itoa(n+1)), 0o644)
}

func mustState(t *testing.T, eng Engine, root string, n int) TurnState {
	t.Helper()
	st, err := TurnStateOf(eng, root, n)
	if err != nil {
		t.Fatalf("TurnStateOf(%d): %v", n, err)
	}
	return st
}

// TestLifecycleLoop drives genesis -> forward -> run -> forward -> run and
// checks the state machine at every step, mirroring the shell-script oracle.
func TestLifecycleLoop(t *testing.T) {
	eng := fakeEngine{}
	root := t.TempDir()

	if err := Genesis(eng, root, 12345, 4); err != nil {
		t.Fatalf("Genesis: %v", err)
	}
	if got := mustState(t, eng, root, 0); got != TurnResolved {
		t.Fatalf("after genesis, turn 0 = %s, want resolved", got)
	}
	if n, ok, _ := ActiveTurn(root); !ok || n != 0 {
		t.Fatalf("active turn = (%d,%v), want (0,true)", n, ok)
	}

	// Two full cycles.
	for want := 1; want <= 2; want++ {
		next, err := FreezeAndForward(eng, root)
		if err != nil {
			t.Fatalf("FreezeAndForward -> %d: %v", want, err)
		}
		if next != want {
			t.Fatalf("FreezeAndForward returned %d, want %d", next, want)
		}
		if got := mustState(t, eng, root, want); got != TurnPending {
			t.Fatalf("after forward, turn %d = %s, want pending", want, got)
		}
		// The prior turn stays resolved (frozen history).
		if got := mustState(t, eng, root, want-1); got != TurnResolved {
			t.Fatalf("frozen turn %d = %s, want resolved", want-1, got)
		}

		resolved, err := RunTurn(eng, root)
		if err != nil {
			t.Fatalf("RunTurn -> %d: %v", want, err)
		}
		if resolved != want {
			t.Fatalf("RunTurn returned %d, want %d", resolved, want)
		}
		if got := mustState(t, eng, root, want); got != TurnResolved {
			t.Fatalf("after run, turn %d = %s, want resolved", want, got)
		}
	}
}

// TestGuards covers the precondition guards each verb enforces.
func TestGuards(t *testing.T) {
	eng := fakeEngine{}

	t.Run("genesis refuses missing data-root", func(t *testing.T) {
		if err := Genesis(eng, filepath.Join(t.TempDir(), "nope"), 1, 1); err == nil {
			t.Fatal("Genesis on missing data-root: want error")
		}
	})

	t.Run("genesis refuses existing game", func(t *testing.T) {
		root := t.TempDir()
		if err := Genesis(eng, root, 1, 1); err != nil {
			t.Fatalf("Genesis: %v", err)
		}
		if err := Genesis(eng, root, 1, 1); err == nil {
			t.Fatal("second Genesis: want refuse-to-overwrite error")
		}
	})

	t.Run("forward refuses no game", func(t *testing.T) {
		if _, err := FreezeAndForward(eng, t.TempDir()); err == nil {
			t.Fatal("FreezeAndForward with no game: want error")
		}
	})

	t.Run("forward refuses pending active turn", func(t *testing.T) {
		root := t.TempDir()
		if err := Genesis(eng, root, 1, 1); err != nil {
			t.Fatalf("Genesis: %v", err)
		}
		if _, err := FreezeAndForward(eng, root); err != nil { // 0 -> 1 (pending)
			t.Fatalf("first forward: %v", err)
		}
		if _, err := FreezeAndForward(eng, root); err == nil { // 1 is pending
			t.Fatal("forward of pending turn: want error")
		}
	})

	t.Run("run refuses no game", func(t *testing.T) {
		if _, err := RunTurn(eng, t.TempDir()); err == nil {
			t.Fatal("RunTurn with no game: want error")
		}
	})

	t.Run("run refuses already-resolved turn", func(t *testing.T) {
		root := t.TempDir()
		if err := Genesis(eng, root, 1, 1); err != nil {
			t.Fatalf("Genesis: %v", err)
		}
		// turn 0 is resolved; running it must be refused.
		if _, err := RunTurn(eng, root); err == nil {
			t.Fatal("RunTurn of resolved turn: want idempotence error")
		}
	})
}

// TestCopyTreeCarriesNumber confirms a byte copy carries the turn marker forward
// (the property freeze-and-forward relies on), independent of the engine.
func TestCopyTreeCarriesNumber(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "turn.txt"), []byte("7"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "3"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "3", "orders"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "copy")
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "turn.txt"))
	if err != nil || strings.TrimSpace(string(b)) != "7" {
		t.Fatalf("copied turn.txt = %q (err %v), want 7", b, err)
	}
	if _, err := os.Stat(filepath.Join(dst, "3", "orders")); errors.Is(err, fs.ErrNotExist) {
		t.Fatal("copyTree did not copy nested species/orders")
	}
}
