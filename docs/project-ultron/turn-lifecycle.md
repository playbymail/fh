# Project Ultron — the turn lifecycle (genesis, freeze-forward, run)

*Design note. Companion to [`fhc-script-data-layout.md`](fhc-script-data-layout.md)
(this doc resolves its open questions #4 turn-id↔`galaxy.dat` and #5 snapshot-vs-mutate)
and [`fhc-script-design.md`](fhc-script-design.md) ("Data-root layout", "Load
mechanism"). No engine code is changed by this document — it describes a
host-side convention layered on the frozen flat-cwd engine.*

## TL;DR

A game lives under one `--data-root` as a series of **integer-named turn
folders** (`0/`, `1/`, `2/`, …). Each folder is a flat engine working directory
(`galaxy.dat`, `stars.dat`, `planets.dat`, `spNN.dat`, plus that turn's
`spNN.ord`/`spNN.rpt` and per-species subdirs). Three host-side scripts drive
the whole game, and every turn moves through the same two-state lifecycle:

| Verb                                              | What it does                                                             | Folder it touches          |
|---------------------------------------------------|--------------------------------------------------------------------------|----------------------------|
| **genesis** (`tools/initialize-ultron-folder.sh`) | create `0/`, generate the galaxy + species                               | makes `0/`                 |
| **freeze-and-forward**                            | freeze turn `N`, copy it to a new `N+1/`                                 | freezes `N`, makes `N+1`   |
| **run-this-turn**                                 | run `fhc` to completion in the active folder, advancing its `galaxy.dat` | resolves the active folder |

The load-bearing rule: **`galaxy.dat`'s `turn_number` is the only semantic
truth; the folder name is an opaque address.** A folder is *resolved* when its
`turn_number` equals its own name and *pending* (orders open, not yet run) when
it is exactly one behind.

## Two numbers, one of them authoritative

Every turn folder carries two numbers that usually — but not always — agree:

- **The folder name `N`** — Ultron's address for "the turn whose orders live
  here." A stable, sortable key. It is *not* a semantic value.
- **`galaxy.dat`'s `turn_number`** — the engine's count of turns *resolved so
  far*. This is the number printed into reports (`spNN.rpt.t<turn>`), so it is
  what `fh` and `fhc` must agree on for byte-parity.

> **Law: never derive a turn's meaning from its folder name. Always read
> `turn_number` from `galaxy.dat`.** The folder name is filing; the `.dat` is
> truth. Because both engines read the same `.dat`, they agree by construction —
> *as long as* the folder name never leaks into any report-affecting logic.
> (Report filenames already take their `t<turn>` suffix from `galaxy.dat`, not
> from a directory name — see the data-layout doc.)

## The turn state machine

Grounded in the engine, not folklore:

- A fresh `create galaxy` writes `turn_number = 0` (`internal/game/galaxy.go`;
  `fhc show turn_number` on a new setup returns `0`).
- Exactly one pipeline step advances it: `finish` does `galaxy.turn_number++`
  (`internal/game/finish.go`). One full turn run = `+1`.

So for any folder `N`:

```
PENDING   turn_number == N-1   orders open, pipeline not yet run
RESOLVED  turn_number == N      pipeline has run; reports written
```

The predicate is uniform for the whole game:

> **`resolved(N)  ⟺  turn_number(N) == N`**

Turn `0` is the *only* folder born resolved — it is created by `create`, not by
resolving orders, so `turn_number(0) == 0` from birth. Every other folder is
born **pending** (a copy of its predecessor, carrying that predecessor's
`turn_number`) and becomes resolved when run. This is why the "setup vs first
turn" seam is not a special case: it is just the first instance of the general
cadence.

## The three lifecycle verbs

### 1. genesis — `tools/initialize-ultron-folder.sh <data-root> [seed] [species]`

The `data-root` is **not** created (the Ultron folder must already exist); the
script makes `0/` inside it and runs `create galaxy` →
`create home-system-templates` → `create species`, leaving `0/galaxy.dat` at
`turn_number = 0`. It refuses to overwrite an existing `0/galaxy.dat`.

`0/` is the genesis. The GM is free to **tweak the setup** (edit `species.cfg`
and regenerate, hand-adjust starting state) for as long as `0/` is the active
folder — i.e. until it is frozen. Immutability is a consequence of *freezing*,
not of being turn 0.

- **Pre:** `data-root` exists; no `0/galaxy.dat`.
- **Post:** `0/` resolved (`turn_number == 0`), active, mutable.

### 2. freeze-and-forward — `tools/freeze-and-forward.sh <data-root>`

Takes the current **resolved, active** folder `N`, marks it **frozen**
(read-only: Ultron may query it but may not stage orders into it), and creates
`N+1/` as a byte copy of `N/`. The copy carries `N`'s `turn_number`, so `N+1/`
is born **pending** (`turn_number == N == (N+1)-1`).

The genesis→turn-1 case is unremarkable under this rule: `0/galaxy.dat` sits at
`turn_number 0`, the folder is `0`, so freeze-and-forward freezes `0/` and
copies it into a new `1/` — which now reads `turn_number 0` in a folder named
`1`: pending, exactly as intended.

Player / Ultron-agent **orders go into the new active folder's species subdirs**
(`N+1/<species>/`), never into a frozen folder.

- **Pre:** `N` is the active folder and is resolved (`turn_number == N`).
- **Post:** `N` frozen (query-only); `N+1` pending, active, accepts orders.

### 3. run-this-turn — `tools/run-this-turn.sh <data-root>`

Runs `fhc` to completion in the active **pending** folder `N`, consuming the
orders staged in `N/<species>/` and advancing `N/galaxy.dat` from `N-1` to `N`.
It first rebuilds the flat order namespace: it drops any `sp*.ord` carried
forward by freeze-and-forward (so stale orders can't be mistaken for this turn's),
copies each `N/<species>/orders` (species id is a bare integer, `1..MAX_SPECIES`)
to the flat `sp<NN>.ord`, and then `create orders` fills defaults for every
species that did not submit (using the bundled `tools/noorders.txt` template,
matching `testdata/cref/generate.sh`). With no staging slots yet, that is a full
default-order turn — the start-of-game case. The sequence is the
canonical pipeline — `locations`, `create orders`, `combat`, `pre-departure`,
`jump`, `production`, `post-arrival`, `finish`, then `report` (and `stats` /
`turn` for summaries). **`report` — not `finish` — is what writes the
`spNN.rpt.t<turn>` files**, so a run that stops at `finish` advances the number
but produces no reports; run-this-turn must include `report`.

- **Pre:** `N` is the active folder and is pending (`turn_number == N-1`).
- **Post:** `N` resolved (`turn_number == N`); reports present in `N/`.

A natural idempotence guard mirrors genesis's refuse-to-overwrite: **refuse to
run an already-resolved folder** (`turn_number == N`).

## Worked example

```
genesis data/alpha            ->  0/  turn_number 0   RESOLVED  (active, GM tweaks here)
freeze-and-forward (N=0)      ->  0/ FROZEN; make 1/  turn_number 0   PENDING (orders -> 1/<sp>/)
run-this-turn      (N=1)      ->  1/  turn_number 1   RESOLVED  (reports in 1/)
freeze-and-forward (N=1)      ->  1/ FROZEN; make 2/  turn_number 1   PENDING (orders -> 2/<sp>/)
run-this-turn      (N=2)      ->  2/  turn_number 2   RESOLVED
...
```

The newest folder is always the single **active** one; everything below it is
frozen history.

## Why this is parity-safe

- The engine reads/writes **bare filenames in its cwd** and never sees the
  folder name. A folder produced by *copy `N→N+1` then run in place* is therefore
  byte-identical to one produced by an in-place run in a throwaway dir — which is
  exactly how `generate.sh` builds the `fhc` reference `turn1/`, `turn2/`, …. So
  a **resolved** folder `N` matches the `fhc` golden for that turn given the same
  seed and orders.
- Parity comparisons are **resolved-to-resolved**. A pending folder is `fh`/Ultron
  staging with no `fhc` counterpart — don't diff it against anything.
- Because the folder name never enters report bytes (the `t<turn>` suffix comes
  from `galaxy.dat`), the transient folder-leads-`.dat` mismatch is invisible to
  the report surface that defines the cross-engine invariant.

## Why not auto-run a default-order turn instead

An alternative would force `turn_number == folder` everywhere by running a turn
with default orders immediately after setup. Rejected as the default because it:

- **silently disenfranchises the opening move** — turn 1, the highest-leverage
  turn, would resolve on autopilot before any player acts;
- **breaks the uniform lifecycle** — turn 1 would be born resolved while every
  other turn is born pending; and
- **doesn't even shrink the tree** — genesis (`0/`) still exists; you've only
  *added* a phantom resolved turn.

Keep it strictly as an emergency escape hatch for a consumer that genuinely
cannot tolerate the pending mismatch — and even then, prefer teaching that
consumer the lifecycle over mutating the game.

## Enforcement: a single source of process truth

The lifecycle rules — what is frozen, which transitions are legal from which
state — are enforced **procedurally, by the commands that control game state**,
not by any on-disk artifact. There is no `.frozen` sentinel and no permission
trick: *"frozen" simply means "not the active turn,"* and the GM control
commands refuse to mutate anything but the active turn, gating every operation
on its lifecycle precondition:

- **stage orders** → requires the active folder, **pending**.
- **run-this-turn** → requires the active folder, **pending** (refuses an
  already-resolved folder).
- **freeze-and-forward** → requires the active folder, **resolved**.

The active turn is the highest-numbered folder; its pending/resolved state is
read from `galaxy.dat` (`turn_number == N`?). Any other (frozen) folder rejects
every mutating operation — it is query-only. Because the rules live in **one
command layer**, there is a single source of process truth, with no second copy
to drift out of sync.

These rules are being baked into the **shell scripts first**, to validate the
process end-to-end, and the *same* rules will then be wired into the **scripting
engine's GM control commands** — the shell scripts and the eventual engine
commands are two embodiments of the one rule set. The read-only script host
(#41/#42) is unaffected: it never mutates, so it can query frozen turns freely.

### One shared predicate (shell phase)

The shell embodiment factors the predicate into a single sourced helper,
[`tools/lib/ultron-lifecycle.sh`](../../tools/lib/ultron-lifecycle.sh), which
every lifecycle script sources — no script re-derives the rule:

- `ultron_turn_state <root> <N>` → `absent` | `pending` | `resolved` |
  `anomalous` (the predicate above; reads `galaxy.dat`'s `turn_number`).
- `ultron_active_turn <root>` → the highest-numbered (active) turn folder.
- `ultron_species_dirs <root> <N>` → the integer-named species staging subdirs
  of turn `N` (the order-staging slots run-this-turn reads).
- `ultron_require_state <root> <N> <want>` → the transition guard the verbs are
  built from (genesis ⇒ `absent`; run-this-turn ⇒ `pending`; freeze-and-forward
  ⇒ `resolved`).

`initialize-ultron-folder.sh` already routes its refuse-to-overwrite check
through `ultron_turn_state … 0 == absent`, so genesis is a real consumer of the
shared predicate rather than a parallel re-implementation.

### Forward-looking — one interface, many engines

The lasting home for these operations is a **Go interface the scripting engine
defines** — query state, stage orders, run a turn, freeze-and-forward, lifecycle
state (`DoFoo() error`-style methods). **`fhc` implements it now; `fh` implements
the *same* interface later.** The scripting engine codes against the interface,
so adding `fh` scripting changes **no scripting-engine code** — it only adds an
implementation. The shell helper is the validation-phase stand-in for that
interface: prove the process here, then move the rules behind the interface
unchanged, keeping the single source of truth as the engine surface grows.

## Implications for the scan and `g:turn(id)` (#41 / #42)

- **Turn 0 is correctly invisible to the scan.** `fh.load{}`'s positive-integer
  filter rejects `0`, and genesis is not an Ultron-addressable *turn* — it's the
  pre-game anchor. The active game surfaces from folder `1` upward. Do **not**
  make `0` scannable.
- **`g:turn(N).turn` reports the `.dat` `turn_number`** (authoritative,
  parity-safe), and the folder id / a `resolved` boolean (`folder == turn_number`)
  are exposed *separately*. A pending turn's `.turn` lagging its folder by one is
  correct, not a bug.
- **Pending turns must be loadable.** An Ultron agent decides turn-`N` orders by
  reading the start-of-turn-`N` state, which *is* the pending folder `N`. Listing
  and loading pending folders is the feature.

## Settled

- **Enforcement** — procedural, in the GM control commands (a single source of
  process truth), not a filesystem marker. Embodied in the shell scripts first
  to validate the process, then wired into the scripting engine's GM commands
  (see above).
- **Tooling path** — the lifecycle verbs are shell scripts now (like genesis);
  the same rules later move into the scripting engine as GM control commands.
  The shell scripts are the validation step *before* that wiring. All three verbs
  exist: `initialize-ultron-folder.sh`, `freeze-and-forward.sh`,
  `run-this-turn.sh`.
- **`<species-id>` format** — a bare integer, `1..MAX_SPECIES`, mapping to the
  engine's `sp%02d.{ord,rpt}`. Staging slots are `<turn>/<species>/` and the
  staged order file is `<turn>/<species>/orders`.
- **`noorders.txt` source** — bundled with the tool (`tools/noorders.txt`),
  copied into the turn dir by run-this-turn so `create orders` can default-fill.

## Deferred decisions

1. **Exact GM command surface** — the names/shape of the control commands once
   they move from shell scripts into the scripting engine.
