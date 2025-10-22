package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	st, err := NewSQLiteStore(dbPath, false)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	if err := st.CreateGame(ctx, "game1", "Test Game"); err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	if err := st.CreateTurn(ctx, "game1", 1, "production"); err != nil {
		t.Fatalf("failed to create turn: %v", err)
	}

	testData := []Entity{
		{ID: "planet-1", Kind: "planet", Data: []byte(`{"name":"Earth","population":1000}`)},
		{ID: "ship-1", Kind: "ship", Data: []byte(`{"name":"Enterprise","tonnage":5000}`)},
		{ID: "species-1", Kind: "species", Data: []byte(`{"name":"Humans","tech_level":10}`)},
	}

	if err := st.SaveSnapshot(ctx, "game1", 1, testData); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	loaded, err := st.LoadSnapshot(ctx, "game1", 1)
	if err != nil {
		t.Fatalf("failed to load snapshot: %v", err)
	}

	if len(loaded) != len(testData) {
		t.Fatalf("expected %d entities, got %d", len(testData), len(loaded))
	}

	for i, entity := range loaded {
		if entity.ID != testData[i].ID {
			t.Errorf("entity %d: expected ID %q, got %q", i, testData[i].ID, entity.ID)
		}
		if entity.Kind != testData[i].Kind {
			t.Errorf("entity %d: expected Kind %q, got %q", i, testData[i].Kind, entity.Kind)
		}
		if string(entity.Data) != string(testData[i].Data) {
			t.Errorf("entity %d: expected Data %q, got %q", i, string(testData[i].Data), string(entity.Data))
		}
	}
}

func TestForeignKeyCascade(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	st, err := NewSQLiteStore(dbPath, false)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	if err := st.CreateGame(ctx, "game1", "Test Game"); err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	if err := st.CreateTurn(ctx, "game1", 1, "production"); err != nil {
		t.Fatalf("failed to create turn: %v", err)
	}

	testData := []Entity{
		{ID: "planet-1", Kind: "planet", Data: []byte(`{"name":"Earth"}`)},
	}

	if err := st.SaveSnapshot(ctx, "game1", 1, testData); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	_, err = st.db.Exec("DELETE FROM game WHERE id = ?", "game1")
	if err != nil {
		t.Fatalf("failed to delete game: %v", err)
	}

	loaded, err := st.LoadSnapshot(ctx, "game1", 1)
	if err != nil {
		t.Fatalf("failed to load snapshot: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("expected cascade delete to remove entities, but got %d entities", len(loaded))
	}
}

func TestSchemaUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	st, err := NewSQLiteStore(dbPath, false)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	st.Close()

	st2, err := OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open existing store: %v", err)
	}
	defer st2.Close()

	version, err := st2.GetSchemaVersion(context.Background())
	if err != nil {
		t.Fatalf("failed to get schema version: %v", err)
	}

	if version != "0001_initial" {
		t.Errorf("expected version 0001_initial, got %q", version)
	}
}

func TestNewSQLiteStoreWithForce(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	st1, err := NewSQLiteStore(dbPath, false)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	st1.Close()

	_, err = NewSQLiteStore(dbPath, false)
	if err == nil {
		t.Fatal("expected error when creating store without force flag")
	}

	st2, err := NewSQLiteStore(dbPath, true)
	if err != nil {
		t.Fatalf("failed to create store with force flag: %v", err)
	}
	defer st2.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file should exist after force create")
	}
}
