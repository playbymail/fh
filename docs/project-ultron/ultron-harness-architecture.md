# Project Ultron — harness architecture

*Design / exploration. No engine code is changed by this document. Companion to
the vision doc [`agentic-project-ultron.md`](agentic-project-ultron.md) and the
turn-generation framework
[`agentic-turn-generation-framework.md`](agentic-turn-generation-framework.md).*

> **Adapted for Far Horizons.** These plans originated on a different PBEM
> engine (one with nobles, gold, skills, and an in-process Lua "immediate-mode"
> GM binding). Far Horizons is structured differently, and — for Ultron's
> purposes — more favorably. The two facts that reshape the whole design:
>
> 1. **Two engines, not one.** `fhc` is the byte-faithful C port and is the
>    *finished, trusted oracle*; `fh` is the idiomatic SQLite engine under
>    active development. **Ultron's engine-under-test is `fh`, with `fhc` as a
>    differential oracle.** See §0.
> 2. **No GM scripting binding exists yet.** The original assumed a *separate*
>    in-process Lua scripting tool for scenario authoring and state queries. FH
>    has no such tool today, and rather than ship a separate binary it will be a
>    **`script` subcommand on each engine** (`fh script` / `fhc script`) so the
>    scripting host stays in lockstep with the engine it drives. It is a
>    *prerequisite* (Gopher Lua vs. an embedded JS runtime is a separate
>    decision). Where this doc says "scenario authoring" or "state oracle via a
>    scripting binding," read it as *to be built*, not *reuse as-is*.

## TL;DR — recommendation

Build the Ultron harness as **three separate roles with one hard rule**, not as
one monolithic agent:

1. **Scenario authoring** — author deterministic pre-turn worlds. FH has no GM
   scripting binding yet, so this is the `fh script` / `fhc script` prerequisite
   (see the framework doc). Until it lands, scenarios are seeded the way the test harness
   already does it: a fixed `FH_SEED` plus `create galaxy` / `create species`
   from a config file (`species.cfg`), captured as the post-setup `.dat`/store
   snapshot the scenario starts from.
2. **State oracle** — give an agent a machine-readable view of "what is here /
   what does this species have / what can it legally do this turn." FH already
   emits a partial export (`fh export json` → `galaxy.json`, `systems.json`,
   `species.NNN.json`); the oracle extends that read surface. In-process reads
   are cheap and lossless; re-parsing the player report text from outside is
   brittle and lossy.
3. **Order submission** — *the agent emits plain-text player order files*
   (`sp0X.ord`) and they run through the **unmodified turn pipeline.** This is
   the thing under test — it must **not** be shortcut.

**The hard rule: Ultron orders never bypass the real order pipeline.** A future
GM/immediate path (the `fh script` authoring verbs) skips the per-order
validation that `do_*` handlers perform during a real turn. The whole point of
Ultron is to stress the *player* order path — parsing (`get_command`),
legality and cost checks, the full `combat → pre-departure → jump → production →
post-arrival → finish` lifecycle. Routing generated orders through a GM path
would bypass the exact layer Ultron exists to break.

A fourth concern is **neither scripting nor order files**: **coverage
instrumentation** — a counter at the command-dispatch chokepoint. That is the
actual heart of "coverage-driven," and it is engine-side.

---

## 0. The two-engine setup (FH-specific, and a gift)

The source engine had one binary, so its only failure signals were "engine
crashed" and "order that should have been rejected wasn't." FH has two engines
that must agree, which adds a third, sharper signal: **report divergence.**

| Engine | Role for Ultron |
|--------|-----------------|
| `fhc`  | Byte-faithful C port, validated against the C engine on `.dat`/s-expr bytes. **Trusted oracle.** Ultron does not hunt for bugs *in* `fhc`. |
| `fh`   | Idiomatic SQLite engine, active development. **Engine-under-test.** |

The phase-2 contract (`CLAUDE.md`) is already that `fh` and `fhc` must produce
**byte-identical `sp0X.rpt.tN` reports** for the same `(seed, orders)`. Ultron
turns that contract into a fuzzer: submit the same generated `sp0X.ord` files,
under the same `FH_SEED`, to **both** engines, and diff their turn reports. The
outcomes:

- **`fh` crashes / errors, `fhc` does not** → defect in `fh`.
- **Reports differ** → defect in `fh` (the report is the parity surface).
- **Both reject the order identically** → the order was illegal; that rejection
  is itself coverage (an exercised validation path), not a failure.
- **Both accept and reports match** → covered, consistent — the boring,
  desired outcome.

So in FH the "validation boundary" the original doc worried about (permission
checks skipped in immediate mode) is replaced by something we can *measure*:
**`fh`-vs-`fhc` report parity**. That is a strictly stronger oracle than a
single engine could offer.

> ### INVARIANT — scripting must be consistent across `fh` and `fhc`
>
> **Project requirement.** The `script` subcommand (`fh script` / `fhc script`)
> must behave *identically* on both engines: the same script, given the same
> `FH_SEED`, must produce **byte-identical engine state** — the same galaxy,
> species, ships, colonies, tech levels, and RNG-stream position. Same language,
> same surface API, same verbs, same execution order, same RNG draws.
>
> **Why it is load-bearing.** The whole differential oracle rests on it. Ultron
> attributes any `fh`-vs-`fhc` report divergence to an `fh` defect — but that
> inference is only valid if the two engines *started a scenario from the same
> state*. If `fh script` and `fhc script` author even slightly different worlds,
> every downstream report diff is ambiguous (engine bug, or authoring skew?) and
> the oracle is worthless. Scenario authoring is the one place where a scripting
> divergence silently poisons every later comparison.
>
> **What it constrains.** The runtime choice (Gopher Lua vs. embedded JS) must be
> shareable by both binaries; the GM-authoring verbs and any RNG they consume
> must be a single shared implementation, not two parallel ones that can drift.
> Treat a `fh script` / `fhc script` state divergence as a release-blocking bug,
> testable the same way reports are: author a scenario on each engine and diff
> the resulting state (or the turn-0 report) byte-for-byte.

---

## 1. Why the split — the validation boundary

> **The order pipeline is the thing under test; reads are not.**

That single fact decides everything:

- Reading state (via `fh export json` / an extended oracle) is **safe and
  ideal** — it touches live state without perturbing it.
- Writing *orders under test* through any GM/scripting shortcut is **wrong** —
  it skips the `get_command` parsing, the inline legality/cost checks in the
  `do_*` handlers, and the per-phase `start`/`finish` lifecycle that Ultron is
  trying to exercise. A bug that manifests as "this order should have been
  rejected but wasn't" is invisible if the order never went through that path.

So queries go through the read-only export; orders-under-test go out as
`sp0X.ord` text and through the real turn runner — on both engines.

## 2. The four roles in detail

| Role               | Mechanism                              | Status in FH        | Reference |
|--------------------|----------------------------------------|---------------------|-----------|
| Scenario authoring | `fh script` GM verbs (or seeded `create`) | **prerequisite** / partial | framework doc; `testdata/cref/generate.sh` |
| State oracle       | `export json` + extended read surface  | **partial**         | `internal/game/export.go`; `fh export json` |
| Order submission   | plain-text `sp0X.ord` files            | **engine native**   | `jump.go`/`production.go`/… `fopen_r("sp%02d.ord")` |
| Coverage tracking  | counter at the dispatch chokepoint     | **new**             | `command.go` `get_command()` + per-phase `switch command` |

### 2.1 Scenario authoring

Ultron's **Scenario Injection** archetypes (resource shortages, excessive
wealth, isolated species, tech imbalance — see the vision doc) are GM-authored
starting conditions. At *authoring* time we *want* a no-validation GM mode,
because we are fabricating state, not playing a turn.

FH has no GM scripting binding yet, so there are two paths:

- **Now:** seed deterministically and snapshot, exactly as the test harness
  does — `FH_SEED` + `create galaxy --species=N` + `create species
  --config=species.cfg`, captured as the post-setup snapshot a scenario starts
  from (`testdata/cref/setup/`). This covers "fresh, varied starts" but not
  arbitrary mid-game fabrication.
- **Later:** the `fh script` prerequisite provides authoring verbs to fabricate
  arbitrary mid-game state (a colony at the edge of starvation, a fleet with no
  fuel). This is the real "Scenario Injection" capability. For the differential
  oracle, `fhc script` must author the *same* fabricated state on the C-port side
  so both engines start a scenario identically — this is the cross-engine
  scripting **invariant** in §0, and it is exactly what makes scenario authoring
  the riskiest place for a scripting divergence to hide.

### 2.2 State oracle (extend the read surface)

Ultron's central question is *"what game systems have not been exercised, and
what can this species legally do this turn?"* The first half is coverage stats
(§2.4); the second half is a **read against live state**.

Today the read surface is `fh export json` →

- `galaxy.json` — turn number, radius, counts, wormholes.
- `systems.json` — stars, planets, coordinates, who has scanned what.
- `species.NNN.json` — per-species tech levels, economic units, ships, named
  planets (colonies), inventories, ally/enemy relations.

To drive coverage-targeted order selection an agent additionally wants, roughly:

- **location / topology** — where each ship is, which systems are reachable
  within jump range, where wormholes lead (to drive **Explorer** / movement
  coverage: `JUMP`, `MOVE`, `WORMHOLE`).
- **inventory & economy** — items, economic units, available population (to
  drive **Accountant** / logistics / production coverage: `TRANSFER`, `UNLOAD`,
  `BUILD`, `PRODUCTION`).
- **tech & capabilities** — the six tech levels (MI, MA, ML, GV, LS, BI) and
  what could be studied or taught (to drive **Mad Scientist** coverage:
  `RESEARCH`, `TEACH`, `ESTIMATE`).
- **neighbors & relations** — which other species are known/co-located and the
  ally/enemy/neutral matrix (to drive **diplomacy** coverage: `ALLY`, `ENEMY`,
  `NEUTRAL`, `MESSAGE` — among the lowest-usage systems in the vision doc's
  table).

These are all reads. They do not perturb the RNG stream and do not run any
`do_*` handler. Building them as extensions of the JSON export keeps the
agent's view **lossless and in sync with engine truth** — strictly better than
re-parsing `sp0X.rpt.tN` out of process.

> Open question for §6: *how much* order legality the oracle should pre-compute
> vs. let the agent attempt-and-observe. Attempting illegal orders on purpose is
> itself coverage (+50 for a validation failure), so the oracle should *inform*
> selection, not gate it.

### 2.3 Order submission (text files, real pipeline, both engines)

Order files are plain text, one order per line, grouped into the sections the
pipeline consumes:

```
START PRE-DEPARTURE
    ; orders for the pre-departure phase
END

START JUMPS
    JUMP PL1, 12 18 5
END

START PRODUCTION
    PRODUCTION PL Home
    BUILD 10 TR1
END

START POST-ARRIVAL
    SCAN 12 18 5
END
```

Each `sp0X.ord` is loaded by the relevant phase command via
`fopen_r("sp%02d.ord")` (see `predeparture.go`, `jump.go`, `production.go`,
`postarrival.go`, `combat1.go`), parsed by `get_command()`, and dispatched by
that phase's `switch command` into the `do_*` handlers — the same path a real
player's orders take.

**Ultron's job is to write these files.** Whatever picks the orders (§3) emits
text; the unmodified engines consume it. The agent runs the standard pipeline on
**both** `fh` and `fhc`:

```
locations → combat → pre-departure → jump → production → post-arrival → finish → report
```

Three payoffs:

- **It tests the real path.** Validation failures and engine exceptions — the
  highest-value coverage events (+50 / +100) — only happen here. And running
  both engines makes **report divergence** a fourth, equally valuable signal
  (§0).
- **It is reproducible.** A posted turn is a pure function of `(FH_SEED,
  sp0X.ord files)`. Every FH command re-seeds from `FH_SEED`, so "produces
  reproducible test scenarios" — a literal Ultron success criterion — falls out
  for free.
- **It avoids RNG perturbation.** Generating orders out of process (text in,
  turn-process out) never interleaves agent draws with the engine's RNG stream,
  so the two engines stay in lockstep.

### 2.4 Coverage instrumentation (engine-side, the real heart)

Every order funnels through one place: `get_command()` in `command.go` resolves
the keyword (via `command_abbr[]`) to one of the command constants in `const.go`
(`ALLY=1` … `WORMHOLE=52`), and each phase's `switch command` dispatches it.

A coverage counter hooks there and records, per command constant, the outcome:

- **rejected** — a `do_*` handler refused the order (wrote an error to the
  species log/report and did not act).
- **executed-ok** — the handler acted and reported success.
- **executed-fail** — the handler ran but the action failed at runtime.

That single hook yields the vision doc's usage table (FH-flavored):

| System (coarse)      | Example verbs                          | Last Used | Count |
|----------------------|----------------------------------------|-----------|-------|
| Jump / movement      | `JUMP` `MOVE` `WORMHOLE`               |           |       |
| Combat               | `ATTACK` `ENGAGE` `AMBUSH` `HIJACK`    |           |       |
| Production / build   | `PRODUCTION` `BUILD` `RECYCLE` `SHIPYARD` |        |       |
| Research             | `RESEARCH` `TEACH` `ESTIMATE`          |           |       |
| Diplomacy            | `ALLY` `ENEMY` `NEUTRAL` `MESSAGE`     |           |       |
| Logistics / transfer | `TRANSFER` `UNLOAD` `SEND` `LAND` `INSTALL` |       |    |
| Recon                | `SCAN` `TELESCOPE` `INTERCEPT` `HIDE`  |           |       |
| Terraform / develop  | `TERRAFORM` `DEVELOP` `REPAIR` `UPGRADE` |         |     |

Emit it as a small machine-readable file (`coverage.tsv`) at end of turn,
alongside the reports. This is the feedback signal that closes the loop: the
harness reads it to choose *next* turn's coverage targets. It is independent of
the order format and of whatever language the brain is written in.

> Note: the command constant is the right grain for *which verb fired*. The
> coarse "systems" column (diplomacy, recon, logistics, …) that groups several
> verbs is a static **verb→system map** maintained next to the command table,
> not a new per-verb hook.
>
> Build the hook in `fh` (the engine-under-test). Adding it to `fhc` would
> change `fhc`'s output and is forbidden by `CLAUDE.md`; if `fhc` ever needs the
> same data, derive it from `fhc`'s logs out of process instead.

## 3. The one real decision — where the agent brain lives

Roles §2.1–§2.4 are settled. The open architectural choice is where the
**decision logic** lives — the archetypes (Bureaucrat, Chaos Goblin, …),
coverage scoring, and order selection.

### Option A — in-process scripting agent

Because the scripting host is a `script` subcommand built into the engine, the
agent runs as `fh script <agent>` in one process. The script queries live state
(§2.2), decides, writes the `sp0X.ord` file, the turn runs.

- **Pro:** single binary, fully deterministic, cheapest possible queries, reuses
  the engine's own (future) `script` host.
- **Con:** logic is pinned to whatever scripting runtime we pick (Gopher Lua /
  embedded JS). LLM-driven or richly stateful archetypes (Chaos Goblin) are
  awkward to embed. Mixing decision logic into the host muddies the engine.

### Option B — out-of-process harness

The engine(s) run a turn and emit a state export + `coverage.tsv`; an external
harness (Go test driver / a Python script / an LLM agent / anything) reads them,
selects coverage-targeted orders, writes `sp0X.ord` files, loops.

- **Pro:** decision logic can be anything; engine stays clean; archetypes and
  scoring evolve without touching engine code; natural fit for "submit unusual
  but legal orders at industrial scale"; and it is the only option that cleanly
  drives **both** `fh` and `fhc` for differential testing (§0).
- **Con:** needs a clean machine-readable **state export**, and care to keep the
  exported view from drifting from engine truth (mitigated: the export is
  generated *by* the engine).

### Recommendation — out-of-process (Option B), with the oracle in-engine

Keep the **state export and (future) scenario authoring in-engine**, and put the
**decision logic out of process**, communicating purely through *(state export
in → `sp0X.ord` files out)*. Concretely:

- The oracle is just the engine's read path surfaced as JSON — `fh export json`
  today, extended per §2.2. The *export format is the contract*, not the
  language the brain is written in.
- The brain reads `(state export, coverage.tsv)`, applies an archetype to pick
  orders, writes `sp0X.ord`, and runs the unmodified pipeline on `fh` **and**
  `fhc`, then diffs reports.

This keeps orders on the validated path, keeps both engines clean, doesn't pin
Ultron's brain to a scripting subset, preserves `(seed, orders)` reproducibility,
and is the natural home for the two-engine differential check that is FH's best
defect signal.

## 4. The loop, end to end

```
            ┌─ (once) scenario authoring ──────────────────────────────┐
            │  seeded create galaxy/species  →  post-setup snapshot     │  (GM/seed)
            │  (later: fh script verbs to fabricate mid-game state)    │
            └──────────────────────────────────────────────────────────┘
                                   │
      ┌────────────────────────────▼─────────────────────────────────────┐
      │  per turn:                                                        │
      │   1. state export     fh export json ─────► galaxy/systems/.json │  (read-only)
      │   2. brain (out-of-proc): state + coverage ► pick orders         │  (archetype)
      │   3. write order files                    ► sp0X.ord             │  (plain text)
      │   4. run pipeline on BOTH engines (fh, fhc): locations…report    │  (real path!)
      │   5a. read coverage   fh ─────────────────► coverage.tsv         │  (dispatch hook)
      │   5b. diff reports    fh.sp0X.rpt  vs  fhc.sp0X.rpt              │  (parity oracle)
      │   6. score + choose next-turn coverage targets                   │
      └──────────────────────────────────────────────────────────────────┘
                                   │ repeat
```

Steps 1, 4, 5a are engine-side; steps 2, 3, 5b, 6 are the out-of-process brain.
`fh` is touched only for the read-only export (§2.2) and the coverage hook
(§2.4) — both additive and both byte-neutral on the existing golden trees (reads
and a counter perturb no report output). `fhc` is **not** touched.

## 5. Why this is well-timed (the harness is already most of the way there)

Ultron's two hardest infrastructure assumptions are **already satisfied** by the
existing test harness (see the framework doc §"Existing test harness" for the
full inventory):

- **Reproducibility from `(seed, orders)`** is a first-class, proven property:
  every command re-seeds from `FH_SEED`, and `testdata/cref/generate.sh` already
  drives the full setup → 4-turn pipeline deterministically.
- **The `(scenario, orders) → reference output` fixture pattern** that Ultron's
  "automated defect reproduction" criterion describes is the repo's *standard
  practice today*: `testdata/scenarios/{build,jump,transfer,combat}/` commit
  hand-written `sp0X.ord` files, run them through the real pipeline, and diff
  **every turn artifact — including `sp0X.rpt.tN` — byte-for-byte** against the
  C oracle (`scenario_test.go`, `run_scenario` / `run_scenario_multi`).

So a reproduced Ultron failure is frozen into a regression fixture exactly the
way `build`/`jump`/`transfer`/`combat` already are: drop the crashing `sp0X.ord`
files under `testdata/scenarios/<name>/`, add a `run_scenario` block to
`generate.sh`, add a golden test. Ultron's "reproducible test scenarios" /
"automated defect reproduction" criteria *are* the fixture pattern the repo
already runs — Ultron just discovers the inputs automatically.

## 6. Open questions

1. **Scripting runtime for the `script` subcommand.** Gopher Lua (closest to the original) vs.
   an embedded JS runtime vs. a small Go-native DSL. Gates §2.1's full scenario
   injection and Option A. Separate decision; see the framework doc.
2. **State-export format & scope.** Extend `fh export json`, or add a dedicated
   `fh export ultron` mode? How much topology/legality to pre-compute vs. let
   the brain attempt-and-observe (attempting illegal orders is itself coverage).
   Smallest useful export first.
3. **Verb→system grouping.** The static map that turns command constants
   (`ALLY`, `JUMP`, …) into the coverage table's coarse systems (diplomacy,
   logistics, recon, …).
4. **Coverage-score bookkeeping.** Where cross-turn usage stats live (the
   brain's state, not the engine's) and how "previously untested interaction"
   (+25) is detected.
5. **Archetype representation.** Data-driven (weights over verb/system classes)
   vs. coded strategies; whether an LLM picks orders or only picks *targets* a
   deterministic selector then expands.
6. **Differential-diff ergonomics.** When `fh` and `fhc` reports diverge, how the
   harness localizes the divergence (first-differing byte is already what
   `report_test.go`/`store_parity_test.go` do — reuse that).
7. **Multi-species turns.** The vision doc's long-term multi-agent diplomacy is
   out of scope for the first cut, but the loop in §4 already supports N
   `sp0X.ord` files per turn (the engine processes all four species per phase).

## 7. First-cut scope (proposed)

Smallest end-to-end slice that proves the architecture, in dependency order:

1. **Coverage hook** in `fh` at the `get_command()` / per-phase dispatch, plus a
   `coverage.tsv` dump (engine-side, byte-neutral on reports). Immediately useful
   even before any agent — it answers "what did *today's* golden turn exercise."
2. **State export** — extend `fh export json` with the minimal oracle fields
   (ship locations, inventories, tech levels, relations) the brain needs.
3. **A trivial brain** — one archetype (the Chaos Goblin is cheapest: pick
   legal-ish orders at random) writing `sp0X.ord` files out of process.
4. **Loop driver** wiring §4 steps 1–6, running both `fh` and `fhc`, diffing
   reports, and freezing any crash *or divergence* as a `testdata/scenarios`
   fixture.

Each step is independently testable, and the first two are valuable on their own
(coverage stats for the existing golden turns; a richer state export for other
tooling).
