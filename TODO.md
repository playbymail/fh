# FH - Far Horizons (Go Rewrite)

This document outlines the major deliverables and milestones for rewriting the classic play-by-email game Far Horizons from C to Go. The project aims for a deterministic, replayable engine with modular architecture.

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

## Sprints (Agile, TDD-Driven)

Each sprint follows TDD principles: write tests first, implement minimal code to pass, refactor. Focus on incremental delivery with working software.

### Sprint 1: Foundation (1-2 weeks)
- [x] Scaffold repo with proper Go structure and internal packages
- [x] Set up Cobra CLI framework and implement "version" command
- [x] Integrate semantic versioning with `github.com/maloquacious/semver`
- [x] Set up linting, formatting, and basic testing
- [x] Configure build system (Makefile)

### Sprint 2: Data Stores (1 week)
- [x] Design persistence interfaces for data stores
- [x] Implement file-based JSON store as default
- [x] Prepare foundation for SQLite store (modernc.org/sqlite)

### Sprint 3: Core Types and RNG (1 week)
- [x] Implement deterministic RNG with PCG/Xoroshiro
- [x] Create test vectors for RNG determinism
- [x] Analyze C source and catalog necessary Go types and interfaces

## Future Sprints

### World Model and Movement (2 weeks)
- [ ] Implement world model v0 (Stars, Wormholes, Fleets)
- [ ] Develop order parser subset for movement orders
- [ ] Add movement rules and validation
- [ ] Create batch executor with dependency DAG

### Execution and Effects (2 weeks)
- [ ] Implement effect reducer and conflict resolution
- [ ] Add movement policies and tie-breaking
- [ ] Build world state commits and snapshots
- [ ] Integrate with storage layer

### Reports and CLI (2 weeks)
- [ ] Develop text-based reports
- [ ] Implement CLI commands (run, replay, diff)
- [ ] Add game management and debugging tools
- [ ] Create basic replay functionality

### Advanced Features (3+ weeks)
- [ ] Expand to full order set (mining, production, combat, etc.)
- [ ] Implement diplomacy, tech, and colony mechanics
- [ ] Add HTML/JSON report formats
- [ ] Complete testing suite and validation

### Analyze C Source (1-2 weeks)
- [ ] Inventory mechanics from original Far Horizons C repository
- [ ] Document orders, phases, entity kinds, resources, and edge cases
- [ ] Freeze semantics in markdown specifications
- [ ] Create thin C-state → JSON exporters for seeding worlds

### Polish and Deployment (1-2 weeks)
- [ ] Performance optimization and edge case handling
- [ ] Final documentation and licensing
- [ ] Set up CI/CD pipelines
- [ ] Deployment setup and release artifacts
