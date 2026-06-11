package engine_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/playbymail/fh/internal/data/store"
	"github.com/playbymail/fh/internal/engine"
)

// TestCrossBackendReportParity drives the same fhc state through all three
// store backends (binary, json, sqlite) and asserts every backend renders turn
// reports byte-identical to fhc's reference — and therefore to each other.
// Report parity across backends is the contract that lets a gamemaster pick
// any store format.
//
// The reference data is git-ignored; the test skips when it is absent (run
// testdata/cref/generate.sh to produce it).
func TestCrossBackendReportParity(t *testing.T) {
	ctx := context.Background()
	refDir, err := filepath.Abs(filepath.Join("../../testdata/cref/turn1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(refDir, "galaxy.json")); err != nil {
		t.Skipf("C reference data not available in %s: %v (run testdata/cref/generate.sh)", refDir, err)
	}

	species := []int{1, 2, 3, 4}
	const gameID = "test"

	// reports[typ][speciesNumber] = rendered bytes
	reports := map[store.StoreType]map[int][]byte{}

	for _, typ := range []store.StoreType{store.StoreBinary, store.StoreJSON, store.StoreSQLite} {
		dir := filepath.Join(t.TempDir(), "game")
		st, err := store.Create(ctx, dir, store.CreateOptions{ID: gameID, Type: typ})
		if err != nil {
			t.Fatalf("%s: create: %v", typ, err)
		}
		eng := engine.New(st, gameID)
		if err := eng.IngestDir(ctx, refDir); err != nil {
			t.Fatalf("%s: ingest: %v", typ, err)
		}

		reports[typ] = map[int][]byte{}
		for _, n := range species {
			got, err := eng.RenderReport(ctx, n, engine.ReportOptions{})
			if err != nil {
				t.Fatalf("%s: sp%02d: render: %v", typ, n, err)
			}
			want, err := os.ReadFile(filepath.Join(refDir, fmt.Sprintf("sp%02d.rpt.t1", n)))
			if err != nil {
				t.Fatalf("%s: sp%02d: read reference: %v", typ, n, err)
			}
			if !bytes.Equal(want, got) {
				t.Errorf("%s: sp%02d report differs from fhc reference", typ, n)
			}
			reports[typ][n] = got
		}
		st.Close()
	}

	// Cross-check: every backend produced identical bytes for each species.
	for _, n := range species {
		base := reports[store.StoreBinary][n]
		for _, typ := range []store.StoreType{store.StoreJSON, store.StoreSQLite} {
			if !bytes.Equal(base, reports[typ][n]) {
				t.Errorf("sp%02d: %s report differs from binary backend", n, typ)
			}
		}
	}
}
