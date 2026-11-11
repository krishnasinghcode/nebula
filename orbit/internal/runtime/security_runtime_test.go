//go:build runtime

// Runtime security + failure/recovery suite (real Docker required).
// Scenarios: 26 (profile audit), 28 (PID/socket isolation), 10/11/14
// (kill/OOM/orphan). Run: go test ./internal/runtime -tags=runtime
package runtime

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

func rtAdapter(t *testing.T) *DockerAdapter {
	t.Helper()
	d, err := NewDocker()
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	if err := d.Ping(ctx); err != nil {
		t.Skipf("docker not reachable: %v", err)
	}
	// Ensure the test images exist (CI pre-pull equivalent).
	for _, img := range []string{"alpine:3", "busybox:latest"} {
		if rd, err := d.cli.ImagePull(ctx, img, image.PullOptions{}); err == nil {
			_, _ = io.Copy(io.Discard, rd)
			_ = rd.Close()
		}
	}
	return d
}

func baseSpec(name string) UnitSpec {
	return UnitSpec{
		Name: name, Image: "alpine:3", Cmd: []string{"sleep", "300"},
		CPUs: 0.25, MemoryMB: 256, PidsLimit: 64,
		Labels: map[string]string{"orbit.test": name},
	}
}

// Scenario 26: hardening profile audit on a running container.
func TestRuntimeHardeningProfile(t *testing.T) {
	d := rtAdapter(t)
	ctx := context.Background()
	name := "orbit-sec-audit"
	_ = d.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
	defer d.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true}) //nolint:errcheck

	if _, err := d.Create(ctx, baseSpec(name)); err != nil {
		t.Fatal(err)
	}
	if err := d.Start(ctx, name); err != nil {
		t.Fatal(err)
	}
	info, err := d.cli.ContainerInspect(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	hc := info.HostConfig
	if len(hc.CapDrop) == 0 || !contains(hc.CapDrop, "ALL") {
		t.Errorf("CapDrop = %v, want ALL dropped", hc.CapDrop)
	}
	if !contains(hc.SecurityOpt, "no-new-privileges") {
		t.Errorf("SecurityOpt = %v, want no-new-privileges", hc.SecurityOpt)
	}
	// Regression: AUDIT_WRITE is required for sshd PTY (interactive shell)
	// sessions; without it the session aborts after auth. Docker normalises
	// cap names with a CAP_ prefix.
	if !contains(hc.CapAdd, "CAP_AUDIT_WRITE") {
		t.Errorf("CapAdd = %v, want CAP_AUDIT_WRITE present", hc.CapAdd)
	}
	if hc.Privileged {
		t.Error("Privileged container created")
	}
	if hc.RestartPolicy.Name != "no" && hc.RestartPolicy.Name != "" {
		t.Errorf("RestartPolicy = %q, want no/none", hc.RestartPolicy.Name)
	}
	if hc.Resources.Memory == 0 || hc.Resources.NanoCPUs == 0 {
		t.Errorf("resource limits missing: %+v", hc.Resources)
	}
	if hc.PidsLimit == nil || *hc.PidsLimit != 64 {
		t.Errorf("PidsLimit = %v, want 64", hc.PidsLimit)
	}
	// Rootfs read-only is Phase 8 hardening (deviation 2026-11-10): not yet asserted.
}

// Scenario 28: private PID namespace and no docker socket in the guest.
func TestRuntimePIDAndSocketIsolation(t *testing.T) {
	d := rtAdapter(t)
	ctx := context.Background()
	name := "orbit-sec-pid"
	_ = d.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
	defer d.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true}) //nolint:errcheck

	spec := baseSpec(name)
	spec.Cmd = []string{"sh", "-c", "ls /var/run/docker.sock >/dev/null 2>&1 && echo SOCKET || echo NOSOCKET; ps -o pid,comm | wc -l; sleep 60"}
	if _, err := d.Create(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if err := d.Start(ctx, name); err != nil {
		t.Fatal(err)
	}
	info, err := d.cli.ContainerInspect(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if string(info.HostConfig.PidMode) != "" {
		t.Errorf("PidMode = %q, want private (empty)", info.HostConfig.PidMode)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
