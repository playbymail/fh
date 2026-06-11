package model

// Galaxy holds the cluster-wide scalars the renderer needs.
type Galaxy struct {
	TurnNumber  int
	NumSpecies  int
	DNumSpecies int
	Radius      int
}

// Planet is one orbit within a star system.
type Planet struct {
	System           *System // owning system (resolved when the World is built)
	Orbit            int     // planet number within the system (1-based)
	TemperatureClass int
	PressureClass    int
	Special          int
	Gas              [4]int
	GasPercent       [4]int
	Diameter         int
	Gravity          int
	MiningDifficulty int // times 100 (the C "base" value, e.g. 206 -> MD 2.06)
	MdIncrease       int
	EconEfficiency   int
	Message          int
}

// System is a star system at a fixed set of coordinates.
type System struct {
	X, Y, Z    int
	Type       int // star type code; informational only for the renderer
	Color      int
	Size       int
	HomeSystem bool
	WormHere   bool
	WormX      int
	WormY      int
	WormZ      int
	Message    int
	VisitedBy  [NumContactWords]uint32
	Planets    []*Planet // ordered by orbit
}

// Nampla is a named planet: a species' home world or a colony.
type Nampla struct {
	Name         string
	X, Y, Z, PN  int
	Status       int
	Hiding       int
	Hidden       int
	SiegeEff     int
	Shipyards    int
	IUsNeeded    int
	AUsNeeded    int
	AutoIUs      int
	AutoAUs      int
	IUsToInstall int
	AUsToInstall int
	MiBase       int
	MaBase       int
	PopUnits     int
	ItemQuantity [MaxItems]int
	UseOnAmbush  int
	Message      int
	Special      int

	Planet *Planet // the planet this colony sits on (resolved when built)
}

// Ship is a single vessel owned by a species.
type Ship struct {
	Name                string
	X, Y, Z, PN         int
	Status              int
	ShipType            int
	DestX, DestY, DestZ int
	JustJumped          int
	ArrivedViaWormhole  int
	Class               int
	Tonnage             int
	ItemQuantity        [MaxItems]int
	Age                 int
	RemainingCost       int
	LoadingPoint        int
	UnloadingPoint      int
	Special             int
}

// Species is one player's complete state.
type Species struct {
	ID               int // species number (1-based)
	Name             string
	GovtName         string
	GovtType         string
	X, Y, Z, PN      int
	RequiredGas      int
	RequiredGasMin   int
	RequiredGasMax   int
	NeutralGas       [6]int
	PoisonGas        [6]int
	AutoOrders       bool
	TechLevel        [6]int
	InitTechLevel    [6]int
	TechKnowledge    [6]int
	TechEps          [6]int
	HpOriginalBase   int
	EconUnits        int
	FleetCost        int
	FleetPercentCost int
	Contact          [NumContactWords]uint32
	Ally             [NumContactWords]uint32
	Enemy            [NumContactWords]uint32
	NumNamplas       int
	NumShips         int
	Namplas          []*Nampla
	Ships            []*Ship

	// Log is the accumulated event log (spNN.log) for the current turn,
	// carried verbatim through ingest and prepended to the turn report.
	Log []byte
}

// Location is a (species, coordinates) entry from the locations index, used by
// the report's "Aliens at ..." section.
type Location struct {
	S       int
	X, Y, Z int
}

// World is the complete game state assembled in memory for rendering.
type World struct {
	Galaxy    Galaxy
	Systems   []*System
	Species   []*Species
	Locations []Location
}

// Resolve links cross-references that are derived rather than stored: each
// nampla's owning planet and (implicitly) each species' home planet. Call it
// once after loading or building a World.
func (w *World) Resolve() {
	index := make(map[[3]int]*System, len(w.Systems))
	for _, s := range w.Systems {
		index[[3]int{s.X, s.Y, s.Z}] = s
	}
	for _, sp := range w.Species {
		for _, np := range sp.Namplas {
			np.Planet = planetAt(index, np.X, np.Y, np.Z, np.PN)
		}
	}
}

// planetAt returns the planet at the given coordinates and orbit, or nil.
func planetAt(index map[[3]int]*System, x, y, z, orbit int) *Planet {
	s := index[[3]int{x, y, z}]
	if s == nil {
		return nil
	}
	for _, p := range s.Planets {
		if p.Orbit == orbit {
			return p
		}
	}
	return nil
}

// SpeciesByNumber returns the species with the given 1-based number, or nil.
func (w *World) SpeciesByNumber(n int) *Species {
	for _, sp := range w.Species {
		if sp.ID == n {
			return sp
		}
	}
	return nil
}

// HomePlanet returns the species' home planet (the planet of its first nampla),
// mirroring the C engine's planet_base[nampla1_base[0].planet_index].
func (s *Species) HomePlanet() *Planet {
	if len(s.Namplas) == 0 {
		return nil
	}
	return s.Namplas[0].Planet
}

// ContactBit reports whether the given 0-based species index bit is set in a
// contact/ally/enemy bitfield.
func ContactBit(bits [NumContactWords]uint32, spIndex int) bool {
	return bits[spIndex/32]&(uint32(1)<<(uint(spIndex)%32)) != 0
}

// SetSpeciesBit sets the bit for a 1-based species number in a bitfield.
func SetSpeciesBit(bits *[NumContactWords]uint32, speciesNumber int) {
	idx := speciesNumber - 1
	bits[idx/32] |= uint32(1) << (uint(idx) % 32)
}
