package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// Scenario 29: fill the budget -> next create 422 insufficient_capacity
// with requested-vs-available numbers (fake host: 4 CPUs / 8192 MB).
// Four orbit.micro instances saturate the 4 vCPU envelope; the 5th trips
// the CPU check (instance quota allows 5, memory is fine).
func TestBudgetFillInsufficient(t *testing.T) {
	h := testAPI(t)
	mk := func(name string) map[string]any {
		return map[string]any{"name": name, "image_id": "ubuntu-24.04", "type_id": "orbit.micro", "key_id": "dev"}
	}
	for i := 1; i <= 4; i++ {
		if rec := do(t, h, "POST", "/v1/instances", mk(fmt.Sprintf("big-%d", i))); rec.Code != http.StatusAccepted {
			t.Fatalf("big-%d: %d (%s)", i, rec.Code, rec.Body.String())
		}
	}
	rec := do(t, h, "POST", "/v1/instances", mk("big-5"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("big-5: %d (%s), want 422", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("insufficient_capacity")) {
		t.Errorf("error code missing: %s", rec.Body.String())
	}
}

// Scenario 30: quota breach -> 422 quota_exceeded.
func TestQuotaBreach(t *testing.T) {
	h := testAPI(t)
	// Max 5 instances.
	for i := 1; i <= 5; i++ {
		body := map[string]any{"name": fmt.Sprintf("q-%d", i), "image_id": "ubuntu-24.04",
			"type_id": "orbit.nano", "key_id": "dev"}
		if rec := do(t, h, "POST", "/v1/instances", body); rec.Code != http.StatusAccepted {
			t.Fatalf("q-%d: %d (%s)", i, rec.Code, rec.Body.String())
		}
	}
	rec := do(t, h, "POST", "/v1/instances",
		map[string]any{"name": "q-6", "image_id": "ubuntu-24.04", "type_id": "orbit.nano", "key_id": "dev"})
	if rec.Code != http.StatusUnprocessableEntity || !bytes.Contains(rec.Body.Bytes(), []byte("quota")) {
		t.Errorf("6th: %d %s", rec.Code, rec.Body.String())
	}
}

// Scenario 31: capacity invariant — used memory + free = schedulable budget.
func TestCapacityInvariant(t *testing.T) {
	h := testAPI(t)
	rec := do(t, h, "POST", "/v1/instances", map[string]any{"name": "inv-1", "image_id": "ubuntu-24.04", "type_id": "orbit.micro", "key_id": "dev"})
	for i := 0; i < 100; i++ {
		rec = do(t, h, "GET", "/v1/capacity", nil)
		if bytes.Contains(rec.Body.Bytes(), []byte(`"instances":1`)) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	var b struct {
		Host struct {
			TotalCPUs     float64 `json:"total_cpus"`
			TotalMemoryMB int     `json:"total_memory_mb"`
		} `json:"host"`
		UsedCPUs     float64 `json:"used_cpus"`
		UsedMemoryMB int     `json:"used_memory_mb"`
		Instances    int     `json:"instances"`
		MaxInstances int     `json:"max_instances"`
	}
	json.Unmarshal(rec.Body.Bytes(), &b)
	if b.Instances != 1 || b.UsedMemoryMB != 1024 || b.UsedCPUs != 1 {
		t.Fatalf("ledger mismatch: %+v", b)
	}
	if b.UsedMemoryMB > b.Host.TotalMemoryMB {
		t.Errorf("used %d > host total %d", b.UsedMemoryMB, b.Host.TotalMemoryMB)
	}
}
