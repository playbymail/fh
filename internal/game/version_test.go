package game

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestVersionCommandMatchesC verifies the `version` command prints the C engine
// version byte-for-byte and returns 2 (version.c returns 2 even on its success
// path). version ignores game state, so no staging is needed.
//
// We cannot use captureStdout here: it fails the test on a non-zero return, and
// version's success path returns 2 by design. So capture stdout inline and
// assert both the bytes and the return value.
func TestVersionCommandMatchesC(t *testing.T) {
	setupDir := refDir(t, "setup")
	requireRef(t, setupDir)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- data
	}()

	ResetState()
	rv := versionCommand([]string{"version"})

	os.Stdout = orig
	_ = w.Close()
	got := <-done
	_ = r.Close()

	if rv != 2 {
		t.Errorf("versionCommand returned %d, want 2 (version.c returns 2 on success)", rv)
	}

	want, err := os.ReadFile(filepath.Join(setupDir, "version.log"))
	if err != nil {
		t.Fatalf("read reference version.log: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("version output differs from C reference: got %q, want %q", got, want)
	}
	// Belt-and-suspenders: the C reference is exactly the engine version + LF.
	if string(want) != cEngineVersion+"\n" {
		t.Errorf("C reference version.log = %q, want %q", want, cEngineVersion+"\n")
	}
}
