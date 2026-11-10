package manager

import (
	"context"
	"testing"

	"orbit/internal/store"
)

// Phase 6: per-instance named root volume, mounted at /var/lib/orbit,
// recorded on the instance, removed on terminate unless kept.
func TestRootVolumeLifecycle(t *testing.T) {
	m, f, _ := testManager(t)
	ctx := context.Background()
	in, _, err := m.Create(ctx, sample())
	if err != nil {
		t.Fatal(err)
	}
	in = waitFor(t, m, in.ID, store.StateRunning)

	want := "orbit-" + in.ID
	if in.RootVolumeID != want {
		t.Errorf("root_volume_id %q, want %q", in.RootVolumeID, want)
	}
	if !f.HasVolume(want) {
		t.Error("root volume not created")
	}

	// Data survives stop/start (fake: volume persists; container not removed).
	if _, err := m.Action(ctx, in.ID, "stop"); err != nil {
		t.Fatal(err)
	}
	if !f.HasVolume(want) {
		t.Error("volume lost on stop")
	}
	waitFor(t, m, in.ID, store.StateRunning)
	if !f.HasVolume(want) {
		t.Error("volume lost on start")
	}

	// Terminate deletes the volume by default.
	if _, err := m.Terminate(ctx, in.ID, false); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, in.ID, store.StateTerminated)
	if f.HasVolume(want) {
		t.Error("volume not removed on terminate")
	}
}

// Phase 6: keep-flag preserves the volume across terminate.
func TestTerminateKeepRootVolume(t *testing.T) {
	m, f, _ := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	in = waitFor(t, m, in.ID, store.StateRunning)
	if _, err := m.Terminate(ctx, in.ID, true); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, in.ID, store.StateTerminated)
	if !f.HasVolume("orbit-" + in.ID) {
		t.Error("kept volume was removed")
	}
}
