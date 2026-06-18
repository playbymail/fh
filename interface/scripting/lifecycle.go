package scripting

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// The turn lifecycle, engine-agnostic. This is the Go port of the validated
// shell predicate (tools/lib/ultron-lifecycle.sh) and verbs
// (tools/{initialize-ultron-folder,freeze-and-forward,run-this-turn}.sh): the
// same state machine expressed once and driven through the Engine interface. The
// engine-specific work (reading turn_number, creating a game, running a turn)
// lives behind Engine; everything here is filesystem + the predicate. See
// docs/project-ultron/turn-lifecycle.md.

// turns lists the integer-named turn folders under dataRoot, ascending. Unlike
// fh.load{}'s scan (which rejects 0), the lifecycle includes turn 0 — genesis is
// the active turn until it is forwarded.
func turns(dataRoot string) ([]int, error) {
	entries, err := os.ReadDir(dataRoot)
	if err != nil {
		return nil, err
	}
	var ts []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if n, ok := atoiTurn(e.Name()); ok {
			ts = append(ts, n)
		}
	}
	sort.Ints(ts)
	return ts, nil
}

// atoiTurn parses a folder name as a non-negative turn id; non-integer names
// (and the empty string) are rejected, mirroring the shell predicate.
func atoiTurn(name string) (int, bool) {
	if name == "" {
		return 0, false
	}
	for _, c := range name {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(name)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ActiveTurn returns the highest-numbered (active) turn folder under dataRoot.
// ok is false when there are no turn folders (no game). Everything below the
// active turn is frozen history.
func ActiveTurn(dataRoot string) (n int, ok bool, err error) {
	ts, err := turns(dataRoot)
	if err != nil {
		return 0, false, err
	}
	if len(ts) == 0 {
		return 0, false, nil
	}
	return ts[len(ts)-1], true, nil
}

// TurnStateOf computes the lifecycle state of folder N: absent (no game),
// resolved (turn_number == N), pending (turn_number == N-1), or anomalous. It is
// the one predicate every verb consults.
func TurnStateOf(eng Engine, dataRoot string, n int) (TurnState, error) {
	dir := filepath.Join(dataRoot, strconv.Itoa(n))
	tn, exists, err := eng.TurnNumber(dir)
	if err != nil {
		return "", err
	}
	if !exists {
		return TurnAbsent, nil
	}
	switch {
	case tn == n:
		return TurnResolved, nil
	case tn == n-1:
		return TurnPending, nil
	default:
		return TurnAnomalous, nil
	}
}

// Genesis creates turn 0 (the genesis state). The data root must already exist —
// genesis never creates the Ultron folder — and must not already hold a game
// (turn 0 must be absent), the refuse-to-overwrite guard.
func Genesis(eng Engine, dataRoot string, seed uint64, numSpecies int) error {
	if fi, err := os.Stat(dataRoot); err != nil || !fi.IsDir() {
		return fmt.Errorf("data-root %q does not exist — create the Ultron folder first", dataRoot)
	}
	st, err := TurnStateOf(eng, dataRoot, 0)
	if err != nil {
		return err
	}
	if st != TurnAbsent {
		return fmt.Errorf("game already exists in %q — refusing to overwrite", dataRoot)
	}
	dir := filepath.Join(dataRoot, "0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return eng.Genesis(dir, seed, numSpecies)
}

// FreezeAndForward opens the next turn: it requires the active turn N to be
// resolved, then byte-copies N/ to a new N+1/ (which, carrying N's turn_number,
// is born pending). "Freeze" is procedural — once N+1 exists, N is no longer the
// active turn, so the mutating verbs refuse it; no marker is written. Returns the
// new turn number.
func FreezeAndForward(eng Engine, dataRoot string) (next int, err error) {
	n, ok, err := ActiveTurn(dataRoot)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("no game in %q — run genesis first", dataRoot)
	}
	st, err := TurnStateOf(eng, dataRoot, n)
	if err != nil {
		return 0, err
	}
	if st != TurnResolved {
		return 0, fmt.Errorf("turn %d is %s (not yet run) — run-turn before forwarding", n, st)
	}

	next = n + 1
	target := filepath.Join(dataRoot, strconv.Itoa(next))
	if _, err := os.Stat(target); err == nil {
		return 0, fmt.Errorf("turn %d already exists at %s — refusing to overwrite", next, target)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}

	if err := copyTree(filepath.Join(dataRoot, strconv.Itoa(n)), target); err != nil {
		return 0, err
	}

	st, err = TurnStateOf(eng, dataRoot, next)
	if err != nil {
		return 0, err
	}
	if st != TurnPending {
		return 0, fmt.Errorf("internal: turn %d is %s after forward, want pending", next, st)
	}
	return next, nil
}

// RunTurn resolves the active turn: it requires the active turn N to be pending,
// runs the engine's pipeline in N/ (which stages orders and advances the turn),
// and verifies N is resolved afterward. Re-running a resolved turn is refused —
// the idempotence guard. Returns the resolved turn number.
func RunTurn(eng Engine, dataRoot string) (resolved int, err error) {
	n, ok, err := ActiveTurn(dataRoot)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("no game in %q — run genesis first", dataRoot)
	}
	st, err := TurnStateOf(eng, dataRoot, n)
	if err != nil {
		return 0, err
	}
	if st != TurnPending {
		if st == TurnResolved {
			return 0, fmt.Errorf("turn %d is already resolved — freeze-and-forward to open the next turn", n)
		}
		return 0, fmt.Errorf("turn %d is %s — must be pending to run", n, st)
	}

	if err := eng.RunTurn(filepath.Join(dataRoot, strconv.Itoa(n))); err != nil {
		return 0, err
	}

	st, err = TurnStateOf(eng, dataRoot, n)
	if err != nil {
		return 0, err
	}
	if st != TurnResolved {
		return 0, fmt.Errorf("internal: turn %d is %s after run, want resolved", n, st)
	}
	return n, nil
}

// copyTree recursively byte-copies the directory src to dst, preserving file
// permission bits. The engine never sees folder names, so the copied flat
// working dir is identical to one produced in place (parity-safe).
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
