package capacity

import (
	"errors"
	"testing"

	"orbit/internal/store"
)

func TestAdmitWithinCapacity(t *testing.T) {
	b := Budget{
		Host:         Host{TotalCPUs: 4, TotalMemoryMB: 8192, SchedulableCPU: 3, SchedulableMB: 6144},
		MaxInstances: 5, QuotaMemoryMB: 8192, QuotaVolumeMB: Quotas.TotalVolumeMB,
	}
	micro := Type{ID: "orbit.micro", VCPU: 1, MemoryMB: 1024}
	if err := b.Admit(micro, 1); err != nil {
		t.Fatalf("admit within capacity: %v", err)
	}
}

func TestAdmitInsufficientMemory(t *testing.T) {
	b := Budget{
		Host:         Host{TotalCPUs: 4, TotalMemoryMB: 2048, SchedulableCPU: 3, SchedulableMB: 1024},
		MaxInstances: 5, QuotaMemoryMB: 8192, QuotaVolumeMB: Quotas.TotalVolumeMB,
	}
	small := Type{ID: "orbit.small", VCPU: 2, MemoryMB: 2048}
	err := b.Admit(small, 1)
	var ice *InsufficientCapacity
	if !errors.As(err, &ice) {
		t.Fatalf("want InsufficientCapacity, got %v", err)
	}
	if ice.Available["memory_mb"] != 1024 {
		t.Errorf("available: %v", ice.Available)
	}
}

func TestAdmitMemoryQuota(t *testing.T) {
	b := Budget{
		Host:         Host{TotalCPUs: 16, TotalMemoryMB: 65536, SchedulableCPU: 15, SchedulableMB: 49152},
		MaxInstances: 5, QuotaMemoryMB: 8192, QuotaVolumeMB: Quotas.TotalVolumeMB,
	}
	small := Type{ID: "orbit.small", VCPU: 2, MemoryMB: 2048}
	err := b.Admit(small, 5) // 5*2048 = 10240 > 8192 quota
	var qe *store.ConflictError
	if !errors.As(err, &qe) {
		t.Fatalf("want quota ConflictError, got %v", err)
	}
}

func TestAdmitInstanceQuota(t *testing.T) {
	b := Budget{
		Host:      Host{TotalCPUs: 16, TotalMemoryMB: 65536, SchedulableCPU: 15, SchedulableMB: 49152},
		Instances: 5, MaxInstances: 5, QuotaMemoryMB: 8192, QuotaVolumeMB: Quotas.TotalVolumeMB,
	}
	nano := Type{ID: "orbit.nano", VCPU: 0.5, MemoryMB: 512}
	if err := b.Admit(nano, 1); err == nil {
		t.Fatal("6th instance admitted; want quota error")
	}
}

func TestAdmitVolumeQuota(t *testing.T) {
	b := Budget{
		Host:         Host{TotalCPUs: 16, TotalMemoryMB: 65536, SchedulableCPU: 15, SchedulableMB: 49152},
		UsedVolumeMB: Quotas.TotalVolumeMB, // quota already exhausted
		MaxInstances: 5, QuotaMemoryMB: 65536, QuotaVolumeMB: Quotas.TotalVolumeMB,
	}
	micro := Type{ID: "orbit.micro", VCPU: 1, MemoryMB: 1024}
	var qe *store.ConflictError
	if err := b.Admit(micro, 1); !errors.As(err, &qe) {
		t.Fatalf("volume quota not enforced: %v", err)
	}
}

func TestAdmitCPUOvercommitAllowed(t *testing.T) {
	// CPU overcommit allowed up to TOTAL cpus (not schedulable) — quotas, not reservations.
	b := Budget{
		Host:     Host{TotalCPUs: 4, TotalMemoryMB: 16384, SchedulableCPU: 3, SchedulableMB: 12288},
		UsedCPUs: 2.5, MaxInstances: 5, QuotaMemoryMB: 16384, QuotaVolumeMB: Quotas.TotalVolumeMB,
	}
	micro := Type{ID: "orbit.micro", VCPU: 1, MemoryMB: 1024}
	if err := b.Admit(micro, 1); err != nil {
		t.Fatalf("cpu overcommit rejected: %v", err)
	}
	// But beyond total physical cores -> insufficient.
	small := Type{ID: "orbit.small", VCPU: 2, MemoryMB: 1024}
	if err := b.Admit(small, 1); err == nil {
		t.Fatal("3.5 vCPU on 4-core host with 2.5 used admitted beyond total")
	}
}
