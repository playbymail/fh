# Project Ultron — data layout & the engine adapter

*Design note. Companion to [`ultron-harness-architecture.md`](ultron-harness-architecture.md)
(see §0 the cross-engine invariant, §2.1 scenario authoring, §2.3 order
submission). No engine code is changed by this document.*

## TL;DR

Ultron uses a **structured, Ultron-optimized directory layout**. The engines
(`fhc`, `fh`) keep their **frozen flat-working-directory** file scheme. A thin
**adapter** (Go package or shell script) bridges the two: it materializes a flat
engine working directory from the Ultron layout before a run, and harvests the
engine's outputs back into the layout after. The engines never learn the Ultron
layout, so byte-parity is preserved.

## Ground truth — how `fhc` lays out files today

`fhc` reads and writes **flat, bare filenames in the current working directory**,
keyed by species **number**. The cwd *is* the game directory. There is no
game-id, no turn-id, no per-species subdirectory, and no configurable data dir.

| File | Purpose | Written by |
|------|---------|-----------|
| `galaxy.dat`, `stars.dat`, `planets.dat` | shared galaxy state | setup / pipeline |
| `sp%02d.dat` | per-species state | setup / pipeline |
| `sp%02d.ord` | per-species orders (input) | player / harness |
| `sp%02d.rpt.t%d` | per-species turn report (output) | `report` phase |
| `sp%02d.log` | per-species log (prepended to reports) | pipeline |

The turn number embedded in `sp%02d.rpt.t<turn>` comes from `galaxy.dat`, **not**
from any directory name. The existing test harness achieves "per turn" purely by
*copying* state into stage dirs (`testdata/cref/turn1/`, `turn2/`, …) and running
`fhc` with cwd set there (`t.Chdir` in tests, `cd` in
`testdata/cref/generate.sh`). `fh` is expected to follow the same flat-file
contract for report parity.

**This flat-cwd, species-numbered scheme is frozen** — changing it changes engine
behavior and breaks byte-parity (forbidden by `CLAUDE.md` and the §0 invariant).

## The Ultron layout (proposed)

```
data/<game-id>/<turn-id>/                     # galaxy/stars/planets/spNN.dat
data/<game-id>/<turn-id>/<species-id>/        # that species' orders + reports
```

- `<game-id>` — names a campaign/run.
- `<turn-id>` — an Ultron label for a snapshot; the engine still derives the real
  turn number from `galaxy.dat`.
- `<species-id>` — groups a species' orders/reports. Format TBD (bare number,
  `sp01`-style, or name); it must round-trip cleanly to the engine's
  `spNN.{ord,rpt}` filenames.

This layout is *ours to optimize*. It can carry whatever Ultron finds convenient
(per-species coverage files, brain state, scenario manifests) without constraint
from the engine's file scheme.

## The adapter — where the divergence is swept under the rug

A small **adapter** (Go package preferred; a shell script is acceptable for a
first cut) is the only thing that knows both layouts:

- **Stage in (Ultron → engine):** create a flat working dir for a
  `(game-id, turn-id)` and populate it — copy/symlink the `.dat` files, and copy
  each `<species-id>/orders` to the flat `spNN.ord` the engine expects.
- **Run:** invoke the unmodified engine pipeline with cwd = that flat dir
  (`combat → pre-departure → jump → production → post-arrival → finish → report`).
- **Harvest out (engine → Ultron):** copy `spNN.rpt.t<turn>` back to
  `<species-id>/report`, and snapshot the mutated `.dat`/`coverage.tsv` into the
  next `<turn-id>` dir.

Because **both** `fh` and `fhc` are fed identical flat working dirs by the
**same** adapter, the adapter actively *reinforces* the §0 invariant: the two
engines start every scenario from byte-identical inputs, so any report divergence
is unambiguously an `fh` defect.

## What this means for `fhc script` (the read-only query slice)

The implemented scripting slice is **read-only**: query the current turn, a
turn's status, a turn's orders/report, and a species' statistics (see
[`fhc-script-design.md`](fhc-script-design.md)). It barely needs the adapter. The
`.dat` files already sit flat inside `<data-root>/<turn-id>/`, so the `fhc` `Game`
implementation reads them directly — and reads `sp%02d.ord` / the
`<species-id>/orders` slot for orders and `sp%02d.rpt.t<turn>` for reports.

The directory-access question is **resolved**: the `fhc` `Game` `chdir`s into the
turn dir and reuses the unmodified loaders for the one query that needs full
state (`SpeciesStats`); the cheaper queries read `galaxy.dat` and the order/report
files directly. Either way the flat-cwd commands stay byte-neutral. The full
adapter (orders staging, report harvesting) is only needed once scripting drives
*runs*, not reads.

## Open questions for the planning session

1. **Adapter boundary & language.** Go package vs. shell script; whether the
   `script` host calls the adapter or it is a separate orchestration step.
2. **`script` host directory access.** *Resolved:* the `fhc` `Game` `chdir`s into
   the turn dir and reuses the unmodified loaders (flat-cwd commands unchanged);
   the lighter queries read `galaxy.dat`/orders/reports directly.
3. **`<species-id>` format.** Bare number / `sp01` / name — must round-trip to
   `spNN.{ord,rpt}`.
4. **`<turn-id>` ↔ `galaxy.dat` turn number.** How the Ultron label maps to the
   engine's authoritative turn number (and how report filenames `…rpt.t<turn>`
   land back under the right `<turn-id>`).
5. **Snapshot vs. mutate.** Whether each turn dir is an immutable snapshot the
   adapter copies forward, matching the repo's existing kill-and-fill model.
