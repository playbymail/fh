package store

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestBinaryByteParity proves the binary codec reproduces fhc's .dat files
// exactly: it decodes the C engine's reference turn-1 binary files into a World
// and re-encodes them, then asserts every .dat file is byte-identical to the
// original. Re-encoding what we decoded must reproduce the source bytes.
//
// The reference data is git-ignored; the test skips when it is absent (run
// testdata/cref/generate.sh to produce it).
func TestBinaryByteParity(t *testing.T) {
	ctx := context.Background()
	ref, err := filepath.Abs(filepath.Join("../../../testdata/cref/turn1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ref, "galaxy.dat")); err != nil {
		t.Skipf("C reference data not available in %s: %v (run testdata/cref/generate.sh)", ref, err)
	}

	src := newBinaryStore(ref)
	w, err := src.LoadWorld(ctx, "")
	if err != nil {
		t.Fatalf("load reference world: %v", err)
	}

	outDir := t.TempDir()
	out := newBinaryStore(outDir)
	if err := out.IngestWorld(ctx, "", w); err != nil {
		t.Fatalf("re-encode world: %v", err)
	}

	files := []string{"galaxy.dat", "stars.dat", "planets.dat", "locations.dat"}
	for n := 1; n <= w.Galaxy.NumSpecies; n++ {
		files = append(files, fmt.Sprintf("sp%02d.dat", n))
	}

	for _, name := range files {
		want, err := os.ReadFile(filepath.Join(ref, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue // not every reference set has every file
			}
			t.Fatalf("read reference %s: %v", name, err)
		}
		got, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Errorf("%s: re-encoded file missing: %v", name, err)
			continue
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s: re-encoded bytes differ from fhc (%d want vs %d got bytes)%s",
				name, len(want), len(got), firstDiff(want, got))
		}
	}
}

// firstDiff returns a short description of the first differing byte offset.
func firstDiff(a, b []byte) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return fmt.Sprintf("; first diff at offset %d: want 0x%02x got 0x%02x", i, a[i], b[i])
		}
	}
	return ""
}
