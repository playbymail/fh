# SQLite Migration System

## Overview

The store package uses a simple, ordered migration system to manage SQLite schema evolution. Migrations are applied sequentially and recorded in the `migrations` table.

## Architecture

### Migration Registry

Migrations are defined in `sqlitestore.go` as an ordered slice:

```go
type migration struct {
    name string
    up   func(*sql.DB) error
}

var migrations = []migration{
    {name: "0001_initial", up: setupSchema},
    {name: "0002_add_indexes", up: migration0002},
    // ...
}
```

### Migration Naming Convention

- Format: `NNNN_description` (e.g., `0001_initial`, `0002_add_colonies`)
- Leading zeros ensure lexicographic ordering matches application order
- Names are stored in the `migrations` table for tracking

### Application Logic

1. **NewSQLiteStore**: Applies all migrations to new database
2. **OpenSQLiteStore**: Checks current version, applies pending migrations if needed
3. **UpgradeSchema**: Finds pending migrations and applies them sequentially

### Schema Version Tracking

```sql
CREATE TABLE migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  applied_at TEXT NOT NULL
);
```

Each migration must insert its own record:
```sql
INSERT OR IGNORE INTO migrations (name, applied_at) 
VALUES ('0002_example', datetime('now'))
```

## Adding New Migrations

### Step 1: Create Migration Function

```go
func migration0002(db *sql.DB) error {
    schema := `
        CREATE TABLE IF NOT EXISTS colony (
            id TEXT PRIMARY KEY,
            planet_id TEXT NOT NULL,
            species_id TEXT NOT NULL,
            population INTEGER NOT NULL,
            FOREIGN KEY (planet_id) REFERENCES entity(id) ON DELETE CASCADE
        );
        
        CREATE INDEX IF NOT EXISTS idx_colony_planet 
        ON colony(planet_id);
    `
    
    if _, err := db.Exec(schema); err != nil {
        return err
    }
    
    // Record migration
    _, err := db.Exec(`
        INSERT OR IGNORE INTO migrations (name, applied_at) 
        VALUES ('0002_add_colony_table', datetime('now'))
    `)
    return err
}
```

### Step 2: Register Migration

Add to the `migrations` slice in order:

```go
var migrations = []migration{
    {name: "0001_initial", up: setupSchema},
    {name: "0002_add_colony_table", up: migration0002},
}
```

### Step 3: Update Expected Version

In `OpenSQLiteStore`, update the expected version:

```go
expected := "0002_add_colony_table"  // Previously "0001_initial"
```

## Migration Best Practices

### Idempotency

Always use `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS`:

```sql
CREATE TABLE IF NOT EXISTS new_table (...);
CREATE INDEX IF NOT EXISTS idx_name ON table(column);
```

Use `INSERT OR IGNORE` for migration records to prevent duplicate application.

### Foreign Keys

- Always specify `ON DELETE CASCADE` or `ON DELETE SET NULL`
- Verify parent tables exist before adding child tables
- Test referential integrity after migration

```sql
FOREIGN KEY (game_id, turn_num) 
REFERENCES turn(game_id, num) ON DELETE CASCADE
```

### Transactions

Migration functions receive `*sql.DB`, not a transaction. If atomicity is required:

```go
func migration0003(db *sql.DB) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Multiple operations...
    
    return tx.Commit()
}
```

### Data Migrations

For data transformations, perform in the same migration:

```go
func migration0004(db *sql.DB) error {
    // 1. Add new column
    if _, err := db.Exec(`ALTER TABLE entity ADD COLUMN version INTEGER DEFAULT 1`); err != nil {
        return err
    }
    
    // 2. Migrate data
    if _, err := db.Exec(`UPDATE entity SET version = 1 WHERE version IS NULL`); err != nil {
        return err
    }
    
    // 3. Record migration
    _, err := db.Exec(`INSERT OR IGNORE INTO migrations (name, applied_at) VALUES ('0004_add_entity_version', datetime('now'))`)
    return err
}
```

### Dropping Columns

SQLite doesn't support `DROP COLUMN` directly. Use table recreation:

```go
func migration0005(db *sql.DB) error {
    schema := `
        -- Create new table without dropped column
        CREATE TABLE entity_new (
            game_id TEXT NOT NULL,
            turn_num INTEGER NOT NULL,
            id TEXT NOT NULL,
            kind TEXT NOT NULL,
            data BLOB NOT NULL,
            PRIMARY KEY (game_id, turn_num, id)
        );
        
        -- Copy data
        INSERT INTO entity_new SELECT game_id, turn_num, id, kind, data FROM entity;
        
        -- Swap tables
        DROP TABLE entity;
        ALTER TABLE entity_new RENAME TO entity;
    `
    
    if _, err := db.Exec(schema); err != nil {
        return err
    }
    
    _, err := db.Exec(`INSERT OR IGNORE INTO migrations (name, applied_at) VALUES ('0005_remove_entity_column', datetime('now'))`)
    return err
}
```

## PRAGMAs

Foreign key enforcement and performance settings are applied via `enablePragmas()`:

```go
PRAGMA foreign_keys = ON        // Enable FK enforcement (required)
PRAGMA journal_mode = WAL       // Write-Ahead Logging for concurrency
PRAGMA busy_timeout = 5000      // Wait 5s for locks
PRAGMA synchronous = NORMAL     // Balance safety/performance
```

These are set per-connection, not per-migration.

## Testing Migrations

### Unit Tests

Test migration application in `sqlitestore_test.go`:

```go
func TestMigration0002(t *testing.T) {
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")
    
    // Create with 0001
    st, _ := NewSQLiteStore(dbPath, false)
    st.Close()
    
    // Apply 0002 by reopening
    st2, err := OpenSQLiteStore(dbPath)
    if err != nil {
        t.Fatalf("migration failed: %v", err)
    }
    defer st2.Close()
    
    // Verify schema changes
    var exists int
    err = st2.db.QueryRow(`
        SELECT 1 FROM sqlite_master 
        WHERE type='table' AND name='colony'
    `).Scan(&exists)
    
    if err != nil || exists != 1 {
        t.Error("colony table not created")
    }
}
```

### Integration Tests

Test end-to-end workflows after migrations:

```go
func TestFullWorkflowAfterMigrations(t *testing.T) {
    st, _ := NewSQLiteStore(t.TempDir()+"/test.db", false)
    defer st.Close()
    
    ctx := context.Background()
    
    // Test all CRUD operations
    if err := st.CreateGame(ctx, "g1", "Test"); err != nil {
        t.Fatal(err)
    }
    // ...
}
```

## Rollback Strategy

This system **does not support automatic rollback**. For production:

1. **Backup before migration**: Copy database file before upgrading
2. **Test migrations**: Use staging environment first
3. **Version compatibility**: Older binaries should detect `ErrSchemaTooNew` and refuse to run

### Manual Rollback

If a migration fails in production:

1. Restore from backup
2. Delete failed migration from registry
3. Fix migration code
4. Redeploy and retry

## Troubleshooting

### "no such table: migrations"

GetSchemaVersion handles this gracefully - returns empty string, triggering full migration application.

### "foreign key constraint failed"

- Verify `PRAGMA foreign_keys = ON` is set
- Check parent tables exist before creating child tables
- Verify FK columns match parent PK types and order

### "UNIQUE constraint failed: migrations.name"

Migration already applied. Check:
- Migration name hasn't changed
- `INSERT OR IGNORE` is used (prevents error but indicates issue)

### Schema version mismatch

`OpenSQLiteStore` returns `ErrSchemaTooNew` if:
- Binary is older than database
- Migration didn't record itself properly

Check migrations table:
```sql
SELECT * FROM migrations ORDER BY id;
```

## Future Enhancements

### Optional: Migration Checksums

For production, add checksum validation:

```go
type migration struct {
    name     string
    checksum string  // SHA256 of up function bytecode
    up       func(*sql.DB) error
}
```

### Optional: Down Migrations

Add rollback capability:

```go
type migration struct {
    name string
    up   func(*sql.DB) error
    down func(*sql.DB) error  // Rollback
}
```

### Optional: External SQL Files

Load migrations from `migrations/*.sql`:

```go
//go:embed migrations/*.sql
var migrationFS embed.FS
```

## References

- SQLite Foreign Keys: https://www.sqlite.org/foreignkeys.html
- SQLite WAL Mode: https://www.sqlite.org/wal.html
- modernc.org/sqlite driver: https://gitlab.com/cznic/sqlite
