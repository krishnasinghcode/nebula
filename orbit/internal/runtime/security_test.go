package runtime

import (
	"context"
	"strings"
	"testing"
)

// Security gate 2 (security.md §8), scenario 25: host-namespace and socket
// mounts are structurally rejected; privileged/pid/network-host modes cannot
// even be expressed through UnitSpec (rejected by construction).
func TestAdversarialSpecsRejected(t *testing.T) {
	valid := UnitSpec{Name: "x", Image: "img", CPUs: 0.5, MemoryMB: 256}
	cases := []struct {
		name    string
		volumes []VolumeMount
		want    string
	}{
		{"docker.sock source", []VolumeMount{{Name: "/var/run/docker.sock", Dest: "/tmp/s"}}, "host-path"},
		{"docker.sock dest", []VolumeMount{{Name: "v", Dest: "/var/run/docker.sock"}}, "not permitted"},
		{"root dest", []VolumeMount{{Name: "v", Dest: "/"}}, "not permitted"},
		{"proc dest", []VolumeMount{{Name: "v", Dest: "/proc/1"}}, "not permitted"},
		{"sys dest", []VolumeMount{{Name: "v", Dest: "/sys"}}, "not permitted"},
		{"etc dest", []VolumeMount{{Name: "v", Dest: "/etc"}}, "not permitted"},
		{"run dest", []VolumeMount{{Name: "v", Dest: "/run"}}, "not permitted"},
	}
	for _, tc := range cases {
		spec := valid
		spec.Volumes = tc.volumes
		err := spec.validate()
		if err == nil {
			t.Errorf("%s: accepted, want rejection", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q, want substring %q", tc.name, err, tc.want)
		}
	}
	// Named volume at a safe destination is fine.
	ok := valid
	ok.Volumes = []VolumeMount{{Name: "orbit-x", Dest: "/var/lib/orbit"}}
	if err := ok.validate(); err != nil {
		t.Errorf("safe mount rejected: %v", err)
	}
	// The adapter's Create path must invoke the same validation.
	f := NewFake()
	f.CreateErr = nil
	_, err := f.Create(context.Background(), UnitSpec{Name: "y", Image: "img", CPUs: 0.5, MemoryMB: 256,
		Volumes: []VolumeMount{{Name: "v", Dest: "/var/run/docker.sock"}}})
	if err == nil {
		t.Error("Create accepted a banned mount")
	}
}
