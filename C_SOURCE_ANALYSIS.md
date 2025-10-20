# FH - Far Horizons (Go Rewrite)

## C Source Analysis: Data Types and Interfaces

Based on analysis of the Far Horizons C source (github.com/playbymail/Far-Horizons), here are the key data structures and concepts that need Go equivalents:

### Core Entities

#### Stars (`binary_star_data_t`)
- **Fields**: x, y, z coordinates; type (dwarf/degenerate/main/giant); color; size; num_planets; home_system flag; wormhole info; planet_index; visited_by bitmask
- **Purpose**: Represents star systems with planets and wormhole connections
- **Go Type**: `Star` struct with embedded `Location`

#### Planets (`binary_planet_data_t`)
- **Fields**: name, x/y/z/pn coords; status; hiding/hidden flags; siege_eff; shipyards; IU/AU needs; mi/ma_base; pop_units; item_quantities; special
- **Purpose**: Named planets (colonies) with economic bases, population, and production
- **Go Type**: `Planet` struct with `Location`, `Resources`, `Production`

#### Ships (`binary_ship_data_t`)
- **Fields**: name, x/y/z/pn coords; status; type (FTL/SUB_LIGHT/STARBASE); class (PB/CT/ES/...); tonnage; item_quantities; age; remaining_cost; loading/unloading points
- **Purpose**: All mobile units including transports, warships, starbases
- **Go Type**: `Ship` struct with `Location`, `Cargo`, `Status`

#### Species (`binary_species_data_t`)
- **Fields**: name, govt info, home coords; gas requirements; tech levels; contacts/allies/enemies bitmasks; econ_units; fleet info
- **Purpose**: Player/AI civilizations with tech, diplomacy, and economic state
- **Go Type**: `Species` struct with `Tech`, `Diplomacy`, `Economy`

### Order Processing Phases

From the turn runner (fh.c and related), turn phases are:
1. **TURN_UPDATE**: Update turn number
2. **LOCATION_UPDATE**: Update ship locations (first pass)
3. **COMBAT**: Resolve combat orders
4. **PRE_DEPARTURE**: Process pre-departure orders
5. **JUMP**: Execute jump orders
6. **PRODUCTION**: Process production orders
7. **POST_ARRIVAL**: Process post-arrival orders
8. **LOCATION_UPDATE**: Update ship locations (second pass)
9. **COMBAT_STRIKE**: Resolve strike combat
10. **FINISH**: Finalize turn (stats, damage, etc.)
11. **REPORT**: Generate player reports
12. **STATS**: Update statistics

### Order Types (from do.h function declarations)

**Movement/Positioning:**
- JUMP, MOVE, LAND, ORBIT, DEEP (space movement)
- WORMHOLE (wormhole navigation)

**Combat:**
- AMBUSH, INTERCEPT (combat positioning)
- COMBAT/STRIKE (resolution)

**Production/Economy:**
- BUILD, INSTALL, SHIPYARD (construction)
- PRODUCTION, DEVELOP, RESEARCH, TECH, UPGRADE (tech/economy)
- RECYCLE, DISBAND, DESTROY (resource management)

**Diplomacy/Intelligence:**
- ALLY, ENEMY, NEUTRAL (relations)
- SCAN, TELESCOPE, ESTIMATE, VISITED (recon)
- MESSAGE, SEND, TRANSFER (communication)

**Colony Management:**
- BASE, HIDE, TERRAFORM, UNLOAD (colony ops)
- TEACH (tech transfer)

### Key Constants and Enums

#### Ship Classes (ship.h)
- PB (0), CT (1), ES (2), FF (3), DD (4), CL (5), CS (6), CA (7), CC (8), BC (9), BS (10), DN (11), SD (12), BM (13), BW (14), BR (15), BA (16), TR (17)

#### Ship Types
- FTL (0), SUB_LIGHT (1), STARBASE (2)

#### Ship Status
- UNDER_CONSTRUCTION (0), ON_SURFACE (1), IN_ORBIT (2), IN_DEEP_SPACE (3), JUMPED_IN_COMBAT (4), FORCED_JUMP (5)

#### Items/Resources (item.h)
- 16 item types: FUEL, GOLD, etc. (need to catalog fully)

### Go Interface Design

```go
// Core entity interfaces
type Entity interface {
    ID() string
    Kind() string
    Location() Location
}

type Location struct {
    X, Y, Z, PN int
}

type Star struct {
    Location
    Type, Color, Size int
    NumPlanets int
    IsHomeSystem bool
    Wormhole Wormhole
    Planets []Planet
}

type Planet struct {
    Location
    Name string
    Status int
    Resources Resources
    Production Production
}

type Ship struct {
    Location
    Name string
    Class, Type int
    Status int
    Cargo []int // item quantities
}

// Order system
type Order interface {
    Execute(world World, species Species, rng RNG) ([]Effect, error)
}

type Phase int
const (
    TurnUpdate Phase = iota
    LocationUpdate
    Combat
    PreDeparture
    Jump
    Production
    PostArrival
    CombatStrike
    Finish
    Report
)
```

This analysis provides the foundation for implementing the world model and order processing in Go.
