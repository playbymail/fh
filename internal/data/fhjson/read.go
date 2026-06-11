package fhjson

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/playbymail/fh/internal/model"
)

// ReadWorld reads a Far Horizons JSON export directory into a model.World.
// The directory must contain galaxy.json and systems.json; species.%03d.json,
// sp%02d.log, and locations.dat are read when present.
func ReadWorld(dir string) (*model.World, error) {
	w := &model.World{}

	var gx galaxyFile
	if err := readJSON(filepath.Join(dir, "galaxy.json"), &gx); err != nil {
		return nil, err
	}
	w.Galaxy = model.Galaxy{
		TurnNumber:  gx.Galaxy.TurnNumber,
		NumSpecies:  gx.Galaxy.NumSpecies,
		DNumSpecies: gx.Galaxy.DNumSpecies,
		Radius:      gx.Galaxy.Radius,
	}

	var sys systemsFile
	if err := readJSON(filepath.Join(dir, "systems.json"), &sys); err != nil {
		return nil, err
	}
	for _, js := range sys.Systems {
		s := &model.System{
			X: js.X, Y: js.Y, Z: js.Z,
			Type: charToType(js.Type), Color: charToColor(js.Color), Size: js.Size,
			HomeSystem: js.HomeSystem,
			WormHere:   js.WormHere, WormX: js.WormX, WormY: js.WormY, WormZ: js.WormZ,
			Message: js.Message,
		}
		for _, n := range js.VisitedBy {
			model.SetSpeciesBit(&s.VisitedBy, n)
		}
		for i, jp := range js.Planets {
			p := &model.Planet{
				System: s, Orbit: i + 1,
				TemperatureClass: jp.TemperatureClass, PressureClass: jp.PressureClass, Special: jp.Special,
				Diameter: jp.Diameter, Gravity: jp.Gravity,
				MiningDifficulty: jp.MiningDifficulty.Base, MdIncrease: jp.MiningDifficulty.Increase,
				EconEfficiency: jp.EconEfficiency, Message: jp.Message,
			}
			for g := 0; g < 4 && g < len(jp.Atmosphere); g++ {
				p.Gas[g] = jp.Atmosphere[g].Code
				p.GasPercent[g] = jp.Atmosphere[g].Percent
			}
			s.Planets = append(s.Planets, p)
		}
		w.Systems = append(w.Systems, s)
	}

	for n := 1; n <= w.Galaxy.NumSpecies; n++ {
		path := filepath.Join(dir, fmt.Sprintf("species.%03d.json", n))
		var sf speciesFile
		if err := readJSON(path, &sf); err != nil {
			return nil, err
		}
		sp := buildSpecies(&sf)
		// Carry the accumulated event log verbatim, if present.
		logBytes, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("sp%02d.log", n)))
		if err == nil {
			sp.Log = logBytes
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		w.Species = append(w.Species, sp)
	}

	// Locations index (4-byte records: species, x, y, z). Absent for setup
	// states; produces an empty "Aliens at" section, which is correct then.
	if locs, err := os.ReadFile(filepath.Join(dir, "locations.dat")); err == nil {
		for i := 0; i+4 <= len(locs); i += 4 {
			w.Locations = append(w.Locations, model.Location{
				S: int(locs[i]), X: int(locs[i+1]), Y: int(locs[i+2]), Z: int(locs[i+3]),
			})
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	w.Resolve()
	return w, nil
}

// buildSpecies converts a parsed species file into the domain model.
func buildSpecies(sf *speciesFile) *model.Species {
	s := sf.Species
	sp := &model.Species{
		ID: s.ID, Name: s.Name, GovtName: s.Government.Name, GovtType: s.Government.Type,
		X: s.HomeWorld.X, Y: s.HomeWorld.Y, Z: s.HomeWorld.Z, PN: s.HomeWorld.Orbit,
		RequiredGas: s.Atmosphere.Required.Code, RequiredGasMin: s.Atmosphere.Required.MinPct, RequiredGasMax: s.Atmosphere.Required.MaxPct,
		AutoOrders:     s.AutoOrders,
		HpOriginalBase: s.HpOriginalBase, EconUnits: s.EconUnits, FleetCost: s.FleetCost, FleetPercentCost: s.FleetPercentCost,
		NumNamplas: len(sf.NamedPlanets), NumShips: len(sf.Ships),
	}
	copyN(sp.NeutralGas[:], s.Atmosphere.Neutral)
	copyN(sp.PoisonGas[:], s.Atmosphere.Poison)
	for key, t := range s.Tech {
		if i, ok := techKeys[key]; ok {
			sp.TechLevel[i] = t.Level
			sp.TechKnowledge[i] = t.Knowledge
			sp.TechEps[i] = t.XP
			sp.InitTechLevel[i] = t.InitLevel
		}
	}
	for _, c := range s.Contacts {
		model.SetSpeciesBit(&sp.Contact, c)
	}
	for _, a := range s.Allies {
		model.SetSpeciesBit(&sp.Ally, a)
	}
	for _, e := range s.Enemies {
		model.SetSpeciesBit(&sp.Enemy, e)
	}

	for _, jn := range sf.NamedPlanets {
		np := &model.Nampla{
			Name: jn.Name, X: jn.Location.X, Y: jn.Location.Y, Z: jn.Location.Z, PN: jn.Location.Orbit,
			Status: jn.Status, Hiding: jn.Hiding, Hidden: jn.Hidden, SiegeEff: jn.SiegeEff, Shipyards: jn.Shipyards,
			IUsNeeded: jn.IUs.Needed, AutoIUs: jn.IUs.Auto, IUsToInstall: jn.IUs.Install,
			AUsNeeded: jn.AUs.Needed, AutoAUs: jn.AUs.Auto, AUsToInstall: jn.AUs.Install,
			MiBase: jn.MiBase, MaBase: jn.MaBase, PopUnits: jn.PopUnits,
			UseOnAmbush: jn.UseOnAmbush, Message: jn.Message, Special: jn.Special,
		}
		for _, it := range jn.Items {
			if it.Code >= 0 && it.Code < model.MaxItems {
				np.ItemQuantity[it.Code] = it.Qty
			}
		}
		sp.Namplas = append(sp.Namplas, np)
	}

	for _, js := range sf.Ships {
		sh := &model.Ship{
			Name: js.Name, X: js.Location.X, Y: js.Location.Y, Z: js.Location.Z, PN: js.Location.Orbit,
			Status: js.Status, ShipType: js.Type, DestX: js.Dest.X, DestY: js.Dest.Y, DestZ: js.Dest.Z,
			JustJumped: boolToInt(js.JustJumped), ArrivedViaWormhole: boolToInt(js.ArrivedViaWormhole),
			Class: js.Class, Tonnage: js.Tonnage, Age: js.Age, RemainingCost: js.RemainingCost,
			LoadingPoint: js.LoadingPoint, UnloadingPoint: js.UnloadingPoint, Special: js.Special,
		}
		for _, it := range js.Cargo {
			if it.Code >= 0 && it.Code < model.MaxItems {
				sh.ItemQuantity[it.Code] = it.Qty
			}
		}
		sp.Ships = append(sp.Ships, sh)
	}

	return sp
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func copyN(dst, src []int) {
	for i := range dst {
		if i < len(src) {
			dst[i] = src[i]
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
