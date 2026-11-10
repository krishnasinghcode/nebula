package capacity

import (
	"context"
	"fmt"
	"math"

	"orbit/internal/store"
)

// Detector reads host resources from the runtime adapter.
type Detector interface {
	HostResources(ctx context.Context) (cpus float64, memoryMB int, err error)
}

// Manager computes budgets and performs admission.
type Manager struct {
	Detector Detector
	St       *store.Store
}

// Budget computes the current budget: host envelope minus reserve, minus
// allocations of live instances.
func (m *Manager) Budget(ctx context.Context) (Budget, error) {
	totalCPUs, totalMB, err := m.Detector.HostResources(ctx)
	if err != nil {
		return Budget{}, fmt.Errorf("detect host resources: %w", err)
	}
	b := Budget{
		Host: Host{
			TotalCPUs:      totalCPUs,
			TotalMemoryMB:  totalMB,
			SchedulableCPU: math.Max(totalCPUs-Quotas.ReserveCPUs, 0.5),
			SchedulableMB:  max(totalMB-int(float64(totalMB)*Quotas.ReserveMemPct/100), 512),
		},
		MaxInstances:  Quotas.MaxInstances,
		QuotaMemoryMB: Quotas.TotalMemoryMB,
	}
	live, err := m.St.ListInstances("")
	if err != nil {
		return b, err
	}
	for _, in := range live {
		if store.Terminal(in.State) {
			continue
		}
		t := GetType(in.TypeID)
		if t == nil {
			continue // unknown type (legacy record): ignore in ledger
		}
		b.Instances++
		b.UsedCPUs += t.VCPU
		b.UsedMemoryMB += t.MemoryMB
	}
	return b, nil
}

// Admit checks a create request of n instances of type t.
func (m *Manager) Admit(ctx context.Context, typeID string, n int) error {
	t := GetType(typeID)
	if t == nil {
		return fmt.Errorf("unknown instance type %q", typeID)
	}
	budget, err := m.Budget(ctx)
	if err != nil {
		return err
	}
	return budget.Admit(*t, n)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
