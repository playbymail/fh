// Package gamescript adapts the byte-faithful fhc engine to the scripting
// engine's Engine interface (interface/scripting). It drives the engine exactly
// as the validated shell scripts do — by invoking fhc subcommands — so its
// behavior matches that oracle and internal/game stays frozen (no new public
// surface on the C-port package). The fh engine will provide its own Engine
// implementation in its own terms; both satisfy the same interface.
package gamescript

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/playbymail/fh/interface/scripting"
)

// Adapter implements scripting.Engine.
var _ scripting.Engine = (*Adapter)(nil)

// noordersTemplate is the create-orders "late orders" note the engine inserts
// for species that did not submit; bundled so the adapter is self-contained.
//
//go:embed noorders.txt
var noordersTemplate []byte

// Adapter implements scripting.Engine by shelling out to the fhc binary. Because
// the scripting host runs inside fhc, fhcPath is normally os.Executable().
type Adapter struct {
	fhcPath string
}

// New returns an Adapter that drives the fhc binary at fhcPath.
func New(fhcPath string) *Adapter { return &Adapter{fhcPath: fhcPath} }

// run invokes `fhc <args...>` with cwd set to dir (the engine reads/writes bare
// filenames in its cwd) and any extra environment appended. Combined stdout+
// stderr is returned for diagnostics.
func (a *Adapter) run(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command(a.fhcPath, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// TurnNumber reports galaxy.dat's turn_number for the game in turnDir. A missing
// galaxy.dat means no game (exists=false → the lifecycle "absent" state).
func (a *Adapter) TurnNumber(turnDir string) (int, bool, error) {
	if _, err := os.Stat(filepath.Join(turnDir, "galaxy.dat")); err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	out, err := a.run(turnDir, nil, "show", "turn_number")
	if err != nil {
		return 0, false, fmt.Errorf("show turn_number in %s: %w: %s", turnDir, err, out)
	}
	n, err := parseEngineInt(out)
	if err != nil {
		return 0, false, fmt.Errorf("parse turn_number %q: %w", out, err)
	}
	return n, true, nil
}

// Genesis creates a new game in turnDir, mirroring initialize-ultron-folder.sh:
// create galaxy (seeded), read the radius to pick a conservative home-system
// separation, create the home-system templates, write a valid species.cfg, and
// create the species. It leaves turn_number == 0.
func (a *Adapter) Genesis(turnDir string, seed uint64, numSpecies int) error {
	env := []string{fmt.Sprintf("FH_SEED=%d", seed)}

	if out, err := a.run(turnDir, env, "create", "galaxy", "--species="+strconv.Itoa(numSpecies)); err != nil {
		return fmt.Errorf("create galaxy: %w: %s", err, out)
	}

	out, err := a.run(turnDir, env, "show", "radius")
	if err != nil {
		return fmt.Errorf("show radius: %w: %s", err, out)
	}
	radius, err := parseEngineInt(out)
	if err != nil {
		return fmt.Errorf("parse radius %q: %w", out, err)
	}
	spRadius := radius / 3 // a smaller minimum home separation is always easier to place
	if spRadius < 1 {
		spRadius = 1
	}

	if out, err := a.run(turnDir, env, "create", "home-system-templates"); err != nil {
		return fmt.Errorf("create home-system-templates: %w: %s", err, out)
	}

	if err := os.WriteFile(filepath.Join(turnDir, "species.cfg"), []byte(speciesCfg(numSpecies)), 0o644); err != nil {
		return err
	}

	if out, err := a.run(turnDir, env, "create", "species", "--config=species.cfg", "--radius="+strconv.Itoa(spRadius)); err != nil {
		return fmt.Errorf("create species: %w: %s", err, out)
	}
	return nil
}

// RunTurn resolves the turn in turnDir, mirroring run-this-turn.sh: rebuild the
// flat order namespace from the per-species staging slots (dropping stale orders
// carried forward), drop in the noorders.txt template, then run the canonical
// pipeline through report. finish advances turn_number; report writes the
// spNN.rpt.t<turn> files.
func (a *Adapter) RunTurn(turnDir string) error {
	if err := removeGlob(turnDir, "sp*.ord"); err != nil {
		return err
	}
	if err := stageOrders(turnDir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(turnDir, "noorders.txt"), noordersTemplate, 0o644); err != nil {
		return err
	}

	for _, step := range [][]string{
		{"locations"}, {"create", "orders"}, {"combat"}, {"pre-departure"},
		{"jump"}, {"production"}, {"post-arrival"}, {"finish"}, {"report"},
		{"stats"}, {"turn"},
	} {
		if out, err := a.run(turnDir, nil, step...); err != nil {
			return fmt.Errorf("%s: %w: %s", strings.Join(step, " "), err, out)
		}
	}
	return nil
}

// stageOrders copies each per-species staging slot <turn>/<species>/orders to the
// flat sp<NN>.ord the engine expects. species id is a bare integer.
func stageOrders(turnDir string) error {
	entries, err := os.ReadDir(turnDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id, ok := speciesID(e.Name())
		if !ok {
			continue
		}
		src := filepath.Join(turnDir, e.Name(), "orders")
		if _, err := os.Stat(src); err != nil {
			continue // no staged orders for this species → defaulted by create orders
		}
		dst := filepath.Join(turnDir, fmt.Sprintf("sp%02d.ord", id))
		b, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// speciesCfg builds a valid species.cfg with numSpecies species: unique 5+ char
// names, govt/homeworld within the engine's 5..31 length limits, and tech levels
// summing to < 16 (MA/MI default to 10 each).
func speciesCfg(numSpecies int) string {
	var b strings.Builder
	for i := 1; i <= numSpecies; i++ {
		fmt.Fprintf(&b, "species\n")
		fmt.Fprintf(&b, "    name      Species %02d\n", i)
		fmt.Fprintf(&b, "    homeworld Homeworld %02d\n", i)
		fmt.Fprintf(&b, "    govtname  Government %02d\n", i)
		fmt.Fprintf(&b, "    govttype  Republic\n")
		fmt.Fprintf(&b, "    ml 3\n    gv 1\n    ls 1\n    bi 3\n")
		fmt.Fprintf(&b, "    email     species%02d@example.com\n\n", i)
	}
	return b.String()
}

// speciesID parses a per-species staging-slot dir name as a bare positive
// integer species id.
func speciesID(name string) (int, bool) {
	n, err := strconv.Atoi(name)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// parseEngineInt extracts the integer an `fhc show <field>` prints, tolerating
// surrounding whitespace (mirrors the shell `tr -dc '0-9'`).
func parseEngineInt(out string) (int, error) {
	var digits strings.Builder
	for _, c := range out {
		if c >= '0' && c <= '9' {
			digits.WriteRune(c)
		}
	}
	if digits.Len() == 0 {
		return 0, fmt.Errorf("no integer in %q", out)
	}
	return strconv.Atoi(digits.String())
}

// removeGlob removes files in dir matching pattern (a filepath.Match pattern).
func removeGlob(dir, pattern string) error {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}
	for _, m := range matches {
		if err := os.Remove(m); err != nil {
			return err
		}
	}
	return nil
}
