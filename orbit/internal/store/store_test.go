package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func mkInst(name, state string) *Instance {
	return &Instance{
		ID: NewID(), Name: name, State: state,
		ImageID: "ubuntu-24.04", TypeID: "orbit.micro", KeyID: "k",
	}
}

// Name uniqueness applies only to non-terminated instances; a terminated
// name must be reusable.
func TestNameReuseAfterTerminate(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	in := mkInst("web-1", StateProvisioning)
	if err := s.CreateInstance(in); err != nil {
		t.Fatal(err)
	}
	// Duplicate live name conflicts.
	var ce *ConflictError
	err = s.CreateInstance(mkInst("web-1", StateProvisioning))
	if !errors.As(err, &ce) {
		t.Fatalf("duplicate live name: got %v, want ConflictError", err)
	}
	// Terminate, then the name is free again.
	if _, err := s.Transition(in.ID, StateTerminated, "terminate", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateInstance(mkInst("web-1", StateProvisioning)); err != nil {
		t.Fatalf("reuse of terminated name: %v", err)
	}
}

// A database created before v2 (table-level UNIQUE on name) migrates without
// losing rows, and terminated names become reusable in it.
func TestLegacySchemaMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	// Old v1 schema: UNIQUE on the column itself.
	if _, err := db.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL);
INSERT INTO schema_version VALUES (1);
CREATE TABLE instances (
  id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, state TEXT NOT NULL,
  image_id TEXT NOT NULL, type_id TEXT NOT NULL, key_id TEXT NOT NULL,
  user_data TEXT NOT NULL DEFAULT '', private_ip TEXT NOT NULL DEFAULT '',
  ports TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
  transition_error TEXT NOT NULL DEFAULT '', owner TEXT NOT NULL DEFAULT 'local',
  root_volume_id TEXT NOT NULL DEFAULT ''
);
CREATE TABLE instance_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  instance_id TEXT NOT NULL REFERENCES instances(id),
  type TEXT NOT NULL, message TEXT NOT NULL DEFAULT '', occurred_at TEXT NOT NULL
);
CREATE TABLE ssh_keys (
  name TEXT PRIMARY KEY, public_key TEXT NOT NULL,
  fingerprint TEXT NOT NULL, created_at TEXT NOT NULL
);
INSERT INTO instances (id, name, state, image_id, type_id, key_id, created_at, updated_at)
VALUES ('orb-old', 'potato', 'terminated', 'ubuntu-24.04', 'orbit.micro', 'k', 't', 't');
INSERT INTO instance_events (instance_id, type, occurred_at) VALUES ('orb-old', 'create', 't');`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Old rows survive.
	old, err := s.GetInstance("orb-old")
	if err != nil || old == nil || old.State != StateTerminated {
		t.Fatalf("legacy row lost: %v %+v", err, old)
	}
	if n, _ := s.GetInstance("potato"); n != nil {
		t.Fatal("GetByName must ignore terminated instances")
	}
	// The terminated name is reusable in the migrated DB.
	if err := s.CreateInstance(mkInst("potato", StateProvisioning)); err != nil {
		t.Fatalf("reuse of terminated name after migration: %v", err)
	}
	// A live duplicate still conflicts.
	var ce *ConflictError
	err = s.CreateInstance(mkInst("potato", StateProvisioning))
	if !errors.As(err, &ce) {
		t.Fatalf("duplicate live name after migration: got %v, want ConflictError", err)
	}
}
