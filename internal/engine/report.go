package engine

// Idiomatic Go port of the C engine's report.c (internal/game/report.go).
//
// The output must be byte-identical to fhc's sp0X.rpt.tN reports: the player
// report is the species' accumulated event log (carried verbatim through
// ingest, see ingest.go) followed by the state-derived status block, planet
// reports, alien sightings, and the order section. Every fixed string,
// integer format, and control-flow branch mirrors the C engine.
//
// Unlike internal/game, this renderer holds no package-level state: report.c's
// file-level statics live on the reporter struct and are threaded explicitly.

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/playbymail/fh/internal/model"
)

// ReportOptions controls report generation.
type ReportOptions struct {
	// SkipLog omits the prepended prior-turn event log (mirrors --skip-log).
	SkipLog bool
	// TestMode suppresses the order section (mirrors the C engine's test mode).
	TestMode bool
}

// RenderReport renders the turn report for one species from world state,
// returning the exact bytes fhc would write to sp%02d.rpt.t%d.
func RenderReport(w *model.World, sp *model.Species, opts ReportOptions) []byte {
	r := &reporter{
		out:           &bytes.Buffer{},
		w:             w,
		species:       sp,
		speciesNumber: sp.ID,
	}
	r.run(opts)
	return r.out.Bytes()
}

// reporter carries the rendering state that report.c keeps in file-level
// statics and globals.
type reporter struct {
	out           *bytes.Buffer
	w             *model.World
	species       *model.Species
	speciesNumber int

	homePlanet *model.Planet

	fleetPercentCost      int
	printingAlien         bool
	shipAlreadyListed     []bool
	truncateName          bool
	ignoreFieldDistorters bool
	fullShipID            string

	// jump-target coordinates (report.c uses the globals x, y, z here).
	jx, jy, jz int

	// log_char line-wrap state for the species-met / allies / enemies block.
	logLine        [128]byte
	logPos         int
	logIndent      int
	logStartOfLine bool
}

func (r *reporter) printf(format string, a ...any) {
	fmt.Fprintf(r.out, format, a...)
}

func (r *reporter) run(opts ReportOptions) {
	sp := r.species
	turnNumber := r.w.Galaxy.TurnNumber
	r.homePlanet = sp.HomePlanet()

	// Prepend the accumulated event log, exactly as report.c copies spNN.log.
	if !opts.SkipLog && len(sp.Log) > 0 {
		if turnNumber > 1 {
			r.printf("\n\n\t\t\tEVENT LOG FOR TURN %d\n", turnNumber-1)
		}
		r.out.Write(sp.Log)
		r.printf("\n\n")
	}

	// Status report header.
	r.printf("\n\t\t\t SPECIES STATUS\n\n\t\t\tSTART OF TURN %d\n\n", turnNumber)
	r.printf("Species name: %s\n", sp.Name)
	r.printf("Government name: %s\n", sp.GovtName)
	r.printf("Government type: %s\n", sp.GovtType)

	r.printf("\nTech Levels:\n")
	for i := 0; i < 6; i++ {
		r.printf("   %s = %d", model.TechName[i], sp.TechLevel[i])
		if sp.TechKnowledge[i] > sp.TechLevel[i] {
			r.printf("/%d", sp.TechKnowledge[i])
		}
		r.printf("\n")
	}

	r.printf("\nAtmospheric Requirement: %d%%-%d%% %s", sp.RequiredGasMin, sp.RequiredGasMax, model.GasString[sp.RequiredGas])
	r.printf("\nNeutral Gases:")
	for i := 0; i < 6; i++ {
		if i != 0 {
			r.printf(",")
		}
		r.printf(" %s", model.GasString[sp.NeutralGas[i]])
	}
	r.printf("\nPoisonous Gases:")
	for i := 0; i < 6; i++ {
		if i != 0 {
			r.printf(",")
		}
		r.printf(" %s", model.GasString[sp.PoisonGas[i]])
	}
	r.printf("\n")

	r.fleetPercentCost = sp.FleetPercentCost
	r.printf("\nFleet maintenance cost = %d (%d.%02d%% of total production)\n",
		sp.FleetCost, r.fleetPercentCost/100, r.fleetPercentCost%100)
	if r.fleetPercentCost > 10000 {
		r.fleetPercentCost = 10000
	}

	// Species met / allies / enemies, using the wrapping log writer.
	r.printContactList("\nSpecies met: ", sp.Contact, false)
	r.printContactList("\nAllies: ", sp.Ally, true)
	r.printContactList("\nEnemies: ", sp.Enemy, true)

	r.printf("\nEconomic units = %d\n", sp.EconUnits)

	r.shipAlreadyListed = make([]bool, sp.NumShips)

	// Report for each producing planet.
	for i := 0; i < sp.NumNamplas; i++ {
		nampla := sp.Namplas[i]
		if nampla.PN == 99 {
			continue
		}
		if nampla.MiBase == 0 && nampla.MaBase == 0 && nampla.Status&model.HomePlanet == 0 {
			continue
		}
		r.printf("\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")
		r.doPlanetReport(nampla, sp.Ships, sp)
	}

	// One-line listing for other planets.
	r.printingAlien = false
	headerPrinted := false
	for i := 0; i < sp.NumNamplas; i++ {
		nampla := sp.Namplas[i]
		if nampla.PN == 99 {
			continue
		}
		if nampla.MiBase > 0 || nampla.MaBase > 0 || nampla.Status&model.HomePlanet != 0 {
			continue
		}
		if !headerPrinted {
			r.printf("\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")
			r.printf("\n\nOther planets and ships:\n\n")
			headerPrinted = true
		}
		r.printf("%4d%3d%3d #%d\tPL %s", nampla.X, nampla.Y, nampla.Z, nampla.PN, nampla.Name)
		for j := 0; j < model.MaxItems; j++ {
			if nampla.ItemQuantity[j] > 0 {
				r.printf(", %d %s", nampla.ItemQuantity[j], model.ItemAbbr[j])
			}
		}
		r.printf("\n")

		for si := 0; si < sp.NumShips; si++ {
			ship := sp.Ships[si]
			if r.shipAlreadyListed[si] {
				continue
			}
			if ship.X != nampla.X || ship.Y != nampla.Y || ship.Z != nampla.Z || ship.PN != nampla.PN {
				continue
			}
			r.printf("\t\t%s", r.shipName(ship))
			for j := 0; j < model.MaxItems; j++ {
				if ship.ItemQuantity[j] > 0 {
					r.printf(", %d %s", ship.ItemQuantity[j], model.ItemAbbr[j])
				}
			}
			r.printf("\n")
			r.shipAlreadyListed[si] = true
		}
	}

	// Ships not associated with a planet.
	for si := 0; si < sp.NumShips; si++ {
		ship := sp.Ships[si]
		ship.Special = 0
		if r.shipAlreadyListed[si] {
			continue
		}
		r.shipAlreadyListed[si] = true
		if ship.PN == 99 {
			continue
		}
		if !headerPrinted {
			r.printf("\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")
			r.printf("\n\nOther planets and ships:\n\n")
			headerPrinted = true
		}
		if ship.Status == model.JumpedInCombat || ship.Status == model.ForcedJump {
			r.printf("  ?? ?? ??\t%s", r.shipName(ship))
		} else {
			r.printf("%4d%3d%3d\t%s", ship.X, ship.Y, ship.Z, r.shipName(ship))
		}
		for i := 0; i < model.MaxItems; i++ {
			if ship.ItemQuantity[i] > 0 {
				r.printf(", %d %s", ship.ItemQuantity[i], model.ItemAbbr[i])
			}
		}
		r.printf("\n")

		if ship.Status == model.JumpedInCombat || ship.Status == model.ForcedJump {
			continue
		}

		for i := si + 1; i < sp.NumShips; i++ {
			ship2 := sp.Ships[i]
			if r.shipAlreadyListed[i] {
				continue
			}
			if ship2.PN == 99 {
				continue
			}
			if ship2.X != ship.X || ship2.Y != ship.Y || ship2.Z != ship.Z {
				continue
			}
			r.printf("\t\t%s", r.shipName(ship2))
			for j := 0; j < model.MaxItems; j++ {
				if ship2.ItemQuantity[j] > 0 {
					r.printf(", %d %s", ship2.ItemQuantity[j], model.ItemAbbr[j])
				}
			}
			r.printf("\n")
			r.shipAlreadyListed[i] = true
		}
	}

	r.printf("\n\n* * * * * * * * * * * * * * * * * * * * * * * * *\n")

	r.reportAliens()

	if !opts.TestMode {
		r.orderSection()
	}
}

// printContactList prints the species-met / allies / enemies block. requireContact
// also requires a contact bit to be set (allies and enemies are only listed if met).
func (r *reporter) printContactList(header string, bits [model.NumContactWords]uint32, requireContact bool) {
	n := 0
	headerPrinted := false
	for spIndex := 0; spIndex < r.w.Galaxy.NumSpecies; spIndex++ {
		alien := r.w.SpeciesByNumber(spIndex + 1)
		if alien == nil {
			continue
		}
		if !model.ContactBit(bits, spIndex) {
			continue
		}
		if requireContact && !model.ContactBit(r.species.Contact, spIndex) {
			continue
		}
		if !headerPrinted {
			r.logString(header)
			headerPrinted = true
		}
		if n > 0 {
			r.logString(", ")
		}
		r.logString("SP ")
		r.logString(alien.Name)
		n++
	}
	if n > 0 {
		r.logChar('\n')
	}
}

func (r *reporter) doPlanetReport(nampla *model.Nampla, ships []*model.Ship, sp *model.Species) {
	planet := nampla.Planet

	r.printf("\n\n")
	switch {
	case nampla.Status&model.HomePlanet != 0:
		r.printf("HOME PLANET")
	case nampla.Status&model.MiningColony != 0:
		r.printf("MINING COLONY")
	case nampla.Status&model.ResortColony != 0:
		r.printf("RESORT COLONY")
	case nampla.Status&model.Populated != 0:
		r.printf("COLONY PLANET")
	default:
		r.printf("PLANET")
	}
	r.printf(": PL %s", nampla.Name)
	r.printf("\n   Coordinates: x = %d, y = %d, z = %d, planet number %d\n", nampla.X, nampla.Y, nampla.Z, nampla.PN)

	if nampla.Status&model.HomePlanet != 0 {
		ib := nampla.MiBase
		ab := nampla.MaBase
		currentBase := ib + ab
		if currentBase < sp.HpOriginalBase {
			n := sp.HpOriginalBase - currentBase
			md := r.homePlanet.MiningDifficulty
			denom := 100 + md
			j := (100*(n+ib) - (md * ab) + denom/2) / denom
			i := n - j
			if i < 0 {
				j = n
				i = 0
			}
			if j < 0 {
				i = n
				j = 0
			}
			r.printf("\nWARNING! Home planet has not yet completely recovered from bombardment!\n")
			r.printf("         %d IUs and %d AUs will have to be installed for complete recovery.\n", i, j)
		}
	}

	if nampla.Status&model.Populated == 0 {
		r.doInventory(nampla, ships, sp)
		return
	}

	if nampla.Status&(model.MiningColony|model.ResortColony) == 0 {
		r.printf("\nAvailable population units = %d\n", nampla.PopUnits)
	}
	if nampla.SiegeEff != 0 {
		r.printf("\nWARNING!  This planet is currently under siege and will remain\n")
		r.printf("  under siege until the combat phase of the next turn!\n")
	}
	if nampla.UseOnAmbush > 0 {
		r.printf("\nIMPORTANT!  This planet has made preparations for an ambush!\n")
	}
	if nampla.Hidden != 0 {
		r.printf("\nIMPORTANT!  This planet is actively hiding from alien observation!\n")
	}

	rawMaterialUnits := (10 * sp.TechLevel[model.MI] * nampla.MiBase) / planet.MiningDifficulty
	productionCapacity := (sp.TechLevel[model.MA] * nampla.MaBase) / 10

	lsNeeded := lifeSupportNeeded(sp, r.homePlanet, planet)
	var productionPenalty int
	if lsNeeded == 0 {
		productionPenalty = 0
	} else {
		productionPenalty = (100 * lsNeeded) / sp.TechLevel[model.LS]
	}

	r.printf("\nProduction penalty = %d%% (LSN = %d)\n", productionPenalty, lsNeeded)
	r.printf("\nEconomic efficiency = %d%%\n", planet.EconEfficiency)

	rawMaterialUnits -= (productionPenalty * rawMaterialUnits) / 100
	rawMaterialUnits = ((planet.EconEfficiency * rawMaterialUnits) + 50) / 100
	productionCapacity -= (productionPenalty * productionCapacity) / 100
	productionCapacity = ((planet.EconEfficiency * productionCapacity) + 50) / 100

	if nampla.MiBase > 0 {
		r.printf("\nMining base = %d.%d", nampla.MiBase/10, nampla.MiBase%10)
		r.printf(" (MI = %d, MD = %d.%02d)\n", sp.TechLevel[model.MI], planet.MiningDifficulty/100, planet.MiningDifficulty%100)
		if nampla.Status&model.MiningColony != 0 {
			n1 := (2 * rawMaterialUnits) / 3
			n2 := ((r.fleetPercentCost * n1) + 5000) / 10000
			n3 := n1 - n2
			r.printf("   This mining colony will generate %d - %d = %d economic units this turn.\n", n1, n2, n3)
			nampla.UseOnAmbush = n3
		} else {
			r.printf("   %d raw material units will be produced this turn.\n", rawMaterialUnits)
		}
	}

	if nampla.MaBase > 0 {
		if nampla.Status&model.ResortColony != 0 {
			r.printf("\n")
		}
		r.printf("Manufacturing base = %d.%d", nampla.MaBase/10, nampla.MaBase%10)
		r.printf(" (MA = %d)\n", sp.TechLevel[model.MA])
		if nampla.Status&model.ResortColony != 0 {
			n1 := (2 * productionCapacity) / 3
			n2 := ((r.fleetPercentCost * n1) + 5000) / 10000
			n3 := n1 - n2
			r.printf("   This resort colony will generate %d - %d = %d economic units this turn.\n", n1, n2, n3)
			nampla.UseOnAmbush = n3
		} else {
			r.printf("   Production capacity this turn will be %d.\n", productionCapacity)
		}
	}

	if nampla.ItemQuantity[model.RM] > 0 {
		r.printf("\n%ss (%s,C%d) carried over from last turn = %d\n",
			model.ItemName[model.RM], model.ItemAbbr[model.RM], model.ItemCarryCapacity[model.RM], nampla.ItemQuantity[model.RM])
	}

	rawMaterialUnits += nampla.ItemQuantity[model.RM]
	var availableToSpend int
	if rawMaterialUnits > productionCapacity {
		availableToSpend = productionCapacity
		nampla.Special = rawMaterialUnits - productionCapacity
	} else {
		availableToSpend = rawMaterialUnits
		nampla.Special = 0
	}

	n1 := availableToSpend
	n2 := ((r.fleetPercentCost * n1) + 5000) / 10000
	n3 := n1 - n2
	if nampla.Status&model.MiningColony == 0 && nampla.Status&model.ResortColony == 0 {
		r.printf("\nTotal available for spending this turn = %d - %d = %d\n", n1, n2, n3)
		nampla.UseOnAmbush = n3
		r.printf("\nShipyard capacity = %d\n", nampla.Shipyards)
	}

	r.doInventory(nampla, ships, sp)
}

// doInventory prints a planet's item inventory and the ships at it (the C
// engine's do_inventory label and the ship-listing loops).
func (r *reporter) doInventory(nampla *model.Nampla, ships []*model.Ship, sp *model.Species) {
	headerPrinted := false
	for i := 0; i < model.MaxItems; i++ {
		if nampla.ItemQuantity[i] > 0 && i != model.RM {
			if !headerPrinted {
				headerPrinted = true
				r.printf("\nPlanetary inventory:\n")
			}
			r.printf("   %ss (%s,C%d) = %d", model.ItemName[i], model.ItemAbbr[i], model.ItemCarryCapacity[i], nampla.ItemQuantity[i])
			if i == model.PD {
				r.printf(" (warship equivalence = %d tons)", 50*nampla.ItemQuantity[i])
			}
			r.printf("\n")
		}
	}

	r.printingAlien = false
	// Starbases, then transports, then everything else.
	r.listShipsAt(nampla, ships, sp, classOnly(model.BA))
	r.listShipsAt(nampla, ships, sp, classOnly(model.TR))
	r.listShipsAt(nampla, ships, sp, nil)
}

type shipFilter func(*model.Ship) bool

func classOnly(class int) shipFilter {
	return func(s *model.Ship) bool { return s.Class == class }
}

func (r *reporter) listShipsAt(nampla *model.Nampla, ships []*model.Ship, sp *model.Species, filter shipFilter) {
	headerPrinted := false
	for si, ship := range ships {
		if ship.X != nampla.X || ship.Y != nampla.Y || ship.Z != nampla.Z || ship.PN != nampla.PN {
			continue
		}
		if filter != nil {
			if !filter(ship) {
				continue
			}
		} else if r.shipAlreadyListed[si] {
			continue
		}
		if !headerPrinted {
			r.printf("\nShips at PL %s:\n", nampla.Name)
			r.printShipHeader()
		}
		headerPrinted = true
		r.printShip(ship, sp, r.speciesNumber)
		r.shipAlreadyListed[si] = true
	}
}

func (r *reporter) printShipHeader() {
	r.printf("  Name                          ")
	if r.printingAlien {
		r.printf("                     Species\n")
	} else {
		r.printf("                 Cap. Cargo\n")
	}
	r.printf(" ---------------------------------------")
	r.printf("-------------------------------------\n")
}

func (r *reporter) printMishapChance(ship *model.Ship, destx, desty, destz int) {
	if destx == 9999 {
		r.printf("Mishap chance = ???")
		return
	}
	mishapGV := r.species.TechLevel[model.GV]
	mishapAge := ship.Age
	mishapChance := (100 * (((destx - ship.X) * (destx - ship.X)) + ((desty - ship.Y) * (desty - ship.Y)) +
		((destz - ship.Z) * (destz - ship.Z)))) / mishapGV
	if mishapAge > 0 && mishapChance < 10000 {
		successChance := 10000 - mishapChance
		successChance -= (2 * mishapAge * successChance) / 100
		mishapChance = 10000 - successChance
	}
	if mishapChance > 10000 {
		mishapChance = 10000
	}
	r.printf("mishap chance = %d.%02d%%", mishapChance/100, mishapChance%100)
}

func (r *reporter) printShip(ship *model.Ship, sp *model.Species, speciesNumber int) {
	r.ignoreFieldDistorters = !r.printingAlien

	r.printf("  %s", r.shipName(ship))

	length := len(r.fullShipID)
	var n int
	if r.printingAlien {
		n = 50
	} else {
		n = 46
	}
	for i := 0; i < (n - length); i++ {
		r.printf(" ")
	}

	var capacity int
	switch {
	case ship.Class == model.BA:
		capacity = 10 * ship.Tonnage
	case ship.Class == model.TR:
		capacity = (10 + (ship.Tonnage / 2)) * ship.Tonnage
	default:
		capacity = ship.Tonnage
	}

	if r.printingAlien {
		r.printf(" ")
	} else {
		r.printf("%4d  ", capacity)
		if ship.Status == model.UnderConstruction {
			r.printf("Left to pay = %d\n", ship.RemainingCost)
			return
		}
	}

	if r.printingAlien {
		if ship.Status == model.OnSurface || ship.ItemQuantity[model.FD] != ship.Tonnage {
			r.printf("SP %s", sp.Name)
		} else {
			r.printf("SP %d", r.distorted(speciesNumber))
		}
	} else {
		needComma := false
		for i := 0; i < model.MaxItems; i++ {
			if ship.ItemQuantity[i] > 0 {
				if needComma {
					r.printf(",")
				}
				r.printf("%d %s", ship.ItemQuantity[i], model.ItemAbbr[i])
				needComma = true
			}
		}
	}
	r.printf("\n")
}

// reportAliens prints the "Aliens at ..." sections for every location the
// current species occupies that also holds another species.
func (r *reporter) reportAliens() {
	sp := r.species
	r.printingAlien = true
	for _, myLoc := range r.w.Locations {
		if myLoc.S != r.speciesNumber {
			continue
		}
		headerPrinted := false
		for _, itsLoc := range r.w.Locations {
			if itsLoc.S == r.speciesNumber {
				continue
			}
			if myLoc.X != itsLoc.X || myLoc.Y != itsLoc.Y || myLoc.Z != itsLoc.Z {
				continue
			}
			alien := r.w.SpeciesByNumber(itsLoc.S)
			if alien == nil {
				continue
			}

			ourNampla := (*model.Nampla)(nil)
			for _, nampla := range sp.Namplas {
				if nampla.X != myLoc.X || nampla.Y != myLoc.Y || nampla.Z != myLoc.Z || nampla.PN == 99 {
					continue
				}
				ourNampla = nampla
				break
			}

			for _, alienNampla := range alien.Namplas {
				if myLoc.X != alienNampla.X || myLoc.Y != alienNampla.Y || myLoc.Z != alienNampla.Z {
					continue
				}
				if alienNampla.Status&model.Populated == 0 {
					continue
				}
				weHaveColonyHere := false
				for _, nampla := range sp.Namplas {
					if alienNampla.X != nampla.X || alienNampla.Y != nampla.Y || alienNampla.Z != nampla.Z || alienNampla.PN != nampla.PN {
						continue
					}
					if nampla.Status&model.Populated == 0 {
						continue
					}
					weHaveColonyHere = true
					break
				}
				if alienNampla.Hidden != 0 && !weHaveColonyHere {
					continue
				}
				if !headerPrinted {
					r.printAlienHeader(myLoc, ourNampla)
					headerPrinted = true
				}

				industry := alienNampla.MiBase + alienNampla.MaBase
				var temp1 string
				switch {
				case alienNampla.Status&model.MiningColony != 0:
					temp1 = "Mining colony"
				case alienNampla.Status&model.ResortColony != 0:
					temp1 = "Resort colony"
				case alienNampla.Status&model.HomePlanet != 0:
					temp1 = "Home planet"
				case industry > 0:
					temp1 = "Colony planet"
				default:
					temp1 = "Uncolonized planet"
				}
				temp2 := fmt.Sprintf("  %s PL %s (pl #%d)", temp1, alienNampla.Name, alienNampla.PN)
				for j := 0; j < 53-len(temp2); j++ {
					temp2 += " "
				}
				r.printf("%sSP %s\n", temp2, alien.Name)

				j := industry
				if industry < 100 {
					industry = (industry + 5) / 10
				} else {
					industry = ((industry + 50) / 100) * 10
				}
				if j == 0 {
					r.printf("      (No economic base.)\n")
				} else {
					r.printf("      (Economic base is approximately %d.)\n", industry)
				}

				if weHaveColonyHere {
					if alienNampla.ItemQuantity[model.PD] == 1 {
						r.printf("      (There is 1 %s on the planet.)\n", model.ItemName[model.PD])
					} else if alienNampla.ItemQuantity[model.PD] > 1 {
						r.printf("      (There are %d %ss on the planet.)\n", alienNampla.ItemQuantity[model.PD], model.ItemName[model.PD])
					}
					if alienNampla.Shipyards == 1 {
						r.printf("      (There is 1 shipyard on the planet.)\n")
					} else if alienNampla.Shipyards > 1 {
						r.printf("      (There are %d shipyards on the planet.)\n", alienNampla.Shipyards)
					}
				}
				if alienNampla.Hidden != 0 {
					r.printf("      (Colony is actively hiding from alien observation.)\n")
				}
			}

			for _, alienShip := range alien.Ships {
				if alienShip.PN == 99 {
					continue
				}
				if myLoc.X != alienShip.X || myLoc.Y != alienShip.Y || myLoc.Z != alienShip.Z {
					continue
				}
				alienCanHide := true
				for _, nampla := range sp.Namplas {
					if alienShip.X != nampla.X || alienShip.Y != nampla.Y || alienShip.Z != nampla.Z || alienShip.PN != nampla.PN {
						continue
					}
					if nampla.Status&model.Populated != 0 {
						alienCanHide = false
						break
					}
				}
				if alienCanHide && alienShip.Status == model.OnSurface {
					continue
				}
				if alienCanHide && alienShip.Status == model.UnderConstruction {
					continue
				}
				if !headerPrinted {
					r.printAlienHeader(myLoc, ourNampla)
					headerPrinted = true
				}
				r.printShip(alienShip, alien, alien.ID)
			}
		}
	}
	r.printingAlien = false
}

func (r *reporter) printAlienHeader(myLoc model.Location, ourNampla *model.Nampla) {
	r.printf("\n\nAliens at x = %d, y = %d, z = %d", myLoc.X, myLoc.Y, myLoc.Z)
	if ourNampla != nil {
		r.printf(" (PL %s star system)", ourNampla.Name)
	}
	r.printf(":\n")
}

// shipName ports ship_name: a complete ship identifier including orbital/landed
// status and age unless truncateName is set.
func (r *reporter) shipName(ship *model.Ship) string {
	shipIsDistorted := ship.ItemQuantity[model.FD] == ship.Tonnage
	if ship.Status == model.OnSurface {
		shipIsDistorted = false
	}
	if r.ignoreFieldDistorters {
		shipIsDistorted = false
	}

	switch {
	case shipIsDistorted:
		switch {
		case ship.Class == model.TR:
			r.fullShipID = fmt.Sprintf("%s%d ???", model.ShipAbbr[ship.Class], ship.Tonnage)
		case ship.Class == model.BA:
			r.fullShipID = "BAS ???"
		default:
			r.fullShipID = fmt.Sprintf("%s ???", model.ShipAbbr[ship.Class])
		}
	case ship.Class == model.TR:
		r.fullShipID = fmt.Sprintf("%s%d%s %s", model.ShipAbbr[ship.Class], ship.Tonnage, model.ShipType[ship.ShipType], ship.Name)
	default:
		r.fullShipID = fmt.Sprintf("%s%s %s", model.ShipAbbr[ship.Class], model.ShipType[ship.ShipType], ship.Name)
	}

	if r.truncateName {
		return r.fullShipID
	}

	r.fullShipID += " ("
	effectiveAge := ship.Age
	if effectiveAge < 0 {
		effectiveAge = 0
	}
	if !shipIsDistorted {
		if ship.Status != model.UnderConstruction {
			r.fullShipID += fmt.Sprintf("A%d,", effectiveAge)
		}
	}
	var temp string
	switch ship.Status {
	case model.UnderConstruction:
		temp = "C"
	case model.InOrbit:
		temp = fmt.Sprintf("O%d", ship.PN)
	case model.OnSurface:
		temp = fmt.Sprintf("L%d", ship.PN)
	case model.InDeepSpace:
		temp = "D"
	case model.ForcedJump:
		temp = "FJ"
	case model.JumpedInCombat:
		temp = "WD"
	default:
		temp = "***???***"
	}
	r.fullShipID += temp
	if ship.ShipType == model.Starbase {
		r.fullShipID += fmt.Sprintf(",%d tons", 10000*ship.Tonnage)
	}
	r.fullShipID += ")"
	return r.fullShipID
}

// distorted ports the species-number distortion used for field-distorter ships.
func (r *reporter) distorted(speciesNumber int) int {
	sp := r.w.SpeciesByNumber(speciesNumber)
	ls := 0
	if sp != nil {
		ls = sp.InitTechLevel[model.LS]
	}
	i := speciesNumber & 0x000F
	j := (speciesNumber >> 4) & 0x000F
	return (ls%5+3)*(4*i+j) + (ls%11 + 7)
}

// lifeSupportNeeded ports life_support_needed.
func lifeSupportNeeded(sp *model.Species, home, colony *model.Planet) int {
	if sp == nil || home == nil || colony == nil {
		return 99
	}
	tc := colony.TemperatureClass - home.TemperatureClass
	if tc < 0 {
		tc = -tc
	}
	pc := colony.PressureClass - home.PressureClass
	if pc < 0 {
		pc = -pc
	}
	hasRequiredGas := false
	poisonGases := 0
	for j := 0; j < 4; j++ {
		if colony.GasPercent[j] != 0 {
			if colony.Gas[j] == sp.RequiredGas {
				if sp.RequiredGasMin <= colony.GasPercent[j] && colony.GasPercent[j] <= sp.RequiredGasMax {
					hasRequiredGas = true
				}
			} else {
				for i := 0; i < 6; i++ {
					if colony.Gas[j] == sp.PoisonGas[i] {
						poisonGases++
						break
					}
				}
			}
		}
	}
	lsNeeded := 3 * (tc + pc + poisonGases)
	if !hasRequiredGas {
		lsNeeded += 3
	}
	return lsNeeded
}

// closestUnvisitedStarReport ports closest_unvisited_star_report: it writes the
// nearest unvisited star's coordinates, marks it visited, and records the
// destination in jx/jy/jz (report.c's globals x, y, z).
func (r *reporter) closestUnvisitedStarReport(ship *model.Ship) {
	speciesArrayIndex := (r.speciesNumber - 1) / 32
	speciesBitNumber := (r.speciesNumber - 1) % 32
	speciesBitMask := uint32(1) << uint(speciesBitNumber)

	r.jx = 9999
	closestDistance := 999999
	var closestStar *model.System
	found := false
	for _, star := range r.w.Systems {
		if star.VisitedBy[speciesArrayIndex]&speciesBitMask != 0 {
			continue
		}
		d := ((ship.X - star.X) * (ship.X - star.X)) + ((ship.Y - star.Y) * (ship.Y - star.Y)) + ((ship.Z - star.Z) * (ship.Z - star.Z))
		if d < closestDistance {
			r.jx = star.X
			r.jy = star.Y
			r.jz = star.Z
			closestDistance = d
			closestStar = star
			found = true
		}
	}
	if found {
		r.printf("%d %d %d", r.jx, r.jy, r.jz)
		closestStar.VisitedBy[speciesArrayIndex] |= speciesBitMask
	} else {
		r.printf("???")
	}
}

// commas ports the comma-grouping integer formatter.
func commas(value int) string {
	neg := value < 0
	abs := value
	if neg {
		abs = -value
	}
	digits := strconv.Itoa(abs)
	var out []byte
	for i, n := 0, len(digits); i < len(digits); i, n = i+1, n-1 {
		if i != 0 && n%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, digits[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// --- log_char / log_string wrapping (report.c uses the log utilities for the
// species-met / allies / enemies block). Ported to instance state. ---

func (r *reporter) logString(s string) {
	for i := 0; i < len(s); i++ {
		r.logChar(s[i])
	}
}

func (r *reporter) logChar(c byte) {
	if (c == ' ' || c == '\n') && r.logPos > 77 {
		tempPosition := r.logPos - 1
		for r.logLine[tempPosition] != ' ' {
			tempPosition--
		}
		tempChar := r.logLine[tempPosition+1]
		r.logLine[tempPosition] = '\n'
		r.logLine[tempPosition+1] = 0
		r.logWriteLine()
		r.logLine[tempPosition+1] = tempChar
		r.logLine[r.logPos] = 0
		r.logPos = r.logIndent + 2
		for i := 0; i < r.logPos; i++ {
			r.logLine[i] = ' '
		}
		src := tempPosition + 1
		dst := r.logPos
		for r.logLine[src] != 0 {
			r.logLine[dst] = r.logLine[src]
			dst++
			src++
		}
		r.logLine[dst] = 0
		r.logPos = dst
		if c == ' ' {
			r.logLine[r.logPos] = ' '
			r.logPos++
			return
		}
	}

	if c == '\n' {
		r.logLine[r.logPos] = '\n'
		r.logLine[r.logPos+1] = 0
		r.logWriteLine()
		r.logPos = 0
		r.logIndent = 0
		r.logStartOfLine = true
		return
	}

	r.logLine[r.logPos] = c
	r.logPos++

	if r.logStartOfLine && c == ' ' {
		r.logIndent++
	} else {
		r.logStartOfLine = false
	}
}

func (r *reporter) logWriteLine() {
	n := 0
	for n < len(r.logLine) && r.logLine[n] != 0 {
		n++
	}
	r.out.Write(r.logLine[:n])
}
