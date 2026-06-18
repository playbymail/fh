package game

import (
	"os"
	"path/filepath"
	"testing"
)

// writeScript writes a Lua script into a temp dir and returns its path.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "script.lua")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

// TestScriptCommandRunsTrivialScript checks the GopherLua host wires up end to
// end: a trivial print("hello") script runs and the command returns 0. The
// host loads no .dat files, so no game staging is required.
func TestScriptCommandRunsTrivialScript(t *testing.T) {
	ResetState()
	dataRoot := t.TempDir()
	script := writeScript(t, `print("hello")`)

	rv := scriptCommand([]string{"script", "--data-root=" + dataRoot, "--gm", script})
	if rv != 0 {
		t.Fatalf("scriptCommand returned %d, want 0", rv)
	}
}

// TestScriptCommandSpaceSeparatedFlags checks the --opt value form (the design
// doc writes flags space-separated) parses the same as --opt=value, including
// a player scope id.
func TestScriptCommandSpaceSeparatedFlags(t *testing.T) {
	ResetState()
	dataRoot := t.TempDir()
	script := writeScript(t, `print("hi")`)

	rv := scriptCommand([]string{"script", "--data-root", dataRoot, "--species", "8", script})
	if rv != 0 {
		t.Fatalf("scriptCommand returned %d, want 0", rv)
	}
}

// TestScriptCommandFlagValidation covers the bad flag combinations, each of
// which must be rejected with exit code 2 before any Lua runs.
func TestScriptCommandFlagValidation(t *testing.T) {
	ResetState()
	dataRoot := t.TempDir()
	script := writeScript(t, `print("hello")`)

	cases := []struct {
		name string
		args []string
	}{
		{"missing data-root", []string{"script", "--gm", script}},
		{"gm and species", []string{"script", "--data-root=" + dataRoot, "--gm", "--species=1", script}},
		{"neither gm nor species", []string{"script", "--data-root=" + dataRoot, script}},
		{"non-integer species", []string{"script", "--data-root=" + dataRoot, "--species=abc", script}},
		{"zero species", []string{"script", "--data-root=" + dataRoot, "--species=0", script}},
		{"missing script path", []string{"script", "--data-root=" + dataRoot, "--gm"}},
		{"unknown option", []string{"script", "--data-root=" + dataRoot, "--gm", "--bogus", script}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rv := scriptCommand(tc.args); rv != 2 {
				t.Errorf("scriptCommand(%v) = %d, want 2", tc.args[1:], rv)
			}
		})
	}
}

// TestScriptCommandBothScopesRun exercises both scopes end to end. #39 does not
// expose the host to Lua yet, so the observable contract is the return code:
// a valid --gm and a valid --species invocation each run the script and return
// 0. Scope is consumed by later slices (#41+).
func TestScriptCommandBothScopesRun(t *testing.T) {
	ResetState()
	dataRoot := t.TempDir()
	script := writeScript(t, `print("ok")`)

	if rv := scriptCommand([]string{"script", "--data-root=" + dataRoot, "--gm", script}); rv != 0 {
		t.Errorf("gm scope returned %d, want 0", rv)
	}
	if rv := scriptCommand([]string{"script", "--data-root=" + dataRoot, "--species=3", script}); rv != 0 {
		t.Errorf("player scope returned %d, want 0", rv)
	}
}

// TestScriptCommandFailingScript checks a Lua runtime error surfaces as exit 2.
func TestScriptCommandFailingScript(t *testing.T) {
	ResetState()
	dataRoot := t.TempDir()
	script := writeScript(t, `error("boom")`)

	if rv := scriptCommand([]string{"script", "--data-root=" + dataRoot, "--gm", script}); rv != 2 {
		t.Errorf("failing script returned %d, want 2", rv)
	}
}
