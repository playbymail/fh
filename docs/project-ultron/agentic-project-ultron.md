# Addendum A: PROJECT ULTRON

## Advanced Coverage-Oriented Testing Initiative

> **Adapted for Far Horizons.** This vision was written for a different PBEM
> engine; the examples below are mapped onto FH's systems and orders. Two FH
> facts shape it: (1) the engine-under-test is **`fh`** (the idiomatic SQLite
> port), with **`fhc`** (the byte-faithful C port) acting as a *differential
> oracle* — see the [harness architecture](ultron-harness-architecture.md) §0;
> (2) much of the infrastructure Ultron needs already exists in the test harness
> (seeded reproducibility, scenario fixtures, byte-for-byte report parity) — see
> the [turn-generation framework](agentic-turn-generation-framework.md)
> §"Existing test harness."

### Overview

Following successful implementation of the autonomous turn-generation framework, the development team may elect to expand testing capabilities through the introduction of directed behavioral objectives.

This initiative, codenamed **PROJECT ULTRON**, focuses on maximizing gameplay system coverage rather than maximizing strategic performance.

The underlying premise is simple:

A highly skilled player may unknowingly avoid large portions of the game.

A deliberately curious player will not.

PROJECT ULTRON seeks to create intentionally varied gameplay behavior designed to expose defects, edge cases, and unexpected interactions between game systems.

---

# Objective

The primary objective of PROJECT ULTRON is to answer the following question:

**"What game systems have not been exercised recently?"**

Once identified, agent behavior is adjusted to increase interaction with those systems.

The goal is comprehensive mechanical coverage across repeated testing cycles.

---

# Coverage-Driven Agent Behavior

Traditional AI agents operate according to strategic goals.

Examples include:

* Expand territory
* Maximize resources
* Win wars
* Research efficiently

PROJECT ULTRON agents operate according to coverage goals.

Examples include (FH orders in parentheses):

* Initiate diplomacy (`ALLY`, `ENEMY`, `NEUTRAL`, `MESSAGE`)
* Conduct resource transfers (`TRANSFER`, `UNLOAD`, `SEND`)
* Attempt colonization (`BUILD` a colony, `LAND`, `INSTALL`)
* Explore unexplored systems (`JUMP`, `MOVE`, `WORMHOLE`, `SCAN`, `TELESCOPE`)
* Research rarely advanced tech levels (`RESEARCH` MI/MA/ML/GV/LS/BI, `TEACH`)
* Build underutilized ship/unit types (`BUILD`, `SHIPYARD`, `UPGRADE`)
* Exercise terraforming and development (`TERRAFORM`, `DEVELOP`, `RECYCLE`)
* Exercise recon / interception (`INTERCEPT`, `AMBUSH`, `HIDE`, `HIJACK`)
* Use obscure order combinations

Success is measured by system interaction rather than strategic outcome.

---

# Coverage Tracking

The test harness maintains usage statistics for major gameplay systems, keyed on
the FH command constant resolved by `get_command()` and grouped into coarse
systems by a static verb→system map (see the harness architecture §2.4).

Example metrics:

| System             | Example verbs                  | Last Used | Usage Count |
| ------------------ | ------------------------------ | --------- | ----------- |
| Jump / movement    | `JUMP` `MOVE` `WORMHOLE`       | Turn 104  | 12,481      |
| Combat             | `ATTACK` `ENGAGE` `AMBUSH`     | Turn 103  | 4,228       |
| Diplomacy          | `ALLY` `ENEMY` `MESSAGE`       | Turn 87   | 212         |
| Technology trading | `TEACH` `TECH` (`TECH_TRANSFER`) | Turn 51 | 14          |
| Recon / intercept  | `TELESCOPE` `INTERCEPT` `HIDE` | Turn 33   | 3           |
| Terraforming       | `TERRAFORM`                    | Never     | 0           |

Systems with low usage become high-priority targets.

---

# Directed Test Objectives

Agents may be assigned explicit mission objectives.

Examples include:

### Diplomatic Exercise

Objectives:

* Contact another species (`MESSAGE`)
* Declare an ally or enemy (`ALLY`, `ENEMY`, `NEUTRAL`)
* Teach or trade technology (`TEACH`, `TECH`)

Success Criteria:

At least one diplomacy action is submitted.

---

### Logistics Exercise

Objectives:

* Transfer economic units or items (`TRANSFER`, `SEND`)
* Move cargo and unload (`UNLOAD`, `LAND`)
* Relocate ships and population (`MOVE`, `INSTALL`)

Success Criteria:

At least one logistics-related action occurs.

---

### Military Exercise

Objectives:

* Build warships (`BUILD` combat-capable classes)
* Position and intercept fleets (`MOVE`, `INTERCEPT`, `AMBUSH`)
* Engage hostile targets (`ATTACK`, `ENGAGE`)

Success Criteria:

Combat systems are activated.

---

### Economic Exercise

Objectives:

* Construct ships and colony infrastructure (`BUILD`, `SHIPYARD`, `DEVELOP`)
* Run and reallocate production (`PRODUCTION`, `RECYCLE`)
* Spend accumulated economic units

Success Criteria:

Economic systems are exercised.

---

# Scenario Injection

PROJECT ULTRON may intentionally create unusual starting conditions. (In FH this
requires the `fh script` / `fhc script` GM-authoring prerequisite for arbitrary mid-game state;
until then, deterministic seeded `create galaxy`/`create species` covers varied
*starts* — see the harness architecture §2.1.)

Examples include:

* Economic-unit or fuel shortages
* Overcrowded colonies (population near the planet limit)
* Isolated species (home system far from any jump neighbor)
* Excessive wealth (large banked economic units)
* Technology imbalance (lopsided MI/MA/ML/GV/LS/BI levels)
* Hostile borders (co-located species set as mutual enemies)

These scenarios encourage interaction with systems that normal gameplay may rarely reach.

---

# Behavioral Archetypes

In addition to standard personalities, PROJECT ULTRON introduces specialized testing archetypes.

### The Bureaucrat

Attempts every administrative action available.

### The Accountant

Optimizes resource movement and economic transactions.

### The Explorer

Aggressively seeks unknown map locations.

### The Mad Scientist

Researches unusual technologies and pursues experimental options.

### The Warmonger

Seeks conflict at every opportunity.

### The Chaos Goblin

Selects legal but unexpected actions.

The Chaos Goblin exists solely because experience has demonstrated that many bugs hide behind decisions no rational player would ever make.

---

# Coverage Score

Each simulation run receives a Coverage Score.

Example formula:

* New system exercised: +10
* Rare system exercised: +5
* Previously untested interaction: +25
* Validation failure discovered: +50
* **`fh`/`fhc` report divergence discovered: +75** *(FH-specific — the two engines must produce byte-identical `sp0X.rpt.tN`; any divergence is an `fh` defect, see harness architecture §0)*
* Engine exception discovered: +100

The score provides a quantitative measure of testing effectiveness.

A low-scoring simulation may still represent successful gameplay but offers limited testing value.

---

# Long-Term Vision

Future versions of PROJECT ULTRON may support:

* Multi-agent diplomacy
* Emergent alliance formation
* Automated regression suites
* Continuous integration execution
* Nightly campaign simulations
* Automated defect reproduction

At sufficient maturity, entire PBEM campaigns may be executed autonomously for the sole purpose of discovering defects.

---

# Success Criteria

PROJECT ULTRON will be considered successful if it:

1. Increases gameplay system coverage.
2. Identifies defects earlier in development.
3. Reduces manual tester workload.
4. Exercises rarely used mechanics.
5. Produces reproducible test scenarios.

Winning games is not a success criterion.

Breaking assumptions is.

---

# Final Note

Despite the project codename, PROJECT ULTRON is not intended to replace human testers, dominate galactic civilization, or declare itself the next stage of evolution.

Its sole purpose is to submit unusual but legal orders at industrial scale until something breaks.

When something breaks, it has succeeded.
