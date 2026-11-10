// Package store persists Orbit state in SQLite (WAL). It is the single
// source of truth; the runtime is only observed (see reconciler).
package store

import (
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
	if _, err := db.Exec(migrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

const migrations = `
CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL);
INSERT INTO schema_version (version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_version);

CREATE TABLE IF NOT EXISTS instances (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL UNIQUE,
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
);
`
