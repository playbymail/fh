package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestInspectCommandMatchesC runs the `inspect` command and compares its
// captured stdout byte-for-byte against the C reference in
// testdata/cref/setup/inspect.log (captured by generate.sh from `fh inspect`).
//
// inspect ignores game state, so no staging is needed. The in-memory struct
// rows are hardcoded reference-platform C ABI sizes; the binary_*_data_t rows
// come from the Go on-disk record-size constants. Either way the table must
// match the C engine exactly.
func TestInspectCommandMatchesC(t *testing.T) {
	setupDir := refDir(t, "setup")
	requireRef(t, setupDir)

	got := captureStdout(t, func() int {
		ResetState()
		return inspectCommand([]string{"inspect"})
	})

	want, err := os.ReadFile(filepath.Join(setupDir, "inspect.log"))
	if err != nil {
		t.Fatalf("read reference inspect.log: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("inspect output differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
			len(got), len(want), firstDiff(got, want))
	}
}
