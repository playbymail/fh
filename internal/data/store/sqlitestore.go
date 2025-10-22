package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/playbymail/fh/internal/cerrs"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLiteStore opens an existing SQLite store.
func OpenSQLiteStore(dbPath string) (*SQLiteStore, error) {
	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return nil, cerrs.ErrNotExist
	}
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}

	// Check schema version
	version, err := store.GetSchemaVersion(context.Background())
	if err != nil {
		store.Close()
		return nil, err
	}
	expected := "v0.8.0_initial"
	if version != expected {
		if version < expected {
			// Upgrade
			if err := store.UpgradeSchema(context.Background()); err != nil {
				store.Close()
				return nil, errors.Join(cerrs.ErrSchemaUpgradeFailed, err)
			}
		} else {
			store.Close()
			return nil, cerrs.ErrSchemaTooNew
		}
	}

	return store, nil
}

// NewSQLiteStore creates a new SQLite store.
func NewSQLiteStore(dbPath string, force bool) (*SQLiteStore, error) {
	_, err := os.Stat(dbPath)
	exists := !os.IsNotExist(err)
	if exists {
		if !force {
			return nil, cerrs.ErrExists
		}
		if err := os.Remove(dbPath); err != nil {
			return nil, errors.Join(cerrs.ErrExists, err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, errors.Join(cerrs.ErrNotOpened, err)
	}

	if err := setupSchema(db); err != nil {
		db.Close()
		return nil, errors.Join(cerrs.ErrSchemaSetupFailed, err)
	}

	return &SQLiteStore{db: db}, nil
}

// setupSchema creates the database tables.
func setupSchema(db *sql.DB) error {
	schema := `
-- migrations
CREATE TABLE IF NOT EXISTS migrations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  applied_at TEXT NOT NULL
);

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
	if err != nil {
		return err
	}

	// Record the initial migration
	_, err = db.Exec(`
		INSERT OR IGNORE INTO migrations (name, applied_at) VALUES ('v0.8.0_initial', datetime('now'))
	`)
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

// GetSchemaVersion returns the current schema version.
func (s *SQLiteStore) GetSchemaVersion(ctx context.Context) (string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT name FROM migrations ORDER BY id DESC LIMIT 1
	`)
	var version string
	err := row.Scan(&version)
	if err == sql.ErrNoRows {
		return "", cerrs.ErrNoMigrations
	}
	return version, err
}

// UpgradeSchema applies pending schema upgrades.
func (s *SQLiteStore) UpgradeSchema(ctx context.Context) error {
	// For now, no upgrades needed beyond initial schema.
	// Future: check current version and apply migrations in order.
	return nil
}

// Close closes the database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
