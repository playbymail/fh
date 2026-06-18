# Far Horizons scripting engine — verb reference

Reference for an agent (Ultron or the GM) that queries Far Horizons game state
through the embedded Lua scripting engine. It describes every verb, its
arguments, its return value, and the conditions under which it fails.

The engine is **read-only**: there is no verb that changes game state. A script
reads state and writes its findings to standard output with `print`.

## How a script is run

A script is a Lua file executed by an engine's `script` subcommand:

```
fhc script --data-root=<dir> ( --gm | --species=<id> ) <file.lua>
fh  script --data-root=<dir> ( --gm | --species=<id> ) <file.lua>
```

- `--data-root=<dir>` — the directory holding the game's turn folders.
- `--gm` or `--species=<id>` — exactly one, mutually exclusive; sets the scope
  (see [Scope](#scope)). Use the `--opt=value` form; space-separated values are
  rejected.
- `<file.lua>` — the script to run.

The script's standard output is the script's result. A Lua error (including one
raised by a verb) aborts the script and the command exits non-zero; see
[Errors](#errors).

## Model

A game is a sequence of **turns** numbered `1, 2, 3, …` (turn 0 is the pre-game
setup and is not addressable). Each turn has a status:

- `"pending"` — orders are open; the turn has not been resolved.
- `"resolved"` — the turn has been run; reports exist.

The **current turn** is the highest-numbered turn. Each turn holds per-species
data: the orders submitted for that turn, the report produced when it resolved,
and the species' statistics.

A **species** is identified by a positive integer id (`1`, `2`, …).

## Scope

Scope is fixed at invocation and cannot be changed from within the script.

| Scope | Flag | Species the script may read |
|---|---|---|
| GM | `--gm` | every species |
| Player | `--species=<id>` | only species `<id>` |

Galaxy-level verbs (`fh.current_turn`, `fh.turn_status`) are unaffected by scope.
Per-species verbs (`fh.orders`, `fh.report`, `fh.species_stats`) take an optional
`species` argument that is resolved against the scope:

- **Player scope.** The species is fixed to the caller's own id. Omit the
  argument (it is implied) or pass the caller's own id; passing any other id
  fails.
- **GM scope.** There is no implied species, so `fh.orders` and `fh.report`
  **require** a `species` argument. `fh.species_stats` may omit it to retrieve
  every species at once.

## Verbs

All verbs live in the global `fh` table. In the signatures below, `[, species]`
marks an optional argument.

---

### `fh.current_turn()`

Returns the current (highest-numbered) turn.

- **Arguments:** none.
- **Returns:** `number` — the current turn number (≥ 1).
- **Fails when:** the game has no turns yet.

```lua
local n = fh.current_turn()   --> e.g. 3
```

---

### `fh.turn_status(turn)`

Returns whether a turn has been resolved.

- **Arguments:** `turn` (`number`) — the turn to query.
- **Returns:** `string` — `"resolved"` or `"pending"`.
- **Fails when:** `turn` does not exist.

```lua
local st = fh.turn_status(3)   --> "pending" or "resolved"
if st == "resolved" then
  -- the report for turn 3 is available
end
```

---

### `fh.orders(turn [, species])`

Returns the entire orders file submitted for a species in a turn.

- **Arguments:**
  - `turn` (`number`) — the turn to query.
  - `species` (`number`, optional) — see [Scope](#scope). Required under `--gm`.
- **Returns:** `string` — the full contents of the orders file, **or** `nil`
  when no orders were submitted for that species and turn.
- **Fails when:** `species` is not visible in the current scope; or (`--gm`) no
  `species` was given.

```lua
-- Player scope: own species implied.
local ord = fh.orders(3)
if ord == nil then
  print("no orders submitted for turn 3")
else
  print(ord)
end

-- GM scope: species required.
local ord8 = fh.orders(3, 8)
```

---

### `fh.report(turn [, species])`

Returns the entire turn report for a species.

- **Arguments:**
  - `turn` (`number`) — the turn to query.
  - `species` (`number`, optional) — see [Scope](#scope). Required under `--gm`.
- **Returns:** `string` — the full contents of the report.
- **Fails when:** `turn` is not `"resolved"`; or no report exists for that
  species; or `species` is not visible in the current scope; or (`--gm`) no
  `species` was given.

Check `fh.turn_status(turn)` first to avoid the unresolved-turn failure.

```lua
if fh.turn_status(3) == "resolved" then
  print(fh.report(3))        -- player scope: own species
end

-- GM scope:
print(fh.report(3, 8))
```

---

### `fh.species_stats(turn [, species])`

Returns a species' statistics for a turn, computed on demand.

- **Arguments:**
  - `turn` (`number`) — the turn to query.
  - `species` (`number`, optional) — see [Scope](#scope).
- **Returns:**
  - With a species (always the case under player scope): a single
    [stats table](#stats-table).
  - Under `--gm` **with `species` omitted**: an array (1-based, `ipairs`-iterable)
    of stats tables, one per species, ordered by ascending species id.
- **Fails when:** `turn` does not exist; `species` is out of range or not present
  in that turn; or `species` is not visible in the current scope.

```lua
-- Single species (player scope, or GM with an id).
local s = fh.species_stats(3)
print(s.name, s.total_production, s.tech.MI)

-- All species (GM scope, id omitted).
for _, s in ipairs(fh.species_stats(3)) do
  print(s.species, s.name, s.econ_units)
end
```

#### Stats table

Each stats table has these fields:

| Field | Type | Meaning |
|---|---|---|
| `species` | number | species id |
| `name` | string | species name |
| `tech` | table | tech levels, keyed `MI`, `MA`, `ML`, `GV`, `LS`, `BI` (each a number) |
| `total_production` | number | total production across all colonies |
| `num_planets` | number | number of populated planets |
| `num_ships` | number | number of completed ships |
| `num_shipyards` | number | number of shipyards |
| `offensive_power` | number | total offensive power |
| `defensive_power` | number | total defensive power |
| `econ_units` | number | banked economic units |

The `tech` keys are the six Far Horizons technologies: Mining (`MI`),
Manufacturing (`MA`), Military (`ML`), Gravitics (`GV`), Life Support (`LS`),
Biology (`BI`).

## Errors

A verb reports a failure by raising a Lua error. An uncaught error aborts the
script and the command exits with a non-zero status, printing the error message
to standard error. To handle a failure within the script, wrap the call in
`pcall`:

```lua
local ok, result = pcall(fh.report, 3, 8)
if ok then
  print(result)
else
  print("could not read report:", result)   -- result is the error message
end
```

Note the distinction:

- `fh.orders` returns `nil` (not an error) when no orders exist — a normal,
  expected absence.
- `fh.report` raises an error when the turn is unresolved or the report is
  missing — querying a report that cannot exist is treated as a fault.

## Sandbox

The script runs in a restricted Lua environment. These standard libraries are
available: `base` (e.g. `print`, `pcall`, `ipairs`, `pairs`, `tostring`,
`tonumber`, `error`, `type`), `string`, `table`, and `math`.

These are **not** available (any reference is `nil`): `os`, `io`, `debug`,
`require`/`package`, `load`, `loadfile`, `loadstring`, `dofile`, and the
randomness functions `math.random` / `math.randomseed`. A script cannot touch the
filesystem or the operating system; its only access to game data is through the
`fh` verbs.

## Determinism

The verbs are read-only and do not draw from the game's random number generator,
so running the same script against the same game state always produces the same
output. The GM all-species form of `fh.species_stats` is ordered by ascending
species id, so its output is stable.
