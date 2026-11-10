package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// FakeAdapter is an in-memory adapter with scriptable failures for
// integration tests (no Docker required).
type FakeAdapter struct {
	mu        sync.Mutex
	units     map[string]*fakeUnit
	CreateErr error // returned by every Create
	StartErr  error // returned by every Start
	OpsErr    error // returned by Stop/Remove/Inspect
}

type fakeUnit struct {
	spec     UnitSpec
	running  bool
	exited   bool
	removed  bool
	exitCode int
	oom      bool
}

func NewFake() *FakeAdapter {
	return &FakeAdapter{units: map[string]*fakeUnit{}}
}

func (f *FakeAdapter) Ping(context.Context) error { return nil }

func (f *FakeAdapter) Create(_ context.Context, spec UnitSpec) (string, error) {
	if err := spec.validate(); err != nil {
		return "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.CreateErr != nil {
		return "", f.CreateErr
	}
	if _, ok := f.units[spec.Name]; ok {
		return "", fmt.Errorf("unit %q already exists", spec.Name)
	}
	f.units[spec.Name] = &fakeUnit{spec: spec, running: false}
	return spec.Name, nil
}

func (f *FakeAdapter) Start(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.StartErr != nil {
		return f.StartErr
	}
	u, ok := f.units[id]
	if !ok {
		return fmt.Errorf("unit %q: %w", id, ErrMissing)
	}
	u.running, u.exited = true, false
	return nil
}

func (f *FakeAdapter) Stop(_ context.Context, id string, _ int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.OpsErr != nil {
		return f.OpsErr
	}
	u, ok := f.units[id]
	if !ok {
		return fmt.Errorf("unit %q: %w", id, ErrMissing)
	}
	u.running, u.exited = false, true
	return nil
}

func (f *FakeAdapter) Remove(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.OpsErr != nil {
		return f.OpsErr
	}
	u, ok := f.units[id]
	if !ok {
		return fmt.Errorf("unit %q: %w", id, ErrMissing)
	}
	u.removed = true
	delete(f.units, id)
	return nil
}

func (f *FakeAdapter) Inspect(_ context.Context, id string) (Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.OpsErr != nil {
		return Status{}, f.OpsErr
	}
	u, ok := f.units[id]
	if !ok {
		return Status{State: "missing"}, nil
	}
	switch {
	case u.running:
		return Status{State: "running"}, nil
	case u.exited:
		return Status{State: "exited", ExitCode: u.exitCode, OOMKilled: u.oom}, nil
	default:
		return Status{State: "exited", ExitCode: u.exitCode}, nil
	}
}

// Scripted helpers for tests.

// Exit simulates the unit's main process exiting.
func (f *FakeAdapter) Exit(id string, exitCode int, oom bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.units[id]; ok {
		u.running, u.exited, u.exitCode, u.oom = false, true, exitCode, oom
	}
}

// Exists reports whether a unit is present (for leak checks).
func (f *FakeAdapter) Exists(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.units[id]
	return ok
}

// ErrMissing is returned when a unit does not exist.
var ErrMissing = errors.New("unit not found")
