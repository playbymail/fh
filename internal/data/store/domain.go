package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/playbymail/fh/internal/model"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// IngestWorld persists a complete game World for gameID, replacing any existing
// domain rows for that game. The whole write is one transaction so a partial
// world is never visible. The game row itself must already exist (the fh_*
// tables reference game(id)).
func (s *SQLiteStore) IngestWorld(ctx context.Context, gameID string, w *model.World) (err error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	defer sqlitex.Transaction(conn)(&err)

	for _, tbl := range []string{
		"fh_galaxy", "fh_system", "fh_planet",
		"fh_species", "fh_nampla", "fh_ship", "fh_location",
	} {
		if err = sqlitex.Execute(conn, "DELETE FROM "+tbl+" WHERE game_id = ?",
			&sqlitex.ExecOptions{Args: []any{gameID}}); err != nil {
			return err
		}
	}

	if err = sqlitex.Execute(conn, `
		INSERT INTO fh_galaxy (game_id, turn_number, num_species, d_num_species, radius)
		VALUES (?, ?, ?, ?, ?)`,
		&sqlitex.ExecOptions{Args: []any{
			gameID, w.Galaxy.TurnNumber, w.Galaxy.NumSpecies, w.Galaxy.DNumSpecies, w.Galaxy.Radius,
		}}); err != nil {
		return err
	}

	for _, sys := range w.Systems {
		if err = sqlitex.Execute(conn, `
			INSERT INTO fh_system
			  (game_id, x, y, z, star_type, color, size, home_system,
			   worm_here, worm_x, worm_y, worm_z, message, visited_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			&sqlitex.ExecOptions{Args: []any{
				gameID, sys.X, sys.Y, sys.Z, sys.Type, sys.Color, sys.Size, boolToInt(sys.HomeSystem),
				boolToInt(sys.WormHere), sys.WormX, sys.WormY, sys.WormZ, sys.Message,
				jsonU32(sys.VisitedBy[:]),
			}}); err != nil {
			return err
		}
		for _, p := range sys.Planets {
			if err = sqlitex.Execute(conn, `
				INSERT INTO fh_planet
				  (game_id, sys_x, sys_y, sys_z, orbit, temperature_class, pressure_class,
				   special, gas, gas_percent, diameter, gravity, mining_difficulty,
				   md_increase, econ_efficiency, message)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				&sqlitex.ExecOptions{Args: []any{
					gameID, sys.X, sys.Y, sys.Z, p.Orbit, p.TemperatureClass, p.PressureClass,
					p.Special, jsonInts(p.Gas[:]), jsonInts(p.GasPercent[:]), p.Diameter, p.Gravity,
					p.MiningDifficulty, p.MdIncrease, p.EconEfficiency, p.Message,
				}}); err != nil {
				return err
			}
		}
	}

	for _, sp := range w.Species {
		if err = sqlitex.Execute(conn, `
			INSERT INTO fh_species
			  (game_id, sp_no, name, govt_name, govt_type, x, y, z, pn,
			   required_gas, required_gas_min, required_gas_max, neutral_gas, poison_gas,
			   auto_orders, tech_level, init_tech_level, tech_knowledge, tech_eps,
			   hp_original_base, econ_units, fleet_cost, fleet_percent_cost,
			   contact, ally, enemy, num_namplas, num_ships, log)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			&sqlitex.ExecOptions{Args: []any{
				gameID, sp.ID, sp.Name, sp.GovtName, sp.GovtType, sp.X, sp.Y, sp.Z, sp.PN,
				sp.RequiredGas, sp.RequiredGasMin, sp.RequiredGasMax, jsonInts(sp.NeutralGas[:]), jsonInts(sp.PoisonGas[:]),
				boolToInt(sp.AutoOrders), jsonInts(sp.TechLevel[:]), jsonInts(sp.InitTechLevel[:]),
				jsonInts(sp.TechKnowledge[:]), jsonInts(sp.TechEps[:]),
				sp.HpOriginalBase, sp.EconUnits, sp.FleetCost, sp.FleetPercentCost,
				jsonU32(sp.Contact[:]), jsonU32(sp.Ally[:]), jsonU32(sp.Enemy[:]),
				sp.NumNamplas, sp.NumShips, sp.Log,
			}}); err != nil {
			return err
		}
		for i, np := range sp.Namplas {
			if err = sqlitex.Execute(conn, `
				INSERT INTO fh_nampla
				  (game_id, sp_no, idx, name, x, y, z, pn, status, hiding, hidden,
				   siege_eff, shipyards, ius_needed, aus_needed, auto_ius, auto_aus,
				   ius_to_install, aus_to_install, mi_base, ma_base, pop_units, items,
				   use_on_ambush, message, special)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				&sqlitex.ExecOptions{Args: []any{
					gameID, sp.ID, i, np.Name, np.X, np.Y, np.Z, np.PN, np.Status, np.Hiding, np.Hidden,
					np.SiegeEff, np.Shipyards, np.IUsNeeded, np.AUsNeeded, np.AutoIUs, np.AutoAUs,
					np.IUsToInstall, np.AUsToInstall, np.MiBase, np.MaBase, np.PopUnits, jsonInts(np.ItemQuantity[:]),
					np.UseOnAmbush, np.Message, np.Special,
				}}); err != nil {
				return err
			}
		}
		for i, sh := range sp.Ships {
			if err = sqlitex.Execute(conn, `
				INSERT INTO fh_ship
				  (game_id, sp_no, idx, name, x, y, z, pn, status, ship_type,
				   dest_x, dest_y, dest_z, just_jumped, arrived_via_wormhole, class, tonnage,
				   items, age, remaining_cost, loading_point, unloading_point, special)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				&sqlitex.ExecOptions{Args: []any{
					gameID, sp.ID, i, sh.Name, sh.X, sh.Y, sh.Z, sh.PN, sh.Status, sh.ShipType,
					sh.DestX, sh.DestY, sh.DestZ, sh.JustJumped, sh.ArrivedViaWormhole, sh.Class, sh.Tonnage,
					jsonInts(sh.ItemQuantity[:]), sh.Age, sh.RemainingCost, sh.LoadingPoint, sh.UnloadingPoint, sh.Special,
				}}); err != nil {
				return err
			}
		}
	}

	for i, loc := range w.Locations {
		if err = sqlitex.Execute(conn, `
			INSERT INTO fh_location (game_id, seq, s, x, y, z) VALUES (?, ?, ?, ?, ?, ?)`,
			&sqlitex.ExecOptions{Args: []any{gameID, i, loc.S, loc.X, loc.Y, loc.Z}}); err != nil {
			return err
		}
	}

	return err
}

// LoadWorld reads the complete game World for gameID back into memory and
// resolves derived cross-references (nampla -> planet). It is the inverse of
// IngestWorld.
func (s *SQLiteStore) LoadWorld(ctx context.Context, gameID string) (*model.World, error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	w := &model.World{}

	if err := sqlitex.Execute(conn, `
		SELECT turn_number, num_species, d_num_species, radius FROM fh_galaxy WHERE game_id = ?`,
		&sqlitex.ExecOptions{
			Args: []any{gameID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				w.Galaxy = model.Galaxy{
					TurnNumber:  stmt.ColumnInt(0),
					NumSpecies:  stmt.ColumnInt(1),
					DNumSpecies: stmt.ColumnInt(2),
					Radius:      stmt.ColumnInt(3),
				}
				return nil
			},
		}); err != nil {
		return nil, err
	}

	systems := make(map[[3]int]*model.System)
	if err := sqlitex.Execute(conn, `
		SELECT x, y, z, star_type, color, size, home_system, worm_here, worm_x, worm_y, worm_z, message, visited_by
		FROM fh_system WHERE game_id = ? ORDER BY rowid`,
		&sqlitex.ExecOptions{
			Args: []any{gameID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				sys := &model.System{
					X: stmt.ColumnInt(0), Y: stmt.ColumnInt(1), Z: stmt.ColumnInt(2),
					Type: stmt.ColumnInt(3), Color: stmt.ColumnInt(4), Size: stmt.ColumnInt(5),
					HomeSystem: stmt.ColumnInt(6) != 0, WormHere: stmt.ColumnInt(7) != 0,
					WormX: stmt.ColumnInt(8), WormY: stmt.ColumnInt(9), WormZ: stmt.ColumnInt(10),
					Message: stmt.ColumnInt(11),
				}
				readU32(stmt.ColumnText(12), sys.VisitedBy[:])
				w.Systems = append(w.Systems, sys)
				systems[[3]int{sys.X, sys.Y, sys.Z}] = sys
				return nil
			},
		}); err != nil {
		return nil, err
	}

	if err := sqlitex.Execute(conn, `
		SELECT sys_x, sys_y, sys_z, orbit, temperature_class, pressure_class, special,
		       gas, gas_percent, diameter, gravity, mining_difficulty, md_increase, econ_efficiency, message
		FROM fh_planet WHERE game_id = ? ORDER BY sys_x, sys_y, sys_z, orbit`,
		&sqlitex.ExecOptions{
			Args: []any{gameID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				sys := systems[[3]int{stmt.ColumnInt(0), stmt.ColumnInt(1), stmt.ColumnInt(2)}]
				if sys == nil {
					return fmt.Errorf("planet references missing system %d,%d,%d", stmt.ColumnInt(0), stmt.ColumnInt(1), stmt.ColumnInt(2))
				}
				p := &model.Planet{
					System: sys, Orbit: stmt.ColumnInt(3),
					TemperatureClass: stmt.ColumnInt(4), PressureClass: stmt.ColumnInt(5), Special: stmt.ColumnInt(6),
					Diameter: stmt.ColumnInt(9), Gravity: stmt.ColumnInt(10), MiningDifficulty: stmt.ColumnInt(11),
					MdIncrease: stmt.ColumnInt(12), EconEfficiency: stmt.ColumnInt(13), Message: stmt.ColumnInt(14),
				}
				readInts(stmt.ColumnText(7), p.Gas[:])
				readInts(stmt.ColumnText(8), p.GasPercent[:])
				sys.Planets = append(sys.Planets, p)
				return nil
			},
		}); err != nil {
		return nil, err
	}

	speciesByNo := make(map[int]*model.Species)
	if err := sqlitex.Execute(conn, `
		SELECT sp_no, name, govt_name, govt_type, x, y, z, pn, required_gas, required_gas_min, required_gas_max,
		       neutral_gas, poison_gas, auto_orders, tech_level, init_tech_level, tech_knowledge, tech_eps,
		       hp_original_base, econ_units, fleet_cost, fleet_percent_cost, contact, ally, enemy,
		       num_namplas, num_ships, log
		FROM fh_species WHERE game_id = ? ORDER BY sp_no`,
		&sqlitex.ExecOptions{
			Args: []any{gameID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				sp := &model.Species{
					ID: stmt.ColumnInt(0), Name: stmt.ColumnText(1), GovtName: stmt.ColumnText(2), GovtType: stmt.ColumnText(3),
					X: stmt.ColumnInt(4), Y: stmt.ColumnInt(5), Z: stmt.ColumnInt(6), PN: stmt.ColumnInt(7),
					RequiredGas: stmt.ColumnInt(8), RequiredGasMin: stmt.ColumnInt(9), RequiredGasMax: stmt.ColumnInt(10),
					AutoOrders:     stmt.ColumnInt(13) != 0,
					HpOriginalBase: stmt.ColumnInt(18), EconUnits: stmt.ColumnInt(19), FleetCost: stmt.ColumnInt(20),
					FleetPercentCost: stmt.ColumnInt(21), NumNamplas: stmt.ColumnInt(25), NumShips: stmt.ColumnInt(26),
				}
				readInts(stmt.ColumnText(11), sp.NeutralGas[:])
				readInts(stmt.ColumnText(12), sp.PoisonGas[:])
				readInts(stmt.ColumnText(14), sp.TechLevel[:])
				readInts(stmt.ColumnText(15), sp.InitTechLevel[:])
				readInts(stmt.ColumnText(16), sp.TechKnowledge[:])
				readInts(stmt.ColumnText(17), sp.TechEps[:])
				readU32(stmt.ColumnText(22), sp.Contact[:])
				readU32(stmt.ColumnText(23), sp.Ally[:])
				readU32(stmt.ColumnText(24), sp.Enemy[:])
				if n := stmt.ColumnLen(27); n > 0 {
					sp.Log = make([]byte, n)
					stmt.ColumnBytes(27, sp.Log)
				}
				w.Species = append(w.Species, sp)
				speciesByNo[sp.ID] = sp
				return nil
			},
		}); err != nil {
		return nil, err
	}

	if err := sqlitex.Execute(conn, `
		SELECT sp_no, name, x, y, z, pn, status, hiding, hidden, siege_eff, shipyards,
		       ius_needed, aus_needed, auto_ius, auto_aus, ius_to_install, aus_to_install,
		       mi_base, ma_base, pop_units, items, use_on_ambush, message, special
		FROM fh_nampla WHERE game_id = ? ORDER BY sp_no, idx`,
		&sqlitex.ExecOptions{
			Args: []any{gameID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				sp := speciesByNo[stmt.ColumnInt(0)]
				if sp == nil {
					return fmt.Errorf("nampla references missing species %d", stmt.ColumnInt(0))
				}
				np := &model.Nampla{
					Name: stmt.ColumnText(1), X: stmt.ColumnInt(2), Y: stmt.ColumnInt(3), Z: stmt.ColumnInt(4), PN: stmt.ColumnInt(5),
					Status: stmt.ColumnInt(6), Hiding: stmt.ColumnInt(7), Hidden: stmt.ColumnInt(8), SiegeEff: stmt.ColumnInt(9),
					Shipyards: stmt.ColumnInt(10), IUsNeeded: stmt.ColumnInt(11), AUsNeeded: stmt.ColumnInt(12),
					AutoIUs: stmt.ColumnInt(13), AutoAUs: stmt.ColumnInt(14), IUsToInstall: stmt.ColumnInt(15), AUsToInstall: stmt.ColumnInt(16),
					MiBase: stmt.ColumnInt(17), MaBase: stmt.ColumnInt(18), PopUnits: stmt.ColumnInt(19),
					UseOnAmbush: stmt.ColumnInt(21), Message: stmt.ColumnInt(22), Special: stmt.ColumnInt(23),
				}
				readInts(stmt.ColumnText(20), np.ItemQuantity[:])
				sp.Namplas = append(sp.Namplas, np)
				return nil
			},
		}); err != nil {
		return nil, err
	}

	if err := sqlitex.Execute(conn, `
		SELECT sp_no, name, x, y, z, pn, status, ship_type, dest_x, dest_y, dest_z,
		       just_jumped, arrived_via_wormhole, class, tonnage, items, age, remaining_cost,
		       loading_point, unloading_point, special
		FROM fh_ship WHERE game_id = ? ORDER BY sp_no, idx`,
		&sqlitex.ExecOptions{
			Args: []any{gameID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				sp := speciesByNo[stmt.ColumnInt(0)]
				if sp == nil {
					return fmt.Errorf("ship references missing species %d", stmt.ColumnInt(0))
				}
				sh := &model.Ship{
					Name: stmt.ColumnText(1), X: stmt.ColumnInt(2), Y: stmt.ColumnInt(3), Z: stmt.ColumnInt(4), PN: stmt.ColumnInt(5),
					Status: stmt.ColumnInt(6), ShipType: stmt.ColumnInt(7), DestX: stmt.ColumnInt(8), DestY: stmt.ColumnInt(9), DestZ: stmt.ColumnInt(10),
					JustJumped: stmt.ColumnInt(11), ArrivedViaWormhole: stmt.ColumnInt(12), Class: stmt.ColumnInt(13), Tonnage: stmt.ColumnInt(14),
					Age: stmt.ColumnInt(16), RemainingCost: stmt.ColumnInt(17), LoadingPoint: stmt.ColumnInt(18), UnloadingPoint: stmt.ColumnInt(19), Special: stmt.ColumnInt(20),
				}
				readInts(stmt.ColumnText(15), sh.ItemQuantity[:])
				sp.Ships = append(sp.Ships, sh)
				return nil
			},
		}); err != nil {
		return nil, err
	}

	if err := sqlitex.Execute(conn, `
		SELECT s, x, y, z FROM fh_location WHERE game_id = ? ORDER BY seq`,
		&sqlitex.ExecOptions{
			Args: []any{gameID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				w.Locations = append(w.Locations, model.Location{
					S: stmt.ColumnInt(0), X: stmt.ColumnInt(1), Y: stmt.ColumnInt(2), Z: stmt.ColumnInt(3),
				})
				return nil
			},
		}); err != nil {
		return nil, err
	}

	w.Resolve()
	return w, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// jsonInts / jsonU32 serialize fixed-width integer vectors to JSON text for
// storage; readInts / readU32 parse them back into a caller-provided slice
// without changing its length.
func jsonInts(v []int) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func jsonU32(v []uint32) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func readInts(s string, dst []int) {
	var v []int
	_ = json.Unmarshal([]byte(s), &v)
	for i := range dst {
		if i < len(v) {
			dst[i] = v[i]
		} else {
			dst[i] = 0
		}
	}
}

func readU32(s string, dst []uint32) {
	var v []uint32
	_ = json.Unmarshal([]byte(s), &v)
	for i := range dst {
		if i < len(v) {
			dst[i] = v[i]
		} else {
			dst[i] = 0
		}
	}
}
