# FH - Far Horizons (Go Rewrite)

## Command Line Interface
A single binary to manage the data, execute game turns, and manage users.

## Layout
Suggested repo layout

```text
fh/
  dist/local          # development artifacts
  internal/
    data/
      store/          # persistence interfaces + sqlite impl
      migrate/        # migrations (goose or atlas)
    engine/           # core simulation engine (pure, deterministic)
      effects/        # write-set buffers, merge semantics
      orders/         # order parsing, schema, validators
      rng/            # scoped PRNG (PCG/Xoroshiro) + seed derivation
      schedule/       # dependency graph, batching, conflict rules
      world/          # world state, entity models, indexes
    reports/          # report builders (plain text/HTML/JSON), replay diffs
  main.go             # entry point for command line interface
  pkg/
    fh/               # stable public SDK (submit orders, parse, test helpers)
  tmp/
  testdata/
  version.go          # semver information for repository
```

## Engine

### Process model
Parse → Normalize → Validate → Stage (build dependency DAG) → Execute (parallel batches) → Resolve conflicts (deterministically) → Commit world → Emit reports/artifacts.

### Storage
* A thin persistence interface so stores can be swapped later
* JSON files by default
* SQLite (modernc/sqlite for portability) for future releases


## Core interfaces (sketches)

```go
// internal/engine/world/types.go
package world

type ID string // stable identifiers, e.g., "SYS:SOL", "FLEET:1234"

type Snapshot interface {
    GetEntity(id ID) (Entity, bool)
    // read-only views & indexes
}

type Mutable interface {
    Snapshot
    Upsert(Entity)
    Delete(ID)
    // internal, used by commit stage
}

type Entity interface {
    ID() ID
    Kind() string // "Star", "Fleet", "Colony", etc.
    // serialize/deserialize methods as needed
}
```

```go
// internal/engine/orders/types.go
package orders

import "github.com/playbymail/fh/internal/engine/rng"

type Context struct {
    GameID string
    Turn   int
    Phase  string // e.g., "Economic", "Movement", "Combat"
    Actor  string // player/faction issuing the order
    Rng    rng.Scoped
}

type Order interface {
    Key() string           // stable key (for logs & seed)
    Actor() string         // which faction
    Validate(w ReadOnly) error
    Dependencies(w ReadOnly) []string // IDs this order reads/writes
    Execute(w ReadWrite, ctx Context) (Effect, error)
}

type Effect interface {
    // pure description of changes; no side effects yet
    Targets() []string
}
```

```go
// internal/engine/rng/rng.go
package rng

type Scoped interface {
    // Deterministic draws:
    Uint64() uint64
    Float64() float64 // [0,1)
    Intn(n int) int
}

type Factory interface {
    // Derive a scoped RNG from a tuple of stable keys
    For(keys ...string) Scoped
}
```

```go
// internal/engine/schedule/schedule.go
package schedule

import "github.com/playbymail/fh/internal/engine/orders"

type Batch struct {
    Orders []orders.Order
}

type Planner interface {
    Plan([]orders.Order, ReadOnlyWorld) ([]Batch, error) // topo-sort by deps
}
```

## Deterministic Turns

Deterministic RNG (concrete plan)
	•	Use PCG-XSH-RR 64 or xoroshiro128+ (fast, well-known).
	•	Derive the 128-bit seed via HMAC-SHA256 of a canonical string:
seedInput := fmt.Sprintf("%s|%06d|%s|%s|%s", gameID, turn, phase, actorID, orderKey)
seed := first16Bytes(HMAC_SHA256(gameMasterSecret, seedInput))
	•	The engine never reads time.Now(); all randomness must flow through rng.Scoped.

Example implementation edge:

```go
// internal/engine/rng/pcg.go
func (f *FactoryImpl) For(keys ...string) Scoped {
    input := strings.Join(keys, "|")
    mac := hmac.New(sha256.New, f.masterKey)
    mac.Write([]byte(input))
    sum := mac.Sum(nil)
    var s0, s1 uint64
    s0 = binary.LittleEndian.Uint64(sum[0:8])
    s1 = binary.LittleEndian.Uint64(sum[8:16])
    return newXoroshiro128Plus(s0, s1)
}
```

## Data Model
Minimal data model (starter)

### Entities
TODO: Examine C source for actual Entities.

Entities (IDs are stable, human-readable):
	•	Star, Wormhole, Fleet, Colony, Industry, Stockpile, Tech, DiplomaticLink, OrderQueue.

### Relations
Core relations:
	•	Fleet at StarID (or en route), contains Ships and Cargo.
	•	Colony at StarID, owner faction, population, installations, stockpiles.
	•	Industry (per colony) with production recipes and throughput limits.
	•	Tech progress per faction.
	•	Map edges (wormholes/lanes) with FTL rules.

## Testing
Testing & replayability
	•	Golden tests for orders → effects → world diffs (store under testdata/).
	•	Property tests: run N random seeds, assert invariants (no negative cargo, mass/energy conservation, etc.).
	•	Replay tests: Given same inputs, checksums of snapshot+reports must match.

## Migration Path

Migration path from the C code
	1.	Inventory mechanics from the original FH C repo (you cited github.com/playbymail/Far-Horizons). Make a table: orders, phases, entity kinds, resources, edge cases.
	2.	Freeze semantics in markdown specs (these become acceptance tests).
	3.	Write thin C-state → JSON exporters (if the old save format is handy) to seed initial worlds for validation.
	4.	Implement the smallest vertical slice: Parsing + Movement phase for Fleets on a tiny map → reports → golden tests.
	5.	Grow outward: Mining/Production → Colonies → Tech → Diplomacy → Combat.

## Milestones
First milestones (2–3 weeks if focused)
	1.	Scaffold repo + CI + lint + modernc.org/sqlite storage.
	2.	RNG factory + test vectors (determinism).
	3.	Order parser v1 (subset) + validator.
	4.	World model v0 (Stars, Wormholes, Fleets) + movement rules.
	5.	Batch executor + effect reducer + movement policy.
	6.	Reports (text) + replay/digest command.

## License
Licenses
	•	Original FH C source: include the original license text verbatim (or, if truly public domain, add LICENSE-ORIGINAL.md clarifying the provenance and your understanding).
	•	New Go code: pick one:
		•	MIT (short, permissive, common in Go), or
		•	Apache-2.0 (patent grant; safer if you expect external contributors & possible patents).

Most Go infra/libs are MIT or Apache-2.0; for engines with many contributors, Apache-2.0 is slightly safer.
