package game

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestVersionCommandMatchesC verifies the `version` command prints the C engine
// version byte-for-byte and returns 0 on success (v7.5.12 changed the success
// path from the historical `return 2` to `return 0`). version ignores game
// state, so no staging is needed.
//
// Capture stdout inline and assert both the bytes and the return value, so the
// v7.5.12 return-code change stays pinned.
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

	if rv != 0 {
		t.Errorf("versionCommand returned %d, want 0 (v7.5.12 returns 0 on success)", rv)
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
