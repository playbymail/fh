# Far Horizons — Makefile (v0.1-alpha)
# Simple, portable targets with sensible defaults.
# Usage examples:
#   make build
#   make test
#   make clean


.PHONY: all build version test tidy clean golden-rng help

all: build

build:
	mkdir -p dist/local
	go build -o dist/local/fh .

version:
	go run . version

test:
	go test ./...

tidy:
	go mod tidy

golden-rng: build
	dist/local/fh update golden rng

clean:
	rm -rf dist/local dist/linux

help:
	@echo "Targets:"
	@echo "  build             Build binary to dist/local/fh"
	@echo "  version           Run version command"
	@echo "  test              Run all tests"
	@echo "  tidy              Run 'go mod tidy'"
	@echo "  golden-rng        Rebuild golden RNG test files"
	@echo "  clean             Remove $(DIST) directory"
