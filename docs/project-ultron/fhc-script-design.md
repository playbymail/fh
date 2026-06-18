# `fhc script` — read-only first slice (design)

*Design note / draft. Companion to
[`ultron-harness-architecture.md`](ultron-harness-architecture.md) (§0 invariant,
§2.2 state oracle) and [`fhc-script-data-layout.md`](fhc-script-data-layout.md)
(Ultron layout + adapter). Tracked by epic #36. **No engine code is changed by
this document.** This describes the first, read-only scripting slice only —
open a game, list turns/species, select a turn, query species/systems. Authoring
/ mutation verbs are a later slice.*

## TL;DR

Add a `script` subcommand to `fhc` that embeds GopherLua (decided, #36) and
exposes a small, **read-only**, **access-controlled** Lua API. The API speaks
**Ultron's language** — turns and species addressed by bare integer id — and
hides the engine's frozen flat-cwd / `sp%02d.dat` file scheme behind the host.
The host runs in one of two scopes set at the command line: **GM** (all turns,
all species) or **player** (all turns, only that species). The script may be an
adversarial agent that tries to read another player's data; the engine **enforces
the scope** so it cannot. The slice is **byte-neutral** on the existing golden
trees and **cannot perturb the RNG stream** (reads never call `rnd()`).

## Guiding principle — address a game the way Ultron thinks about it

Ultron reasons in stable identifiers: *"turn 3, species 8, give me that data."*
The Lua surface honours that directly. The engine's on-disk reality — bare
filenames in cwd, species keyed by number, turn number embedded in `galaxy.dat`
— is an **implementation detail the host resolves**, not something the script
author sees. Today the host resolves a turn by computing a directory under the
data root and `chdir`-ing into it before calling the unmodified loaders;
tomorrow that resolution could be a virtual filesystem mapped in. Either way the
script just selects a turn and the bytes appear. **`chdir` is a mechanism, not
the API.**

## Security model — Ultron agents may try to cheat

An Ultron player-agent is **untrusted**: its decision logic may attempt to read
another species' state to gain an unfair view. The design denies that with three
independent layers, all enforced by the host, none defeatable from inside the
script:

1. **Sandbox — no raw filesystem.** The Lua environment has no `io`, `os`,
   `require`/`package`, `load`/`loadfile`/`dofile`, or `debug`. A script cannot
   open `sp03.dat` by hand; its *only* path to game data is the scoped query API.
2. **Scoped query API — species reads are checked.** Every species-addressed
   query validates the target against the caller's scope. A `--species 8` engine
   serves species 8 and denies any other; `g:species()` (the list form) returns
   only the species the caller may see.
3. **Scope is set by the CLI, immutable in-script.** Whether the engine is GM or
   player-scoped is fixed by the `--gm` / `--species` flag at invocation. There
   is no Lua verb to change scope, so an adversarial script cannot widen it. The
   **trust boundary is the CLI invoker**: the Ultron harness chooses
   `--species N` for a player-agent and **never** passes `--gm`. `--gm` exists
   only for the human GM / trusted tooling.

| Scope  | Selected by      | Turns visible | Species data visible |
|--------|------------------|---------------|----------------------|
| GM     | `--gm`           | all           | all                  |
| Player | `--species <id>` | all           | only `<id>`          |

Galaxy-level metadata (turn number, radius, species *count*) lives in
`galaxy.dat` and is shared, so it is visible in both scopes. Per-species data (a
species' tech, economy, colonies, ships) is scope-restricted. The **map is
fog-of-war'd** too: a player-scoped engine filters `t:systems()` / `t:system()`
to systems that species has scanned — `star_data_t.visited_by` must include the
caller's species. Unscanned systems are omitted from the list and denied by
coordinate lookup, so an agent cannot read unexplored galaxy. (First cut is
binary visibility: a scanned system returns its full `marshalSystem` record; an
unscanned one is invisible. Modelling *partial* knowledge — knowing a system
exists without full planet detail — is a possible later refinement.)

## What the engine code already gives us (ground truth)

| Fact                                                                             | Location                                                                         | Consequence for the design                                                                                                                                     |
|----------------------------------------------------------------------------------|----------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Dispatch is a linear `if/else if` chain                                          | `internal/game/fh.go:31`                                                         | A `script` verb is one more branch → `scriptCommand(args)`, returns 0/2 like every command.                                                                    |
| Loaders read **bare filenames in cwd**, no path arg                              | `galaxyio.go:61`, `stario.go`, `planetio.go`, `speciesio.go:142`                 | Reuse them unmodified by `chdir`-ing into the resolved turn dir. **Parameterizing them would edit frozen loaders → forbidden.**                                |
| No loader touches the PRNG                                                       | (all `get_*_data`)                                                               | Loading is RNG-neutral.                                                                                                                                        |
| `prngSeed` is lazily set only on first `rnd()`                                   | `prng.go:14`                                                                     | A read-only script never calls `rnd()`, so `prngSeed` stays `0` — a crisp determinism assertion.                                                               |
| `export` is the exact analog: load four files, serialize                         | `export.go:255` (`exportToJson`)                                                 | Our read path is the same shape, emitting Lua tables instead of cJSON.                                                                                         |
| `marshal*` already define "what a query returns"                                 | `marshalGalaxy`/`marshalSystem`/`marshalSpecies`/`marshalPlanet` in `marshal.go` | Lua return tables mirror those field sets so the script view and `fh export json` never drift.                                                                 |
| State is package-level globals; `ResetState()` zeroes them and sets `prngSeed=0` | `vars.go:137`                                                                    | Selecting a turn calls `ResetState()` then loads, so turn switches are clean. Only one turn is loaded at a time (global state).                                |
| Turn number comes from `galaxy.dat`, not a dir name                              | `galaxyio.go:73` (`galaxy.turn_number`)                                          | Selecting a turn = resolve to the right turn dir; the authoritative number is read back out. The engine never enumerates turns — the host scans the data root. |
| Species addressed by 1-based number → `sp%02d.dat`                               | `speciesio.go:142`                                                               | Ultron's `species=8` maps directly to `sp08.dat` / `spec_data[7]`; species count is fixed at galaxy creation, so the roster is stable across turns.            |

## Data-root layout (the host's, not the engine's)

```
<data-root>/                     # the single game; --data-root points here
  <turn-id>/                     # bare integer, e.g. 3
    galaxy.dat stars.dat planets.dat sp01.dat … spNN.dat   # the engine's flat store
    <species-id>/                # bare integer, e.g. 8
      …orders, reports           # Ultron writes orders here directly (not this slice)
```

The data root never holds `.dat` files directly — only turn folders. `fh.load{}`
**scans** it once (turn folders → turn list; species subfolders → species roster)
and caches the result; the actual `.dat` store is loaded only when a turn is
selected. Game-id is dropped: the engine is always limited to the single game the
data root points at.

## Decisions (settled with the maintainer)

- **Security model: GM vs. player scope, enforced by the host** (above). `--gm`
  and `--species` mutually exclusive, exactly one required.
- **`--data-root` is required** and names the dir of turn folders.
- **Host lives inside package `game`.** New files (`script.go`, `script_api.go`)
  in `internal/game`, reaching the unexported loaders/globals/structs directly —
  no new public surface on the frozen package. (`fh`'s later host lives in its own
  idiomatic package and must satisfy the cross-engine invariant against `fhc`.)
- **Game-handle API**: `fh.load{}` returns a handle; `g:turn(id)` → turn handle;
  `t:species()`, `t:systems()`.
- **`fh.load{}` scans + scopes**; turn `.dat` is loaded lazily on `g:turn(id)`.
  These are distinct units of work (scan/scope vs. load).
- **Species query is top-level + tech only this slice.** Namplas (colonies) and
  ships are a follow-up.
- **Bare integer keys** for turn and species ids.
- **Write-up lives here** (`docs/project-ultron/fhc-script-design.md`).

## CLI shape

```
fhc script --data-root <dir> ( --gm | --species <id> ) <file.lua>
```

- `--data-root <dir>` — required; the dir of turn folders.
- `--gm` | `--species <id>` — exactly one required, mutually exclusive. Sets the
  immutable scope.
- `<file.lua>` — read relative to the **original** cwd, *before* any turn `chdir`,
  so the script path is unaffected by turn selection.
- Slots into `fh.go` as `else if arg == "script" { runCommand(scriptCommand(args)) }`.
  Bad combinations (`--gm` with `--species`, neither, missing `--data-root`,
  unknown species) error with exit 2 before any Lua runs.

## Lua API (read-only first slice)

```lua
-- Open the single game in --data-root. Scans + applies scope; loads no .dat yet.
local g, turns, species = fh.load{}
-- turns:   { 1, 2, 3 }            all turns, in both scopes
-- species: { 8 }                  player scope; { 1, 2, … N } under --gm
-- (also available as g.turns / g.species)

-- Select a turn → loads <data-root>/<id>/*.dat (ResetState + chdir + loaders).
local t = g:turn(3)
print(t.turn, t.radius, t.num_species)   -- galaxy-level metadata (shared)

-- Species, addressed by id. SCOPE-ENFORCED.
local sp = t:species(8)                   -- ok in --species 8 or --gm
-- t:species(3)  -> error/denied in --species 8 scope
for _, sp in ipairs(t:species()) do       -- only species the caller may see
  print(sp.number, sp.name, sp.government.type, sp.tech.MI, sp.econ_units)
end

-- Systems and their planets.
for _, sys in ipairs(t:systems()) do       -- ordered by star_base index
  print(sys.x, sys.y, sys.z, sys.num_planets)
end
local sys = t:system(x, y, z)
for _, pl in ipairs(sys.planets) do        -- ordered by orbit
  print(pl.orbit, pl.temperature_class, pl.gravity)
end
```

Return-shape rules:

- Tables **mirror the `marshal*` field sets** for the corresponding entity so the
  Lua view and `fh export json` stay in lockstep.
- Collections are returned as **ordered array tables** iterated with `ipairs`:
  turns ascending, species by number, systems by `star_base` index, planets by
  orbit. Returning explicit sequences makes ordering a guarantee, so scripted
  output is deterministic.
- A turn handle reflects the **currently loaded** turn (global state); selecting a
  new turn `ResetState`s and reloads, so interleaving two turn handles is not
  supported in this slice (documented constraint).

## Load mechanism (host-side, hidden from the script)

`fh.load{}`:

1. Scan `--data-root`: integer-named turn dirs → turn list; species subdirs →
   species roster.
2. Apply scope: `--gm` → all species; `--species N` → roster filtered to `{N}`
   (error if `N` absent). Turn list is unrestricted in both.
3. Return a handle carrying the immutable scope, plus the turn and species lists.

`g:turn(id)`:

1. `ResetState()` — clean globals (and `prngSeed=0`).
2. Resolve `dir = <data-root>/<id>/`; `os.Chdir(dir)` (host Go; never exposed to
   Lua).
3. `get_galaxy_data()`, `get_star_data()`, `get_planet_data()`,
   `get_species_data()` — the unmodified loaders.
4. Return a turn handle exposing `turn`/`radius`/`num_species` and the scoped
   query methods.

## Sandboxing (a security control, not just hygiene)

- `lua.NewState(lua.Options{SkipOpenLibs: true})`, then open only **base, string,
  table, math**.
- Strip the rest: `os`, `io`, `package`/`require`, `dofile`, `loadfile`, `load`,
  `loadstring`, `debug`; and `math.random` / `math.randomseed`.
- The host's own `os.Chdir` / `os.ReadFile` are Go-side and never exposed as Lua
  functions — the script reaches game data *only* through the scoped query API.
- This realises both the determinism invariant (no `os.time`/`os.clock`/
  `math.random`/`io`) and security layer 1 (no raw filesystem). No engine-PRNG
  host binding is exposed yet — reads don't need randomness.

## Determinism, byte-neutrality & security (how we test this slice)

- **Scope enforced.** A `--species 8` script that calls `t:species(3)` is denied;
  `t:species()` returns only species 8; under `--gm` it returns all. (Plus: `io`,
  `os`, `math.random` are `nil` in-script; no Lua verb mutates scope.)
- **RNG-neutral, provable.** After a load+query script over a golden turn dir,
  assert `prngSeed == 0`.
- **Golden-neutral.** `make test-golden` unaffected: `script` is invoked only
  explicitly and writes nothing into the golden trees.
- **Reproducible output.** Repeated runs of the same script over the same turn dir
  produce byte-identical stdout; the ordered-sequence rule guarantees it.
- **Cross-engine (`fh` vs `fhc`)** is not exercised yet (`fh` has no `script`);
  when it lands, the §0 invariant test applies.

## Proposed issues (under epic #36, label `harness`) — for approval

Right-sized to the code; each references #36.

1. **script: CLI subcommand skeleton + GopherLua wiring + scope flags.** Add the
   `script` dispatch branch in `fh.go`; `scriptCommand(args)` in `internal/game`;
   pin GopherLua; parse/validate `--data-root` (required) and `--gm` |
   `--species <id>` (exactly one, mutually exclusive); run a trivial
   `print("hello")` script. Byte-neutral.
2. **script: sandbox the stdlib (security layer 1).** `SkipOpenLibs` + open only
   base/string/table/math; drop `os`/`io`/`package`/`require`/`dofile`/`loadfile`/
   `load`/`debug` and `math.random`/`math.randomseed`. Test they are `nil`.
3. **script: `fh.load{}` — scan data-root, build scoped handle.** Scan turn dirs +
   species subdirs, cache the lists, apply gm/player scope, return
   `handle, turns, species`. No `.dat` loaded yet.
4. **script: `g:turn(id)` — select + load a turn.** Resolve `<data-root>/<id>/`,
   `ResetState` + `chdir` + the four loaders; expose `turn`/`radius`/
   `num_species`.
5. **script: query species entity (scope-enforced).** `t:species([id])`,
   top-level + tech fields mirroring the species JSON export; deny out-of-scope
   ids; list form returns only visible species; ordered; id→`sp%02d` mapping.
6. **script: query system/star entity (+ planets), fog-of-war'd.** `t:systems()` /
   `t:system(x,y,z)` mirroring the systems JSON export; ordered; nested planets by
   orbit. Player scope filters to systems whose `star_data_t.visited_by` includes
   the caller's species (unscanned omitted from the list, denied by coordinate
   lookup); GM scope sees all.
7. **script: security + determinism test for the read-only slice.** Scope
   enforcement (cross-species denial, list filtering, sandbox `nil`s, no
   scope-mutation verb); `prngSeed == 0` after load+query; `make test-golden`
   unaffected; repeated-run output identical.

Dependency order: 1 → 2 → 3 → 4 → {5, 6} → 7. Follow-ups (not this slice):
species namplas/ships query; topology/reachability reads (§2.2); orders staging
+ report harvesting (the full adapter); `fh script` mirroring `fhc script`.

## Resolved with the maintainer

- **Player fog-of-war on systems/planets: yes.** `t:systems()` / `t:system()` are
   filtered by `star_data_t.visited_by` for player scope (binary visibility,
   first cut); GM sees all. Baked into #6 above.

## Open questions (minor — can settle while implementing)

1. **GM-only exposure.** Anything GM scope should expose that a player must not,
   beyond all-species reads / the full map, in this read-only slice? (Assumed no —
   same query verbs, wider roster + unfiltered map.)
2. **Partial map knowledge.** Whether to later model "knows a system exists but
   not its planets" vs. the first-cut binary visited/invisible. Follow-up, not
   this slice.
