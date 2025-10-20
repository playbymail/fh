# Amp Agent: Far Horizons (fh)

We are going to create the game engine and CLI for Far Horizons using Go.

## Objectives
1. Convert the existing game engine (C source) to idiomatic Go.
2. Use Cobra to implement the command line interface.
3. Update the game I/O to read and write JSON data files instead of binary data files.
4. Use the `github.com/maloquacious/semver` for semantic versioning.

## Project Structure
    fh/
    ├── .gitattributes
    ├── .gitignore
    ├── AGENT.md
    ├── dist/                # Build artifacts, one directory per deploy target
    │   ├── linux/           # Linux (production artifacts)
    │   └── local/           # Local (development artifacts)
    ├── far-horizons-source/ # Original C source code
    │   └── CMakeLists.txt   # Manifest for building original C source
    ├── go.mod
    ├── go.work              # Development may use local repositories
    ├── internal/
    │   ├── engine/
    │   └── store/
    ├── LICENSE
    ├── main.go              # entry point for the fhx application
    ├── Makefile
    ├── testdata/            # Directory for testing larger problems
    ├── tmp/                 # temporary directory for testing artifacts
    ├── README.md
    ├── TODO.md
    └── ... (CI/CD configs, etc.)

## Commands
* CLI command:
  * Build CLI: `go build -o dist/local/fhx ./cmd/fhx`
  * Version info: `dist/local/fhx version`
  * Tests: `go test ./...`
  * Format code: `go fmt ./...`
  * Build for Linux: get version then `GOOS=linux GOARCH=amd64 go build -o dist/linux/fhx-${VERSION}`

## Code Style
- Standard Go formatting using `gofmt`
- Imports organized by stdlib first, then external packages
- Error handling: return errors to caller, log.Fatal only in main
- Function comments use Go standard format `// FunctionName does X`
- Variable naming follows camelCase
- File structure follows standard Go package conventions
- Type naming follows standard Go conventions (no special suffixes)
