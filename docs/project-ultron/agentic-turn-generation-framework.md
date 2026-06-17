# AI Agent Framework for Automated PBEM Game Testing

> **Adapted for Far Horizons.** This framework was written for a different PBEM
> engine and is mapped here onto FH. The engine-under-test is **`fh`** (the
> idiomatic SQLite port); **`fhc`** (the byte-faithful C port) is a trusted
> *differential oracle*. Crucially, FH **already has a well-structured test
> harness** that satisfies most of this framework's infrastructure assumptions —
> see the [Existing test harness](#existing-test-harness) section below before
> building anything. Companion docs:
> [vision](agentic-project-ultron.md), [harness architecture](ultron-harness-architecture.md).

## Executive Summary

As development of the modern port enters the iterative testing phase, the project faces a common challenge: generating sufficient gameplay activity to validate game systems, identify defects, and exercise edge cases without requiring constant participation from human testers.

This proposal outlines an AI-agent-based testing framework that can autonomously generate player orders for a Play-By-Email (PBEM) style 4X strategy game. The primary objective is not to create highly competitive opponents, but rather to accelerate testing cycles by producing plausible, legal, and varied gameplay decisions.

The proposed approach leverages modern large language models (LLMs) to interpret game documentation, analyze turn reports, and generate orders for subsequent turns. Combined with deterministic validation and automated execution, this framework can create a continuous test harness capable of simulating dozens or hundreds of game turns with minimal human intervention.

---

# Problem Statement

The project is approaching a development stage characterized by the following workflow:

1. Create a game setup.
2. Execute several turns.
3. Observe results.
4. Identify defects and balance issues.
5. Restart and repeat.

Historically, this process relies heavily on human testers. While humans provide valuable insight, they introduce several limitations:

* Scheduling constraints
* Limited availability
* Inconsistent play patterns
* Slow iteration cycles
* Difficulty reproducing specific behaviors

The objective is to supplement human testing with automated participants capable of generating realistic game activity at scale.

---

# Goals

The proposed system is designed to achieve the following goals:

## Primary Goals

* Generate legal orders automatically.
* Exercise a broad range of game systems.
* Increase testing throughput.
* Reduce dependence on human availability.
* Create reproducible test scenarios.

## Secondary Goals

* Produce plausible player behavior.
* Generate diverse strategic styles.
* Discover unusual state interactions.
* Support regression testing.

## Non-Goals

The system is not intended to:

* Defeat expert human players.
* Optimize strategic outcomes.
* Create a production-quality AI opponent.
* Replace human playtesting entirely.

The focus is test coverage rather than strategic excellence.

---

# Existing test harness

**FH already has a well-structured testing harness.** Before building anything,
inventory what is here — several of this framework's "components" and most of
Ultron's hard infrastructure assumptions are already satisfied. The table maps
each capability to where it lives.

| Capability this framework assumes | Already in FH? | Where |
|---|---|---|
| Deterministic, reproducible runs from a seed | **Yes** | Every command re-seeds from `FH_SEED`; a turn is a pure function of `(FH_SEED, sp0X.ord)`. |
| Setup → run N turns → observe → repeat | **Yes** | `testdata/cref/generate.sh` drives `create galaxy` → templates → `create species` → a 4-turn accumulating pipeline; `turn_pipeline_test.go` asserts it. |
| Plain-text orders through the *unmodified* pipeline | **Yes** | `sp0X.ord` files consumed by `combat → pre-departure → jump → production → post-arrival → finish → report` via `fopen_r("sp%02d.ord")`; no shortcut path. |
| Regression fixtures: `(scenario, orders) → reference output` | **Yes** | `testdata/scenarios/{build,jump,transfer,combat}/` commit hand-written `sp0X.ord`; `run_scenario`/`run_scenario_multi` run them; golden tests diff **every artifact byte-for-byte** against the C oracle (`scenario_test.go`). |
| A trusted oracle to compare against | **Yes** | `fhc` (byte-faithful C port) is validated against the C engine and is the reference `fh` is measured on. Run `make test-golden`. |
| Report-level validation surface | **Yes** | `TestReportMatchesC` + `TestCrossBackendReportParity` assert `sp0X.rpt.tN` byte-identical to `fhc` across all store backends. This is the phase-2 contract in `CLAUDE.md`. |
| A read-only state export | **Partial** | `fh export json` → `galaxy.json`, `systems.json`, `species.NNN.json` (`internal/game/export.go`). Designed for engine ingest; the oracle (§State oracle) extends it. |
| Coverage instrumentation (usage stats per system) | **No — to build** | The dispatch chokepoint exists (`get_command()` + per-phase `switch command`), but no counter / `coverage.tsv` yet. This is Ultron's "real heart." |
| GM scenario-authoring binding (`fh script` / `fhc script`) | **No — prerequisite** | The original used a separate Lua tool; FH adds it as a `script` subcommand on each engine instead, so the host stays in sync with the engine. **Invariant:** `fh script` and `fhc script` must author byte-identical state from the same seed (see harness architecture §0) — the differential oracle depends on it. Runtime choice (Gopher Lua vs. embedded JS) is a separate decision. |

**Net:** the reproducibility, the order-file pipeline, the oracle, and the
regression-fixture pattern are all done. What Ultron adds on top is (1) a
**coverage counter** at the dispatch chokepoint, (2) an **extended state export**
for the agent, and (3) eventually a **`script` subcommand** (`fh script` /
`fhc script`) for arbitrary scenario injection. Items (1) and (2) are additive and byte-neutral on the existing
golden trees; item (3) is the only true prerequisite, and it is independently
useful. See the [harness architecture](ultron-harness-architecture.md) §§2,7 for
the build order.

---

# Proposed Architecture

The framework consists of six major components.

## 1. Rules Knowledge Base

The game manual and supporting documentation are ingested and transformed into a structured reference source.

Information extracted includes:

* Order syntax
* Unit capabilities
* Resource rules
* Technology effects
* Diplomacy mechanics
* Turn-processing rules

This knowledge base serves as the authoritative source for agent behavior.

---

## 2. Historical State Processor

Each turn report is parsed into a structured game state.

Examples include:

* Species assets (ships, economic units, population)
* Fleets and their locations
* Named planets / colonies (namplas)
* Tech levels (MI, MA, ML, GV, LS, BI)
* Item inventories
* Ally / enemy / neutral relationships
* Known systems and scan data

The processor also maintains a timeline of previous turns.

This allows agents to understand not only the current state but also recent trends.

---

## 3. Agent Layer

LLM agents generate candidate orders.

Agents receive:

* Relevant rule excerpts
* Current game state
* Historical context
* Testing objectives

The output is a proposed order set.

Example prompt objective:

"Generate a legal turn that expands exploration activity and attempts at least one research, movement, and production action if permitted."

---

## 4. Validation Layer

This is the most important component of the system.

Rather than trusting AI output directly, all generated orders pass through deterministic validation.

In FH the validation *is* the engine: there is no separate validator to
re-implement. `get_command()` parses each keyword, and the per-order `do_*`
handlers enforce legality inline, writing rejection messages to the species
log/report. Validation checks the handlers perform include:

* Syntax correctness (`get_command`, `get_ship`, `get_location`, …)
* Legal command usage for the current phase
* Economic-unit and item availability
* Ship / colony ownership
* Jump-range and mishap restrictions
* Technology-level prerequisites
* Turn-specific and section-specific constraints

Invalid orders are rejected by the engine automatically. Note that an order
Ultron *intends* to be illegal is not a harness failure — exercising a rejection
path is itself coverage (+50 in the vision doc's scoring).

**FH differential check.** Because the engine-under-test is `fh` and `fhc` is a
trusted oracle, every generated turn runs on **both** engines and their
`sp0X.rpt.tN` reports are diffed. A divergence is an `fh` defect even when
neither engine crashes — a signal a single-engine harness cannot produce (see
the [harness architecture](ultron-harness-architecture.md) §0).

Where practical, the agent may be given the species log's rejection messages as
feedback and allowed to repair its submission.

---

## 5. Execution Layer

Validated orders are written as `sp0X.ord` files and run through the existing
turn pipeline (`combat → pre-departure → jump → production → post-arrival →
finish → report`) — on **both** `fh` and `fhc`, so their reports can be diffed.

Results are processed normally through that pipeline. No special handling is
required within game logic; the agent system behaves as an ordinary player.

---

## 6. Analytics and Reporting

Each run records:

* Orders generated
* Validation failures
* Turn outcomes
* Engine errors
* Unexpected game states

Metrics collected can identify:

* Frequently exercised systems
* Under-tested mechanics
* Crash triggers
* Rule ambiguities

---

# Agent Design Philosophy

A key design decision is to prioritize behavioral diversity over strategic optimization.

The objective is not to find the best move.

The objective is to create useful gameplay activity.

Recommended agent profiles include:

## Expansionist

Prioritizes:

* Exploration
* Colonization
* Economic growth

## Militarist

Prioritizes:

* Fleet construction
* Aggressive movement
* Combat engagement

## Researcher

Prioritizes:

* Technology advancement
* Long-term development

## Diplomat

Prioritizes:

* Alliances
* Communication
* Resource exchanges

## Conservative

Prioritizes:

* Defensive actions
* Resource preservation
* Low-risk decisions

## Chaotic

Intentionally favors unusual but legal actions to increase test coverage.

---

# Model Selection

The recommended implementation uses a mid-tier model — **Claude Sonnet 4.6** is
the current workhorse choice; **Claude Haiku 4.5** is worth trying for the
cheapest archetypes (e.g. the Chaos Goblin, which barely reasons). Reserve
**Claude Opus 4.8** for occasional hard cases, not the high-volume loop.

Reasons include:

* Lower cost per turn at scale
* Faster execution
* Sufficient reasoning capability for constrained order generation
* High-volume operation support

The project does not currently require premium frontier-model reasoning for the
bulk loop. Order generation is primarily a constrained decision-making task
rather than a deep strategic challenge — and much of the "decision" can be a
deterministic selector that the model only *steers* (picks targets/archetype
weights), keeping token cost low.

Resources should be invested in validation infrastructure rather than larger
models. In FH most of that infrastructure already exists (see
[Existing test harness](#existing-test-harness)); the marginal spend is the
coverage hook and the state export, not a bigger model.

---

# Testing Modes

> FH already has the building blocks for each mode. **Regression Mode** in
> particular is essentially done: `make test-golden` plus the
> `testdata/scenarios/` fixtures are deterministic, seeded, byte-for-byte
> outcome comparisons against the `fhc` oracle. The Ultron modes below are the
> *agent-driven* extensions of that existing machinery.

## Smoke Test Mode

Purpose:

Verify core systems function.

Characteristics:

* Small galaxies (small radius, few stars)
* Few species
* Short simulations

---

## Coverage Mode

Purpose:

Exercise as many mechanics as possible.

Characteristics:

* Multiple agent personalities
* Forced action diversity
* Broad system interaction

---

## Long-Haul Mode

Purpose:

Detect issues that emerge over many turns.

Characteristics:

* Automated multi-turn execution
* Hundreds of simulated turns
* State consistency monitoring

---

## Regression Mode

Purpose:

Validate bug fixes.

Characteristics:

* Repeated execution of known scenarios
* Deterministic seeds
* Automated comparison of outcomes

---

# Expected Benefits

The proposed system is expected to provide:

* Faster development feedback loops
* Increased gameplay coverage
* Reduced tester scheduling dependency
* Earlier defect discovery
* Better regression testing support
* Improved confidence in game stability

Most importantly, the framework allows the team to move from manually creating gameplay activity to automatically generating it at scale.

---

# Risks and Mitigations

## Risk: Invalid Orders

Mitigation:

Deterministic validation layer.

---

## Risk: Repetitive Behavior

Mitigation:

Multiple agent profiles and randomized objectives.

---

## Risk: Rules Misinterpretation

Mitigation:

Rule retrieval from authoritative documentation and validation against game logic.

---

## Risk: False Confidence

Mitigation:

Continue human playtesting for strategic, usability, and balance evaluation.

Agents supplement testing; they do not replace players.

---

# Recommended Pilot

Phase 1 should target a minimal proof-of-concept.

Success criteria:

1. Read the state export (`fh export json`) for a species.
2. Generate a legal `sp0X.ord` order set.
3. Run a complete turn pipeline on `fh` (and, for the differential check, `fhc`).
4. Repeat for 20 consecutive turns without human intervention, with `fh`/`fhc`
   reports matching at every turn.

Once stable, additional agent profiles and coverage objectives can be
introduced. Because reproducibility from `(FH_SEED, sp0X.ord)` already holds, any
crash or report divergence found during the pilot can be frozen straight into a
`testdata/scenarios/<name>/` fixture (see harness architecture §5).

This incremental approach minimizes implementation risk while quickly demonstrating practical value.

---

# Conclusion

An AI-assisted PBEM testing framework offers a practical method for increasing testing throughput during active development. By focusing on legal, varied, and reproducible gameplay rather than sophisticated strategic intelligence, the team can automate large portions of the setup-run-observe-repeat cycle that currently consumes developer and tester time.

The central design principle is straightforward:

**Use AI to generate activity, use deterministic systems to enforce correctness, and use automated analytics to identify defects.**

This approach provides a scalable path toward continuous gameplay testing while preserving the value of human testers for balance, usability, and strategic evaluation.
