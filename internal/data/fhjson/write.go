package fhjson

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/playbymail/fh/internal/model"
)

// WriteWorld writes a model.World to dir as the Far Horizons JSON export:
// galaxy.json, systems.json, species.%03d.json, plus sp%02d.log sidecars and
// the locations.dat index. dir must already exist.
func WriteWorld(dir string, w *model.World) error {
	gx := galaxyFile{
		Version: 1,
		Galaxy: galaxyJSON{
			TurnNumber:  w.Galaxy.TurnNumber,
			NumSpecies:  w.Galaxy.NumSpecies,
			DNumSpecies: w.Galaxy.DNumSpecies,
			Radius:      w.Galaxy.Radius,
		},
	}
	if err := writeJSON(filepath.Join(dir, "galaxy.json"), gx); err != nil {
		return err
	}

	var sys systemsFile
	sys.Systems = make([]systemJSON, 0, len(w.Systems))
	for _, s := range w.Systems {
		js := systemJSON{
			X: s.X, Y: s.Y, Z: s.Z,
			Type: typeToChar(s.Type), Color: colorToChar(s.Color), Size: s.Size,
			HomeSystem: s.HomeSystem,
			WormHere:   s.WormHere, WormX: s.WormX, WormY: s.WormY, WormZ: s.WormZ,
			VisitedBy: bitfieldToList(s.VisitedBy),
			Message:   s.Message,
		}
		for _, p := range s.Planets {
			pj := planetJSON{
				TemperatureClass: p.TemperatureClass, PressureClass: p.PressureClass, Special: p.Special,
				Diameter: p.Diameter, Gravity: p.Gravity,
				EconEfficiency: p.EconEfficiency, Message: p.Message,
			}
			pj.MiningDifficulty.Base = p.MiningDifficulty
			pj.MiningDifficulty.Increase = p.MdIncrease
			pj.Atmosphere = make([]gasJSON, 4)
			for g := 0; g < 4; g++ {
				pj.Atmosphere[g] = gasJSON{Code: p.Gas[g], Percent: p.GasPercent[g]}
			}
			js.Planets = append(js.Planets, pj)
		}
		sys.Systems = append(sys.Systems, js)
	}
	if err := writeJSON(filepath.Join(dir, "systems.json"), sys); err != nil {
		return err
	}

	for _, sp := range w.Species {
		sf := buildSpeciesFile(sp)
		if err := writeJSON(filepath.Join(dir, fmt.Sprintf("species.%03d.json", sp.ID)), sf); err != nil {
			return err
		}
		logPath := filepath.Join(dir, fmt.Sprintf("sp%02d.log", sp.ID))
		if len(sp.Log) > 0 {
			if err := os.WriteFile(logPath, sp.Log, 0o644); err != nil {
				return err
			}
		} else {
			// Keep the directory consistent: an absent log reads back as nil.
			_ = os.Remove(logPath)
		}
	}

	return writeLocations(filepath.Join(dir, "locations.dat"), w.Locations)
}

// buildSpeciesFile converts a domain species into its JSON form.
func buildSpeciesFile(sp *model.Species) speciesFile {
	var sf speciesFile
	s := &sf.Species
	s.ID = sp.ID
	s.Name = sp.Name
	s.Government.Name = sp.GovtName
	s.Government.Type = sp.GovtType
	s.HomeWorld.X, s.HomeWorld.Y, s.HomeWorld.Z, s.HomeWorld.Orbit = sp.X, sp.Y, sp.Z, sp.PN
	s.Atmosphere.Required.Code = sp.RequiredGas
	s.Atmosphere.Required.MinPct = sp.RequiredGasMin
	s.Atmosphere.Required.MaxPct = sp.RequiredGasMax
	s.Atmosphere.Neutral = intSlice(sp.NeutralGas[:])
	s.Atmosphere.Poison = intSlice(sp.PoisonGas[:])
	s.AutoOrders = sp.AutoOrders
	s.Tech = make(map[string]techJSON, 6)
	for i := 0; i < 6; i++ {
		s.Tech[techKeyByIndex[i]] = techJSON{
			Level:     sp.TechLevel[i],
			Knowledge: sp.TechKnowledge[i],
			XP:        sp.TechEps[i],
			InitLevel: sp.InitTechLevel[i],
		}
	}
	s.HpOriginalBase = sp.HpOriginalBase
	s.EconUnits = sp.EconUnits
	s.FleetCost = sp.FleetCost
	s.FleetPercentCost = sp.FleetPercentCost
	s.Contacts = bitfieldToList(sp.Contact)
	s.Allies = bitfieldToList(sp.Ally)
	s.Enemies = bitfieldToList(sp.Enemy)

	sf.NamedPlanets = make([]namplaJSON, 0, len(sp.Namplas))
	for _, np := range sp.Namplas {
		jn := namplaJSON{
			Name:     np.Name,
			Location: coordOrbitJSON{X: np.X, Y: np.Y, Z: np.Z, Orbit: np.PN},
			Status:   np.Status, Hiding: np.Hiding, Hidden: np.Hidden,
			SiegeEff: np.SiegeEff, Shipyards: np.Shipyards,
			IUs:    consJSON{Needed: np.IUsNeeded, Auto: np.AutoIUs, Install: np.IUsToInstall},
			AUs:    consJSON{Needed: np.AUsNeeded, Auto: np.AutoAUs, Install: np.AUsToInstall},
			MiBase: np.MiBase, MaBase: np.MaBase, PopUnits: np.PopUnits,
			Items:       items(np.ItemQuantity[:]),
			UseOnAmbush: np.UseOnAmbush, Message: np.Message, Special: np.Special,
		}
		sf.NamedPlanets = append(sf.NamedPlanets, jn)
	}

	sf.Ships = make([]shipJSON, 0, len(sp.Ships))
	for _, sh := range sp.Ships {
		js := shipJSON{
			Name:     sh.Name,
			Location: coordOrbitJSON{X: sh.X, Y: sh.Y, Z: sh.Z, Orbit: sh.PN},
			Status:   sh.Status, Type: sh.ShipType,
			Dest:               coordJSON{X: sh.DestX, Y: sh.DestY, Z: sh.DestZ},
			JustJumped:         sh.JustJumped != 0,
			ArrivedViaWormhole: sh.ArrivedViaWormhole != 0,
			Class:              sh.Class, Tonnage: sh.Tonnage,
			Cargo: items(sh.ItemQuantity[:]),
			Age:   sh.Age, RemainingCost: sh.RemainingCost,
			LoadingPoint: sh.LoadingPoint, UnloadingPoint: sh.UnloadingPoint, Special: sh.Special,
		}
		sf.Ships = append(sf.Ships, js)
	}

	return sf
}

// items lists the non-zero item slots as {code, qty} pairs.
func items(q []int) []itemJSON {
	out := make([]itemJSON, 0)
	for code, qty := range q {
		if qty != 0 {
			out = append(out, itemJSON{Code: code, Qty: qty})
		}
	}
	return out
}

func intSlice(a []int) []int {
	out := make([]int, len(a))
	copy(out, a)
	return out
}

func writeLocations(path string, locs []model.Location) error {
	if len(locs) == 0 {
		_ = os.Remove(path)
		return nil
	}
	buf := make([]byte, 0, len(locs)*4)
	for _, l := range locs {
		buf = append(buf, byte(l.S), byte(l.X), byte(l.Y), byte(l.Z))
	}
	return os.WriteFile(path, buf, 0o644)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
