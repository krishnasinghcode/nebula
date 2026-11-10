package store

import (
	"database/sql"
	"fmt"
	"time"
)

// SSHKey is a registered public key. Private keys never reach the server.
type SSHKey struct {
	Name        string    `json:"name"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateSSHKey registers a key.
func (s *Store) CreateSSHKey(k *SSHKey) error {
	k.CreatedAt = time.Now().UTC()
	_, err := s.db.Exec(`INSERT INTO ssh_keys (name, public_key, fingerprint, created_at) VALUES (?,?,?,?)`,
		k.Name, k.PublicKey, k.Fingerprint, k.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert ssh key: %w", err)
	}
	return nil
}

// GetSSHKey returns one key or nil.
func (s *Store) GetSSHKey(name string) (*SSHKey, error) {
	row := s.db.QueryRow(`SELECT name, public_key, fingerprint, created_at FROM ssh_keys WHERE name=?`, name)
	var k SSHKey
	var at string
	if err := row.Scan(&k.Name, &k.PublicKey, &k.Fingerprint, &at); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	k.CreatedAt, _ = time.Parse(time.RFC3339, at)
	return &k, nil
}

// ListSSHKeys returns all registered keys.
func (s *Store) ListSSHKeys() ([]SSHKey, error) {
	rows, err := s.db.Query(`SELECT name, public_key, fingerprint, created_at FROM ssh_keys ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SSHKey
	for rows.Next() {
		var k SSHKey
		var at string
		if err := rows.Scan(&k.Name, &k.PublicKey, &k.Fingerprint, &at); err != nil {
			return nil, err
		}
		k.CreatedAt, _ = time.Parse(time.RFC3339, at)
		out = append(out, k)
	}
	return out, rows.Err()
}

// DeleteSSHKey removes a key unless an instance references it.
func (s *Store) DeleteSSHKey(name string) error {
	var n int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM instances WHERE key_id=? AND state != ?`, name, StateTerminated).
		Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return &ConflictError{Message: fmt.Sprintf("key %q is referenced by %d instance(s)", name, n)}
	}
	_, err := s.db.Exec(`DELETE FROM ssh_keys WHERE name=?`, name)
	return err
}
