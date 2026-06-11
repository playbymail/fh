package store

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/playbymail/fh/internal/model"
)

// BinaryStore persists a game as the classic Far Horizons binary .dat files in
// a directory: galaxy.dat, stars.dat, planets.dat, one sp%02d.dat per species
// (a species record followed by its nampla and ship records), and
// locations.dat. Per-species event logs ride alongside as sp%02d.log sidecars,
// matching fhc (the binary species record does not carry the log).
//
// The store is bound to a single game directory; the gameID arguments on the
// Store interface are accepted for compatibility and otherwise unused.
type BinaryStore struct {
	dir string
}

// newBinaryStore binds a BinaryStore to a game directory.
func newBinaryStore(dir string) *BinaryStore { return &BinaryStore{dir: dir} }

// Close releases resources (none for a file-backed store).
func (b *BinaryStore) Close() error { return nil }

// IngestWorld writes the complete World as binary .dat files, replacing any
// prior state in the directory.
func (b *BinaryStore) IngestWorld(_ context.Context, _ string, w *model.World) error {
	w.Resolve() // ensure nampla -> planet links for planet_index computation

	if err := os.WriteFile(b.path("galaxy.dat"), encodeGalaxy(w.Galaxy), 0o644); err != nil {
		return err
	}

	// stars.dat references a flat planets array via (planet_index, num_planets);
	// build that flat array as we encode the stars.
	var flat []*model.Planet
	planetIndex := make(map[*model.Planet]int)
	starsBuf := make([]byte, 4+len(w.Systems)*binStarSize)
	binary.LittleEndian.PutUint32(starsBuf[0:], uint32(int32(len(w.Systems))))
	for i, s := range w.Systems {
		pi := len(flat)
		for _, p := range s.Planets {
			planetIndex[p] = len(flat)
			flat = append(flat, p)
		}
		encodeStar(starsBuf[4+i*binStarSize:], s, pi, len(s.Planets))
	}
	if err := os.WriteFile(b.path("stars.dat"), starsBuf, 0o644); err != nil {
		return err
	}

	planetsBuf := make([]byte, 4+len(flat)*binPlanetSize)
	binary.LittleEndian.PutUint32(planetsBuf[0:], uint32(int32(len(flat))))
	for i, p := range flat {
		encodePlanet(planetsBuf[4+i*binPlanetSize:], p)
	}
	if err := os.WriteFile(b.path("planets.dat"), planetsBuf, 0o644); err != nil {
		return err
	}

	for _, sp := range w.Species {
		buf := make([]byte, binSpeciesSize+len(sp.Namplas)*binNamplaSize+len(sp.Ships)*binShipSize)
		encodeSpecies(buf[0:], sp)
		off := binSpeciesSize
		for _, np := range sp.Namplas {
			encodeNampla(buf[off:], np, planetIndex[np.Planet])
			off += binNamplaSize
		}
		for _, sh := range sp.Ships {
			encodeShip(buf[off:], sh)
			off += binShipSize
		}
		if err := os.WriteFile(b.path(fmt.Sprintf("sp%02d.dat", sp.ID)), buf, 0o644); err != nil {
			return err
		}
		logPath := b.path(fmt.Sprintf("sp%02d.log", sp.ID))
		if len(sp.Log) > 0 {
			if err := os.WriteFile(logPath, sp.Log, 0o644); err != nil {
				return err
			}
		} else {
			_ = os.Remove(logPath)
		}
	}

	return b.writeLocations(w.Locations)
}

// LoadWorld reads the binary .dat files back into a model.World.
func (b *BinaryStore) LoadWorld(_ context.Context, _ string) (*model.World, error) {
	w := &model.World{}

	gdata, err := b.readRecords("galaxy.dat", binGalaxySize, 1)
	if err != nil {
		return nil, err
	}
	w.Galaxy = decodeGalaxy(gdata)

	stars, err := os.ReadFile(b.path("stars.dat"))
	if err != nil {
		return nil, err
	}
	numStars, err := recordCount("stars.dat", stars, binStarSize)
	if err != nil {
		return nil, err
	}
	type starMeta struct {
		sys         *model.System
		planetIndex int
		numPlanets  int
	}
	metas := make([]starMeta, 0, numStars)
	for i := 0; i < numStars; i++ {
		sys, pi, npl := decodeStar(stars[4+i*binStarSize:])
		w.Systems = append(w.Systems, sys)
		metas = append(metas, starMeta{sys, pi, npl})
	}

	planets, err := os.ReadFile(b.path("planets.dat"))
	if err != nil {
		return nil, err
	}
	numPlanets, err := recordCount("planets.dat", planets, binPlanetSize)
	if err != nil {
		return nil, err
	}
	flat := make([]*model.Planet, numPlanets)
	for i := 0; i < numPlanets; i++ {
		flat[i] = decodePlanet(planets[4+i*binPlanetSize:])
	}
	for _, m := range metas {
		for k := 0; k < m.numPlanets; k++ {
			idx := m.planetIndex + k
			if idx < 0 || idx >= len(flat) {
				return nil, fmt.Errorf("stars.dat: planet index %d out of range", idx)
			}
			p := flat[idx]
			p.System = m.sys
			p.Orbit = k + 1
			m.sys.Planets = append(m.sys.Planets, p)
		}
	}

	for n := 1; n <= w.Galaxy.NumSpecies; n++ {
		data, err := os.ReadFile(b.path(fmt.Sprintf("sp%02d.dat", n)))
		if os.IsNotExist(err) {
			continue // extinct species: no data file
		}
		if err != nil {
			return nil, err
		}
		if len(data) < binSpeciesSize {
			return nil, fmt.Errorf("sp%02d.dat: short species record", n)
		}
		sp, numNamplas, numShips := decodeSpecies(data)
		sp.ID = n
		off := binSpeciesSize
		for k := 0; k < numNamplas; k++ {
			if off+binNamplaSize > len(data) {
				return nil, fmt.Errorf("sp%02d.dat: truncated nampla %d", n, k)
			}
			sp.Namplas = append(sp.Namplas, decodeNampla(data[off:]))
			off += binNamplaSize
		}
		for k := 0; k < numShips; k++ {
			if off+binShipSize > len(data) {
				return nil, fmt.Errorf("sp%02d.dat: truncated ship %d", n, k)
			}
			sp.Ships = append(sp.Ships, decodeShip(data[off:]))
			off += binShipSize
		}
		logBytes, err := os.ReadFile(b.path(fmt.Sprintf("sp%02d.log", n)))
		if err == nil {
			sp.Log = logBytes
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		w.Species = append(w.Species, sp)
	}

	if locs, err := os.ReadFile(b.path("locations.dat")); err == nil {
		for i := 0; i+binLocationSize <= len(locs); i += binLocationSize {
			w.Locations = append(w.Locations, model.Location{
				S: int(locs[i]), X: int(locs[i+1]), Y: int(locs[i+2]), Z: int(locs[i+3]),
			})
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	w.Resolve()
	return w, nil
}

func (b *BinaryStore) writeLocations(locs []model.Location) error {
	path := b.path("locations.dat")
	if len(locs) == 0 {
		_ = os.Remove(path)
		return nil
	}
	buf := make([]byte, 0, len(locs)*binLocationSize)
	for _, l := range locs {
		buf = append(buf, byte(l.S), byte(l.X), byte(l.Y), byte(l.Z))
	}
	return os.WriteFile(path, buf, 0o644)
}

func (b *BinaryStore) path(name string) string { return filepath.Join(b.dir, name) }

// readRecords reads a headerless fixed-size record file and returns its bytes
// after verifying it holds at least want records.
func (b *BinaryStore) readRecords(name string, recSize, want int) ([]byte, error) {
	data, err := os.ReadFile(b.path(name))
	if err != nil {
		return nil, err
	}
	if len(data) < recSize*want {
		return nil, fmt.Errorf("%s: short file (%d bytes)", name, len(data))
	}
	return data, nil
}

// recordCount reads the 4-byte little-endian record count header and validates
// the file is large enough to hold that many fixed-size records.
func recordCount(name string, data []byte, recSize int) (int, error) {
	if len(data) < 4 {
		return 0, fmt.Errorf("%s: missing record count header", name)
	}
	n := int(int32(binary.LittleEndian.Uint32(data[0:])))
	if n < 0 || len(data) < 4+n*recSize {
		return 0, fmt.Errorf("%s: header claims %d records but file is %d bytes", name, n, len(data))
	}
	return n, nil
}
