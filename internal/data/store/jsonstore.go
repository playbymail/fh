package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/playbymail/fh/internal/cerrs"
)

// JSONStore implements Store using JSON files.
// Data is stored in a directory structure:
// baseDir/
//   games/
//     {gameID}/
//       game.json
//       turns/
//         {turnNum}-{phase}.json
//       snapshots/
//         {turnNum}.json
//       orders/
//         {turnNum}/{actor}.json
//       reports/
//         {turnNum}/{actor}/{mime}
type JSONStore struct {
	baseDir string
}

// NewJSONStore creates a new JSON file store.
func NewJSONStore(baseDir string) (*JSONStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &JSONStore{baseDir: baseDir}, nil
}

// CreateGame creates a new game directory and metadata.
func (s *JSONStore) CreateGame(ctx context.Context, id, name string) error {
	gameDir := filepath.Join(s.baseDir, "games", id)
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		return err
	}

	game := Game{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	path := filepath.Join(gameDir, "game.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(game)
}

// GetGame retrieves game metadata.
func (s *JSONStore) GetGame(ctx context.Context, id string) (*Game, error) {
	path := filepath.Join(s.baseDir, "games", id, "game.json")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, cerrs.ErrNotImplemented // TODO: proper error
		}
		return nil, err
	}
	defer file.Close()

	var game Game
	if err := json.NewDecoder(file).Decode(&game); err != nil {
		return nil, err
	}
	return &game, nil
}

// CreateTurn creates a turn directory and metadata.
func (s *JSONStore) CreateTurn(ctx context.Context, gameID string, turnNum int, phase string) error {
	turnDir := filepath.Join(s.baseDir, "games", gameID, "turns")
	if err := os.MkdirAll(turnDir, 0755); err != nil {
		return err
	}

	turn := Turn{
		GameID:    gameID,
		Num:       turnNum,
		Phase:     phase,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	path := filepath.Join(turnDir, fmt.Sprintf("%d-%s.json", turnNum, phase))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(turn)
}

// GetCurrentTurn finds the latest turn for a game.
func (s *JSONStore) GetCurrentTurn(ctx context.Context, gameID string) (*Turn, error) {
	return nil, cerrs.ErrNotImplemented
}

// SaveSnapshot saves world entities for a turn.
func (s *JSONStore) SaveSnapshot(ctx context.Context, gameID string, turnNum int, entities []Entity) error {
	snapDir := filepath.Join(s.baseDir, "games", gameID, "snapshots")
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(snapDir, fmt.Sprintf("%d.json", turnNum))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(entities)
}

// LoadSnapshot loads world entities for a turn.
func (s *JSONStore) LoadSnapshot(ctx context.Context, gameID string, turnNum int) ([]Entity, error) {
	path := filepath.Join(s.baseDir, "games", gameID, "snapshots", fmt.Sprintf("%d.json", turnNum))
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, cerrs.ErrNotImplemented
		}
		return nil, err
	}
	defer file.Close()

	var entities []Entity
	if err := json.NewDecoder(file).Decode(&entities); err != nil {
		return nil, err
	}
	return entities, nil
}

// SaveOrders saves player orders for a turn.
func (s *JSONStore) SaveOrders(ctx context.Context, gameID string, turnNum int, actor string, orders []Order) error {
	ordersDir := filepath.Join(s.baseDir, "games", gameID, "orders", fmt.Sprintf("%d", turnNum))
	if err := os.MkdirAll(ordersDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(ordersDir, fmt.Sprintf("%s.json", actor))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(orders)
}

// GetOrders retrieves player orders for a turn.
func (s *JSONStore) GetOrders(ctx context.Context, gameID string, turnNum int, actor string) ([]Order, error) {
	path := filepath.Join(s.baseDir, "games", gameID, "orders", fmt.Sprintf("%d", turnNum), fmt.Sprintf("%s.json", actor))
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, cerrs.ErrNotImplemented
		}
		return nil, err
	}
	defer file.Close()

	var orders []Order
	if err := json.NewDecoder(file).Decode(&orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// SaveReport saves a report.
func (s *JSONStore) SaveReport(ctx context.Context, gameID string, turnNum int, actor string, mime string, body io.Reader) error {
	reportDir := filepath.Join(s.baseDir, "games", gameID, "reports", fmt.Sprintf("%d", turnNum), actor)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(reportDir, mime)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, body)
	return err
}

// GetReport retrieves a report.
func (s *JSONStore) GetReport(ctx context.Context, gameID string, turnNum int, actor string, mime string) (io.ReadCloser, error) {
	path := filepath.Join(s.baseDir, "games", gameID, "reports", fmt.Sprintf("%d", turnNum), actor, mime)
	return os.Open(path)
}

// Close is a no-op for JSON store.
func (s *JSONStore) Close() error {
	return nil
}
