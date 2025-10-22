# Amp Agent: Far Horizons (fh)

We are going to create the game engine and CLI for Far Horizons using Go.

Note that the ../Far-Horizons/ folder contains a clone of the github.com/playbymail/Far-Horizons repository.

## Objectives
1. Convert the existing game engine (C source) to idiomatic Go.
2. Use Cobra to implement the command line interface.
3. Update the game I/O to read and write JSON data files instead of binary data files.
4. Use the `github.com/maloquacious/semver` for semantic versioning.


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
