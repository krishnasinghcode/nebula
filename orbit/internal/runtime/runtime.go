// Package runtime defines the runtime adapter seam between the Orbit control
// plane and the execution environment (Docker today, QEMU/KVM later).
// Only this package and its implementations may import Docker.
package runtime

import (
	"context"
	"fmt"
)

// UnitSpec describes a compute unit to create, in Orbit terms.
// It intentionally has no privileged/host-namespace knobs: the adapter type
// system makes those unrepresentable, and the Docker implementation enforces
// the hardening profile (docs/features/orbit/design/security.md §3.1).
type UnitSpec struct {
	Name           string
	Image          string
	Cmd            []string
	Labels         map[string]string
	Env            []string
	CPUs           float64 // cores
	MemoryMB       int
	PidsLimit      int64
	ReadOnlyRootFS bool
}

func (s UnitSpec) validate() error {
	if s.Name == "" || s.Image == "" {
		return fmt.Errorf("unit spec requires name and image")
	}
	if s.MemoryMB <= 0 {
		return fmt.Errorf("unit spec requires a positive memory limit (memory is never overcommitted)")
	}
	if s.CPUs <= 0 {
		return fmt.Errorf("unit spec requires a positive CPU limit")
	}
	return nil
}

// Status is the observed state of a unit.
type Status struct {
	State     string // "running", "exited", or "missing"
	ExitCode  int
	OOMKilled bool
	Error     string // runtime error detail, if any
}

// Adapter is the runtime seam. Implementations: Docker (Phase 1), Fake (tests), QEMU (future).
type Adapter interface {
	Ping(ctx context.Context) error
	Create(ctx context.Context, spec UnitSpec) (id string, err error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string, timeoutSecs int) error
	Remove(ctx context.Context, id string) error
	Inspect(ctx context.Context, id string) (Status, error)
}
