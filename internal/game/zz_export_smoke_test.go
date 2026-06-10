package game

import (
	"os"
	"testing"
)

func TestExportSmokeTemp(t *testing.T) {
	if os.Getenv("FH_SMOKE_DIR") == "" {
		t.Skip("no smoke dir")
	}
	ResetState()
	if err := os.Chdir(os.Getenv("FH_SMOKE_DIR")); err != nil {
		t.Fatal(err)
	}
	if rc := exportCommand([]string{"export", "json"}); rc != 0 {
		t.Fatalf("export rc %d", rc)
	}
}

func TestImportSmokeTemp(t *testing.T) {
	if os.Getenv("FH_SMOKE_DIR") == "" {
		t.Skip("no smoke dir")
	}
	ResetState()
	if err := os.Chdir(os.Getenv("FH_SMOKE_DIR")); err != nil {
		t.Fatal(err)
	}
	if rc := importCommand([]string{"import", "json"}); rc != 0 {
		t.Fatalf("import rc %d", rc)
	}
}
