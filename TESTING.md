# Testing

This document describes how to run tests and update golden test files.

## Running Tests

Run all tests:
```
go test ./...
```

Run tests for a specific package:
```
go test ./internal/engine/rng
```

## Golden Test Files

Some tests use golden files to compare expected output. These files are located in `testdata/` directories.

### Updating Golden Files

To regenerate golden test files, use the CLI command:

```
fh update golden rng
```

This will update the RNG-related golden files in `internal/engine/rng/testdata/`.

Note: Only run this command when the test logic or expected output has changed intentionally.

## SQLite Store Testing

The store package (`internal/data/store`) is backed by SQLite via
`zombiezen.com/go/sqlite` with migrations managed by `sqlitemigration`. Tests
use temporary, file-based databases so each test is isolated.

```go
dbPath := filepath.Join(t.TempDir(), "test.db")
st, err := NewSQLiteStore(dbPath, false)
```

Key points:

- **Schema version** is the count of applied migrations, tracked by
  `PRAGMA user_version`. `GetSchemaVersion` returns it as a string. Assert
  against `strconv.Itoa(len(appSchema.Migrations))` so tests stay correct as
  migrations are appended.
- **Raw connection access** in tests goes through the pool, not a `*sql.DB`:

  ```go
  conn, err := st.pool.Take(ctx)
  if err != nil {
      t.Fatal(err)
  }
  err = sqlitex.Execute(conn, "DELETE FROM game WHERE id = ?",
      &sqlitex.ExecOptions{Args: []any{"game1"}})
  st.pool.Put(conn)
  ```

- **Foreign keys / cascade** are enforced (`PRAGMA foreign_keys = ON` is set in
  the pool's `PrepareConn` hook), so cascade-delete behavior is testable.
- **Migration compatibility**: migrations use `CREATE TABLE IF NOT EXISTS`, so a
  database created by older schema code (left at `user_version = 0`) migrates
  cleanly on open. `appSchema.AppID` is intentionally `0` to keep such databases
  openable.

See `internal/data/store/MIGRATIONS.md` for how to add and test new migrations.

## JSON Compatibility Testing

We validate our Go implementation against the original C version using golden file testing with shared JSON files.

### C JSON Implementation Quirks

The C version uses cJSON library with custom helpers (`Far-Horizons/src/cjson/`). Key behaviors to account for:

#### Parsing (Input)
- **Case-sensitive field names**: Uses `cJSON_GetObjectItemCaseSensitive()`
- **Optional fields**: Missing fields return NULL, no error (e.g., `experimental` object)
- **Unknown fields ignored**: Extra JSON properties are silently skipped
- **Type checking**: Validates types (string, number, boolean) at parse time

#### Writing (Output)
- **Pretty-printed format**: Uses `cJSON_Print()` which produces indented JSON with 4 spaces
- **Trailing newline**: All output files end with `\n`
- **Number formatting**: Stored as doubles, integers may appear as `1.0` instead of `1`
- **All fields present**: C writes all struct fields, including zeros and NULLs (no `omitempty`)

### Go Implementation Considerations

When comparing Go output to C golden files:

1. **Use `json.MarshalIndent()`** with 4-space indentation, not compact `json.Marshal()`
2. **Avoid `omitempty` tags** when generating output for comparison (C doesn't omit zero values)
3. **Field name casing** must match exactly (e.g., `"govt-name"` not `"govtName"`)
4. **Add trailing newline** when writing JSON files

### Comparison Strategies

For golden file tests:

- **Semantic comparison** (recommended): Unmarshal both JSON files and compare structures
- **jq normalization** (simple): Use `jq` to normalize before text comparison
- **Normalized text comparison**: Strip/normalize whitespace before byte comparison
- **Avoid byte-for-byte comparison**: Different JSON libraries format numbers/whitespace differently

### Using jq for Normalization

Normalize both C and Go JSON output with `jq` before comparison:

```bash
# Normalize C output (strips whitespace, sorts keys, consistent formatting)
jq -S . c-output.json > c-normalized.json

# Normalize Go output
jq -S . go-output.json > go-normalized.json

# Compare normalized files
diff c-normalized.json go-normalized.json
```

The `-S` flag sorts object keys alphabetically, and `jq` normalizes number formatting and whitespace.

### Example Test Pattern (Go)

```go
// Load C-generated golden file
cData, _ := os.ReadFile("testdata/species.golden.json")
var cSpecies []*SpeciesConfig
json.Unmarshal(cData, &cSpecies)

// Load with Go implementation
goSpecies, _ := config.LoadSpecies("testdata/species.json")

// Semantic comparison
if !reflect.DeepEqual(cSpecies, goSpecies) {
    t.Errorf("species mismatch")
}
```

### Example Test Pattern (Shell)

```bash
# In test script
jq -S . testdata/c-golden.json > /tmp/c-norm.json
jq -S . testdata/go-output.json > /tmp/go-norm.json
diff -u /tmp/c-norm.json /tmp/go-norm.json || exit 1
```
