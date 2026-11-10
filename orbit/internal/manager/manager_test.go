package manager

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"orbit/internal/runtime"
	"orbit/internal/store"
)

func testManager(t *testing.T) (*Manager, *runtime.FakeAdapter, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	// Seed the key the sample instances reference.
	if err := st.CreateSSHKey(&store.SSHKey{
		Name: "dev", PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB9mXhE0+7T0mKc1V0T7XyO9mE0samplekeyfortests dev",
		Fingerprint: "SHA256:test",
	}); err != nil {
		t.Fatal(err)
	}
	f := runtime.NewFake()
	return New(st, f), f, st
}

func sample() *store.Instance {
	return &store.Instance{Name: "web-1", ImageID: "ubuntu-24.04", TypeID: "orbit.micro", KeyID: "dev"}
}

func waitFor(t *testing.T, m *Manager, id, state string) *store.Instance {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		in, err := m.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if in.State == state {
			return in
		}
		if time.Now().After(deadline) {
			t.Fatalf("instance %s: state %q, want %q (err: %q)", id, in.State, state, in.TransitionError)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCreateProvisionsAsync(t *testing.T) {
	m, _, _ := testManager(t)
	ctx := context.Background()
	in, existing, err := m.Create(ctx, sample())
	if err != nil || existing {
		t.Fatalf("create: existing=%v err=%v", existing, err)
	}
	in = waitFor(t, m, in.ID, store.StateRunning)
	if in.ID == "" || in.Name != "web-1" {
		t.Errorf("unexpected record: %+v", in)
	}
}

func TestCreateIdempotentByName(t *testing.T) {
	m, _, _ := testManager(t)
	ctx := context.Background()
	first, _, err := m.Create(ctx, sample())
	if err != nil {
		t.Fatal(err)
	}
	dup, existing, err := m.Create(ctx, sample())
	if err != nil || !existing {
		t.Fatalf("duplicate create: existing=%v err=%v", existing, err)
	}
	if dup.ID != first.ID {
		t.Errorf("duplicate returned new id %s, want %s", dup.ID, first.ID)
	}
}

func TestFullLifecycle(t *testing.T) {
	m, f, _ := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	waitFor(t, m, in.ID, store.StateRunning)

	if _, err := m.Action(ctx, in.ID, "stop"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	waitFor(t, m, in.ID, store.StateStopped)

	if _, err := m.Action(ctx, in.ID, "start"); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitFor(t, m, in.ID, store.StateRunning)

	if _, err := m.Action(ctx, in.ID, "reboot"); err != nil {
		t.Fatalf("reboot: %v", err)
	}
	waitFor(t, m, in.ID, store.StateRunning)

	if _, err := m.Action(ctx, in.ID, "terminate"); err != nil {
		t.Fatalf("terminate: %v", err)
	}
	waitFor(t, m, in.ID, store.StateTerminated)
	if f.Exists(in.ID) {
		t.Error("runtime unit not removed after terminate")
	}
}

func TestEventsAppended(t *testing.T) {
	m, _, _ := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	waitFor(t, m, in.ID, store.StateRunning)
	events, err := m.Events(in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 2 { // create + provisioning->running
		t.Errorf("want >=2 events, got %d", len(events))
	}
}

func TestCrashDetectedByReconciler(t *testing.T) {
	m, f, _ := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	waitFor(t, m, in.ID, store.StateRunning)

	f.Exit(in.ID, 1, false) // crash
	m.Reconcile(ctx)
	in = waitFor(t, m, in.ID, store.StateFailed)
	if in.TransitionError == "" {
		t.Error("failed without reason")
	}
}

func TestOOMFail(t *testing.T) {
	m, f, _ := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	waitFor(t, m, in.ID, store.StateRunning)
	f.Exit(in.ID, 137, true)
	m.Reconcile(ctx)
	in = waitFor(t, m, in.ID, store.StateFailed)
	if in.TransitionError == "" {
		t.Error("OOM failure without reason")
	}
}

func TestOrphanedDetected(t *testing.T) {
	m, f, _ := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	waitFor(t, m, in.ID, store.StateRunning)
	_ = f.Remove(ctx, in.ID) // out-of-band deletion
	m.Reconcile(ctx)
	waitFor(t, m, in.ID, store.StateFailed)
}

func TestStartupReconcileHostReboot(t *testing.T) {
	// Host reboot: runtime objects gone (fresh fake), records say running.
	m, _, st := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	waitFor(t, m, in.ID, store.StateRunning)
	m2 := New(st, runtime.NewFake()) // fresh adapter, empty runtime
	m2.Reconcile(ctx)
	waitFor(t, m2, in.ID, store.StateFailed)
}

func TestConflictMidTransition(t *testing.T) {
	m, _, _ := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	waitFor(t, m, in.ID, store.StateRunning)
	if _, err := m.Action(ctx, in.ID, "stop"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, in.ID, store.StateStopped)
	// stopped -> stopping is illegal.
	if _, err := m.Action(ctx, in.ID, "stop"); err != ErrConflict {
		t.Errorf("stopping a stopped instance: err=%v, want conflict", err)
	}
}
