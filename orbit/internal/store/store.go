// Package store persists Orbit state in SQLite (WAL). It is the single
// source of truth; the runtime is only observed (see reconciler).
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

// Open opens (and migrates) the database at path.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// migrate applies schema migrations in order, skipping any already recorded
// in schema_version. Each migration runs in a transaction (SQLite DDL is
// transactional, so a crash mid-migration rolls back cleanly).
func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return err
	}
	var cur int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&cur); err != nil {
		return err
	}
	for i, m := range migrations {
		v := i + 1
		if v <= cur {
			continue
		}
		conn, err := db.Conn(context.Background())
		if err != nil {
			return err
		}
		// The v2 rebuild drops and renames tables; FK enforcement would
		// reject the swap. Disabled on this one connection only.
		if _, err := conn.ExecContext(context.Background(), `PRAGMA foreign_keys=OFF`); err != nil {
			conn.Close()
			return err
		}
		tx, err := conn.BeginTx(context.Background(), nil)
		if err != nil {
			conn.Close()
			return err
		}
		if _, err := tx.Exec(m); err != nil {
			tx.Rollback() //nolint:errcheck
			conn.Close()
			return fmt.Errorf("migration %d: %w", v, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES (?)`, v); err != nil {
			tx.Rollback() //nolint:errcheck
			conn.Close()
			return err
		}
		if err := tx.Commit(); err != nil {
			conn.Close()
			return err
		}
		conn.Close()
	}
	return nil
}

var migrations = []string{
	// v1: initial schema. Instance names are unique among non-terminated
	// instances only (partial index below); terminated names may be reused.
	`CREATE TABLE IF NOT EXISTS instances (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  state         TEXT NOT NULL,
  image_id      TEXT NOT NULL,
  type_id       TEXT NOT NULL,
  key_id        TEXT NOT NULL,
  user_data     TEXT NOT NULL DEFAULT '',
  private_ip    TEXT NOT NULL DEFAULT '',
  ports         TEXT NOT NULL DEFAULT '[]',
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL,
  transition_error TEXT NOT NULL DEFAULT '',
  owner         TEXT NOT NULL DEFAULT 'local',
  root_volume_id TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_live_name ON instances(name) WHERE state != 'terminated';

CREATE TABLE IF NOT EXISTS instance_events (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  instance_id TEXT NOT NULL REFERENCES instances(id),
  type        TEXT NOT NULL,
  message     TEXT NOT NULL DEFAULT '',
  occurred_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_instance ON instance_events(instance_id);

CREATE TABLE IF NOT EXISTS ssh_keys (
  name        TEXT PRIMARY KEY,
  public_key  TEXT NOT NULL,
  fingerprint TEXT NOT NULL,
  created_at  TEXT NOT NULL
);`,
	// v2: rebuild instances to drop the original table-level UNIQUE on name,
	// which blocked reuse of terminated names (see v1 partial index).
	`CREATE TABLE IF NOT EXISTS instances_rebuild (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  state         TEXT NOT NULL,
  image_id      TEXT NOT NULL,
  type_id       TEXT NOT NULL,
  key_id        TEXT NOT NULL,
  user_data     TEXT NOT NULL DEFAULT '',
  private_ip    TEXT NOT NULL DEFAULT '',
  ports         TEXT NOT NULL DEFAULT '[]',
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL,
  transition_error TEXT NOT NULL DEFAULT '',
  owner         TEXT NOT NULL DEFAULT 'local',
  root_volume_id TEXT NOT NULL DEFAULT ''
);
INSERT INTO instances_rebuild (id, name, state, image_id, type_id, key_id, user_data,
  private_ip, ports, created_at, updated_at, transition_error, owner, root_volume_id)
SELECT id, name, state, image_id, type_id, key_id, user_data,
  private_ip, ports, created_at, updated_at, transition_error, owner, root_volume_id
FROM instances;
CREATE TABLE IF NOT EXISTS events_rebuild (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  instance_id TEXT NOT NULL REFERENCES instances(id),
  type        TEXT NOT NULL,
  message     TEXT NOT NULL DEFAULT '',
  occurred_at TEXT NOT NULL
);
INSERT INTO events_rebuild (id, instance_id, type, message, occurred_at)
SELECT id, instance_id, type, message, occurred_at FROM instance_events;
DROP TABLE instance_events;
DROP TABLE instances;
ALTER TABLE instances_rebuild RENAME TO instances;
ALTER TABLE events_rebuild RENAME TO instance_events;
CREATE INDEX IF NOT EXISTS idx_events_instance ON instance_events(instance_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_live_name ON instances(name) WHERE state != 'terminated';`,
}
