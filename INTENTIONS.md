# FH

An approach:
* deterministic engine
* reproducible RNG
* clean server/runner split

## Engin

### Process model
Parse → Normalize → Validate → Stage (build dependency DAG) → Execute (parallel batches) → Resolve conflicts (deterministically) → Commit world → Emit reports/artifacts.

### Determinisms
All randomness comes from a scoped PRNG seeded from stable keys (e.g., {gameID, turn, phase, entityID, orderIndex}) so replays are byte-for-byte.

### Storage
* A thin persistence interface so stores can be swapped later
* JSON files by default
* SQLite (modernc/sqlite for portability) for future releases

## Command Line Interface
A single binary to manage the data, execute game turns, and manage users.

## Layout
Suggested repo layout

```text
fh/
  internal/
    engine/           # core simulation engine (pure, deterministic)
      orders/         # order parsing, schema, validators
      world/          # world state, entity models, indexes
      rng/            # scoped PRNG (PCG/Xoroshiro) + seed derivation
      schedule/       # dependency graph, batching, conflict rules
      effects/        # write-set buffers, merge semantics
    data/
      store/          # persistence interfaces + sqlite impl
      migrate/        # migrations (goose or atlas)
    reports/          # report builders (plain text/HTML/JSON), replay diffs
    api/              # wire types, request/response DTOs
    auth/             # (optional) if you expose sessions; JWT later
  pkg/
    fh/               # stable public SDK (submit orders, parse, test helpers)
  testdata/
  LICENSE
  README.md
  main.go
```

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

## Order Execution
Order execution model
	1.	Parse & Normalize
	•	Strict grammar (keep FH order names), produce typed orders with canonical args.
	2.	Validate
	•	Fast read-only checks (ownership, resource presence, legal targets).
	3.	Stage (build DAG)
	•	Each order declares read/write sets (e.g., reads: system S; writes: fleet F, colony C).
	•	Build batches by non-overlapping write sets; reads don’t conflict unless later writes collide.
	4.	Execute (parallel per batch)
	•	N workers run orders; each returns an Effect into a per-order buffer.
	5.	Conflict resolution (deterministic reducer)
	•	Conflicts resolved by policy + tie-break sort key (e.g., lowest ActorID, then lexicographic orderKey).
	•	Policies are per “domain”: movement lane capacity, mineral claim, colony build slots, combat target selection, etc.
	6.	Commit
	•	Apply merged Effects to a Mutable world, producing the next Snapshot.
	7.	Reports
	•	Generate per-player views + GM audit (diffs vs prior turn).

“Parallel” + determinism tricks
	•	No global maps mutated during execution; effects are local until merge.
	•	Stable iteration everywhere (sort keys: kind, ID, etc.).
	•	Index snapshots are immutable; new indexes derived after commit.
	•	Batching ensures true independence; if in doubt, push to next batch.

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

### Schema
SQLite schema (outline)

```sql
-- games & turns
CREATE TABLE game (
  id TEXT PRIMARY KEY,
  name TEXT, created_at TEXT
);

CREATE TABLE turn (
  game_id TEXT, num INTEGER, phase TEXT, started_at TEXT, ended_at TEXT,
  PRIMARY KEY (game_id, num, phase)
);

-- world snapshots (event-sourced or current-state tables)
CREATE TABLE entity (
  game_id TEXT, turn_num INTEGER, id TEXT, kind TEXT, blob BLOB,
  PRIMARY KEY (game_id, turn_num, id)
);

-- orders
CREATE TABLE orders (
  game_id TEXT, turn_num INTEGER, actor TEXT, seq INTEGER,
  raw TEXT, normalized JSON, status TEXT, error TEXT,
  PRIMARY KEY (game_id, turn_num, actor, seq)
);

-- reports
CREATE TABLE report (
  game_id TEXT, turn_num INTEGER, actor TEXT, mime TEXT, body BLOB,
  PRIMARY KEY (game_id, turn_num, actor, mime)
);

-- indexes as needed for lookups (entity_by_kind, by_location, etc.)
```

Use blob/json to store serialized entities (e.g., msgpack or JSON). Keep some columns duplicated (kind, location) for query speed.

## Turn Processing

Turn runner
	•	fh-runner run --game <id> --turn <n>: ingest orders, run phases, commit, write reports.
	•	fh-runner replay --game <id> --from 1 --to N: deterministic replay, emit digest per turn.
	•	fh-runner diff --game <id> --a N --b N: structural diff of snapshots and reports.

## Server Endpoints
Server endpoints (sketch)

```text
POST   /v1/games                 -> create game
GET    /v1/games/{id}            -> game info
POST   /v1/games/{id}/orders     -> submit orders (text or JSON list)
POST   /v1/games/{id}/turns/run  -> GM: trigger run
GET    /v1/games/{id}/reports/{turn}/{actor} -> fetch report
GET    /v1/games/{id}/state/{turn}           -> GM snapshot (debug)
```

Auth can be deferred (one-user GM secret) and later replaced by JWT.

## Policies
Conflict policies (examples to codify early)
	•	Movement lane capacity: If >capacity, sort applicants by (priority tag?, tech?, RNG draw seeded on lane+turn+fleet); losers queue next turn at origin.
	•	Mining/production slots: Fixed slots per colony; tie-break by colony initiative (derived RNG) then actor ID.
	•	Combat: Deterministic initiative order; RNG seeded by (battleID, round, side, unitID); all target rolls reproducible.

These policies belong under internal/engine/schedule/policy/.

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
