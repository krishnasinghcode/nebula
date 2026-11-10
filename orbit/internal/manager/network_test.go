package manager

import (
	"context"
	"errors"
	"testing"

	"orbit/internal/capacity"
	"orbit/internal/store"
)

// Phase 5: create must attach the unit to the orbit bridge and record its
// private IP after start.
func TestProvisionSetsPrivateIPAndNetwork(t *testing.T) {
	m, f, _ := testManager(t)
	ctx := context.Background()
	in, _, err := m.Create(ctx, sample())
	if err != nil {
		t.Fatal(err)
	}
	in = waitFor(t, m, in.ID, store.StateRunning)
	if len(in.PrivateIP) < 8 || in.PrivateIP[:7] != "172.28." {
		t.Fatalf("private_ip %q, want 172.28.0.x", in.PrivateIP)
	}
	if f.NetName != capacity.NetName {
		t.Errorf("fake network %q, want %q", f.NetName, capacity.NetName)
	}
	// Two instances get distinct IPs.
	in2, _, err := m.Create(ctx, &store.Instance{Name: "web-2", ImageID: "ubuntu-24.04", TypeID: "orbit.micro", KeyID: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	in2 = waitFor(t, m, in2.ID, store.StateRunning)
	if in2.PrivateIP == in.PrivateIP {
		t.Errorf("duplicate private IP %s", in.PrivateIP)
	}
}

// Phase 5: PUT /ports — validation, duplicates, bind defaults, recreation.
func TestUpdatePorts(t *testing.T) {
	m, f, _ := testManager(t)
	ctx := context.Background()
	in, _, _ := m.Create(ctx, sample())
	in = waitFor(t, m, in.ID, store.StateRunning)
	sshHost := 0
	for _, p := range in.Ports {
		if p.InstancePort == 22 {
			sshHost = p.HostPort
		}
	}

	// Running instance -> conflict.
	if _, err := m.UpdatePorts(ctx, in.ID, nil); err == nil {
		t.Fatal("update ports on running instance allowed")
	}
	if _, err := m.Action(ctx, in.ID, "stop"); err != nil {
		t.Fatal(err)
	}

	// Duplicate instance_port rejected.
	if _, err := m.UpdatePorts(ctx, in.ID, []store.Port{{InstancePort: 80}, {InstancePort: 80}}); err == nil {
		t.Fatal("duplicate instance_port accepted")
	}
	// Invalid bind rejected.
	if _, err := m.UpdatePorts(ctx, in.ID, []store.Port{{InstancePort: 80, Bind: "everywhere"}}); err == nil {
		t.Fatal("invalid bind accepted")
	}
	// Cross-instance host_port collision rejected (SSH port of instance A).
	var ce *store.ConflictError
	if _, err := m.UpdatePorts(ctx, in.ID, []store.Port{{InstancePort: 80, HostPort: sshHost}}); !errors.As(err, &ce) {
		t.Fatalf("host_port collision not rejected: %v", err)
	}

	// Valid update: empty bind defaults to loopback; unit dropped for recreation.
	got, err := m.UpdatePorts(ctx, in.ID, []store.Port{{InstancePort: 80, HostPort: 8080}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Ports) != 1 || got.Ports[0].Bind != "loopback" || got.Ports[0].HostPort != 8080 {
		t.Errorf("ports %+v, want single loopback 80->8080", got.Ports)
	}
	if f.Exists(in.ID) {
		t.Error("old unit not dropped for recreation")
	}

	// Start recreates the unit with the new bindings and re-resolves.
	waitFor(t, m, in.ID, store.StateRunning)
	specPorts := f.UnitSpecPorts(in.ID)
	if len(specPorts) != 1 || specPorts[0].InstancePort != 80 || specPorts[0].HostPort != 8080 {
		t.Errorf("recreated spec ports %+v, want 80->8080", specPorts)
	}
}
