# SQLite Migration System

> **Scope: SQLite store only.** This document describes schema migrations for
> the SQLite (`sqlite`) data store backend exclusively. The `binary` (.dat) and
> `json` data stores serialize `model.World` to fixed, well-defined file formats
> and have **no migration system** — nothing here applies to them. If we are
> ever required to version or migrate the binary or JSON formats, we'll add
> their own documentation (and update this note).

## Overview

The store package uses [`zombiezen.com/go/sqlite/sqlitemigration`](https://pkg.go.dev/zombiezen.com/go/sqlite/sqlitemigration)
to manage SQLite schema evolution. Migrations are an ordered list of SQL
scripts. The number of applied migrations is tracked by the database's
`PRAGMA user_version`, so there is no separate `migrations` table to maintain.

`sqlitemigration.Pool` guarantees that all pending migrations have run
successfully before it hands out a connection to the application.

## Architecture

### Schema Definition

Migrations are defined in `sqlitestore.go` as a `sqlitemigration.Schema`:

```go
var appSchema = sqlitemigration.Schema{
    Migrations: []string{
        `CREATE TABLE IF NOT EXISTS game (...); ...`, // migration 1 -> user_version 1
        // append new migrations here; never edit or reorder released entries
    },
}
```

Each entry is a SQL script that may contain multiple statements.
`sqlitemigration` wraps each script in its own transaction (and appends
`PRAGMA user_version = N`), rolling back on any error.

### Version Tracking

- The applied version is the integer `PRAGMA user_version`.
- After applying `appSchema.Migrations[0]`, `user_version` becomes `1`.
- `GetSchemaVersion` returns this integer as a string (e.g. `"1"`).
- On open, `sqlitemigration` runs every migration whose index is `>= user_version`.

### Application Logic

Both `NewSQLiteStore` and `OpenSQLiteStore` call `openPool`, which constructs a
`sqlitemigration.Pool` and then takes (and immediately returns) a connection to
force migrations to complete and surface any error at construction time.

- `NewSQLiteStore`: enforces file existence/`--force` semantics, then opens the
  pool with `OpenCreate`.
- `OpenSQLiteStore`: requires the file to exist, then opens the pool without
  `OpenCreate`.
- `UpgradeSchema`: migrations are applied automatically by the pool, so this is
  effectively a readiness check that takes and returns a connection.

## Adding New Migrations

### Step 1: Append a Migration Script

Add a new string to the end of `appSchema.Migrations`. Never edit or reorder
existing entries — they have already been applied to live databases and their
position defines the `user_version` they correspond to.

```go
var appSchema = sqlitemigration.Schema{
    Migrations: []string{
        `CREATE TABLE IF NOT EXISTS game (...); ...`, // existing, do not touch
        `CREATE TABLE IF NOT EXISTS colony (
            id TEXT PRIMARY KEY,
            planet_id TEXT NOT NULL,
            species_id TEXT NOT NULL,
            population INTEGER NOT NULL,
            FOREIGN KEY (planet_id) REFERENCES entity(id) ON DELETE CASCADE
        );
        CREATE INDEX IF NOT EXISTS idx_colony_planet ON colony(planet_id);`,
    },
}
```

That is the entire change. Do **not** write `PRAGMA user_version` yourself;
`sqlitemigration` sets it after each migration's transaction.

### Step 2: Update Tests

Tests that assert on the schema version compare against
`strconv.Itoa(len(appSchema.Migrations))`, so they update automatically.

## Migration Best Practices

### Idempotency

Use `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS`. This is what
allows databases created by the previous hand-rolled schema (which left
`user_version = 0`) to migrate cleanly: migration 1 re-runs harmlessly and
`user_version` is advanced to `1`.

### Foreign Keys

- Always specify `ON DELETE CASCADE` or `ON DELETE SET NULL`.
- If a migration must temporarily disable foreign key enforcement (for example,
  a table rebuild), set `MigrationOptions[i].DisableForeignKeys = true` for that
  migration; `sqlitemigration` toggles the PRAGMA around the migration's
  transaction.

```go
var appSchema = sqlitemigration.Schema{
    Migrations: []string{ /* ... */ },
    MigrationOptions: []*sqlitemigration.MigrationOptions{
        nil, // migration 1
        {DisableForeignKeys: true}, // migration 2
    },
}
```

### Transactions

Each migration script is already wrapped in a transaction by
`sqlitemigration`, so multiple statements in one script are atomic.

### Dropping Columns

SQLite doesn't support `DROP COLUMN` on older versions. Use table recreation
inside a single migration script (rename, create, copy, drop), and set
`DisableForeignKeys` for that migration if needed.

## PRAGMAs

Per-connection settings are applied via the pool's `PrepareConn` hook
(`prepareConn`):

```go
PRAGMA foreign_keys = ON        // Enable FK enforcement (required)
PRAGMA busy_timeout = 5000      // Wait 5s for locks
PRAGMA synchronous = NORMAL     // Balance safety/performance
```

WAL journal mode is enabled via the `sqlite.OpenWAL` flag passed to the pool,
not a PRAGMA.

## Testing Migrations

Create a store, close it, reopen it, and assert the schema version:

```go
func TestSchemaUpgrade(t *testing.T) {
    dbPath := filepath.Join(t.TempDir(), "test.db")

    st, _ := NewSQLiteStore(dbPath, false)
    st.Close()

    st2, err := OpenSQLiteStore(dbPath)
    if err != nil {
        t.Fatalf("reopen: %v", err)
    }
    defer st2.Close()

    version, _ := st2.GetSchemaVersion(context.Background())
    want := strconv.Itoa(len(appSchema.Migrations))
    if version != want {
        t.Errorf("expected version %q, got %q", want, version)
    }
}
```

To inspect schema objects directly, take a connection from the pool and run a
`sqlitex.Execute` query against `sqlite_master`.

## Application ID

`appSchema.AppID` is intentionally left unset (`0`). Setting a non-zero AppID
would cause `sqlitemigration` to reject databases created before the AppID was
introduced (their `application_id` is `0`). Leaving it `0` keeps existing
databases openable.

## Rollback Strategy

`sqlitemigration` does **not** support automatic down-migrations. For
production:

1. **Backup before migration**: copy the database file before upgrading.
2. **Test migrations**: use a staging environment first.
3. **Forward-only**: write a new migration to undo a change rather than editing
   history.

## Troubleshooting

### `database application_id = 0x0 (expected 0x...)`

A non-zero `AppID` was set against a database that predates it. Keep `AppID`
unset, or migrate the database's `application_id` separately.

### `foreign key constraint failed`

- Verify `PRAGMA foreign_keys = ON` is set (it is, via `prepareConn`).
- Ensure parent tables/rows exist before child rows.
- For table rebuilds, set `MigrationOptions[i].DisableForeignKeys = true`.

### Migration appears not to run

`sqlitemigration` only runs migrations whose index is `>= user_version`. If a
database has a higher `user_version` than `len(appSchema.Migrations)`, it was
created by a newer binary; opening it is safe but no migrations run.

## References

- sqlitemigration package: https://pkg.go.dev/zombiezen.com/go/sqlite/sqlitemigration
- SQLite Foreign Keys: https://www.sqlite.org/foreignkeys.html
- SQLite WAL Mode: https://www.sqlite.org/wal.html
- zombiezen.com/go/sqlite driver: https://github.com/zombiezen/go-sqlite
