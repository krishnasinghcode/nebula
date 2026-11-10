//go:build runtime

// Runtime suite: requires a running Docker Engine.
package runtime

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types/image"
)

const busybox = "busybox:latest"

func dockerAdapter(t *testing.T) *DockerAdapter {
	t.Helper()
	a, err := NewDocker()
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	ctx := t.Context()
	if err := a.Ping(ctx); err != nil {
		t.Skipf("docker unreachable: %v", err)
	}
	if rd, err := a.cli.ImagePull(ctx, busybox, image.PullOptions{}); err == nil {
		_, _ = io.Copy(io.Discard, rd)
		_ = rd.Close()
	}
	return a
}

func TestDockerLifecycle(t *testing.T) {
	a := dockerAdapter(t)
	ctx := t.Context()
	id, err := a.Create(ctx, UnitSpec{
		Name: "orbit-rt-test", Image: busybox,
		CPUs: 0.5, MemoryMB: 64, PidsLimit: 256,
		Labels: map[string]string{"orbit.test": "true"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer a.Remove(context.Background(), id)

	if err := a.Start(ctx, id); err != nil {
		t.Fatalf("start: %v", err)
	}
	st, err := a.Inspect(ctx, id)
	if err != nil || st.State != "running" {
		t.Fatalf("inspect after start: state=%q err=%v", st.State, err)
	}
	if err := a.Stop(ctx, id, 5); err != nil {
		t.Fatalf("stop: %v", err)
	}
	st, _ = a.Inspect(ctx, id)
	if st.State == "running" {
		t.Error("still running after stop")
	}
	if err := a.Remove(ctx, id); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if st, _ := a.Inspect(ctx, id); st.State != "missing" {
		t.Errorf("state after remove: %q, want missing", st.State)
	}
}

func TestDockerHardeningApplied(t *testing.T) {
	a := dockerAdapter(t)
	ctx := t.Context()
	id, err := a.Create(ctx, UnitSpec{
		Name: "orbit-rt-harden", Image: busybox,
		CPUs: 0.5, MemoryMB: 64, PidsLimit: 128, ReadOnlyRootFS: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer a.Remove(context.Background(), id)
	info, err := a.cli.ContainerInspect(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	hc := info.HostConfig
	if len(hc.CapDrop) == 0 || hc.CapDrop[0] != "ALL" {
		t.Error("CapDrop ALL not applied")
	}
	if hc.Memory != 64*1024*1024 || hc.MemorySwap != hc.Memory {
		t.Errorf("memory limits: mem=%d swap=%d", hc.Memory, hc.MemorySwap)
	}
	if hc.NanoCPUs != int64(0.5*1e9) {
		t.Errorf("cpu quota: %d", hc.NanoCPUs)
	}
	if !hc.ReadonlyRootfs {
		t.Error("readonly rootfs not applied")
	}
	if hc.Privileged {
		t.Error("privileged container created")
	}
	if hc.RestartPolicy.Name != "no" {
		t.Errorf("restart policy: %q, want no", hc.RestartPolicy.Name)
	}
}

func TestDockerOOMDetected(t *testing.T) {
	a := dockerAdapter(t)
	ctx := t.Context()
	id, err := a.Create(ctx, UnitSpec{
		Name: "orbit-rt-oom", Image: busybox,
		CPUs: 0.5, MemoryMB: 32, PidsLimit: 128,
		// Exponential variable growth -> genuine cgroup OOM kill.
		Cmd: []string{"sh", "-c", "a=x; while true; do a=$a$a; done"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer a.Remove(context.Background(), id)
	if err := a.Start(ctx, id); err != nil {
		t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		st, err := a.Inspect(ctx, id)
		if err != nil {
			t.Fatalf("inspect: %v", err)
		}
		if st.State == "exited" {
			if !st.OOMKilled {
				t.Fatalf("exited without OOM flag: %+v", st)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("unit still %q after 30s; want OOM exit", st.State)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
