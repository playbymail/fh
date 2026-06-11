# CLAUDE.md

Guidance for Claude Code (and other agents) working in this repository.

## What this project is

A port of the classic play-by-mail game **Far Horizons** from C to Go.

The authoritative C source is cloned at **`../Far-Horizons/`** (package
`github.com/playbymail/Far-Horizons`). Consult it whenever porting — the goal of
phase 1 is to match it exactly.

**Target version: Far Horizons `v7.5.12`.** The golden reference data
(`testdata/cref/`) and every parity test are generated from that tag, so keep
the `../Far-Horizons/` clone on `v7.5.12`. This release fixes the
`version`/`import` dispatcher argument offset in `fh.c` (the branches used
`argv + 1` instead of the surrounding `argv + i`, so a preceding global flag
like `fh -v version` misparsed the command token), fixes `versionCommand`'s
unknown-option message passing a NULL `val` to `%s`, and changes
`versionCommand`'s success path to `return 0` (was `return 2`) and its printed
string to `7.5.12` — all mirrored in the Go port. It builds on v7.5.11's
deterministic `max_tech_level` fix in `finish.c` (v7.5.10) plus a set of
read-before-init fixes (`stats.c` `totalBankedEconUnits`, `do.c`
`do_BUILD_command`'s `new_ship`, `command.c`/`combat.c` `best_species_index`
guards, and removal of the `do_UNLOAD` `current_pop` dead store). If the
maintainer adopts a newer upstream release, bump this version note and
regenerate the goldens (`make golden-ref`) in the same change.

## Strategy: two phases, in order

1. **Byte-faithful C port first** — `internal/game`, run by the `cmd/fhc` binary.
   A direct, auditable translation of the C engine that reproduces the original
   binary `.dat` files, s-expression `.txt` exports, logs, and reports
   **byte-for-byte**. This is the trusted reference implementation. **Finish and
   validate this before leaning on the idiomatic package.**

2. **Idiomatic Go second** — clean `internal/...` packages run by the `cmd/fh`
   binary, storing game state in a **SQLite datastore (ZombieZen)** instead of
   the original binary files (JSON is used for config input and snapshots). It is
   built and validated *against* the byte-faithful port: only port a subsystem to
   the idiomatic side once its `fhc` counterpart is correct.

Keep the two cleanly separated. When in doubt, work on the byte-faithful port.

## Layout

```
cmd/
  fhc/        # byte-faithful C-port runner -> internal/game CommandRunner
  fh/         # idiomatic Go runner -> ff/v4 CLI + internal/data/store (SQLite)
internal/
  game/       # the byte-faithful C port (phase 1) — the bulk of the work
  data/store/ # idiomatic SQLite store (ZombieZen) for phase 2
  config/     # JSON config parsing (species, etc.)
  engine/rng/ # deterministic PRNG + golden vectors (used by golden-rng tool)
  cerrs/      # error helpers
testdata/cref/ # C-engine reference output for parity tests (see generate.sh)
```

> Note: `internal/engine/{world,orders,effects,schedule}` are early
> architectural scaffolding from the original plan and are not yet wired into
> either binary. Don't build on them without checking with the maintainer.

## Phase 1: the byte-faithful port (`internal/game`)

- Translates the C file-for-file. Examples: `combat.c` → `combat1.go` +
  `combat2.go`; `do.c` + `do_*.c` → `do1.go`/`do2.go`/`do3.go`; the C `*vars.c`
  global-variable files collapse into `vars.go`; the CLI dispatcher `fh.c` →
  `fh.go` (`CommandRunner`).
- **Preserve the C names.** Struct and field identifiers stay verbatim
  (`species_data_t`, `nampla_base`, `num_stars`, …) so the Go can be diffed
  against the C. This is the one place we deviate from idiomatic Go naming.
- **On-disk formats are part of the contract.** Binary `.dat` records go through
  `marshal.go`/`unmarshal.go` and the `*io.go` codecs; s-expression text exports
  go through the `*DataAsSExpr` writers. The C engine writes `galaxy.hs.txt`,
  `stars.hs.txt`, `planets.hs.txt`, and `species%03d.txt`, so the port must too —
  **do not remove the `*AsSExpr` writers.**
- **Globals + tests.** The port relies on package-level globals (mirroring the C).
  Tests must **not** run in parallel and should call `ResetState()` first when
  they touch globals.
- **`sexpr` is not ported.** The C `sexpr.c` is a Lisp/s-expression *interpreter*
  used only for debugging; it is redundant given direct `.dat`/JSON comparison
  and has been intentionally dropped (including its CLI dispatch).

### Validating against the C engine

`testdata/cref/generate.sh` builds the C engine
(`../Far-Horizons/build/fh`), runs it under a fixed `FH_SEED`, and writes
reference `.dat`/`.txt`/`.log` outputs. The Go port must produce byte-identical
results.

- Covered today: setup phase — `galaxy_create_test.go`, `species_create_test.go`,
  `templates_create_test.go` — plus record sizes and round-trips
  (`io_roundtrip_test.go`).
- Not yet covered: the turn pipeline (locations → combat → pre-departure → jump →
  production → post-arrival → finish → report). Adding these golden tests is
  active work; add them as subsystems are confirmed.

## Phase 2: the idiomatic SQLite store (`internal/data/store`)

Behind the `Store` interface.

- **Driver**: `zombiezen.com/go/sqlite` (CGo-free; builds with `CGO_ENABLED=0`).
  Do **not** use `database/sql`. `modernc.org/sqlite` is only an indirect dep.
- **Connections**: go through a `*sqlitemigration.Pool`. Every method does
  `conn, err := s.pool.Take(ctx)` then `defer s.pool.Put(conn)`. A leaked
  connection deadlocks the pool.
- **Queries**: `sqlitex.Execute` / `sqlitex.ExecuteScript` with
  `ExecOptions.Args` to bind and `ResultFunc` to scan. Wrap multi-statement
  writes in `sqlitex.Transaction(conn)`.
- **Pragmas**: `foreign_keys`, `busy_timeout`, `synchronous` set per connection in
  the pool's `PrepareConn` hook; WAL via the `sqlite.OpenWAL` flag.
- **Migrations**: `sqlitemigration` over an append-only
  `appSchema.Migrations []string`; version tracked by `PRAGMA user_version`.
  Never edit or reorder a released migration — only append. See
  `internal/data/store/MIGRATIONS.md`.

## Commands

```sh
go build -o dist/local/fhc ./cmd/fhc   # byte-faithful C-port runner
go build -o dist/local/fh  ./cmd/fh    # idiomatic Go runner
go test ./...                          # all tests
go fmt ./...                           # format
testdata/cref/generate.sh              # regenerate C reference data
```

## Code style

- `gofmt`; imports stdlib first, then external packages.
- Return errors to the caller; `log.Fatal` only in `main`.
- `// FunctionName does X` comments; camelCase variables.
- Standard Go type naming **except** inside `internal/game`, which keeps the C
  identifiers verbatim (see above).
