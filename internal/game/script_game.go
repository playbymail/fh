package game

// fhcGame implements scripting.Game for the byte-faithful fhc engine. It serves
// read-only queries over the Ultron data-root layout — integer-named turn
// folders, each a flat engine working directory (galaxy.dat, sp%02d.dat,
// sp%02d.ord, sp%02d.rpt.t<turn>). Access policy (which species a caller may see)
// is the host's concern; this type serves any (turn, species) it is asked for.
//
// All methods are read-only and never call rnd(), so a script is byte-neutral on
// the golden trees and cannot perturb the PRNG stream.

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/playbymail/fh/interface/scripting"
)

// fhcGame answers scripting queries against a data-root of turn folders.
type fhcGame struct {
	dataRoot string
}

// newFHCGame returns a Game serving the data-root at dataRoot.
func newFHCGame(dataRoot string) *fhcGame { return &fhcGame{dataRoot: dataRoot} }

var _ scripting.Game = (*fhcGame)(nil)

// turnDir is the working directory for turn n.
func (g *fhcGame) turnDir(n int) string {
	return filepath.Join(g.dataRoot, strconv.Itoa(n))
}

// galaxyMeta reads the four ints in a turn's galaxy.dat without disturbing engine
// globals or the PRNG. exists is false when the turn has no galaxy.dat.
func (g *fhcGame) galaxyMeta(n int) (numSpecies, turnNumber int, exists bool, err error) {
	data, err := os.ReadFile(filepath.Join(g.turnDir(n), "galaxy.dat"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, false, nil
		}
		return 0, 0, false, err
	}
	if len(data) < binary_galaxy_data_size {
		return 0, 0, false, fmt.Errorf("turn %d: galaxy.dat is truncated", n)
	}
	// Layout (see get_galaxy_data): d_num_species@0, num_species@4, radius@8,
	// turn_number@12.
	numSpecies = int(int32(binary.LittleEndian.Uint32(data[4:])))
	turnNumber = int(int32(binary.LittleEndian.Uint32(data[12:])))
	return numSpecies, turnNumber, true, nil
}

// turnFolders lists the positive-integer turn folders under the data-root,
// ascending. Turn 0 (genesis) is not an Ultron-addressable turn and is omitted.
func (g *fhcGame) turnFolders() ([]int, error) {
	entries, err := os.ReadDir(g.dataRoot)
	if err != nil {
		return nil, err
	}
	var turns []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if n, err := strconv.Atoi(e.Name()); err == nil && n > 0 {
			turns = append(turns, n)
		}
	}
	sort.Ints(turns)
	return turns, nil
}

// CurrentTurn returns the highest-numbered (active) turn folder.
func (g *fhcGame) CurrentTurn() (int, error) {
	turns, err := g.turnFolders()
	if err != nil {
		return 0, err
	}
	if len(turns) == 0 {
		return 0, fmt.Errorf("no turns in %q", g.dataRoot)
	}
	return turns[len(turns)-1], nil
}

// TurnStatus reports whether turn n has been resolved. The lifecycle predicate is
// resolved ⟺ galaxy.dat's turn_number == n; otherwise the turn is pending.
func (g *fhcGame) TurnStatus(turn int) (string, error) {
	_, turnNumber, exists, err := g.galaxyMeta(turn)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("turn %d does not exist", turn)
	}
	if turnNumber == turn {
		return scripting.TurnResolved, nil
	}
	return scripting.TurnPending, nil
}

// SpeciesIDs returns the roster 1..num_species, read from the current turn's
// galaxy.dat (the roster is fixed at galaxy creation, so any turn agrees).
func (g *fhcGame) SpeciesIDs() ([]int, error) {
	n, err := g.CurrentTurn()
	if err != nil {
		return nil, err
	}
	numSpecies, _, exists, err := g.galaxyMeta(n)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("turn %d has no galaxy.dat", n)
	}
	ids := make([]int, numSpecies)
	for i := range ids {
		ids[i] = i + 1
	}
	return ids, nil
}

// Orders returns the entire orders file for (turn, species): the flat
// sp%02d.ord the engine consumes, or the staged <species>/orders slot if the
// flat file is absent. ok is false when neither exists.
func (g *fhcGame) Orders(turn, species int) (string, bool, error) {
	candidates := []string{
		filepath.Join(g.turnDir(turn), fmt.Sprintf("sp%02d.ord", species)),
		filepath.Join(g.turnDir(turn), strconv.Itoa(species), "orders"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data), true, nil
		}
		if !os.IsNotExist(err) {
			return "", false, err
		}
	}
	return "", false, nil
}

// Report returns the entire turn report for (turn, species). It is an error if
// the turn has not been resolved. The report filename's t<turn> suffix is the
// galaxy turn_number, which equals the folder number once resolved.
func (g *fhcGame) Report(turn, species int) (string, error) {
	_, turnNumber, exists, err := g.galaxyMeta(turn)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("turn %d does not exist", turn)
	}
	if turnNumber != turn {
		return "", fmt.Errorf("turn %d is not resolved", turn)
	}
	path := filepath.Join(g.turnDir(turn), fmt.Sprintf("sp%02d.rpt.t%d", species, turnNumber))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("turn %d: no report for species %d", turn, species)
		}
		return "", err
	}
	return string(data), nil
}

// SpeciesStats loads the turn's state into engine globals and computes the
// per-species statistics on demand. It validates the turn and species, then
// chdirs into the turn dir (restoring cwd afterward) and runs the unmodified
// loaders — read-only, so RNG-neutral.
func (g *fhcGame) SpeciesStats(turn, species int) (scripting.SpeciesStats, error) {
	numSpecies, _, exists, err := g.galaxyMeta(turn)
	if err != nil {
		return scripting.SpeciesStats{}, err
	}
	if !exists {
		return scripting.SpeciesStats{}, fmt.Errorf("turn %d does not exist", turn)
	}
	if species < 1 || species > numSpecies {
		return scripting.SpeciesStats{}, fmt.Errorf("species %d out of range (1..%d)", species, numSpecies)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return scripting.SpeciesStats{}, err
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(g.turnDir(turn)); err != nil {
		return scripting.SpeciesStats{}, err
	}

	ResetState()
	get_galaxy_data()
	get_planet_data()
	get_species_data()

	if data_in_memory[species-1] == FALSE {
		return scripting.SpeciesStats{}, fmt.Errorf("turn %d: species %d not present", turn, species)
	}
	return computeSpeciesStats(species), nil
}

// computeSpeciesStats mirrors the per-species block of statsCommand, returning
// the figures in structured form instead of printing them. It must stay aligned
// with stats.go; script_game_test.go asserts the two agree on a golden turn.
func computeSpeciesStats(spNo int) scripting.SpeciesStats {
	sp := &spec_data[spNo-1]
	namplas := namp_data[spNo-1]
	ships := ship_data[spNo-1]

	fleetPercentCost := sp.fleet_percent_cost
	if fleetPercentCost > 10000 {
		fleetPercentCost = 10000
	}

	totalProduction := 0
	totalDefensivePower := 0
	numYards := 0
	numPopPlanets := 0
	homePlanet := planet_base[namplas[0].planet_index]

	for ni := 0; ni < sp.num_namplas; ni++ {
		np := namplas[ni]
		if np.pn == 99 {
			continue
		}
		numYards += np.shipyards
		pl := planet_base[np.planet_index]

		rawMaterialUnits := (10 * sp.tech_level[MI] * np.mi_base) / pl.mining_difficulty
		productionCapacity := (sp.tech_level[MA] * np.ma_base) / 10

		lsNeeded := life_support_needed(sp, homePlanet, pl)
		productionPenalty := 0
		if lsNeeded != 0 {
			productionPenalty = (100 * lsNeeded) / sp.tech_level[LS]
		}

		rawMaterialUnits -= (productionPenalty * rawMaterialUnits) / 100
		rawMaterialUnits = ((pl.econ_efficiency * rawMaterialUnits) + 50) / 100
		productionCapacity -= (productionPenalty * productionCapacity) / 100
		productionCapacity = ((pl.econ_efficiency * productionCapacity) + 50) / 100

		var n1 int
		switch {
		case np.status&MINING_COLONY != 0:
			n1 = (2 * rawMaterialUnits) / 3
		case np.status&RESORT_COLONY != 0:
			n1 = (2 * productionCapacity) / 3
		case productionCapacity > rawMaterialUnits:
			n1 = rawMaterialUnits
		default:
			n1 = productionCapacity
		}

		n2 := ((fleetPercentCost * n1) + 5000) / 10000
		totalProduction += n1 - n2

		tons := np.item_quantity[PD] / 200
		if tons < 1 && np.item_quantity[PD] > 0 {
			tons = 1
		}
		totalDefensivePower += power(tons)

		if np.status&POPULATED != 0 {
			numPopPlanets++
		}
	}

	numShips := 0
	totalOffensivePower := 0
	for si := 0; si < sp.num_ships; si++ {
		s := ships[si]
		if s.pn == 99 || s.status == UNDER_CONSTRUCTION {
			continue
		}
		numShips++
		switch {
		case s.ship_type == STARBASE:
			totalDefensivePower += power(s.tonnage)
		case s.class == TR:
			// transport: contributes no combat power
		case s.ship_type == SUB_LIGHT:
			totalDefensivePower += power(s.tonnage)
		default:
			totalOffensivePower += power(s.tonnage)
		}
	}

	totalOffensivePower += (sp.tech_level[ML] * totalOffensivePower) / 50
	totalDefensivePower += (sp.tech_level[ML] * totalDefensivePower) / 50
	if sp.tech_level[ML] == 0 {
		totalDefensivePower = 0
		totalOffensivePower = 0
	}
	totalOffensivePower /= 10
	totalDefensivePower /= 10

	return scripting.SpeciesStats{
		Species: spNo,
		Name:    sp.name,
		Tech: map[string]int{
			"MI": sp.tech_level[MI],
			"MA": sp.tech_level[MA],
			"ML": sp.tech_level[ML],
			"GV": sp.tech_level[GV],
			"LS": sp.tech_level[LS],
			"BI": sp.tech_level[BI],
		},
		TotalProduction: totalProduction,
		NumPlanets:      numPopPlanets,
		NumShips:        numShips,
		NumShipyards:    numYards,
		OffensivePower:  totalOffensivePower,
		DefensivePower:  totalDefensivePower,
		EconUnits:       sp.econ_units,
	}
}
