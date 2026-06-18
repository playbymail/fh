# `fhc script` — read-only query API (design)

*Design note. Companion to
[`ultron-harness-architecture.md`](ultron-harness-architecture.md) (§0 invariant,
§2.2 state oracle) and [`fhc-script-data-layout.md`](fhc-script-data-layout.md)
(Ultron layout). Tracked by epic #36; the current shape is #53. This describes
the implemented read-only scripting slice: query the current turn, a turn's
status, a turn's orders/report, and a species' statistics. Authoring / mutation
verbs are out of scope and are a later slice.*

## TL;DR

`fhc` (and later `fh`) embeds GopherLua (decided, #36) behind a `script`
subcommand and exposes a small, **read-only**, **access-controlled** set of Lua
query verbs. The API speaks **Ultron's language** — turns and species addressed
by bare integer id — and hides the engine's frozen flat-cwd / `sp%02d.dat` file
scheme behind the host. The host runs in one of two scopes set at the command
line: **GM** (every species) or **player** (only its own species). The script may
be an adversarial agent that tries to read another player's data; the host
**enforces the scope** so it cannot. The slice is **byte-neutral** on the golden
trees and **cannot perturb the RNG stream** (reads never call `rnd()`).

The verbs:

| Verb | Returns | Scope |
|---|---|---|
| `fh.current_turn()` | highest turn number N (N>0) | both |
| `fh.turn_status(n)` | `"pending"` \| `"resolved"` | both |
| `fh.orders(n [, sp])` | the whole orders file, or `nil` | per-species |
| `fh.report(n [, sp])` | the whole report (errors if turn unresolved) | per-species |
| `fh.species_stats(n [, sp])` | a structured stats table | per-species |

## Architecture — one host, many engines

The scripting host is **engine-agnostic** and lives in
[`interface/scripting`](../../interface/scripting). It owns the GopherLua
sandbox, the immutable scope, and the Lua verb bindings. The engine-specific data
access sits behind one interface:

```go
type Game interface {
    CurrentTurn() (n int, err error)
    TurnStatus(turn int) (status string, err error)
    SpeciesIDs() (ids []int, err error)
    Orders(turn, species int) (content string, ok bool, err error)
    Report(turn, species int) (content string, err error)
    SpeciesStats(turn, species int) (SpeciesStats, err error)
}
```

`fhc` implements `Game` **in-process** in
[`internal/game/script_game.go`](../../internal/game/script_game.go) — it reads
`galaxy.dat`/orders/reports straight from a turn dir and computes
`SpeciesStats` on demand. `fh` will implement the **same** `Game` in its own
terms (its SQLite store) later; the host never changes. There is no subprocess
and no `.dat`-format dependency in the interface — the contract is the verbs.

**Policy lives only in the host.** A `Game` implementation serves any
`(turn, species)` it is asked for; the host gates access against the scope. So
the security model is written once, engine-agnostic, and each `Game` stays
trivial.

## Guiding principle — address a game the way Ultron thinks about it

Ultron reasons in stable identifiers: *"turn 3, species 8, give me that data."*
The Lua surface honours that directly. The engine's on-disk reality — bare
filenames in cwd, species keyed by number, turn number embedded in `galaxy.dat`
— is an **implementation detail the host/Game resolves**, not something the
script author sees. The `fhc` Game resolves a turn by computing a directory under
the data root (and, for stats, `chdir`-ing into it before calling the unmodified
loaders); a different engine could resolve it however it likes. **`chdir` is a
mechanism, not the API.**

## Security model — Ultron agents may try to cheat

An Ultron player-agent is **untrusted**: its decision logic may attempt to read
another species' state to gain an unfair view. The design denies that with three
independent layers, all enforced by the host, none defeatable from inside the
script:

1. **Sandbox — no raw filesystem.** The Lua environment has no `io`, `os`,
   `require`/`package`, `load`/`loadfile`/`dofile`, or `debug`, and no
   `math.random`/`math.randomseed`. A script cannot open `sp03.dat` by hand; its
   *only* path to game data is the scoped query verbs.
2. **Scoped verbs — species reads are checked.** Every per-species verb
   (`orders`, `report`, `species_stats`) resolves its species through the host's
   scope check. A `--species 8` engine serves species 8 and denies any other id;
   under `--gm`, `species_stats(turn)` with no id returns the whole roster.
3. **Scope is set by the CLI, immutable in-script.** Whether the engine is GM or
   player-scoped is fixed by the `--gm` / `--species` flag at invocation. There
   is no Lua verb to change scope, so an adversarial script cannot widen it. The
   **trust boundary is the CLI invoker**: the Ultron harness chooses
   `--species N` for a player-agent and **never** passes `--gm`.

| Scope  | Selected by      | Turns visible | Species data visible |
|--------|------------------|---------------|----------------------|
| GM     | `--gm`           | all           | all                  |
| Player | `--species <id>` | all           | only `<id>`          |

Galaxy-level metadata (the current turn, a turn's status) is shared, so it is
visible in both scopes. Per-species data (orders, report, stats) is
scope-restricted.

Scope resolution for the per-species verbs:

- **Player scope**: the species is fixed to the caller's id. The argument may be
  omitted (implied) or stated explicitly; any other id is denied.
- **GM scope**: a species id is required for `orders`/`report` (there is no
  implied single species). `species_stats` with the id omitted returns an array
  of every species' stats, ascending.

## What the engine code already gives us (ground truth)

| Fact | Location | Consequence |
|---|---|---|
| Dispatch is a linear `if/else if` chain | `internal/game/fh.go` | `script` is one more branch → `scriptCommand(args)`, returns 0/2 like every command. |
| Loaders read **bare filenames in cwd** | `galaxyio.go`, `stario.go`, `planetio.go`, `speciesio.go` | The fhc Game `chdir`s into the turn dir and reuses them unmodified for `SpeciesStats`. Parameterizing them would edit frozen loaders → forbidden. |
| No loader touches the PRNG | (all `get_*_data`) | Loading is RNG-neutral; a read-only script leaves `prngSeed == 0`. |
| Turn number lives in `galaxy.dat`, not a dir name | `galaxyio.go` (`galaxy.turn_number`) | `CurrentTurn`/`TurnStatus` read it; the report filename's `t<turn>` suffix comes from it too. |
| `galaxy.dat` is a 16-byte record | `binary_galaxy_data_size` | `TurnStatus`/`SpeciesIDs` read the four ints directly, without disturbing globals or the PRNG. |
| `stats` computes per-species figures | `stats.go` | `SpeciesStats` recomputes the same figures structurally; a drift test asserts they match the frozen `statsCommand` output. |
| State is package-level globals; `ResetState()` zeroes them | `vars.go` | `SpeciesStats` calls `ResetState()` then loads, so each query is clean. Only one turn is loaded at a time. |
| Reports are `sp%02d.rpt.t<turn>` | `report.go` | `Report` reads that file; orders are `sp%02d.ord`. |

## Data-root layout (the host's, not the engine's)

```
<data-root>/                     # the single game; --data-root points here
  <turn-id>/                     # bare integer, e.g. 3
    galaxy.dat stars.dat planets.dat sp01.dat … spNN.dat   # the engine's flat store
    sp01.ord … spNN.ord          # per-species orders for the turn
    sp01.rpt.t<turn> …           # per-species reports (resolved turns)
    <species-id>/                # bare integer, e.g. 8
      orders                     # staged orders (Ultron writes here)
```

The fhc Game reads the turn directory directly: the `.dat` store for stats, the
flat `sp%02d.ord` (or the staged `<species>/orders` slot) for orders, and the
`sp%02d.rpt.t<turn>` files for reports. The data root never holds `.dat` files
directly — only turn folders. Turn 0 (genesis) is not an Ultron-addressable turn
and is omitted; the active game surfaces from folder 1 upward.

## CLI shape

```
fhc script --data-root=<dir> ( --gm | --species=<id> ) <file.lua>
```

- `--data-root=<dir>` — required; the dir of turn folders. Resolved to an
  absolute path before any turn `chdir`.
- `--gm` | `--species=<id>` — exactly one required, mutually exclusive. Sets the
  immutable scope.
- `<file.lua>` — resolved to an absolute path against the **original** cwd,
  *before* any turn `chdir`, so the script path is unaffected by turn selection.
- Slots into `fh.go` as `else if arg == "script" { runCommand(scriptCommand(args)) }`.
  Bad combinations (`--gm` with `--species`, neither, missing `--data-root`,
  invalid species) error with exit 2 before any Lua runs. Flags use the
  `--opt=val` form (the `internal/game` convention); space-separated values are
  rejected.

## Lua API

```lua
-- Galaxy-level metadata (shared; both scopes).
local n  = fh.current_turn()          -- highest turn number, e.g. 3
local st = fh.turn_status(3)          -- "pending" | "resolved"

-- Per-species, scope-enforced. Under --species the id is implied.
local ord = fh.orders(3)              -- whole orders file, or nil if none
local rpt = fh.report(3)              -- whole report; raises if turn 3 is pending
local s   = fh.species_stats(3)       -- one stats table

-- Under --gm the id is required for orders/report:
local ord8 = fh.orders(3, 8)
local rpt8 = fh.report(3, 8)

-- Under --gm, species_stats with no id returns the whole roster (ascending):
for _, s in ipairs(fh.species_stats(3)) do
  print(s.species, s.name, s.total_production, s.tech.MI, s.econ_units)
end
```

The `species_stats` table mirrors the per-species columns of the `stats` report
so the scripted view and the stats command never drift:

```
{ species, name,
  tech = { MI, MA, ML, GV, LS, BI },
  total_production, num_planets, num_ships, num_shipyards,
  offensive_power, defensive_power, econ_units }
```

Return-shape rules:

- `orders`/`report` return the **entire file** as a string (or `nil` for absent
  orders; an error for an unresolved report) — they are opaque text the engine
  produced, not parsed structures.
- `species_stats` returns a **structured table** so Ultron can ingest the figures
  directly; the GM all-species form is an ordered array (ascending species id),
  making output deterministic.

## Sandboxing (a security control, not just hygiene)

- `lua.NewState(lua.Options{SkipOpenLibs: true})`, then open only **base, string,
  table, math**.
- Strip the rest: `os`, `io`, `package`/`require`, `dofile`, `loadfile`, `load`,
  `loadstring`, `debug`; and `math.random` / `math.randomseed`.
- The host's own `os.Chdir` / `os.ReadFile` are Go-side and never exposed as Lua
  functions — the script reaches game data *only* through the scoped verbs.
- This realises both the determinism invariant (no `os.time`/`os.clock`/
  `math.random`/`io`) and security layer 1 (no raw filesystem). No engine-PRNG
  host binding is exposed — reads don't need randomness.

## Determinism, byte-neutrality & security (how we test this slice)

- **Scope enforced.** A `--species 8` script that calls `fh.report(1, 3)` is
  denied; under `--gm`, `fh.species_stats(turn)` returns all species. (Plus: `io`,
  `os`, `math.random` are `nil` in-script; no Lua verb mutates scope.)
- **RNG-neutral, provable.** After a query script over a golden turn dir, assert
  `prngSeed == 0`.
- **Stats parity.** `SpeciesStats` is recomputed structurally; a test asserts the
  re-rendered per-species row matches the frozen `statsCommand` output, so the two
  cannot silently diverge.
- **Golden-neutral.** `make test-golden` unaffected: `script` is invoked only
  explicitly and writes nothing into the golden trees.
- **Cross-engine (`fh` vs `fhc`)** is not exercised yet (`fh` has no `script`);
  when it lands, both engines implement the same `Game` and the §0 invariant test
  applies.

## Follow-ups (not this slice)

- `fh script` mirroring `fhc script` (its own `Game` over the SQLite store).
- Richer entity reads (species tech/economy detail, colonies/ships inventory,
  systems/topology) — with player fog-of-war on the map.
- GM-authoring / mutation verbs and the seeded-PRNG host binding (#36 epic).
- The earlier read-only proposals #42–#45 (`g:turn(id)`, species/system entity
  queries, fog-of-war) are superseded by this query-verb shape and folded into
  the follow-ups above.
