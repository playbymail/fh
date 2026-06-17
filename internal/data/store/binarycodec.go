package store

// Binary record codecs for the classic Far Horizons .dat files. The on-disk
// layouts mirror the binary_*_data_t structs in the C engine (data.h) with
// native x86-64 alignment, little-endian, and reserved/padding bytes written
// as zero — byte-identical to fhc's files. They are re-implemented here from
// model.World so the binary store never depends on internal/game (the C port
// with its package globals).

import (
	"encoding/binary"

	"github.com/playbymail/fh/internal/model"
)

const (
	binGalaxySize   = 16
	binStarSize     = 52
	binPlanetSize   = 40
	binSpeciesSize  = 264
	binNamplaSize   = 288
	binShipSize     = 172
	binLocationSize = 4
)

func boolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// copyName encodes a Go string into a NUL-padded fixed byte array (C zstrcpy
// semantics: zero-filled, always NUL-terminated).
func copyName(dst []byte, name string) {
	for i := range dst {
		dst[i] = 0
	}
	n := len(name)
	if n > len(dst)-1 {
		n = len(dst) - 1
	}
	copy(dst, name[:n])
}

// nameString decodes a NUL-padded fixed byte array, truncating at the first NUL.
func nameString(b []byte) string {
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// --- galaxy (16 bytes) ---

func encodeGalaxy(g model.Galaxy) []byte {
	b := make([]byte, binGalaxySize)
	binary.LittleEndian.PutUint32(b[0:], uint32(int32(g.DNumSpecies)))
	binary.LittleEndian.PutUint32(b[4:], uint32(int32(g.NumSpecies)))
	binary.LittleEndian.PutUint32(b[8:], uint32(int32(g.Radius)))
	binary.LittleEndian.PutUint32(b[12:], uint32(int32(g.TurnNumber)))
	return b
}

func decodeGalaxy(b []byte) model.Galaxy {
	return model.Galaxy{
		DNumSpecies: int(int32(binary.LittleEndian.Uint32(b[0:]))),
		NumSpecies:  int(int32(binary.LittleEndian.Uint32(b[4:]))),
		Radius:      int(int32(binary.LittleEndian.Uint32(b[8:]))),
		TurnNumber:  int(int32(binary.LittleEndian.Uint32(b[12:]))),
	}
}

// --- star / system (52 bytes) ---

func encodeStar(dst []byte, s *model.System, planetIndex, numPlanets int) {
	for i := range dst[:binStarSize] {
		dst[i] = 0
	}
	dst[0] = byte(s.X)
	dst[1] = byte(s.Y)
	dst[2] = byte(s.Z)
	dst[3] = byte(s.Type)
	dst[4] = byte(s.Color)
	dst[5] = byte(s.Size)
	dst[6] = byte(numPlanets)
	dst[7] = boolByte(s.HomeSystem)
	dst[8] = boolByte(s.WormHere)
	dst[9] = byte(s.WormX)
	dst[10] = byte(s.WormY)
	dst[11] = byte(s.WormZ)
	binary.LittleEndian.PutUint16(dst[16:], uint16(int16(planetIndex)))
	binary.LittleEndian.PutUint32(dst[20:], uint32(int32(s.Message)))
	for j := 0; j < model.NumContactWords; j++ {
		binary.LittleEndian.PutUint32(dst[24+4*j:], s.VisitedBy[j])
	}
}

// decodeStar returns the system plus its flat planet_index and planet count.
func decodeStar(b []byte) (sys *model.System, planetIndex, numPlanets int) {
	sys = &model.System{
		X: int(b[0]), Y: int(b[1]), Z: int(b[2]),
		Type: int(b[3]), Color: int(b[4]), Size: int(b[5]),
		HomeSystem: b[7] != 0, WormHere: b[8] != 0,
		WormX: int(b[9]), WormY: int(b[10]), WormZ: int(b[11]),
		Message: int(int32(binary.LittleEndian.Uint32(b[20:]))),
	}
	numPlanets = int(b[6])
	planetIndex = int(int16(binary.LittleEndian.Uint16(b[16:])))
	for j := 0; j < model.NumContactWords; j++ {
		sys.VisitedBy[j] = binary.LittleEndian.Uint32(b[24+4*j:])
	}
	return sys, planetIndex, numPlanets
}

// --- planet (40 bytes) ---

func encodePlanet(dst []byte, p *model.Planet) {
	for i := range dst[:binPlanetSize] {
		dst[i] = 0
	}
	dst[0] = byte(p.TemperatureClass)
	dst[1] = byte(p.PressureClass)
	dst[2] = byte(p.Special)
	for g := 0; g < 4; g++ {
		dst[4+g] = byte(p.Gas[g])
		dst[8+g] = byte(p.GasPercent[g])
	}
	binary.LittleEndian.PutUint16(dst[14:], uint16(int16(p.Diameter)))
	binary.LittleEndian.PutUint16(dst[16:], uint16(int16(p.Gravity)))
	binary.LittleEndian.PutUint16(dst[18:], uint16(int16(p.MiningDifficulty)))
	binary.LittleEndian.PutUint16(dst[20:], uint16(int16(p.EconEfficiency)))
	binary.LittleEndian.PutUint16(dst[22:], uint16(int16(p.MdIncrease)))
	binary.LittleEndian.PutUint32(dst[24:], uint32(int32(p.Message)))
}

func decodePlanet(b []byte) *model.Planet {
	p := &model.Planet{
		TemperatureClass: int(b[0]), PressureClass: int(b[1]), Special: int(b[2]),
		Diameter:         int(int16(binary.LittleEndian.Uint16(b[14:]))),
		Gravity:          int(int16(binary.LittleEndian.Uint16(b[16:]))),
		MiningDifficulty: int(int16(binary.LittleEndian.Uint16(b[18:]))),
		EconEfficiency:   int(int16(binary.LittleEndian.Uint16(b[20:]))),
		MdIncrease:       int(int16(binary.LittleEndian.Uint16(b[22:]))),
		Message:          int(int32(binary.LittleEndian.Uint32(b[24:]))),
	}
	for g := 0; g < 4; g++ {
		p.Gas[g] = int(b[4+g])
		p.GasPercent[g] = int(b[8+g])
	}
	return p
}

// --- species record (264 bytes) ---

func encodeSpecies(dst []byte, sp *model.Species) {
	for i := range dst[:binSpeciesSize] {
		dst[i] = 0
	}
	copyName(dst[0:32], sp.Name)
	copyName(dst[32:64], sp.GovtName)
	copyName(dst[64:96], sp.GovtType)
	dst[96] = byte(sp.X)
	dst[97] = byte(sp.Y)
	dst[98] = byte(sp.Z)
	dst[99] = byte(sp.PN)
	dst[100] = byte(sp.RequiredGas)
	dst[101] = byte(sp.RequiredGasMin)
	dst[102] = byte(sp.RequiredGasMax)
	for g := 0; g < 6; g++ {
		dst[104+g] = byte(sp.NeutralGas[g])
		dst[110+g] = byte(sp.PoisonGas[g])
	}
	dst[116] = boolByte(sp.AutoOrders)
	for j := 0; j < 6; j++ {
		binary.LittleEndian.PutUint16(dst[120+2*j:], uint16(int16(sp.TechLevel[j])))
		binary.LittleEndian.PutUint16(dst[132+2*j:], uint16(int16(sp.InitTechLevel[j])))
		binary.LittleEndian.PutUint16(dst[144+2*j:], uint16(int16(sp.TechKnowledge[j])))
		binary.LittleEndian.PutUint32(dst[164+4*j:], uint32(int32(sp.TechEps[j])))
	}
	binary.LittleEndian.PutUint32(dst[156:], uint32(int32(len(sp.Namplas))))
	binary.LittleEndian.PutUint32(dst[160:], uint32(int32(len(sp.Ships))))
	binary.LittleEndian.PutUint32(dst[188:], uint32(int32(sp.HpOriginalBase)))
	binary.LittleEndian.PutUint32(dst[192:], uint32(int32(sp.EconUnits)))
	binary.LittleEndian.PutUint32(dst[196:], uint32(int32(sp.FleetCost)))
	binary.LittleEndian.PutUint32(dst[200:], uint32(int32(sp.FleetPercentCost)))
	for j := 0; j < model.NumContactWords; j++ {
		binary.LittleEndian.PutUint32(dst[204+4*j:], sp.Contact[j])
		binary.LittleEndian.PutUint32(dst[220+4*j:], sp.Ally[j])
		binary.LittleEndian.PutUint32(dst[236+4*j:], sp.Enemy[j])
	}
}

// decodeSpecies returns the species plus its nampla and ship counts.
func decodeSpecies(b []byte) (sp *model.Species, numNamplas, numShips int) {
	sp = &model.Species{
		Name: nameString(b[0:32]), GovtName: nameString(b[32:64]), GovtType: nameString(b[64:96]),
		X: int(b[96]), Y: int(b[97]), Z: int(b[98]), PN: int(b[99]),
		RequiredGas: int(b[100]), RequiredGasMin: int(b[101]), RequiredGasMax: int(b[102]),
		AutoOrders:       b[116] != 0,
		HpOriginalBase:   int(int32(binary.LittleEndian.Uint32(b[188:]))),
		EconUnits:        int(int32(binary.LittleEndian.Uint32(b[192:]))),
		FleetCost:        int(int32(binary.LittleEndian.Uint32(b[196:]))),
		FleetPercentCost: int(int32(binary.LittleEndian.Uint32(b[200:]))),
	}
	for g := 0; g < 6; g++ {
		sp.NeutralGas[g] = int(b[104+g])
		sp.PoisonGas[g] = int(b[110+g])
	}
	for j := 0; j < 6; j++ {
		sp.TechLevel[j] = int(int16(binary.LittleEndian.Uint16(b[120+2*j:])))
		sp.InitTechLevel[j] = int(int16(binary.LittleEndian.Uint16(b[132+2*j:])))
		sp.TechKnowledge[j] = int(int16(binary.LittleEndian.Uint16(b[144+2*j:])))
		sp.TechEps[j] = int(int32(binary.LittleEndian.Uint32(b[164+4*j:])))
	}
	for j := 0; j < model.NumContactWords; j++ {
		sp.Contact[j] = binary.LittleEndian.Uint32(b[204+4*j:])
		sp.Ally[j] = binary.LittleEndian.Uint32(b[220+4*j:])
		sp.Enemy[j] = binary.LittleEndian.Uint32(b[236+4*j:])
	}
	numNamplas = int(int32(binary.LittleEndian.Uint32(b[156:])))
	numShips = int(int32(binary.LittleEndian.Uint32(b[160:])))
	sp.NumNamplas = numNamplas
	sp.NumShips = numShips
	return sp, numNamplas, numShips
}

// --- nampla record (288 bytes) ---

func encodeNampla(dst []byte, np *model.Nampla, planetIndex int) {
	for i := range dst[:binNamplaSize] {
		dst[i] = 0
	}
	copyName(dst[0:32], np.Name)
	dst[32] = byte(np.X)
	dst[33] = byte(np.Y)
	dst[34] = byte(np.Z)
	dst[35] = byte(np.PN)
	dst[36] = byte(np.Status)
	dst[38] = byte(np.Hiding)
	dst[39] = byte(np.Hidden)
	binary.LittleEndian.PutUint16(dst[42:], uint16(int16(planetIndex)))
	binary.LittleEndian.PutUint16(dst[44:], uint16(int16(np.SiegeEff)))
	binary.LittleEndian.PutUint16(dst[46:], uint16(int16(np.Shipyards)))
	binary.LittleEndian.PutUint32(dst[52:], uint32(int32(np.IUsNeeded)))
	binary.LittleEndian.PutUint32(dst[56:], uint32(int32(np.AUsNeeded)))
	binary.LittleEndian.PutUint32(dst[60:], uint32(int32(np.AutoIUs)))
	binary.LittleEndian.PutUint32(dst[64:], uint32(int32(np.AutoAUs)))
	binary.LittleEndian.PutUint32(dst[72:], uint32(int32(np.IUsToInstall)))
	binary.LittleEndian.PutUint32(dst[76:], uint32(int32(np.AUsToInstall)))
	binary.LittleEndian.PutUint32(dst[80:], uint32(int32(np.MiBase)))
	binary.LittleEndian.PutUint32(dst[84:], uint32(int32(np.MaBase)))
	binary.LittleEndian.PutUint32(dst[88:], uint32(int32(np.PopUnits)))
	for j := 0; j < model.MaxItems; j++ {
		binary.LittleEndian.PutUint32(dst[92+4*j:], uint32(int32(np.ItemQuantity[j])))
	}
	binary.LittleEndian.PutUint32(dst[248:], uint32(int32(np.UseOnAmbush)))
	binary.LittleEndian.PutUint32(dst[252:], uint32(int32(np.Message)))
	binary.LittleEndian.PutUint32(dst[256:], uint32(int32(np.Special)))
}

func decodeNampla(b []byte) *model.Nampla {
	np := &model.Nampla{
		Name: nameString(b[0:32]),
		X:    int(b[32]), Y: int(b[33]), Z: int(b[34]), PN: int(b[35]),
		Status: int(b[36]), Hiding: int(b[38]), Hidden: int(b[39]),
		SiegeEff:     int(int16(binary.LittleEndian.Uint16(b[44:]))),
		Shipyards:    int(int16(binary.LittleEndian.Uint16(b[46:]))),
		IUsNeeded:    int(int32(binary.LittleEndian.Uint32(b[52:]))),
		AUsNeeded:    int(int32(binary.LittleEndian.Uint32(b[56:]))),
		AutoIUs:      int(int32(binary.LittleEndian.Uint32(b[60:]))),
		AutoAUs:      int(int32(binary.LittleEndian.Uint32(b[64:]))),
		IUsToInstall: int(int32(binary.LittleEndian.Uint32(b[72:]))),
		AUsToInstall: int(int32(binary.LittleEndian.Uint32(b[76:]))),
		MiBase:       int(int32(binary.LittleEndian.Uint32(b[80:]))),
		MaBase:       int(int32(binary.LittleEndian.Uint32(b[84:]))),
		PopUnits:     int(int32(binary.LittleEndian.Uint32(b[88:]))),
		UseOnAmbush:  int(int32(binary.LittleEndian.Uint32(b[248:]))),
		Message:      int(int32(binary.LittleEndian.Uint32(b[252:]))),
		Special:      int(int32(binary.LittleEndian.Uint32(b[256:]))),
	}
	for j := 0; j < model.MaxItems; j++ {
		np.ItemQuantity[j] = int(int32(binary.LittleEndian.Uint32(b[92+4*j:])))
	}
	return np
}

// --- ship record (172 bytes) ---

func encodeShip(dst []byte, sh *model.Ship) {
	for i := range dst[:binShipSize] {
		dst[i] = 0
	}
	copyName(dst[0:32], sh.Name)
	dst[32] = byte(sh.X)
	dst[33] = byte(sh.Y)
	dst[34] = byte(sh.Z)
	dst[35] = byte(sh.PN)
	dst[36] = byte(sh.Status)
	dst[37] = byte(sh.ShipType)
	dst[38] = byte(sh.DestX)
	dst[39] = byte(sh.DestY)
	dst[40] = byte(sh.DestZ)
	dst[41] = byte(sh.JustJumped)
	dst[42] = byte(sh.ArrivedViaWormhole)
	binary.LittleEndian.PutUint16(dst[48:], uint16(int16(sh.Class)))
	binary.LittleEndian.PutUint16(dst[50:], uint16(int16(sh.Tonnage)))
	for j := 0; j < model.MaxItems; j++ {
		binary.LittleEndian.PutUint16(dst[52+2*j:], uint16(int16(sh.ItemQuantity[j])))
	}
	binary.LittleEndian.PutUint16(dst[128:], uint16(int16(sh.Age)))
	binary.LittleEndian.PutUint16(dst[130:], uint16(int16(sh.RemainingCost)))
	binary.LittleEndian.PutUint16(dst[134:], uint16(int16(sh.LoadingPoint)))
	binary.LittleEndian.PutUint16(dst[136:], uint16(int16(sh.UnloadingPoint)))
	binary.LittleEndian.PutUint32(dst[140:], uint32(int32(sh.Special)))
}

func decodeShip(b []byte) *model.Ship {
	sh := &model.Ship{
		Name: nameString(b[0:32]),
		X:    int(b[32]), Y: int(b[33]), Z: int(b[34]), PN: int(b[35]),
		Status: int(b[36]), ShipType: int(b[37]),
		DestX: int(b[38]), DestY: int(b[39]), DestZ: int(b[40]),
		JustJumped: int(b[41]), ArrivedViaWormhole: int(b[42]),
		Class:          int(int16(binary.LittleEndian.Uint16(b[48:]))),
		Tonnage:        int(int16(binary.LittleEndian.Uint16(b[50:]))),
		Age:            int(int16(binary.LittleEndian.Uint16(b[128:]))),
		RemainingCost:  int(int16(binary.LittleEndian.Uint16(b[130:]))),
		LoadingPoint:   int(int16(binary.LittleEndian.Uint16(b[134:]))),
		UnloadingPoint: int(int16(binary.LittleEndian.Uint16(b[136:]))),
		Special:        int(int32(binary.LittleEndian.Uint32(b[140:]))),
	}
	for j := 0; j < model.MaxItems; j++ {
		sh.ItemQuantity[j] = int(int16(binary.LittleEndian.Uint16(b[52+2*j:])))
	}
	return sh
}
