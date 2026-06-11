package engine

// Ingest builds an in-memory model.World from the structured state the
// byte-faithful fhc engine emits: its `export json` output (galaxy.json,
// systems.json, species.NNN.json) plus the per-species event logs (spNN.log)
// and the locations index (locations.dat).
//
// This is a one-way bridge that lets fh validate its report path against fhc
// without first reproducing galaxy generation. The .dat/JSON formats are NOT
// part of fh's contract — once a subsystem is ported, fh will generate this
// state itself. For this first slice the accumulated spNN.log is carried
// through verbatim rather than regenerated.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/playbymail/fh/internal/model"
)

// jsonGalaxy mirrors fhc's galaxy.json.
type jsonGalaxy struct {
	Galaxy struct {
		TurnNumber  int `json:"turn_number"`
		NumSpecies  int `json:"num_species"`
		DNumSpecies int `json:"d_num_species"`
		Radius      int `json:"radius"`
	} `json:"galaxy"`
}

// jsonSystems mirrors fhc's systems.json.
type jsonSystems struct {
	Systems []struct {
		X         int   `json:"x"`
		Y         int   `json:"y"`
		Z         int   `json:"z"`
		VisitedBy []int `json:"visited_by"`
		Message   int   `json:"message"`
		WormHere  bool  `json:"worm_here"`
		WormX     int   `json:"worm_x"`
		WormY     int   `json:"worm_y"`
		WormZ     int   `json:"worm_z"`
		Size      int   `json:"size"`
		Planets   []struct {
			TemperatureClass int `json:"temperature_class"`
			PressureClass    int `json:"pressure_class"`
			Special          int `json:"special"`
			Atmosphere       []struct {
				Code    int `json:"code"`
				Percent int `json:"percent"`
			} `json:"atmosphere"`
			Diameter         int `json:"diameter"`
			Gravity          int `json:"gravity"`
			MiningDifficulty struct {
				Base     int `json:"base"`
				Increase int `json:"increase"`
			} `json:"mining_difficulty"`
			EconEfficiency int `json:"econ_efficiency"`
			Message        int `json:"message"`
		} `json:"planets"`
	} `json:"systems"`
}

// Note: the system "x" key is shared by the y/z fields below via the struct
// tags; declare them explicitly to avoid the duplicate-tag pitfall.
type jsonCoord struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

type jsonItem struct {
	Code int `json:"code"`
	Qty  int `json:"qty"`
}

// jsonSpeciesFile mirrors fhc's species.NNN.json.
type jsonSpeciesFile struct {
	Species struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Government struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"government"`
		HomeWorld struct {
			X     int `json:"x"`
			Y     int `json:"y"`
			Z     int `json:"z"`
			Orbit int `json:"orbit"`
		} `json:"home_world"`
		Atmosphere struct {
			Required struct {
				Code   int `json:"code"`
				MinPct int `json:"min_pct"`
				MaxPct int `json:"max_pct"`
			} `json:"required"`
			Neutral []int `json:"neutral"`
			Poison  []int `json:"poison"`
		} `json:"atmosphere"`
		AutoOrders       bool                `json:"auto_orders"`
		Tech             map[string]jsonTech `json:"tech"`
		HpOriginalBase   int                 `json:"hp_original_base"`
		EconUnits        int                 `json:"econ_units"`
		FleetCost        int                 `json:"fleet_cost"`
		FleetPercentCost int                 `json:"fleet_percent_cost"`
		Contacts         []int               `json:"contacts"`
		Allies           []int               `json:"allies"`
		Enemies          []int               `json:"enemies"`
	} `json:"species"`
	NamedPlanets []struct {
		Name        string     `json:"name"`
		Location    jsonCoordO `json:"location"`
		Status      int        `json:"status"`
		Hiding      int        `json:"hiding"`
		Hidden      int        `json:"hidden"`
		SiegeEff    int        `json:"siege_eff"`
		Shipyards   int        `json:"shipyards"`
		IUs         jsonCons   `json:"ius"`
		AUs         jsonCons   `json:"aus"`
		MiBase      int        `json:"mi_base"`
		MaBase      int        `json:"ma_base"`
		PopUnits    int        `json:"pop_units"`
		Items       []jsonItem `json:"items"`
		UseOnAmbush int        `json:"use_on_ambush"`
		Message     int        `json:"message"`
		Special     int        `json:"special"`
	} `json:"named_planets"`
	Ships []struct {
		Name               string     `json:"name"`
		Location           jsonCoordO `json:"location"`
		Status             int        `json:"status"`
		Type               int        `json:"type"`
		Dest               jsonCoord  `json:"dest"`
		JustJumped         bool       `json:"just_jumped"`
		ArrivedViaWormhole bool       `json:"arrived_via_wormhole"`
		Class              int        `json:"class"`
		Tonnage            int        `json:"tonnage"`
		Cargo              []jsonItem `json:"cargo"`
		Age                int        `json:"age"`
		RemainingCost      int        `json:"remaining_cost"`
		LoadingPoint       int        `json:"loading_point"`
		UnloadingPoint     int        `json:"unloading_point"`
		Special            int        `json:"special"`
	} `json:"ships"`
}

type jsonTech struct {
	Level     int `json:"level"`
	Knowledge int `json:"knowledge"`
	XP        int `json:"xp"`
	InitLevel int `json:"init_level"`
}

type jsonCoordO struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Z     int `json:"z"`
	Orbit int `json:"orbit"`
}

type jsonCons struct {
	Needed  int `json:"needed"`
	Auto    int `json:"auto"`
	Install int `json:"install"`
}

// techIndex maps fhc's tech-discipline keys to model tech indices.
var techIndex = map[string]int{
	"MI": model.MI, "MA": model.MA, "ML": model.ML,
	"GV": model.GV, "LS": model.LS, "BI": model.BI,
}

// LoadWorldFromDir reads an fhc state directory into a model.World.
func LoadWorldFromDir(dir string) (*model.World, error) {
	w := &model.World{}

	var gx jsonGalaxy
	if err := readJSON(filepath.Join(dir, "galaxy.json"), &gx); err != nil {
		return nil, err
	}
	w.Galaxy = model.Galaxy{
		TurnNumber:  gx.Galaxy.TurnNumber,
		NumSpecies:  gx.Galaxy.NumSpecies,
		DNumSpecies: gx.Galaxy.DNumSpecies,
		Radius:      gx.Galaxy.Radius,
	}

	var sys jsonSystems
	if err := readJSON(filepath.Join(dir, "systems.json"), &sys); err != nil {
		return nil, err
	}
	for _, js := range sys.Systems {
		s := &model.System{
			X: js.X, Y: js.Y, Z: js.Z, Size: js.Size,
			WormHere: js.WormHere, WormX: js.WormX, WormY: js.WormY, WormZ: js.WormZ,
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
		var sf jsonSpeciesFile
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
func buildSpecies(sf *jsonSpeciesFile) *model.Species {
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
		if i, ok := techIndex[key]; ok {
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
