# FH - Far Horizons (Go Rewrite)

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

