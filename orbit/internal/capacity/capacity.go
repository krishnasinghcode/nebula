// Package capacity implements the instance type catalogue, host capacity
// detection, admission control, and platform quotas
// (docs/features/orbit/design/technical-design.md §3, §12).
package capacity

import (
	"fmt"

	"orbit/internal/store"
)

// Type is a predefined instance flavour.
type Type struct {
	ID          string  `json:"id"`
	VCPU        float64 `json:"vcpu"`
	MemoryMB    int     `json:"memory_mb"`
	Description string  `json:"description"`
}

// DefaultTypes is the MVP catalogue (technical-design.md §3).
var DefaultTypes = []Type{
	{ID: "orbit.nano", VCPU: 0.5, MemoryMB: 512, Description: "0.5 vCPU / 512 MB"},
	{ID: "orbit.micro", VCPU: 1, MemoryMB: 1024, Description: "1 vCPU / 1 GB"},
	{ID: "orbit.small", VCPU: 2, MemoryMB: 2048, Description: "2 vCPU / 2 GB"},
}

// GetType returns a type by id or nil.
func GetType(id string) *Type {
	for i := range DefaultTypes {
		if DefaultTypes[i].ID == id {
			return &DefaultTypes[i]
		}
	}
	return nil
}

// Quotas are platform-wide defaults (technical-design.md §12).
// shortcut: types/quotas are code constants; making them config-file driven
// is deferred until a user actually needs to tune them (upgrade path: load
// from <data-dir>/config.json in the constructor).
var Quotas = struct {
	MaxInstances  int
	TotalMemoryMB int
	TotalVolumeMB int
	ReserveCPUs   float64 // kept for the host
	ReserveMemPct float64 // percent of detected memory kept for the host
}{MaxInstances: 5, TotalMemoryMB: 8192, TotalVolumeMB: 20480, ReserveCPUs: 1, ReserveMemPct: 25}

// VolumeEstimateMB is the accounting size charged per instance data volume.
// shortcut: Docker local volumes have no native size enforcement; we count
// a flat 1 GB estimate per instance against the 20 GB quota (technical-design
// §4 limitation). Real disk limits need per-instance disk images (P1).
const VolumeEstimateMB = 1024

// Host is the detected schedulable capacity envelope.
type Host struct {
	TotalCPUs      float64 `json:"total_cpus"`
	TotalMemoryMB  int     `json:"total_memory_mb"`
	SchedulableCPU float64 `json:"schedulable_cpus"`
	SchedulableMB  int     `json:"schedulable_memory_mb"`
	// Docker Desktop caveat: on macOS/Windows these are the Desktop VM's
	// resources, which is exactly the envelope that matters.
}

// Budget is the allocation ledger: what is used, free, and capped.
type Budget struct {
	Host          Host    `json:"host"`
	UsedCPUs      float64 `json:"used_cpus"`
	UsedMemoryMB  int     `json:"used_memory_mb"`
	Instances     int     `json:"instances"`
	MaxInstances  int     `json:"max_instances"`
	QuotaMemoryMB int     `json:"quota_memory_mb"`
	UsedVolumeMB  int     `json:"used_volume_mb"`
	QuotaVolumeMB int     `json:"quota_volume_mb"`
	Subnet        string  `json:"subnet"`
}

// FreeCPUs / FreeMB are the schedulable leftovers.
func (b Budget) FreeCPUs() float64 { return b.Host.SchedulableCPU - b.UsedCPUs }
func (b Budget) FreeMB() int       { return b.Host.SchedulableMB - b.UsedMemoryMB }

// Admit validates a request against the budget and quotas.
// CPU may be overcommitted (quotas, not reservations — like cloud bursting);
// memory may NOT (an OOM-killed instance teaches the wrong lesson).
func (b Budget) Admit(t Type, n int) error {
	if b.Instances+n > b.MaxInstances {
		return fmt.Errorf("quota exceeded: %d running + %d requested > max %d instances",
			b.Instances, n, b.MaxInstances)
	}
	if b.UsedMemoryMB+t.MemoryMB*n > b.QuotaMemoryMB {
		return &store.ConflictError{Message: fmt.Sprintf(
			"quota exceeded: %d MB used + %d MB requested > quota %d MB",
			b.UsedMemoryMB, t.MemoryMB*n, b.QuotaMemoryMB)}
	}
	if b.UsedVolumeMB+VolumeEstimateMB*n > b.QuotaVolumeMB {
		return &store.ConflictError{Message: fmt.Sprintf(
			"quota exceeded: %d MB of volumes used + %d MB requested > quota %d MB",
			b.UsedVolumeMB, VolumeEstimateMB*n, b.QuotaVolumeMB)}
	}
	if t.MemoryMB*n > b.FreeMB() {
		return &InsufficientCapacity{
			Requested: map[string]any{"memory_mb": t.MemoryMB * n},
			Available: map[string]any{"memory_mb": b.FreeMB()},
		}
	}
	// CPU overcommit allowed by design; only sanity-check against total.
	if b.UsedCPUs+t.VCPU*float64(n) > b.Host.TotalCPUs {
		return &InsufficientCapacity{
			Requested: map[string]any{"vcpu": t.VCPU * float64(n)},
			Available: map[string]any{"vcpus": b.Host.TotalCPUs},
		}
	}
	return nil
}

// InsufficientCapacity is a structured 422 error with requested-vs-available.
type InsufficientCapacity struct {
	Requested map[string]any
	Available map[string]any
}

func (e *InsufficientCapacity) Error() string {
	return fmt.Sprintf("insufficient capacity: requested %v, available %v", e.Requested, e.Available)
}
