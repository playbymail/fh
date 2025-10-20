# Far Horizons (Go Rewrite) - TODO

This document outlines the major deliverables and milestones for rewriting the classic play-by-email game Far Horizons from C to Go. The project aims for a deterministic, replayable engine with modular architecture.

## Broad Deliverables

### 1. Project Setup and Structure
- [ ] Establish Go project structure per INTENTIONS.md layout
- [ ] Set up Cobra CLI framework for command-line interface
- [ ] Implement "version" command as proof of concept for project structure
- [ ] Integrate semantic versioning with `github.com/maloquacious/semver`
- [ ] Set up linting, formatting, and testing infrastructure
- [ ] Configure build system (Makefile)

### 2. Data Storage Interfaces
- [ ] Design persistence interfaces for data stores
- [ ] Implement file-based JSON store as default
- [ ] Prepare foundation for SQLite store (modernc.org/sqlite)

### 3. Core Engine Development
- [ ] Define Go types and interfaces (world, orders, RNG, schedule, effects)
- [ ] Implement deterministic RNG factory (PCG/Xoroshiro with HMAC-SHA256 seeding)
- [ ] Develop order parsing, normalization, and validation
- [ ] Build world model (stars, wormholes, fleets, colonies, etc.)
- [ ] Implement dependency-aware order execution with parallel batches
- [ ] Create conflict resolution policies (movement, mining, combat, etc.)
- [ ] Develop effect merging and world state commits

### 4. Data Storage and Persistence
- [ ] Design JSON-based data file format for default storage
- [ ] Implement SQLite schema for optional advanced storage
- [ ] Create migration system (goose or atlas)
- [ ] Build persistence interfaces and store implementations

### 5. CLI and User Interface
- [ ] Implement core CLI commands (version, run, replay, diff)
- [ ] Add game management commands (create, info)
- [ ] Develop order submission and report retrieval
- [ ] Create debugging and administration tools

### 6. Reporting and Output
- [ ] Implement report builders (text, HTML, JSON formats)
- [ ] Add replay diffs and audit logs
- [ ] Develop per-player views and GM snapshots

### 7. Testing and Validation
- [ ] Write golden tests for orders → effects → world diffs
- [ ] Implement property tests with random seeds and invariant checks
- [ ] Create replay tests for deterministic checksums

### 8. Analyze Original C Source
- [ ] Inventory mechanics from original Far Horizons C repository
- [ ] Document orders, phases, entity kinds, resources, and edge cases
- [ ] Freeze semantics in markdown specifications (acceptance tests)
- [ ] Create thin C-state → JSON exporters for seeding initial worlds

### 9. Documentation and Deployment
- [ ] Update README.md and create user documentation
- [ ] Set up deployment artifacts (Linux binaries, containers)
- [ ] Handle licensing (MIT for new code, original FH license reference)

## Sprints (Agile, TDD-Driven)

Each sprint follows TDD principles: write tests first, implement minimal code to pass, refactor. Focus on incremental delivery with working software.

### Sprint 1: Foundation (1-2 weeks)
- [ ] Scaffold repo with proper Go structure and internal packages
- [ ] Set up Cobra CLI framework and implement "version" command
- [ ] Integrate semantic versioning with `github.com/maloquacious/semver`
- [ ] Set up linting, formatting, and basic testing
- [ ] Configure build system (Makefile)

### Sprint 2: Data Stores (1 week)
- [ ] Design persistence interfaces for data stores
- [ ] Implement file-based JSON store as default
- [ ] Prepare foundation for SQLite store (modernc.org/sqlite)

### Sprint 3: Core Types and RNG (1 week)
- [ ] Define core interfaces (world, orders, RNG, schedule)
- [ ] Implement deterministic RNG with PCG/Xoroshiro
- [ ] Create test vectors for RNG determinism
- [ ] Set up basic order parsing framework

### Sprint 4: World Model and Movement (2 weeks)
- [ ] Implement world model v0 (Stars, Wormholes, Fleets)
- [ ] Develop order parser subset for movement orders
- [ ] Add movement rules and validation
- [ ] Create batch executor with dependency DAG

### Sprint 5: Execution and Effects (2 weeks)
- [ ] Implement effect reducer and conflict resolution
- [ ] Add movement policies and tie-breaking
- [ ] Build world state commits and snapshots
- [ ] Integrate with storage layer

### Sprint 6: Reports and CLI (2 weeks)
- [ ] Develop text-based reports
- [ ] Implement CLI commands (run, replay, diff)
- [ ] Add game management and debugging tools
- [ ] Create basic replay functionality

### Sprint 7: Advanced Features (3+ weeks)
- [ ] Expand to full order set (mining, production, combat, etc.)
- [ ] Implement diplomacy, tech, and colony mechanics
- [ ] Add HTML/JSON report formats
- [ ] Complete testing suite and validation

### Sprint 8: Analyze C Source (1-2 weeks)
- [ ] Inventory mechanics from original Far Horizons C repository
- [ ] Document orders, phases, entity kinds, resources, and edge cases
- [ ] Freeze semantics in markdown specifications
- [ ] Create thin C-state → JSON exporters for seeding worlds

### Sprint 9: Polish and Deployment (1-2 weeks)
- [ ] Performance optimization and edge case handling
- [ ] Final documentation and licensing
- [ ] Set up CI/CD pipelines
- [ ] Deployment setup and release artifacts

## Dependencies and Risks
- Need thorough understanding of original C mechanics before implementation
- Deterministic RNG and parallel execution require careful design
- JSON vs. SQLite storage options may affect performance
- Cobra CLI integration should be straightforward
- Semantic versioning library is specified

## Completion Criteria
- All original game mechanics preserved in Go implementation
- Deterministic turn processing with byte-for-byte replays
- Clean, idiomatic Go code with comprehensive tests
- Functional CLI for game management and turn execution
- Documentation sufficient for new contributors
