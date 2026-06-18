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

// TestScriptCommandSpaceSeparatedFlagsRejected checks the --opt value form is
// rejected: fhc uses the --opt=val convention, so --data-root with no '=' has
// no value and is an error (exit 2) before any Lua runs.
func TestScriptCommandSpaceSeparatedFlagsRejected(t *testing.T) {
	ResetState()
	dataRoot := t.TempDir()
	script := writeScript(t, `print("hi")`)

	rv := scriptCommand([]string{"script", "--data-root", dataRoot, "--species", "8", script})
	if rv != 2 {
		t.Fatalf("scriptCommand returned %d, want 2 (space-separated flags are not supported)", rv)
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

// runSandboxScript runs a Lua body under the --gm scope and returns the exit
// code. The sandbox tests below each write a script that error()s when a global
// they expect to be nil is in fact present, so a leak surfaces as exit code 2.
func runSandboxScript(t *testing.T, body string) int {
	t.Helper()
	ResetState()
	dataRoot := t.TempDir()
	script := writeScript(t, body)
	return scriptCommand([]string{"script", "--data-root=" + dataRoot, "--gm", script})
}

// TestScriptSandboxDangerousGlobalsAbsent checks security layer 1 (#40): the
// filesystem/OS/introspection libraries and the code-loading globals are absent
// in the sandboxed interpreter. Each script returns 0 only if the global is
// nil, and exit 2 (via error()) if it leaked.
func TestScriptSandboxDangerousGlobalsAbsent(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"os", `if os ~= nil then error("os leaked") end`},
		{"io", `if io ~= nil then error("io leaked") end`},
		{"debug", `if debug ~= nil then error("debug leaked") end`},
		{"require", `if require ~= nil then error("require leaked") end`},
		{"load", `if load ~= nil then error("load leaked") end`},
		{"loadfile", `if loadfile ~= nil then error("loadfile leaked") end`},
		{"loadstring", `if loadstring ~= nil then error("loadstring leaked") end`},
		{"dofile", `if dofile ~= nil then error("dofile leaked") end`},
		{"math.random", `if math.random ~= nil then error("math.random leaked") end`},
		{"math.randomseed", `if math.randomseed ~= nil then error("math.randomseed leaked") end`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rv := runSandboxScript(t, tc.body); rv != 0 {
				t.Errorf("sandbox script for %s returned %d, want 0 (global should be nil)", tc.name, rv)
			}
		})
	}
}

// TestScriptSandboxAllowedLibsWork is the positive control: the pure libraries
// we deliberately keep (base print, string, table, math) must still be
// callable. A failure here would mean the sandbox stripped too much.
func TestScriptSandboxAllowedLibsWork(t *testing.T) {
	body := `
		print("hello")
		if string.upper("ab") ~= "AB" then error("string.upper broken") end
		local t = {}
		table.insert(t, 1)
		if #t ~= 1 then error("table.insert broken") end
		if math.floor(1.9) ~= 1 then error("math.floor broken") end
	`
	if rv := runSandboxScript(t, body); rv != 0 {
		t.Errorf("allowed-libs script returned %d, want 0", rv)
	}
}
