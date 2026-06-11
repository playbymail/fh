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

// TestReportMatchesC drives the full SQLite-backed path end to end and asserts
// the rendered species report is byte-identical to fhc's reference.
//
//	fhc state dir (export-json + sp0X.log + locations.dat)
//	  -> ingest -> SQLite store
//	  -> load World from store -> render report
//	  -> diff against testdata/cref/turn1/sp01.rpt.t1
//
// The reference data is git-ignored; the test skips when it is absent (run
// testdata/cref/generate.sh to produce it).
func TestReportMatchesC(t *testing.T) {
	ctx := context.Background()
	refDir, err := filepath.Abs(filepath.Join("../../testdata/cref/turn1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(refDir, "galaxy.json")); err != nil {
		t.Skipf("C reference data not available in %s: %v (run testdata/cref/generate.sh)", refDir, err)
	}

	st, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "fh.db"), false)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer st.Close()

	const gameID = "test"
	if err := st.CreateGame(ctx, gameID, "parity test"); err != nil {
		t.Fatalf("create game: %v", err)
	}

	eng := engine.New(st, gameID)
	if err := eng.IngestDir(ctx, refDir); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	// The first slice targets sp01 (the simplest report), but the renderer
	// generalizes: every turn-1 species report must match byte-for-byte.
	for _, n := range []int{1, 2, 3, 4} {
		got, err := eng.RenderReport(ctx, n, engine.ReportOptions{})
		if err != nil {
			t.Fatalf("sp%02d: render report: %v", n, err)
		}
		want, err := os.ReadFile(filepath.Join(refDir, fmt.Sprintf("sp%02d.rpt.t1", n)))
		if err != nil {
			t.Fatalf("sp%02d: read reference report: %v", n, err)
		}
		if !bytes.Equal(got, want) {
			d := firstDiff(got, want)
			t.Errorf("sp%02d: rendered report differs from C reference (got %d bytes, want %d); first divergence at byte %d",
				n, len(got), len(want), d)
			t.Logf("got:\n%s", excerpt(got, d))
			t.Logf("want:\n%s", excerpt(want, d))
		}
	}
}

// firstDiff returns the index of the first differing byte, or -1 if equal.
func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

// excerpt returns a window of bytes around pos for diagnostics.
func excerpt(b []byte, pos int) string {
	if pos < 0 {
		return ""
	}
	start := pos - 80
	if start < 0 {
		start = 0
	}
	end := pos + 80
	if end > len(b) {
		end = len(b)
	}
	return string(b[start:end])
}
