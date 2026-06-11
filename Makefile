# Far Horizons — Makefile (v0.1-alpha)
# Simple, portable targets with sensible defaults.
# Usage examples:
#   make build
#   make test
#   make clean
#
# Two binaries:
#   fh  — idiomatic Go runner (ff/v4 CLI + ZombieZen SQLite store)
#   fhc — byte-faithful C-port runner (internal/game CommandRunner)


.PHONY: all build build-fh build-fhc version test test-golden golden-ref tidy clean golden-rng help

all: build

build: build-fh build-fhc

build-fh:
	mkdir -p dist/local
	go build -o dist/local/fh ./cmd/fh

build-fhc:
	mkdir -p dist/local
	go build -o dist/local/fhc ./cmd/fhc

version:
	go run ./cmd/fh version

test:
	go test ./...

# Run the C-parity golden tests (setup + turns 1-4 pipeline). These compare
# the Go port's output byte-for-byte against the C reference data in
# testdata/cref; they skip automatically if that data has not been
# generated (see golden-ref).
test-golden:
	go test ./internal/game/ -run MatchesC -v

# Regenerate the C-engine reference data the golden tests compare against.
# Requires the C engine built at ../Far-Horizons/build/fh.
golden-ref:
	sh testdata/cref/generate.sh

tidy:
	go mod tidy

golden-rng: build-fh
	dist/local/fh update golden rng

clean:
	rm -rf dist/local dist/linux

help:
	@echo "Targets:"
	@echo "  build             Build both binaries to dist/local/{fh,fhc}"
	@echo "  build-fh          Build idiomatic Go runner to dist/local/fh"
	@echo "  build-fhc         Build byte-faithful C-port runner to dist/local/fhc"
	@echo "  version           Run version command"
	@echo "  test              Run all tests"
	@echo "  test-golden       Run the C-parity golden tests (setup + turns 1-4)"
	@echo "  golden-ref        Regenerate C reference data (needs C engine)"
	@echo "  tidy              Run 'go mod tidy'"
	@echo "  golden-rng        Rebuild golden RNG test files"
	@echo "  clean             Remove dist/local and dist/linux directories"
