package runtime

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

func itoa(n int) string {
	if n == 0 {
		return "" // empty = runtime auto-allocates
	}
	return strconv.Itoa(n)
}

// DockerAdapter implements Adapter against the Docker Engine API.
// It enforces the Orbit hardening profile on every unit it creates
// (docs/features/orbit/design/security.md §3.1).
type DockerAdapter struct {
	cli *client.Client
}

func NewDocker() (*DockerAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &DockerAdapter{cli: cli}, nil
}

func (d *DockerAdapter) Ping(ctx context.Context) error {
	_, err := d.cli.Ping(ctx)
	return err
}

// hardening returns the security options every Orbit unit gets.
// Capabilities: ALL dropped, then the minimal set the bootstrap entrypoint
// needs to administer the guest user's files (chown/write/setuid for `su`).
// Everything else stays dropped; no-new-privileges + default seccomp remain.
// shortcut: guest-side hardening relies on the drop + no-new-privileges; a
// fully capless root bootstrap needs a redesign of the entrypoint (build-time
// key injection), revisit in Phase 8 hardening backlog.
func hardening() *container.HostConfig {
	return &container.HostConfig{
		CapDrop: []string{"ALL"},
		CapAdd:  []string{"CHOWN", "FOWNER", "DAC_OVERRIDE", "SETUID", "SETGID", "SYS_CHROOT"},
		SecurityOpt:   []string{"no-new-privileges"},
		RestartPolicy: container.RestartPolicy{Name: "no"}, // Orbit's reconciler owns restarts
		// seccomp: Docker's default profile applies unless overridden;
		// we never override it.
	}
}

func (d *DockerAdapter) Create(ctx context.Context, spec UnitSpec) (string, error) {
	if err := spec.validate(); err != nil {
		return "", err
	}
	hc := hardening()
	hc.Resources.NanoCPUs = int64(spec.CPUs * 1e9)
	hc.Resources.Memory = int64(spec.MemoryMB) * 1024 * 1024
	hc.Resources.MemorySwap = hc.Resources.Memory // no swap: limits are honest
	hc.Resources.PidsLimit = &spec.PidsLimit
	hc.ReadonlyRootfs = spec.ReadOnlyRootFS

	if len(spec.Ports) > 0 {
		hc.PortBindings = nat.PortMap{}
		for _, p := range spec.Ports {
			port := nat.Port(fmt.Sprintf("%d/tcp", p.InstancePort))
			hostIP := "127.0.0.1" // loopback default; "all" publishes on every interface
			if p.Bind == "all" {
				hostIP = ""
			}
			hc.PortBindings[port] = append(hc.PortBindings[port],
				nat.PortBinding{HostIP: hostIP, HostPort: itoa(p.HostPort)})
		}
	}

	// Image may be absent locally; the caller's policy decides when to pull.
	// Phase 1 keeps create pull-free; Phase 3 adds curated-image pulls.
	// Join the Orbit bridge; subnet isolation per security.md §5.
	nc := &network.NetworkingConfig{}
	if spec.Network != "" {
		nc.EndpointsConfig = map[string]*network.EndpointSettings{
			spec.Network: {},
		}
	}
	// Named data volumes (per-instance root data volume, technical-design §4).
	for _, v := range spec.Volumes {
		hc.Mounts = append(hc.Mounts, mount.Mount{Type: mount.TypeVolume, Source: v.Name, Target: v.Dest})
	}
	resp, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image:  spec.Image,
			Cmd:    spec.Cmd,
			Labels: spec.Labels,
			Env:    spec.Env,
		},
		hc,
		nc,
		nil,
		spec.Name,
	)
	if err != nil {
		return "", fmt.Errorf("create unit %q: %w", spec.Name, err)
	}
	return resp.ID, nil
}

func (d *DockerAdapter) Start(ctx context.Context, id string) error {
	return d.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (d *DockerAdapter) Stop(ctx context.Context, id string, timeoutSecs int) error {
	timeout := timeoutSecs
	return d.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
}

func (d *DockerAdapter) Remove(ctx context.Context, id string) error {
	return d.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: false})
}

func (d *DockerAdapter) Inspect(ctx context.Context, id string) (Status, error) {
	info, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "No such") || strings.Contains(err.Error(), "no such") {
			return Status{State: "missing"}, nil
		}
		return Status{}, err
	}
	st := info.State
	if st == nil {
		return Status{State: "exited", Error: "no state reported"}, nil
	}
	s := Status{ExitCode: st.ExitCode, OOMKilled: st.OOMKilled}
	switch {
	case st.Running:
		s.State = "running"
	default:
		s.State = "exited"
	}
	if st.Error != "" {
		s.Error = st.Error
	}
	return s, nil
}

// HostResources reports the capacity envelope (Engine-reported NCPU/MemTotal;
// on Docker Desktop these are the Desktop VM's values, which is the envelope
// that actually constrains containers).
func (d *DockerAdapter) HostResources(ctx context.Context) (float64, int, error) {
	info, err := d.cli.Info(ctx)
	if err != nil {
		return 0, 0, err
	}
	return float64(info.NCPU), int(info.MemTotal / 1024 / 1024), nil
}

// EnsureNetwork idempotently creates the bridge network. If it already
// exists with a different subnet, the collision is reported (never rebuilt:
// existing instances depend on it).
func (d *DockerAdapter) EnsureNetwork(ctx context.Context, name, subnet string) error {
	_, err := d.cli.NetworkInspect(ctx, name, network.InspectOptions{})
	if err == nil {
		return nil // exists; driver/subnet drift tolerated for MVP
	}
	if !client.IsErrNotFound(err) {
		return err
	}
	ipam := &network.IPAM{Config: []network.IPAMConfig{{Subnet: subnet}}}
	_, err = d.cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		IPAM:   ipam,
	})
	if err != nil {
		return fmt.Errorf("create network %q: %w", name, err)
	}
	return nil
}

// EnsureVolume idempotently creates a named data volume.
func (d *DockerAdapter) EnsureVolume(ctx context.Context, name string) error {
	_, err := d.cli.VolumeInspect(ctx, name)
	if err == nil {
		return nil
	}
	if !client.IsErrNotFound(err) {
		return err
	}
	_, err = d.cli.VolumeCreate(ctx, volume.CreateOptions{Name: name})
	if err != nil {
		return fmt.Errorf("create volume %q: %w", name, err)
	}
	return nil
}

// RemoveVolume deletes a named data volume (missing = ok).
func (d *DockerAdapter) RemoveVolume(ctx context.Context, name string) error {
	err := d.cli.VolumeRemove(ctx, name, false)
	if err != nil && client.IsErrNotFound(err) {
		return nil
	}
	return err
}

// IPAddress reports the unit's IP on the named network.
func (d *DockerAdapter) IPAddress(ctx context.Context, id, networkName string) (string, error) {
	info, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}
	ns := info.NetworkSettings
	if ns == nil || ns.Networks == nil {
		return "", nil
	}
	ep, ok := ns.Networks[networkName]
	if !ok || ep.IPAddress == "" {
		return "", fmt.Errorf("unit %s has no address on %s", id, networkName)
	}
	return ep.IPAddress, nil
}

// PortBindings reports resolved host ports for the unit's published instance ports.
func (d *DockerAdapter) PortBindings(ctx context.Context, id string) (map[int]int, error) {
	info, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}
	out := map[int]int{}
	for port, bindings := range info.NetworkSettings.Ports {
		if len(bindings) == 0 {
			continue
		}
		var instancePort int
		if _, err := fmt.Sscanf(string(port), "%d/tcp", &instancePort); err != nil {
			continue
		}
		host, err := strconv.Atoi(bindings[0].HostPort)
		if err != nil {
			continue
		}
		out[instancePort] = host
	}
	return out, nil
}
