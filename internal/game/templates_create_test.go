package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCreateHomeSystemTemplatesMatchesC runs the ported
// createHomeSystemTemplates with the C reference seed and compares the
// generated homesystem files against the C engine's output
// (fh create home-system-templates, FH_SEED=1924085713).
func TestCreateHomeSystemTemplatesMatchesC(t *testing.T) {
	refDir, err := filepath.Abs("../../testdata/cref/setup")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(refDir, "homesystem3.dat")); err != nil {
		t.Skipf("C reference data not available: %v", err)
	}

	ResetState()
	t.Setenv("FH_SEED", "1924085713")
	t.Chdir(t.TempDir())

	if rc := createHomeSystemTemplates(); rc != 0 {
		t.Fatalf("createHomeSystemTemplates returned %d", rc)
	}

	for n := 3; n <= 9; n++ {
		for _, ext := range []string{"dat", "txt"} {
			name := fmt.Sprintf("homesystem%d.%s", n, ext)
			got, err := os.ReadFile(name)
			if err != nil {
				t.Errorf("%s: %v", name, err)
				continue
			}
			want, err := os.ReadFile(filepath.Join(refDir, name))
			if err != nil {
				t.Errorf("%s: %v", name, err)
				continue
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s differs from C reference (got %d bytes, want %d bytes); first divergence at byte %d",
					name, len(got), len(want), firstDiff(got, want))
			}
		}
	}
}
