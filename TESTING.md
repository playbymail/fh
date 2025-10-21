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
