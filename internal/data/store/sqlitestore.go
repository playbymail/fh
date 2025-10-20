package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"

	"github.com/playbymail/fh/internal/cerrs"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := setupSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// setupSchema creates the database tables.
func setupSchema(db *sql.DB) error {
	schema := `
-- games & turns
CREATE TABLE IF NOT EXISTS game (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS turn (
  game_id TEXT NOT NULL,
  num INTEGER NOT NULL,
  phase TEXT NOT NULL,
  started_at TEXT NOT NULL,
  ended_at TEXT,
  PRIMARY KEY (game_id, num, phase),
  FOREIGN KEY (game_id) REFERENCES game(id)
);

-- world snapshots
CREATE TABLE IF NOT EXISTS entity (
  game_id TEXT NOT NULL,
  turn_num INTEGER NOT NULL,
  id TEXT NOT NULL,
  kind TEXT NOT NULL,
  data BLOB NOT NULL,
  PRIMARY KEY (game_id, turn_num, id),
  FOREIGN KEY (game_id, turn_num) REFERENCES turn(game_id, num)
);

-- orders
CREATE TABLE IF NOT EXISTS orders (
  game_id TEXT NOT NULL,
  turn_num INTEGER NOT NULL,
  actor TEXT NOT NULL,
  seq INTEGER NOT NULL,
  raw TEXT NOT NULL,
  normalized TEXT,
  status TEXT NOT NULL,
  error TEXT,
  PRIMARY KEY (game_id, turn_num, actor, seq),
  FOREIGN KEY (game_id, turn_num) REFERENCES turn(game_id, num)
);

-- reports
CREATE TABLE IF NOT EXISTS report (
  game_id TEXT NOT NULL,
  turn_num INTEGER NOT NULL,
  actor TEXT NOT NULL,
  mime TEXT NOT NULL,
  body BLOB NOT NULL,
  PRIMARY KEY (game_id, turn_num, actor, mime),
  FOREIGN KEY (game_id, turn_num) REFERENCES turn(game_id, num)
);
`
	_, err := db.Exec(schema)
	return err
}

// CreateGame inserts a new game.
func (s *SQLiteStore) CreateGame(ctx context.Context, id, name string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO game (id, name, created_at) VALUES (?, ?, datetime('now'))
	`, id, name)
	return err
}

// GetGame retrieves game metadata.
func (s *SQLiteStore) GetGame(ctx context.Context, id string) (*Game, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, created_at FROM game WHERE id = ?
	`, id)

	var game Game
	err := row.Scan(&game.ID, &game.Name, &game.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, cerrs.ErrNotImplemented // TODO: proper not found error
	}
	return &game, err
}

// CreateTurn inserts a new turn.
func (s *SQLiteStore) CreateTurn(ctx context.Context, gameID string, turnNum int, phase string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO turn (game_id, num, phase, started_at) VALUES (?, ?, ?, datetime('now'))
	`, gameID, turnNum, phase)
	return err
}

// GetCurrentTurn finds the latest turn.
func (s *SQLiteStore) GetCurrentTurn(ctx context.Context, gameID string) (*Turn, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT game_id, num, phase, started_at, ended_at
		FROM turn
		WHERE game_id = ?
		ORDER BY num DESC, started_at DESC
		LIMIT 1
	`, gameID)

	var turn Turn
	err := row.Scan(&turn.GameID, &turn.Num, &turn.Phase, &turn.StartedAt, &turn.EndedAt)
	if err == sql.ErrNoRows {
		return nil, cerrs.ErrNotImplemented
	}
	return &turn, err
}

// SaveSnapshot saves entities.
func (s *SQLiteStore) SaveSnapshot(ctx context.Context, gameID string, turnNum int, entities []Entity) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing entities for this turn
	_, err = tx.ExecContext(ctx, `
		DELETE FROM entity WHERE game_id = ? AND turn_num = ?
	`, gameID, turnNum)
	if err != nil {
		return err
	}

	// Insert new entities
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO entity (game_id, turn_num, id, kind, data) VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entity := range entities {
		data, err := json.Marshal(entity.Data) // assuming Data is already JSON, but marshal to bytes
		if err != nil {
			return err
		}
		_, err = stmt.ExecContext(ctx, gameID, turnNum, entity.ID, entity.Kind, data)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LoadSnapshot loads entities.
func (s *SQLiteStore) LoadSnapshot(ctx context.Context, gameID string, turnNum int) ([]Entity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, data FROM entity WHERE game_id = ? AND turn_num = ?
	`, gameID, turnNum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var entity Entity
		var data []byte
		err := rows.Scan(&entity.ID, &entity.Kind, &data)
		if err != nil {
			return nil, err
		}
		entity.Data = data
		entities = append(entities, entity)
	}
	return entities, rows.Err()
}

// SaveOrders saves orders.
func (s *SQLiteStore) SaveOrders(ctx context.Context, gameID string, turnNum int, actor string, orders []Order) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing orders
	_, err = tx.ExecContext(ctx, `
		DELETE FROM orders WHERE game_id = ? AND turn_num = ? AND actor = ?
	`, gameID, turnNum, actor)
	if err != nil {
		return err
	}

	// Insert new orders
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO orders (game_id, turn_num, actor, seq, raw, normalized, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, order := range orders {
		_, err = stmt.ExecContext(ctx, gameID, turnNum, actor, order.Seq, order.Raw, order.Normalized, order.Status, order.Error)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetOrders retrieves orders.
func (s *SQLiteStore) GetOrders(ctx context.Context, gameID string, turnNum int, actor string) ([]Order, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT seq, raw, normalized, status, error FROM orders
		WHERE game_id = ? AND turn_num = ? AND actor = ?
		ORDER BY seq
	`, gameID, turnNum, actor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		err := rows.Scan(&order.Seq, &order.Raw, &order.Normalized, &order.Status, &order.Error)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// SaveReport saves a report.
func (s *SQLiteStore) SaveReport(ctx context.Context, gameID string, turnNum int, actor string, mime string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO report (game_id, turn_num, actor, mime, body) VALUES (?, ?, ?, ?, ?)
	`, gameID, turnNum, actor, mime, data)
	return err
}

// GetReport retrieves a report.
func (s *SQLiteStore) GetReport(ctx context.Context, gameID string, turnNum int, actor string, mime string) (io.ReadCloser, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT body FROM report WHERE game_id = ? AND turn_num = ? AND actor = ? AND mime = ?
	`, gameID, turnNum, actor, mime).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, cerrs.ErrNotImplemented
		}
		return nil, err
	}

	return io.NopCloser(NewByteReader(data)), nil
}

// ByteReader implements io.Reader for byte slice.
type ByteReader struct {
	data []byte
	pos  int
}

func NewByteReader(data []byte) *ByteReader {
	return &ByteReader{data: data}
}

func (r *ByteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// Close is a no-op.
func (r *ByteReader) Close() error {
	return nil
}

// Close closes the database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
