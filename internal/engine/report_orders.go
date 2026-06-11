package engine

// Port of report.c's order-section generator (the test_mode == FALSE block):
// the editable START COMBAT / PRE-DEPARTURE / JUMPS / PRODUCTION /
// POST-ARRIVAL / STRIKES template plus the auto-order generation the engine
// fills in from species state.

import "github.com/playbymail/fh/internal/model"

func (r *reporter) orderSection() {
	sp := r.species

	r.truncateName = true
	tempIgnoreFieldDistorters := r.ignoreFieldDistorters
	r.ignoreFieldDistorters = true

	r.printf("\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")
	r.printf("\n\nORDER SECTION. Remove these two lines and everything above\n")
	r.printf("  them, and submit only the orders below.\n\n")

	r.printf("START COMBAT\n")
	r.printf("; Place combat orders here.\n\n")
	r.printf("END\n\n")

	r.orderPreDeparture(sp)
	r.orderJumps(sp)
	r.orderProduction(sp)
	r.orderPostArrival(sp)

	r.printf("START STRIKES\n")
	r.printf("; Place strike orders here.\n\n")
	r.printf("END\n")

	r.truncateName = false
	r.ignoreFieldDistorters = tempIgnoreFieldDistorters
}

func (r *reporter) orderPreDeparture(sp *model.Species) {
	r.printf("START PRE-DEPARTURE\n")
	r.printf("; Place pre-departure orders here.\n\n")

	for namplaIndex := 0; namplaIndex < sp.NumNamplas; namplaIndex++ {
		nampla := sp.Namplas[namplaIndex]
		if nampla.PN == 99 {
			continue
		}

		if nampla.AutoIUs != 0 {
			r.printf("\tInstall\t%d IU\tPL %s\n", nampla.AutoIUs, nampla.Name)
		}
		if nampla.AutoAUs != 0 {
			r.printf("\tInstall\t%d AU\tPL %s\n", nampla.AutoAUs, nampla.Name)
		}
		if nampla.AutoIUs != 0 || nampla.AutoAUs != 0 {
			r.printf("\n")
		}

		if !sp.AutoOrders {
			continue
		}

		for j := 0; j < sp.NumShips; j++ {
			ship := sp.Ships[j]
			if ship.PN == 99 {
				continue
			}
			if ship.X != nampla.X || ship.Y != nampla.Y || ship.Z != nampla.Z || ship.PN != nampla.PN {
				continue
			}
			if ship.Status == model.JumpedInCombat || ship.Status == model.ForcedJump {
				continue
			}
			if ship.Class != model.TR {
				continue
			}
			if ship.ItemQuantity[model.CU] < 1 {
				continue
			}

			doUnload := false
			if ship.LoadingPoint != 0 {
				n := ship.UnloadingPoint
				if n == namplaIndex || (n == 9999 && namplaIndex == 0) {
					doUnload = true
				}
			}
			if !doUnload {
				if nampla.Status&model.Populated == 0 {
					continue
				}
				if (nampla.MiBase + nampla.MaBase) >= 2000 {
					continue
				}
				if nampla.X == sp.Namplas[0].X && nampla.Y == sp.Namplas[0].Y && nampla.Z == sp.Namplas[0].Z {
					continue
				}
			}

			n := ship.LoadingPoint
			if n == 9999 {
				n = 0
			}
			if n == namplaIndex {
				continue
			}
			r.printf("\tUnload\tTR%d%s %s\n\n", ship.Tonnage, model.ShipType[ship.ShipType], ship.Name)
			ship.Special = ship.LoadingPoint
			n = namplaIndex
			if n == 0 {
				n = 9999
			}
			ship.UnloadingPoint = n
		}
	}

	r.printf("END\n\n")
}

func (r *reporter) orderJumps(sp *model.Species) {
	r.printf("START JUMPS\n")
	r.printf("; Place jump orders here.\n\n")

	for i := 0; i < sp.NumShips; i++ {
		ship := sp.Ships[i]
		ship.JustJumped = 0
		if ship.PN == 99 {
			continue
		}
		if ship.Status == model.JumpedInCombat || ship.Status == model.ForcedJump {
			continue
		}

		j := ship.Special
		if j != 0 {
			if j == 9999 {
				j = 0
			}
			tempNampla := sp.Namplas[j]
			r.printf("\tJump\t%s, PL %s\t; Age %d, ", r.shipName(ship), tempNampla.Name, ship.Age)
			r.printMishapChance(ship, tempNampla.X, tempNampla.Y, tempNampla.Z)
			r.printf("\n\n")
			ship.JustJumped = 1
			continue
		}

		n := ship.UnloadingPoint
		if n != 0 {
			if n == 9999 {
				n = 0
			}
			tempNampla := sp.Namplas[n]
			r.printf("\tJump\t%s, PL %s\t; ", r.shipName(ship), tempNampla.Name)
			r.printMishapChance(ship, tempNampla.X, tempNampla.Y, tempNampla.Z)
			r.printf("\n\n")
			ship.JustJumped = 1
		}
	}

	if sp.AutoOrders {
		for i := 0; i < sp.NumShips; i++ {
			ship := sp.Ships[i]
			if ship.PN == 99 {
				continue
			}
			if ship.JustJumped != 0 {
				continue
			}
			if ship.Status == model.UnderConstruction || ship.Status == model.JumpedInCombat || ship.Status == model.ForcedJump {
				continue
			}
			if ship.ShipType != model.FTL {
				continue
			}

			r.printf("\tJump\t%s, ", r.shipName(ship))
			if ship.Class == model.TR && ship.Tonnage == 1 {
				r.closestUnvisitedStarReport(ship)
				r.printf("\n\t\t\t; Age %d, now at %d %d %d, ", ship.Age, ship.X, ship.Y, ship.Z)
				switch ship.Status {
				case model.InOrbit:
					r.printf("O%d, ", ship.PN)
				case model.OnSurface:
					r.printf("L%d, ", ship.PN)
				default:
					r.printf("D, ")
				}
				r.printMishapChance(ship, r.jx, r.jy, r.jz)
			} else {
				r.printf("???\t; Age %d, now at %d %d %d", ship.Age, ship.X, ship.Y, ship.Z)
				switch ship.Status {
				case model.InOrbit:
					r.printf(", O%d", ship.PN)
				case model.OnSurface:
					r.printf(", L%d", ship.PN)
				default:
					r.printf(", D")
				}
				r.jx = 9999
			}
			r.printf("\n")

			if r.jx == 9999 {
				ship.DestX = -1
			} else {
				ship.DestX = r.jx
				ship.DestY = r.jy
				ship.DestZ = r.jz
			}
		}
	}

	r.printf("END\n\n")
}

func (r *reporter) orderProduction(sp *model.Species) {
	r.printf("START PRODUCTION\n\n")
	r.printf(";   Economic units at start of turn = %d\n\n", sp.EconUnits)

	for namplaIndex := sp.NumNamplas - 1; namplaIndex >= 0; namplaIndex-- {
		nampla := sp.Namplas[namplaIndex]
		if nampla.PN == 99 {
			continue
		}
		if nampla.MiBase == 0 && nampla.Status&model.ResortColony == 0 {
			continue
		}
		if nampla.MaBase == 0 && nampla.Status&model.MiningColony == 0 {
			continue
		}

		r.printf("    PRODUCTION PL %s\n", nampla.Name)

		switch {
		case nampla.Status&model.MiningColony != 0:
			r.printf("    ; The above PRODUCTION order is required for this mining colony, even\n")
			r.printf("    ;  if no other production orders are given for it. This mining colony\n")
			r.printf("    ;  will generate %d economic units this turn.\n", nampla.UseOnAmbush)
		case nampla.Status&model.ResortColony != 0:
			r.printf("    ; The above PRODUCTION order is required for this resort colony, even\n")
			r.printf("    ;  though no other production orders can be given for it.  This resort\n")
			r.printf("    ;  colony will generate %d economic units this turn.\n", nampla.UseOnAmbush)
		default:
			r.printf("    ; Place production orders here for planet %s", nampla.Name)
			r.printf(" (sector %d %d %d #%d).\n", nampla.X, nampla.Y, nampla.Z, nampla.PN)
			r.printf("    ;  Avail pop = %d, shipyards = %d, to spend = %d", nampla.PopUnits, nampla.Shipyards, nampla.UseOnAmbush)
			n := nampla.UseOnAmbush
			if nampla.Status&model.HomePlanet != 0 {
				if sp.HpOriginalBase != 0 {
					r.printf(" (max = %d)", 5*n)
				} else {
					r.printf(" (max = no limit)")
				}
			} else {
				r.printf(" (max = %d)", 2*n)
			}
			r.printf(".\n\n")
		}

		if nampla.IUsNeeded != 0 {
			r.printf("\tBuild\t%d IU\n", nampla.IUsNeeded)
		}
		if nampla.AUsNeeded != 0 {
			r.printf("\tBuild\t%d AU\n", nampla.AUsNeeded)
		}
		if nampla.IUsNeeded != 0 || nampla.AUsNeeded != 0 {
			r.printf("\n")
		}

		if !sp.AutoOrders {
			continue
		}
		if nampla.Status&model.MiningColony != 0 {
			continue
		}
		if nampla.Status&model.ResortColony != 0 {
			continue
		}

		n := nampla.Special / 5
		if n > 0 {
			r.printf("\tRecycle\t%d RM\n\n", 5*n)
		}

		// DEVELOP for ships arriving here because of AUTO.
		for i := 0; i < sp.NumShips; i++ {
			ship := sp.Ships[i]
			if ship.PN == 99 {
				continue
			}
			k := ship.Special
			if k == 0 {
				continue
			}
			if k == 9999 {
				k = 0
			}
			if nampla != sp.Namplas[k] {
				continue
			}
			k = ship.UnloadingPoint
			if k == 9999 {
				k = 0
			}
			tempNampla := sp.Namplas[k]
			r.printf("\tDevelop\tPL %s, TR%d%s %s\n\n", tempNampla.Name, ship.Tonnage, model.ShipType[ship.ShipType], ship.Name)
		}

		// Continue unfinished ships and starbases.
		for i := 0; i < sp.NumShips; i++ {
			ship := sp.Ships[i]
			if ship.PN == 99 {
				continue
			}
			if ship.X != nampla.X || ship.Y != nampla.Y || ship.Z != nampla.Z || ship.PN != nampla.PN {
				continue
			}
			if ship.Status == model.UnderConstruction {
				r.printf("\tContinue\t%s, %d\t; Left to pay = %d\n\n", r.shipName(ship), ship.RemainingCost, ship.RemainingCost)
				continue
			}
			if ship.ShipType != model.Starbase {
				continue
			}
			j := (sp.TechLevel[model.MA] / 2) - ship.Tonnage
			if j < 1 {
				continue
			}
			r.printf("\tContinue\tBAS %s, %d\t; Current tonnage = %s\n\n", ship.Name, 100*j, commas(10000*ship.Tonnage))
		}

		// DEVELOP this colony if its economic base is under 200.
		n = nampla.MiBase + nampla.MaBase + nampla.IUsNeeded + nampla.AUsNeeded
		nn := nampla.ItemQuantity[model.CU]
		for i := 0; i < sp.NumShips; i++ {
			ship := sp.Ships[i]
			if ship.X != nampla.X || ship.Y != nampla.Y || ship.Z != nampla.Z || ship.PN != nampla.PN {
				continue
			}
			nn += ship.ItemQuantity[model.CU]
		}
		n += nn
		if nampla.Status&model.Colony != 0 && n < 2000 && nampla.PopUnits > 0 {
			if nampla.PopUnits > (2000 - n) {
				nn = 2000 - n
			} else {
				nn = nampla.PopUnits
			}
			r.printf("\tDevelop\t%d\n\n", 2*nn)
			nampla.IUsNeeded += nn
		}

		// Home planets / large colonies DEVELOP other planets in the sector.
		if n >= 2000 || nampla.Status&model.HomePlanet != 0 {
			for i := 1; i < sp.NumNamplas; i++ {
				if i == namplaIndex {
					continue
				}
				tempNampla := sp.Namplas[i]
				if tempNampla.PN == 99 {
					continue
				}
				if tempNampla.X != nampla.X || tempNampla.Y != nampla.Y || tempNampla.Z != nampla.Z {
					continue
				}
				n = tempNampla.MiBase + tempNampla.MaBase + tempNampla.IUsNeeded + tempNampla.AUsNeeded
				if n == 0 {
					continue
				}
				nn = tempNampla.ItemQuantity[model.IU] + tempNampla.ItemQuantity[model.AU]
				if nn > tempNampla.ItemQuantity[model.CU] {
					nn = tempNampla.ItemQuantity[model.CU]
				}
				n += nn
				if n >= 2000 {
					continue
				}
				nn = 2000 - n
				if nn > nampla.PopUnits {
					nn = nampla.PopUnits
				}
				r.printf("\tDevelop\t%d\tPL %s\n\n", 2*nn, tempNampla.Name)
				tempNampla.AUsNeeded += nn
			}
		}
	}

	r.printf("END\n\n")
}

func (r *reporter) orderPostArrival(sp *model.Species) {
	r.printf("START POST-ARRIVAL\n")
	r.printf("; Place post-arrival orders here.\n\n")

	if !sp.AutoOrders {
		r.printf("END\n\n")
		return
	}

	r.printf("\tAuto\n\n")

	for i := 0; i < sp.NumShips; i++ {
		ship := sp.Ships[i]
		if ship.PN == 99 {
			continue
		}
		if ship.Status == model.UnderConstruction {
			continue
		}
		if ship.Class != model.TR {
			continue
		}
		if ship.Tonnage != 1 {
			continue
		}
		if ship.ShipType != model.FTL {
			continue
		}

		found := false
		for j := 0; j < sp.NumNamplas; j++ {
			if ship.DestX == -1 {
				break
			}
			nampla := sp.Namplas[j]
			if nampla.PN == 99 {
				continue
			}
			if nampla.X != ship.DestX || nampla.Y != ship.DestY || nampla.Z != ship.DestZ {
				continue
			}
			if nampla.Status&model.Populated != 0 {
				found = true
				break
			}
		}
		if !found {
			r.printf("\tScan\tTR1 %s\n", ship.Name)
		}
	}

	r.printf("END\n\n")
}
