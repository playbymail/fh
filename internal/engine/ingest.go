package engine

// Ingest builds an in-memory model.World from the structured state the
// byte-faithful fhc engine emits: its `export json` output (galaxy.json,
// systems.json, species.NNN.json) plus the per-species event logs (spNN.log)
// and the locations index (locations.dat).
//
// This is a one-way bridge that lets fh validate its report path against fhc
// without first reproducing galaxy generation. The parsing lives in
// internal/data/fhjson so the JSON store backend can share it.

import (
	"github.com/playbymail/fh/internal/data/fhjson"
	"github.com/playbymail/fh/internal/model"
)

// LoadWorldFromDir reads an fhc state directory into a model.World.
func LoadWorldFromDir(dir string) (*model.World, error) {
	return fhjson.ReadWorld(dir)
}
