package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// NewID returns a fresh instance id.
func NewID() string {
	b := make([]byte, 10)
	rand.Read(b)
	return "orb-" + hex.EncodeToString(b)
}

func scanInstance(row interface{ Scan(...any) error }) (*Instance, error) {
	var in Instance
	var portsJSON string
	var created, updated string
	if err := row.Scan(&in.ID, &in.Name, &in.State, &in.ImageID, &in.TypeID, &in.KeyID,
		&in.UserData, &in.PrivateIP, &portsJSON, &created, &updated,
		&in.TransitionError, &in.Owner, &in.RootVolumeID); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(portsJSON), &in.Ports); err != nil && portsJSON != "[]" {
		return nil, fmt.Errorf("decode ports for %s: %w", in.ID, err)
	}
	in.CreatedAt, _ = time.Parse(time.RFC3339, created)
	in.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &in, nil
}

const instanceCols = `id, name, state, image_id, type_id, key_id, user_data, private_ip, ports,
created_at, updated_at, transition_error, owner, root_volume_id`

// CreateInstance inserts an instance in its initial state.
func (s *Store) CreateInstance(in *Instance) error {
	now := time.Now().UTC().Format(time.RFC3339)
	in.CreatedAt, in.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	ports, _ := json.Marshal(in.Ports)
	_, err := s.db.Exec(
		`INSERT INTO instances (`+instanceCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		in.ID, in.Name, in.State, in.ImageID, in.TypeID, in.KeyID,
		in.UserData, in.PrivateIP, string(ports), now, now,
		in.TransitionError, in.Owner, in.RootVolumeID,
	)
	if err != nil {
		return fmt.Errorf("insert instance: %w", err)
	}
	return nil
}

// GetInstance returns one instance or nil if not found.
func (s *Store) GetInstance(id string) (*Instance, error) {
	row := s.db.QueryRow(`SELECT `+instanceCols+` FROM instances WHERE id=?`, id)
	in, err := scanInstance(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return in, err
}

// GetInstanceByName returns one non-terminated instance by name or nil.
func (s *Store) GetInstanceByName(name string) (*Instance, error) {
	row := s.db.QueryRow(
		`SELECT `+instanceCols+` FROM instances WHERE name=? AND state != ?`, name, StateTerminated)
	in, err := scanInstance(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return in, err
}

// ListInstances returns instances, optionally filtered by state.
func (s *Store) ListInstances(state string) ([]*Instance, error) {
	q := `SELECT ` + instanceCols + ` FROM instances`
	args := []any{}
	if state != "" {
		q += ` WHERE state=?`
		args = append(args, state)
	}
	q += ` ORDER BY created_at`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Instance
	for rows.Next() {
		in, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

// Transition updates instance state and appends an event in one transaction.
// It fails if the current state does not permit the transition.
func (s *Store) Transition(id, to, eventType, message string) (*Instance, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var cur string
	if err := tx.QueryRow(`SELECT state FROM instances WHERE id=?`, id).Scan(&cur); err != nil {
		return nil, err
	}
	if !CanTransition(cur, to) {
		return nil, &ConflictError{Message: fmt.Sprintf("instance %s: transition %s→%s not permitted", id, cur, to)}
	}
	if _, err := tx.Exec(`UPDATE instances SET state=?, updated_at=? WHERE id=?`, to, now, id); err != nil {
		return nil, err
	}
	if to == StateFailed {
		// Surface the failure reason on the record (describe shows it).
		if _, err := tx.Exec(`UPDATE instances SET transition_error=? WHERE id=?`, message, id); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO instance_events (instance_id, type, message, occurred_at) VALUES (?,?,?,?)`,
		id, eventType, message, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetInstance(id)
}

// SetInstancePorts updates the resolved port mappings.
func (s *Store) SetInstancePorts(id string, ports []Port) error {
	b, err := json.Marshal(ports)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE instances SET ports=?, updated_at=? WHERE id=?`,
		string(b), time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// SetInstanceRootVolume records the named data volume after provisioning.
func (s *Store) SetInstanceRootVolume(id, volumeID string) error {
	_, err := s.db.Exec(`UPDATE instances SET root_volume_id=?, updated_at=? WHERE id=?`,
		volumeID, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// SetInstanceFields updates mutable metadata (private_ip, transition_error).
func (s *Store) SetInstanceFields(id string, privateIP, transitionError string) error {
	_, err := s.db.Exec(`UPDATE instances SET private_ip=?, transition_error=?, updated_at=? WHERE id=?`,
		privateIP, transitionError, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// AddEvent appends a lifecycle event outside a state transition
// (e.g. the initial create event).
func (s *Store) AddEvent(instanceID, eventType, message string) error {
	_, err := s.db.Exec(
		`INSERT INTO instance_events (instance_id, type, message, occurred_at) VALUES (?,?,?,?)`,
		instanceID, eventType, message, time.Now().UTC().Format(time.RFC3339))
	return err
}

// Events returns an instance's lifecycle events, newest last.
func (s *Store) Events(instanceID string) ([]Event, error) {
	rows, err := s.db.Query(
		`SELECT id, instance_id, type, message, occurred_at FROM instance_events
		 WHERE instance_id=? ORDER BY id`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var at string
		if err := rows.Scan(&e.ID, &e.InstanceID, &e.Type, &e.Message, &at); err != nil {
			return nil, err
		}
		e.OccurredAt, _ = time.Parse(time.RFC3339, at)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ConflictError signals an illegal state transition or duplicate/in-use resource.
type ConflictError struct{ Message string }

func (e *ConflictError) Error() string { return e.Message }
