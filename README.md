# Far Horizons (Go Rewrite)

**Repository:** [github.com/playbymail/fh](https://github.com/playbymail/fh)

This project is a reimplementation of the classic play-by-email (PBEM) game **Far Horizons**, originally written in C.
The new version is written in Go and designed for deterministic, replayable turn processing.

---

## Overview

- Deterministic results via seed-based RNG derived from `{gameID, playerID, turnID}`.
- Parallel, dependency-aware order execution.
- Modular architecture separating world model, orders, reports, and storage.
  - JSON data files by default.
  - Optional in-memory Sqlite datastore for testing.

---

## Acknowledgements

This project was inspired by and references the original *Far Horizons* source code maintained at [github.com/playbymail/Far-Horizons](https://github.com/playbymail/Far-Horizons), created by Bruce A. Holloway and the PBEM community.
All credit for the original design, rules, and world concepts belongs to them.

The rewrite effort aims to preserve gameplay fidelity while modernizing the engine.

---

## Version Information

This project uses [github.com/maloquacious/semver](https://github.com/maloquacious/semver) for version metadata.

```go
import "github.com/maloquacious/semver"

var Version = semver.Version{
    Major: 0,
    Minor: 1,
    Patch: 0,
    PreRelease: "alpha",
}
```

---

## License

This Go version is licensed under the [MIT License](LICENSE).
The original *Far Horizons* C source remains under its original license or public domain status as applicable.
