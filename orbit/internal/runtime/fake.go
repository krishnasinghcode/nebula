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
	nextPort  int   // deterministic auto port allocation
	nextIP    int   // deterministic fake private IPs (172.28.0.x)
	NetName   string
	NetSubnet string
	NetErr    error // scripted EnsureNetwork failure
	ips       map[string]string
	volumes   map[string]bool
	HostCPUs  float64
	HostMemMB int
}

type fakeUnit struct {
	spec     UnitSpec
	running  bool
	exited   bool
	removed  bool
	exitCode int
	oom      bool
	ports    map[int]int
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
	f.units[spec.Name] = &fakeUnit{spec: spec, running: false, ports: map[int]int{}}
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
	// Resolve auto host ports deterministically (fake: 31001, 31002, ...).
	for _, p := range u.spec.Ports {
		if p.HostPort == 0 {
			f.nextPort++
			u.ports[p.InstancePort] = 31000 + f.nextPort
		} else {
			u.ports[p.InstancePort] = p.HostPort
		}
	}
	// Assign a private IP on the unit's network (172.28.0.2, .3, ...).
	if u.spec.Network != "" {
		f.nextIP++
		if f.ips == nil {
			f.ips = map[string]string{}
		}
		f.ips[id] = fmt.Sprintf("172.28.0.%d", f.nextIP+1)
	}
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

// HostResources reports the fake host envelope (tests configure it).
func (f *FakeAdapter) HostResources(context.Context) (float64, int, error) {
	if f.HostCPUs == 0 {
		return 4, 8192, nil // test defaults
	}
	return f.HostCPUs, f.HostMemMB, nil
}

// PortBindings reports resolved host ports for published instance ports.
func (f *FakeAdapter) PortBindings(_ context.Context, id string) (map[int]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.units[id]
	if !ok {
		return nil, fmt.Errorf("unit %q: %w", id, ErrMissing)
	}
	out := map[int]int{}
	for k, v := range u.ports {
		out[k] = v
	}
	return out, nil
}

// EnsureNetwork records the network (idempotent like the Docker path).
func (f *FakeAdapter) EnsureNetwork(_ context.Context, name, subnet string) error {
	if f.NetErr != nil {
		return f.NetErr
	}
	if f.NetName != "" && f.NetName != name {
		return fmt.Errorf("network %q already exists as %q", name, f.NetName)
	}
	f.NetName, f.NetSubnet = name, subnet
	return nil
}

// EnsureVolume idempotently creates a named volume.
func (f *FakeAdapter) EnsureVolume(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.volumes == nil {
		f.volumes = map[string]bool{}
	}
	f.volumes[name] = true
	return nil
}

// RemoveVolume deletes a named volume (missing = ok).
func (f *FakeAdapter) RemoveVolume(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.volumes, name)
	return nil
}

// HasVolume reports whether a volume exists (test assertions).
func (f *FakeAdapter) HasVolume(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.volumes[name]
}

// IPAddress reports the fake unit's deterministic address (172.28.0.2, .3, ...).
func (f *FakeAdapter) IPAddress(_ context.Context, id, networkName string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.units[id]
	if !ok {
		return "", fmt.Errorf("unit %q: %w", id, ErrMissing)
	}
	if networkName != u.spec.Network {
		return "", fmt.Errorf("unit %s not on network %s", id, networkName)
	}
	if !u.running {
		return "", nil
	}
	return f.ips[id], nil
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

// UnitSpecPorts exposes the created unit's port spec (test assertions).
func (f *FakeAdapter) UnitSpecPorts(id string) []PortBinding {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.units[id]; ok {
		return u.spec.Ports
	}
	return nil
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
