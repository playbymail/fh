# Agent Guide: Far Horizons (fh)

We are porting the classic play-by-mail game Far Horizons from C to Go.

Note that `../Far-Horizons/` contains a clone of the github.com/playbymail/Far-Horizons
repository — the authoritative C source. Always consult it when porting. We are
porting **Far Horizons v7.5.11** (the `v7.5.11` tag); the golden reference data
and parity tests are generated from that release. When updating the C clone,
stay on v7.5.11 unless the maintainer bumps the target version here.

> The canonical, detailed guidance lives in **CLAUDE.md**. This file is the
> short Amp-facing summary; CLAUDE.md and this file must agree.

## Strategy: two phases, two binaries

We are doing this in two phases, in order:

1. **Byte-faithful C port first** (`internal/game`, run by `cmd/fhc`).
   A direct, line-by-line translation of the C engine that reproduces the
   original binary `.dat` files, s-expression `.txt` exports, logs, and reports
   **byte-for-byte**. This is the reference implementation we trust.

2. **Idiomatic Go second** (`internal/...` clean packages, run by `cmd/fh`).
   A clean rewrite that stores game state in a SQLite datastore (ZombieZen)
   instead of the original binary files, using JSON for config input and
   test/export snapshots. This is built and validated *against* the
   byte-faithful port — only start a subsystem here once its `fhc` counterpart
   is correct.

Do not blur the two. Work on the byte-faithful port until it is complete and
validated; the idiomatic package consumes its verified behavior.

### Binaries
- `cmd/fhc` — the **C port** runner. `main` calls `game.CommandRunner(os.Args)`,
  a direct port of the C `fh.c` argument dispatcher in `internal/game`.
- `cmd/fh` — the **idiomatic Go** runner. Uses `peterbourgon/ff/v4` for the CLI
  and `internal/data/store` (ZombieZen SQLite) for persistence. Many subcommands
  are still stubs (`cerrs.ErrNotImplemented`).

## The byte-faithful port (`internal/game`)

- Translates the C engine file-for-file (e.g. `combat.c` → `combat1.go` +
  `combat2.go`; `do.c`/`do_*.c` → `do1.go`/`do2.go`/`do3.go`; the C `*vars.c`
  globals collapse into `vars.go`).
- Keeps the original on-disk formats: binary `.dat` records (`marshal.go`,
  `unmarshal.go`, and the `*io.go` codecs) and the s-expression text exports
  (`*DataAsSExpr`). **These `*AsSExpr` writers are load-bearing** — the C engine
  writes `galaxy.hs.txt`, `stars.hs.txt`, `planets.hs.txt`, `species%03d.txt`,
  so the Go port must too. Do not remove them.
- The `sexpr` debug command (the Lisp/s-expression *interpreter* from the C
  `sexpr.c`) is intentionally **not ported** — it is a debugging tool, made
  redundant by direct `.dat`/JSON comparison.
- Tests mutate package-level globals and must **not** run in parallel; call
  `ResetState()` first where needed.

### Validating against C
- `testdata/cref/generate.sh` builds and runs the C engine (seeded with
  `FH_SEED`) to emit reference `.dat`/`.txt`/`.log` outputs under
  `testdata/cref/`. The Go port must produce byte-identical results.
- Existing golden tests cover the **setup** phase (`galaxy_create_test.go`,
  `species_create_test.go`, `templates_create_test.go`) plus binary record-size
  and round-trip checks (`io_roundtrip_test.go`). Extending coverage to the
  **turn pipeline** (locations → combat → pre-departure → jump → production →
  post-arrival → finish → report) is ongoing work.

## The idiomatic data store (`internal/data/store`)

The SQLite store lives behind the `Store` interface.

- **Driver**: `zombiezen.com/go/sqlite` (CGo-free, builds with `CGO_ENABLED=0`).
  We deliberately do **not** use `database/sql`; `modernc.org/sqlite` remains
  only as an indirect dependency of zombiezen.
- **Connections**: access goes through a `*sqlitemigration.Pool`. Every method
  must `conn, err := s.pool.Take(ctx)` and `defer s.pool.Put(conn)`. A leaked
  connection deadlocks the pool.
- **Queries**: use `sqlitex.Execute` / `sqlitex.ExecuteScript` with
  `ExecOptions.Args` for binding and `ResultFunc` for row scanning. Wrap
  multi-statement writes in `sqlitex.Transaction(conn)`.
- **Pragmas**: `foreign_keys`, `busy_timeout`, and `synchronous` are set per
  connection in the pool's `PrepareConn` hook; WAL is enabled via the
  `sqlite.OpenWAL` flag.
- **Migrations**: managed by `zombiezen.com/go/sqlite/sqlitemigration`. The
  schema is the append-only `appSchema.Migrations []string` slice; version is
  tracked by `PRAGMA user_version`. Never edit or reorder a released migration —
  only append. See `internal/data/store/MIGRATIONS.md`.

## Commands
- Build C-port runner:    `go build -o dist/local/fhc ./cmd/fhc`
- Build idiomatic runner: `go build -o dist/local/fh  ./cmd/fh`
- Tests:        `go test ./...`
- Format code:  `go fmt ./...`
- Regenerate C reference data: `testdata/cref/generate.sh`

## Code Style
- Standard Go formatting via `gofmt`; imports stdlib first, then external.
- Error handling: return errors to the caller; `log.Fatal` only in `main`.
- Function comments use `// FunctionName does X`.
- camelCase variables; standard Go type naming (no special suffixes) — except
  inside `internal/game`, where C struct/field names are kept verbatim (e.g.
  `species_data_t`, `nampla_base`) to make the port auditable against the C.
