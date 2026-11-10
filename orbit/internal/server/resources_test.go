package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// Scenario 29: fill the schedulable budget -> next create 422
// insufficient_capacity with requested-vs-available numbers.
func TestBudgetFillInsufficient(t *testing.T) {
	h := testAPI(t)
	// Fake host defaults: 4 CPUs / 8192 MB, schedulable 3 / 6144 MB.
	mk := func(name string) map[string]any {
		return map[string]any{"name": name, "image_id": "ubuntu-24.04", "type_id": "orbit.small", "key_id": "dev"}
	}
	rec := do(t, h, "POST", "/v1/instances", mk("big-1")) // 2 GB
	if rec.Code != http.StatusAccepted {
		t.Fatalf("big-1: %d (%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, h, "POST", "/v1/instances", mk("big-2")) // 2 GB -> 4 GB used
	if rec.Code != http.StatusAccepted {
		t.Fatalf("big-2: %d (%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, h, "POST", "/v1/instances", mk("big-3")) // +2 GB = 6 GB = schedulable
	if rec.Code != http.StatusAccepted {
		t.Fatalf("big-3: %d (%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, h, "POST", "/v1/instances", mk("big-4")) // 8 GB > 6 GB schedulable
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("big-4: %d (%s), want 422", rec.Code, rec.Body.String())
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
		SchedulableCPU float64 `json:"schedulable_cpus"`
		SchedulableMB  int     `json:"schedulable_memory_mb"`
		UsedCPUs       float64 `json:"used_cpus"`
		UsedMemoryMB   int     `json:"used_memory_mb"`
		Instances      int     `json:"instances"`
		MaxInstances   int     `json:"max_instances"`
	}
	json.Unmarshal(rec.Body.Bytes(), &b)
	if b.Instances != 1 || b.UsedMemoryMB != 1024 || b.UsedCPUs != 1 {
		t.Fatalf("ledger mismatch: %+v", b)
	}
	if b.UsedMemoryMB > b.SchedulableMB {
		t.Errorf("used %d > schedulable %d", b.UsedMemoryMB, b.SchedulableMB)
	}
}
