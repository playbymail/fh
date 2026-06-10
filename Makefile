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


.PHONY: all build build-fh build-fhc version test tidy clean golden-rng help

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
	@echo "  tidy              Run 'go mod tidy'"
	@echo "  golden-rng        Rebuild golden RNG test files"
	@echo "  clean             Remove dist/local and dist/linux directories"
