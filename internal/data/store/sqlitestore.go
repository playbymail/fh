package store

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"

	"github.com/playbymail/fh/internal/cerrs"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitemigration"
	"zombiezen.com/go/sqlite/sqlitex"
)

// SQLiteStore implements Store using SQLite database.
type SQLiteStore struct {
	pool *sqlitemigration.Pool
}

// prepareConn applies per-connection pragmas. WAL journal mode is set via the
// OpenWAL flag; foreign keys, busy timeout, and synchronous mode are not
// persisted and must be set on every connection.
func prepareConn(conn *sqlite.Conn) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, pragma := range pragmas {
		if err := sqlitex.ExecuteTransient(conn, pragma, nil); err != nil {
			return err
		}
	}
	return nil
}

// appSchema defines the ordered list of schema migrations applied by
// sqlitemigration. The index of each script (1-based once applied) is tracked
// in the database via PRAGMA user_version, so scripts must only ever be
// appended, never reordered or edited once released. Statements use
// "IF NOT EXISTS" so databases created by earlier schema-management code
// migrate cleanly.
var appSchema = sqlitemigration.Schema{
	Migrations: []string{
		`
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
  PRIMARY KEY (game_id, num),
  FOREIGN KEY (game_id) REFERENCES game(id) ON DELETE CASCADE
);

-- world snapshots
CREATE TABLE IF NOT EXISTS entity (
  game_id TEXT NOT NULL,
  turn_num INTEGER NOT NULL,
  id TEXT NOT NULL,
  kind TEXT NOT NULL,
  data BLOB NOT NULL,
  PRIMARY KEY (game_id, turn_num, id),
  FOREIGN KEY (game_id, turn_num) REFERENCES turn(game_id, num) ON DELETE CASCADE
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
  FOREIGN KEY (game_id, turn_num) REFERENCES turn(game_id, num) ON DELETE CASCADE
);

-- reports
CREATE TABLE IF NOT EXISTS report (
  game_id TEXT NOT NULL,
  turn_num INTEGER NOT NULL,
  actor TEXT NOT NULL,
  mime TEXT NOT NULL,
  body BLOB NOT NULL,
  PRIMARY KEY (game_id, turn_num, actor, mime),
  FOREIGN KEY (game_id, turn_num) REFERENCES turn(game_id, num) ON DELETE CASCADE
);

-- indexes
CREATE INDEX IF NOT EXISTS idx_turn_game_started ON turn(game_id, started_at);
`,
	},
}

// openPool opens a migration-managed connection pool and blocks until all
// migrations have been applied, surfacing any migration error.
func openPool(dbPath string, flags sqlite.OpenFlags) (*sqlitemigration.Pool, error) {
	pool := sqlitemigration.NewPool(dbPath, appSchema, sqlitemigration.Options{
		Flags:       flags,
		PrepareConn: prepareConn,
	})

	// Take a connection to force migrations to complete and surface any error.
	conn, err := pool.Take(context.Background())
	if err != nil {
		pool.Close()
		return nil, err
	}
	pool.Put(conn)

	return pool, nil
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

	pool, err := openPool(dbPath, sqlite.OpenReadWrite|sqlite.OpenWAL|sqlite.OpenURI)
	if err != nil {
		return nil, errors.Join(cerrs.ErrSchemaUpgradeFailed, err)
	}

	return &SQLiteStore{pool: pool}, nil
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

	pool, err := openPool(dbPath, sqlite.OpenReadWrite|sqlite.OpenCreate|sqlite.OpenWAL|sqlite.OpenURI)
	if err != nil {
		return nil, errors.Join(cerrs.ErrSchemaSetupFailed, err)
	}

	return &SQLiteStore{pool: pool}, nil
}

// CreateGame inserts a new game.
func (s *SQLiteStore) CreateGame(ctx context.Context, id, name string) error {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn, `
		INSERT INTO game (id, name, created_at) VALUES (?, ?, datetime('now'))
	`, &sqlitex.ExecOptions{Args: []any{id, name}})
}

// GetGame retrieves game metadata.
func (s *SQLiteStore) GetGame(ctx context.Context, id string) (*Game, error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var game *Game
	err = sqlitex.Execute(conn, `
		SELECT id, name, created_at FROM game WHERE id = ?
	`, &sqlitex.ExecOptions{
		Args: []any{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			game = &Game{
				ID:        stmt.ColumnText(0),
				Name:      stmt.ColumnText(1),
				CreatedAt: stmt.ColumnText(2),
			}
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, cerrs.ErrNotImplemented // TODO: proper not found error
	}
	return game, nil
}

// CreateTurn inserts a new turn.
func (s *SQLiteStore) CreateTurn(ctx context.Context, gameID string, turnNum int, phase string) error {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn, `
		INSERT INTO turn (game_id, num, phase, started_at) VALUES (?, ?, ?, datetime('now'))
	`, &sqlitex.ExecOptions{Args: []any{gameID, turnNum, phase}})
}

// GetCurrentTurn finds the latest turn.
func (s *SQLiteStore) GetCurrentTurn(ctx context.Context, gameID string) (*Turn, error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var turn *Turn
	err = sqlitex.Execute(conn, `
		SELECT game_id, num, phase, started_at, ended_at
		FROM turn
		WHERE game_id = ?
		ORDER BY num DESC, started_at DESC
		LIMIT 1
	`, &sqlitex.ExecOptions{
		Args: []any{gameID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			turn = &Turn{
				GameID:    stmt.ColumnText(0),
				Num:       stmt.ColumnInt(1),
				Phase:     stmt.ColumnText(2),
				StartedAt: stmt.ColumnText(3),
				EndedAt:   stmt.ColumnText(4),
			}
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	if turn == nil {
		return nil, cerrs.ErrNotImplemented
	}
	return turn, nil
}

// SaveSnapshot saves entities.
func (s *SQLiteStore) SaveSnapshot(ctx context.Context, gameID string, turnNum int, entities []Entity) (err error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	defer sqlitex.Transaction(conn)(&err)

	// Delete existing entities for this turn
	if err = sqlitex.Execute(conn, `
		DELETE FROM entity WHERE game_id = ? AND turn_num = ?
	`, &sqlitex.ExecOptions{Args: []any{gameID, turnNum}}); err != nil {
		return err
	}

	// Insert new entities
	for _, entity := range entities {
		if err = sqlitex.Execute(conn, `
			INSERT INTO entity (game_id, turn_num, id, kind, data) VALUES (?, ?, ?, ?, ?)
		`, &sqlitex.ExecOptions{Args: []any{gameID, turnNum, entity.ID, entity.Kind, entity.Data}}); err != nil {
			return err
		}
	}

	return err
}

// LoadSnapshot loads entities.
func (s *SQLiteStore) LoadSnapshot(ctx context.Context, gameID string, turnNum int) ([]Entity, error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var entities []Entity
	err = sqlitex.Execute(conn, `
		SELECT id, kind, data FROM entity WHERE game_id = ? AND turn_num = ?
	`, &sqlitex.ExecOptions{
		Args: []any{gameID, turnNum},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			data := make([]byte, stmt.ColumnLen(2))
			stmt.ColumnBytes(2, data)
			entities = append(entities, Entity{
				ID:   stmt.ColumnText(0),
				Kind: stmt.ColumnText(1),
				Data: data,
			})
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return entities, nil
}

// SaveOrders saves orders.
func (s *SQLiteStore) SaveOrders(ctx context.Context, gameID string, turnNum int, actor string, orders []Order) (err error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	defer sqlitex.Transaction(conn)(&err)

	// Delete existing orders
	if err = sqlitex.Execute(conn, `
		DELETE FROM orders WHERE game_id = ? AND turn_num = ? AND actor = ?
	`, &sqlitex.ExecOptions{Args: []any{gameID, turnNum, actor}}); err != nil {
		return err
	}

	// Insert new orders
	for _, order := range orders {
		if err = sqlitex.Execute(conn, `
			INSERT INTO orders (game_id, turn_num, actor, seq, raw, normalized, status, error)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, &sqlitex.ExecOptions{Args: []any{gameID, turnNum, actor, order.Seq, order.Raw, order.Normalized, order.Status, order.Error}}); err != nil {
			return err
		}
	}

	return err
}

// GetOrders retrieves orders.
func (s *SQLiteStore) GetOrders(ctx context.Context, gameID string, turnNum int, actor string) ([]Order, error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var orders []Order
	err = sqlitex.Execute(conn, `
		SELECT seq, raw, normalized, status, error FROM orders
		WHERE game_id = ? AND turn_num = ? AND actor = ?
		ORDER BY seq
	`, &sqlitex.ExecOptions{
		Args: []any{gameID, turnNum, actor},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			orders = append(orders, Order{
				Seq:        stmt.ColumnInt(0),
				Raw:        stmt.ColumnText(1),
				Normalized: stmt.ColumnText(2),
				Status:     stmt.ColumnText(3),
				Error:      stmt.ColumnText(4),
			})
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// SaveReport saves a report.
func (s *SQLiteStore) SaveReport(ctx context.Context, gameID string, turnNum int, actor string, mime string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	conn, err := s.pool.Take(ctx)
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn, `
		INSERT OR REPLACE INTO report (game_id, turn_num, actor, mime, body) VALUES (?, ?, ?, ?, ?)
	`, &sqlitex.ExecOptions{Args: []any{gameID, turnNum, actor, mime, data}})
}

// GetReport retrieves a report.
func (s *SQLiteStore) GetReport(ctx context.Context, gameID string, turnNum int, actor string, mime string) (io.ReadCloser, error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var data []byte
	found := false
	err = sqlitex.Execute(conn, `
		SELECT body FROM report WHERE game_id = ? AND turn_num = ? AND actor = ? AND mime = ?
	`, &sqlitex.ExecOptions{
		Args: []any{gameID, turnNum, actor, mime},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			found = true
			data = make([]byte, stmt.ColumnLen(0))
			stmt.ColumnBytes(0, data)
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, cerrs.ErrNotImplemented
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

// GetSchemaVersion returns the current schema version, which is the number of
// applied migrations as tracked by PRAGMA user_version.
func (s *SQLiteStore) GetSchemaVersion(ctx context.Context) (string, error) {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return "", err
	}
	defer s.pool.Put(conn)

	var version int
	err = sqlitex.Execute(conn, "PRAGMA user_version;", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			version = stmt.ColumnInt(0)
			return nil
		},
	})
	if err != nil {
		return "", err
	}
	return strconv.Itoa(version), nil
}

// UpgradeSchema ensures all pending migrations have been applied. Migrations
// are applied automatically by sqlitemigration when the pool is opened, so this
// simply forces a connection acquisition to surface any migration error.
func (s *SQLiteStore) UpgradeSchema(ctx context.Context) error {
	conn, err := s.pool.Take(ctx)
	if err != nil {
		return errors.Join(cerrs.ErrSchemaUpgradeFailed, err)
	}
	s.pool.Put(conn)
	return nil
}

// Close closes the database.
func (s *SQLiteStore) Close() error {
	return s.pool.Close()
}
