# Amp Agent: Far Horizons (fh)

We are going to create the game engine and CLI for Far Horizons using Go.

Note that the ../Far-Horizons/ folder contains a clone of the github.com/playbymail/Far-Horizons repository.

## Objectives
1. Convert the existing game engine (C source) to idiomatic Go.
2. Use `github.com/peterbourgon/ff/v4` to implement the command line interface.
3. Store game state in a SQLite datastore (replacing the original binary data files); use JSON for config input and test/export snapshots.
4. Use the `github.com/maloquacious/semver` for semantic versioning.

## Data Store
The SQLite store lives in `internal/data/store` behind the `Store` interface.

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
* CLI command:
  * Build CLI: `go build -o dist/local/fh .`
  * Version info: `dist/local/fh version`
  * Tests: `go test ./...`
  * Format code: `go fmt ./...`
  * Build for Linux: get version then `GOOS=linux GOARCH=amd64 go build -o dist/linux/fh-${VERSION} .`

## Code Style
- Standard Go formatting using `gofmt`
- Imports organized by stdlib first, then external packages
- Error handling: return errors to caller, log.Fatal only in main
- Function comments use Go standard format `// FunctionName does X`
- Variable naming follows camelCase
- File structure follows standard Go package conventions
- Type naming follows standard Go conventions (no special suffixes)
