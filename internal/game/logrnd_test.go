package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestLogRndCommandStdoutMatchesC runs the wired `logrnd` command through the
// same os.Stdout-capture path the CLI uses and compares its output byte-for-byte
// against the C reference in testdata/cref/setup/logrnd.log (captured by
// testdata/cref/generate.sh from `fh logrnd`).
//
// logrnd seeds from the historical default seed and ignores game state, so no
// staging is needed.
func TestLogRndCommandStdoutMatchesC(t *testing.T) {
	setupDir := refDir(t, "setup")
	requireRef(t, setupDir)

	got := captureStdout(t, func() int {
		ResetState()
		return logRandomCommand(os.Stdout)
	})

	want, err := os.ReadFile(filepath.Join(setupDir, "logrnd.log"))
	if err != nil {
		t.Fatalf("read reference logrnd.log: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("logrnd output differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
			len(got), len(want), firstDiff(got, want))
	}
}
