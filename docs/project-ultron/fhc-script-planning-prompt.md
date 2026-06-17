# `fhc script` — planning-session prompt

*Paste the block below into a fresh session to start the `fhc script` design
work. It is a planning session: produce a design and create GitHub issues — no
engine code. Companion docs:
[`ultron-harness-architecture.md`](ultron-harness-architecture.md),
[`fhc-script-data-layout.md`](fhc-script-data-layout.md).*

---

```
We're starting the design for the `fhc script` subcommand — the embedded Lua
scripting host for the byte-faithful Far Horizons C-port engine. This is a
PLANNING session: produce a design and create GitHub issues. Do NOT write
engine code.

## Context (read these first)
- Repo: /Users/wraith/Software/playbymail/fh (branch main). GitHub: playbymail/fh,
  `gh` is authed. CLAUDE.md governs the repo — read it.
- Two engines: `fhc` (byte-faithful C port in internal/game, run by cmd/fhc) is
  FEATURE-COMPLETE and is the trusted oracle. `fh` (idiomatic SQLite port) is a
  work in progress. We implement scripting on `fhc` FIRST.
- Decision already made: the scripting engine is GopherLua (pure-Go Lua 5.1).
  It ships as a `script` subcommand on each engine (`fhc script` / `fh script`),
  not a separate binary. Tracked by epic #36 (read it: `gh issue view 36`).
- Design docs live in docs/project-ultron/. Read them, especially:
    * ultron-harness-architecture.md — §0 (cross-engine determinism INVARIANT),
      §2.1 (scenario authoring), §2.2 (state oracle / queries), §2.3 (orders).
    * fhc-script-data-layout.md — the Ultron directory layout + engine adapter,
      and what the first read-only scripting slice actually needs.
  Scripting is a prerequisite for Project Ultron.

## Hard constraints
- No code. Plans, design notes, and GitHub issues only.
- Never change existing `fhc` behavior. A new `script` subcommand must be
  byte-neutral on the existing golden trees (`make test-golden`). Query/read
  operations must NOT perturb the RNG stream.
- internal/game keeps the C identifiers verbatim and uses package-level globals
  cleared by ResetState() — understand how that interacts with loading state.
- Determinism invariant: scripting must be reproducible and (eventually) produce
  byte-identical state across fhc and fh from the same seed. For this read-only
  first slice that mostly means: don't expose os/io/math.random/time; any RNG
  comes from the engine's seeded PRNG.

## Directory / file layout — already decided, mind the boundary
Ultron uses an Ultron-optimized layout; the engines keep their FROZEN flat
working-directory scheme; a thin ADAPTER (Go package preferred, shell acceptable)
bridges them. Full detail in docs/project-ultron/fhc-script-data-layout.md.

  Ultron layout:
    data/<game-id>/<turn-id>/                  # galaxy/stars/planets/spNN.dat
    data/<game-id>/<turn-id>/<species-id>/     # that species' orders + reports

  Engine ground truth (FROZEN — do NOT teach the engine the nested layout):
    flat, bare filenames in cwd, keyed by species NUMBER — galaxy.dat,
    stars.dat, planets.dat, sp%02d.dat, sp%02d.ord, sp%02d.log, and reports
    sp%02d.rpt.t<turn> (turn number comes from galaxy.dat, not a dir name).
    cwd IS the game dir; no configurable data dir. The harness "does turns" by
    copying state into stage dirs and running with cwd there (see
    testdata/cref/generate.sh and the Chdir in internal/game tests).

  The adapter stages Ultron-layout files into a flat engine working dir before a
  run and harvests outputs back after. Feeding BOTH engines identical flat dirs
  from the SAME adapter reinforces the §0 invariant.

  For THIS read-only first slice the adapter is barely needed: the .dat files
  already sit flat inside data/<game-id>/<turn-id>/, so the script host can read
  galaxy/stars/planets/spNN.dat straight from a turn dir; the <species-id>/
  subdir only holds orders/reports, which entity queries don't touch. The open
  question that remains: does `fhc script` chdir into the turn dir, or do we
  parameterize the .dat loaders to read from an explicit path? Keep existing
  flat-cwd commands byte-neutral either way. (Full adapter — orders staging,
  report harvesting — is a later slice, once scripting drives RUNS not reads.)

## First investigation (ground the design in the real code)
- How does `fhc` load game state today? Find the galaxy/stars/planets/species
  .dat load path (the CommandRunner dispatch in cmd/fhc + internal/game, the
  marshal/unmarshal + *io.go codecs, and the C-named get_*_data loaders).
- What does "a turn" mean on disk? State in .dat is "current"; the harness
  snapshots per-turn dirs. Confirm how the script host should select a turn
  (point cwd / path at the right data/<game-id>/<turn-id>/ snapshot).
- What do the species and system/star entity structures look like
  (species_data_t, star_data_t/planet, nampla, etc.) so we know what a query
  returns.

## Scope for THIS round — start small, read-only
Design a minimal Lua surface and the commands behind it:
  1. Load a game (point the script host at a game's state directory).
  2. Load/select a specific turn.
  3. Query a few entities: species, and systems (maybe planets). Read-only.
No authoring/mutation verbs yet — that's a later slice.

Propose: the `fhc script <file.lua>` CLI shape; the Lua API names and return
shapes for load/turn/species/system; how each maps to existing loaders; how the
host points at a data/<game-id>/<turn-id>/ dir without perturbing RNG or breaking
flat-cwd commands; sandboxing (which stdlib to drop); and how we'd test
determinism for this slice.

## Deliverables
- A short design write-up (propose where it lives, e.g. a new doc under
  docs/project-ultron/ or a comment on #36) — discuss the shape with me before
  finalizing.
- A set of SMALL, incremental GitHub issues under epic #36 (reference #36 in
  each, label `harness`; suggest others). Keep them narrowly scoped, e.g.:
    * script: CLI subcommand skeleton (`fhc script <file.lua>`, GopherLua wiring)
    * script: sandbox the stdlib (drop os/io/math.random/time)
    * script: point host at a data/<game-id>/<turn-id>/ dir (chdir vs. path)
    * script: load game state (read galaxy/stars/planets/spNN.dat, no RNG touch)
    * script: select turn
    * script: query species entity
    * script: query system/star entity
    * script: determinism test for the read-only slice
  Right-size these after reading the code; the layout decision (above) shapes the
  load/turn/query commands, so settle the chdir-vs-path question before drafting
  those issues.

Start by reading CLAUDE.md, epic #36, and the docs/project-ultron/ docs
(especially fhc-script-data-layout.md), then investigate the load/query code
paths and propose the design. Discuss with me before creating the issues.
```
