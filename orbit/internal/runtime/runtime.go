// Package runtime defines the runtime adapter seam between the Orbit control
// plane and the execution environment (Docker today, QEMU/KVM later).
// Only this package and its implementations may import Docker.
package runtime

import (
	"context"
	"fmt"
	"strings"
)

// PortBinding publishes an instance port on the host.
type PortBinding struct {
	InstancePort int
	HostPort     int    // 0 = auto-allocate
	Bind         string // "loopback" (default) or "all"
}

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
	Network        string // bridge network the unit joins (empty = default)
	Volumes        []VolumeMount
	Ports          []PortBinding
}

// VolumeMount attaches a named data volume at a fixed destination.
type VolumeMount struct {
	Name string
	Dest string
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
	return s.validateMounts()
}

// validateMounts enforces security.md §8 gate 2: no host-path mounts, no
// docker socket, no sensitive host directories. Volumes are named volumes
// only; the API cannot express privileged/pid/network-host modes at all
// (rejected by construction — scenario 25).
func (s UnitSpec) validateMounts() error {
	banned := map[string]bool{
		"/": true, "/etc": true, "/proc": true, "/sys": true,
		"/var/run": true, "/var/run/docker.sock": true, "/run": true,
	}
	for _, v := range s.Volumes {
		if v.Name == "" || v.Dest == "" {
			return fmt.Errorf("volume mounts require name and destination")
		}
		if strings.HasPrefix(v.Name, "/") || strings.HasPrefix(v.Name, ".") {
			return fmt.Errorf("volume %q: host-path mounts are not permitted (named volumes only)", v.Name)
		}
		if banned[v.Dest] || strings.Contains(v.Dest, "docker.sock") ||
			strings.HasPrefix(v.Dest, "/proc") || strings.HasPrefix(v.Dest, "/sys") {
			return fmt.Errorf("volume %q: mounting %q is not permitted", v.Name, v.Dest)
		}
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
	// HostResources reports the capacity envelope (see Docker Desktop caveat
	// in the capacity package).
	HostResources(ctx context.Context) (cpus float64, memoryMB int, err error)
	// EnsureNetwork idempotently creates the bridge network with the given
	// subnet. Collision with an existing different subnet is an error.
	EnsureNetwork(ctx context.Context, name, subnet string) error
	// IPAddress reports the unit's IP on the named network.
	IPAddress(ctx context.Context, id, network string) (string, error)
	// EnsureVolume idempotently creates a named data volume.
	EnsureVolume(ctx context.Context, name string) error
	// RemoveVolume deletes a named data volume (missing = ok).
	RemoveVolume(ctx context.Context, name string) error
	// PortBindings reports resolved host ports for the unit's published
	// instance ports (host port 0 = auto-allocated by the runtime).
	PortBindings(ctx context.Context, id string) (map[int]int, error)
}
