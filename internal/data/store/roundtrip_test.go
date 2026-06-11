package store_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/playbymail/fh/internal/data/store"
	"github.com/playbymail/fh/internal/model"
)

// TestRoundTripAllBackends builds a representative World, persists it through
// each store backend, reloads it, and asserts the reloaded World is identical
// to the original. This proves all three backends serialize model.World
// losslessly (the engine renders reports from the loaded World, so a lossless
// round-trip is the foundation of cross-backend report parity).
func TestRoundTripAllBackends(t *testing.T) {
	ctx := context.Background()
	const gameID = "alpha"

	for _, typ := range []store.StoreType{store.StoreBinary, store.StoreJSON, store.StoreSQLite} {
		t.Run(string(typ), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "game")
			want := buildWorld()

			st, err := store.Create(ctx, dir, store.CreateOptions{ID: gameID, Type: typ})
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if err := st.IngestWorld(ctx, gameID, want); err != nil {
				t.Fatalf("ingest: %v", err)
			}
			if err := st.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}

			st2, err := store.Open(ctx, dir)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer st2.Close()
			got, err := st2.LoadWorld(ctx, gameID)
			if err != nil {
				t.Fatalf("load: %v", err)
			}

			if wj, gj := dump(t, want), dump(t, got); wj != gj {
				t.Errorf("round-trip mismatch for %s store\n--- want ---\n%s\n--- got ---\n%s", typ, wj, gj)
			}
		})
	}
}

// dump normalizes a World (clearing derived back-references that Resolve
// rebuilds) and renders it as indented JSON for comparison.
func dump(t *testing.T, w *model.World) string {
	t.Helper()
	for _, s := range w.Systems {
		for _, p := range s.Planets {
			p.System = nil
		}
	}
	for _, sp := range w.Species {
		for _, np := range sp.Namplas {
			np.Planet = nil
		}
	}
	b, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		t.Fatalf("marshal world: %v", err)
	}
	return string(b)
}

// buildWorld returns a small but fully-populated World exercising every model
// field that the stores persist. Star Type/Color use values whose char
// encodings are unique (the JSON store encodes them as single chars), and all
// scalars stay within the binary record field widths.
func buildWorld() *model.World {
	w := &model.World{
		Galaxy: model.Galaxy{TurnNumber: 3, NumSpecies: 2, DNumSpecies: 4, Radius: 12},
	}

	sys1 := &model.System{
		X: 5, Y: 6, Z: 7, Type: 2, Color: 3, Size: 4,
		HomeSystem: true, WormHere: false, Message: 11,
	}
	model.SetSpeciesBit(&sys1.VisitedBy, 1)
	model.SetSpeciesBit(&sys1.VisitedBy, 2)
	sys1.Planets = []*model.Planet{
		{System: sys1, Orbit: 1, TemperatureClass: 30, PressureClass: 2, Special: 0,
			Gas: [4]int{1, 5, 0, 0}, GasPercent: [4]int{53, 47, 0, 0},
			Diameter: 13, Gravity: 99, MiningDifficulty: 106, MdIncrease: 3, EconEfficiency: 100, Message: 0},
		{System: sys1, Orbit: 2, TemperatureClass: 40, PressureClass: 1, Special: 1,
			Gas: [4]int{5, 0, 0, 0}, GasPercent: [4]int{100, 0, 0, 0},
			Diameter: 9, Gravity: 50, MiningDifficulty: 220, MdIncrease: 0, EconEfficiency: 0, Message: 7},
	}

	sys2 := &model.System{
		X: 8, Y: 9, Z: 10, Type: 1, Color: 5, Size: 3,
		WormHere: true, WormX: 5, WormY: 6, WormZ: 7, Message: 0,
	}
	sys2.Planets = []*model.Planet{
		{System: sys2, Orbit: 1, TemperatureClass: 25, PressureClass: 3, Special: 0,
			Gas: [4]int{3, 9, 0, 0}, GasPercent: [4]int{20, 80, 0, 0},
			Diameter: 11, Gravity: 72, MiningDifficulty: 150, MdIncrease: 1, EconEfficiency: 88, Message: 0},
	}
	w.Systems = []*model.System{sys1, sys2}

	sp1 := &model.Species{
		ID: 1, Name: "Humans", GovtName: "Earth Federation", GovtType: "Democracy",
		X: 5, Y: 6, Z: 7, PN: 1,
		RequiredGas: 5, RequiredGasMin: 20, RequiredGasMax: 80,
		NeutralGas: [6]int{1, 2, 3, 0, 0, 0}, PoisonGas: [6]int{9, 0, 0, 0, 0, 0},
		AutoOrders:     true,
		TechLevel:      [6]int{10, 12, 8, 5, 7, 6},
		InitTechLevel:  [6]int{10, 10, 5, 1, 1, 1},
		TechKnowledge:  [6]int{15, 15, 9, 6, 8, 7},
		TechEps:        [6]int{120, 340, 50, 0, 10, 0},
		HpOriginalBase: 1000, EconUnits: 250, FleetCost: 42, FleetPercentCost: 1234,
		NumNamplas: 1, NumShips: 1,
		Log: []byte("Turn 3 events for Humans\n"),
	}
	model.SetSpeciesBit(&sp1.Contact, 2)
	model.SetSpeciesBit(&sp1.Ally, 2)
	np := &model.Nampla{
		Name: "Earth", X: 5, Y: 6, Z: 7, PN: 1,
		Status: model.HomePlanet | model.Populated, SiegeEff: 0, Shipyards: 2,
		IUsNeeded: 3, AUsNeeded: 4, AutoIUs: 1, AutoAUs: 2, IUsToInstall: 5, AUsToInstall: 6,
		MiBase: 100, MaBase: 90, PopUnits: 500, UseOnAmbush: 0, Message: 0, Special: 0,
	}
	np.ItemQuantity[model.IU] = 12
	np.ItemQuantity[model.AU] = 7
	sp1.Namplas = []*model.Nampla{np}
	sh := &model.Ship{
		Name: "Enterprise", X: 5, Y: 6, Z: 7, PN: 1,
		Status: model.InOrbit, ShipType: model.FTL, Class: model.TR, Tonnage: 5,
		Age: 1, RemainingCost: 0, LoadingPoint: 0, UnloadingPoint: 0,
	}
	sh.ItemQuantity[model.CU] = 3
	sp1.Ships = []*model.Ship{sh}

	sp2 := &model.Species{
		ID: 2, Name: "Vulcans", GovtName: "High Command", GovtType: "Logic",
		X: 8, Y: 9, Z: 10, PN: 1,
		RequiredGas: 5, RequiredGasMin: 10, RequiredGasMax: 90,
		TechLevel:      [6]int{8, 8, 8, 8, 8, 8},
		HpOriginalBase: 800, EconUnits: 100,
		NumNamplas: 1, NumShips: 0,
	}
	np2 := &model.Nampla{
		Name: "Vulcan", X: 8, Y: 9, Z: 10, PN: 1,
		Status: model.HomePlanet | model.Populated, MiBase: 80, MaBase: 80, PopUnits: 400,
	}
	sp2.Namplas = []*model.Nampla{np2}
	w.Species = []*model.Species{sp1, sp2}

	w.Locations = []model.Location{
		{S: 1, X: 5, Y: 6, Z: 7},
		{S: 2, X: 8, Y: 9, Z: 10},
	}

	w.Resolve()
	return w
}
