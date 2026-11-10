// Package manager orchestrates the instance lifecycle: state machine,
// per-instance serialized transitions, and the reconciler loop.
package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"orbit/internal/capacity"
	"orbit/internal/catalogue"
	"orbit/internal/runtime"
	"orbit/internal/store"
)

// ErrConflict is returned for illegal transitions or duplicate names.
var ErrConflict = errors.New("conflict")

// ErrNotFound is returned when an instance does not exist.
var ErrNotFound = errors.New("not found")

// Manager owns instance lifecycle orchestration.
type Manager struct {
	st    *store.Store
	rt    runtime.Adapter
	cap   *capacity.Manager
	mu    sync.Mutex             // guards locks map
	locks map[string]*sync.Mutex // one transition at a time per instance
}

func New(st *store.Store, rt runtime.Adapter) *Manager {
	return &Manager{
		st:  st,
		rt:  rt,
		cap: &capacity.Manager{Detector: rt, St: st},
	}
}

func (m *Manager) lock(id string) func() {
	m.mu.Lock()
	l, ok := m.locks[id]
	if !ok {
		l = &sync.Mutex{}
		m.locks[id] = l
	}
	m.mu.Unlock()
	l.Lock()
	return l.Unlock
}

func conflictErr(err error) error {
	var ce *store.ConflictError
	if errors.As(err, &ce) {
		return ErrConflict
	}
	return err
}

// Create validates and records a new instance (state: provisioning), then
// provisions asynchronously via the runtime adapter.
// Idempotency: name is unique among non-terminated instances; a repeated
// create for a live name returns the existing instance (existing=true).
func (m *Manager) Create(ctx context.Context, in *store.Instance) (created *store.Instance, existing bool, err error) {
	if dup, derr := m.st.GetInstanceByName(in.Name); derr == nil && dup != nil {
		return dup, true, nil
	}
	in.ID = store.NewID()
	in.State = store.StateProvisioning

	// Admission control: capacity + quotas (memory never overcommitted).
	if err := m.cap.Admit(ctx, in.TypeID, 1); err != nil {
		return nil, false, err // caller maps to 422 (insufficient_capacity / quota_exceeded)
	}

	if err := m.st.CreateInstance(in); err != nil {
		return nil, false, err
	}
	_ = m.st.AddEvent(in.ID, store.EventCreate, "instance created")
	go m.provision(context.Background(), in.ID)
	return in, false, nil
}

// ensureUnit recreates the runtime unit if it is missing (e.g. after a
// port-update recreation or an external removal).
func (m *Manager) ensureUnit(ctx context.Context, in *store.Instance) error {
	st, err := m.rt.Inspect(ctx, in.ID)
	if err != nil {
		return err
	}
	if st.State != "missing" {
		return nil
	}
	spec, err := m.unitSpec(ctx, in)
	if err != nil {
		return err
	}
	if err := m.rt.EnsureNetwork(ctx, spec.Network, m.subnet()); err != nil {
		return err
	}
	_, err = m.rt.Create(ctx, spec)
	return err
}

// resolve saves the running unit's host ports and private IP into the record.
func (m *Manager) resolve(ctx context.Context, id string) {
	in, err := m.st.GetInstance(id)
	if err != nil || in == nil {
		return
	}
	if bindings, err := m.rt.PortBindings(ctx, id); err == nil {
		changed := false
		for i := range in.Ports {
			if host, ok := bindings[in.Ports[i].InstancePort]; ok && host != in.Ports[i].HostPort {
				in.Ports[i].HostPort = host
				changed = true
			}
		}
		if changed {
			if err := m.st.SetInstancePorts(id, in.Ports); err != nil {
				log.Printf("orbit: resolve(%s): save ports: %v", id, err)
			}
		}
	}
	if ip, err := m.rt.IPAddress(ctx, id, capacity.NetName); err == nil && ip != "" && ip != in.PrivateIP {
		if err := m.st.SetInstanceFields(id, ip, ""); err != nil {
			log.Printf("orbit: resolve(%s): save ip: %v", id, err)
		}
	}
}

// provision runs the runtime work for an asynchronously created instance.
func (m *Manager) provision(ctx context.Context, id string) {
	unlock := m.lock(id)
	defer unlock()

	in, err := m.st.GetInstance(id)
	if err != nil || in == nil {
		return
	}
	spec, err := m.unitSpec(ctx, in)
	if err != nil {
		m.fail(id, "build unit spec: "+err.Error())
		return
	}
	if err := m.rt.EnsureNetwork(ctx, spec.Network, m.subnet()); err != nil {
		m.fail(id, "ensure network: "+err.Error())
		return
	}
	unitID, err := m.rt.Create(ctx, spec)
	if err != nil {
		m.fail(id, "runtime create: "+err.Error())
		return
	}
	if err := m.rt.Start(ctx, unitID); err != nil {
		_ = m.rt.Remove(ctx, unitID) // roll back the half-created unit
		m.fail(id, "runtime start: "+err.Error())
		return
	}
	// Resolve auto-allocated host ports and the private IP into the record.
	m.resolve(ctx, id)
	if _, err := m.st.Transition(id, store.StateRunning, store.EventTransition,
		"provisioned as unit "+unitID); err != nil {
		m.fail(id, "state transition: "+err.Error())
	}
}

func (m *Manager) unitSpec(ctx context.Context, in *store.Instance) (runtime.UnitSpec, error) {
	dockerImage, err := catalogue.MustDockerImage(in.ImageID)
	if err != nil {
		return runtime.UnitSpec{}, err
	}
	key, err := m.st.GetSSHKey(in.KeyID)
	if err != nil {
		return runtime.UnitSpec{}, err
	}
	if key == nil {
		return runtime.UnitSpec{}, fmt.Errorf("ssh key %q not found", in.KeyID)
	}

	meta, _ := json.Marshal(map[string]string{
		"instance_id": in.ID, "name": in.Name, "type": in.TypeID,
	})
	env := []string{
		"ORBIT_INSTANCE_META=" + string(meta),
		"ORBIT_PUBKEY=" + key.PublicKey,
	}
	if in.UserData != "" {
		env = append(env, "ORBIT_USER_DATA_B64="+base64.StdEncoding.EncodeToString([]byte(in.UserData)))
	}

	// SSH (22) is auto-published to loopback so `orbit ssh` works on every
	// platform, incl. Docker Desktop where private IPs are unreachable.
	ports := in.Ports
	hasSSH := false
	for _, p := range ports {
		if p.InstancePort == 22 {
			hasSSH = true
		}
	}
	if !hasSSH {
		ports = append(ports, store.Port{InstancePort: 22, HostPort: 0, Bind: "loopback"})
	}
	if len(ports) != len(in.Ports) {
		if err := m.st.SetInstancePorts(in.ID, ports); err != nil {
			return runtime.UnitSpec{}, err
		}
	}
	specPorts := make([]runtime.PortBinding, len(ports))
	for i, p := range ports {
		specPorts[i] = runtime.PortBinding{InstancePort: p.InstancePort, HostPort: p.HostPort, Bind: p.Bind}
	}

	t := capacity.GetType(in.TypeID)
	if t == nil {
		return runtime.UnitSpec{}, fmt.Errorf("unknown instance type %q", in.TypeID)
	}
	return runtime.UnitSpec{
		Name:      in.ID,
		Image:     dockerImage,
		Labels:    map[string]string{"orbit.instance-id": in.ID, "orbit.name": in.Name},
		Env:       env,
		Network:   capacity.NetName,
		Ports:     specPorts,
		CPUs:      t.VCPU,
		MemoryMB:  t.MemoryMB,
		PidsLimit: 512,
	}, nil
}

// subnet is the configured Orbit bridge subnet.
func (m *Manager) subnet() string { return capacity.Subnet() }

// UpdatePorts replaces the instance's published ports (PUT /ports).
// Only stopped instances may be re-published; the unit is recreated on
// the next start so the new bindings take effect.
func (m *Manager) UpdatePorts(ctx context.Context, id string, ports []store.Port) (*store.Instance, error) {
	unlock := m.lock(id)
	defer unlock()

	in, err := m.st.GetInstance(id)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, ErrNotFound
	}
	if in.State != store.StateStopped {
		return nil, &store.ConflictError{Message: fmt.Sprintf(
			"instance %s is %s: stop it before changing port publishing", id, in.State)}
	}
	seenInstance, seenHost := map[int]bool{}, map[int]bool{}
	for i := range ports {
		p := &ports[i]
		if p.InstancePort <= 0 || p.InstancePort > 65535 {
			return nil, &store.ConflictError{Message: fmt.Sprintf("port %d: instance_port out of range", p.InstancePort)}
		}
		if seenInstance[p.InstancePort] {
			return nil, &store.ConflictError{Message: fmt.Sprintf("duplicate instance_port %d", p.InstancePort)}
		}
		seenInstance[p.InstancePort] = true
		if p.Bind == "" {
			p.Bind = "loopback"
		}
		if p.Bind != "loopback" && p.Bind != "all" {
			return nil, &store.ConflictError{Message: fmt.Sprintf("port %d: bind must be \"loopback\" or \"all\"", p.InstancePort)}
		}
		if p.Bind == "all" {
			log.Printf("orbit: update-ports(%s): port %d published on all interfaces (0.0.0.0) — reachable from your LAN", id, p.InstancePort)
		}
		if p.HostPort != 0 && seenHost[p.HostPort] {
			return nil, &store.ConflictError{Message: fmt.Sprintf("duplicate host_port %d", p.HostPort)}
		}
		seenHost[p.HostPort] = true
	}
	// Cross-instance host-port collisions.
	live, err := m.st.ListInstances("")
	if err != nil {
		return nil, err
	}
	for _, other := range live {
		if other.ID == id {
			continue
		}
		for _, p := range other.Ports {
			for i := range ports {
				if ports[i].HostPort != 0 && ports[i].HostPort == p.HostPort {
					return nil, &store.ConflictError{Message: fmt.Sprintf(
						"host_port %d already used by instance %s", p.HostPort, other.ID)}
				}
			}
		}
	}
	// Auto-allocated host ports reset: the recreated unit re-allocates.
	if err := m.st.SetInstancePorts(id, ports); err != nil {
		return nil, err
	}
	// Drop the old unit so the next start recreates it with new bindings.
	_ = m.rt.Remove(ctx, id) // absent unit is fine
	_ = m.st.AddEvent(id, store.EventTransition, "ports updated; unit will be recreated on next start")
	return m.st.GetInstance(id)
}

// fail records an error state.
func (m *Manager) fail(id, msg string) {
	if _, err := m.st.Transition(id, store.StateFailed, store.EventError, msg); err != nil {
		log.Printf("orbit: fail(%s): %v", id, err)
	}
}

// Action performs a lifecycle action. stop/start/reboot complete inline;
// terminate moves to terminating and finishes in the background.
func (m *Manager) Action(ctx context.Context, id, action string) (*store.Instance, error) {
	unlock := m.lock(id)
	defer unlock()

	in, err := m.st.GetInstance(id)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, ErrNotFound
	}

	switch action {
	case "start":
		if in, err = m.toState(in, store.StateRunning, "start"); err != nil {
			return nil, err
		}
		if err := m.ensureUnit(ctx, in); err != nil {
			m.fail(in.ID, "ensure unit: "+err.Error())
			return m.st.GetInstance(in.ID)
		}
		if err := m.rt.Start(ctx, in.ID); err != nil {
			m.fail(in.ID, "runtime start: "+err.Error())
			return m.st.GetInstance(in.ID)
		}
		m.resolve(ctx, in.ID)
		return m.st.GetInstance(in.ID)
	case "stop":
		if in, err = m.toState(in, store.StateStopping, "stop"); err != nil {
			return nil, err
		}
		return m.finishStop(ctx, in)
	case "reboot":
		if in, err = m.toState(in, store.StateRebooting, "reboot"); err != nil {
			return nil, err
		}
		if err := m.rt.Stop(ctx, in.ID, 30); err != nil {
			m.fail(in.ID, "runtime stop (reboot): "+err.Error())
			return m.st.GetInstance(in.ID)
		}
		if err := m.rt.Start(ctx, in.ID); err != nil {
			m.fail(in.ID, "runtime start (reboot): "+err.Error())
			return m.st.GetInstance(in.ID)
		}
		return m.st.Transition(in.ID, store.StateRunning, store.EventTransition, "rebooted")
	case "terminate":
		if in, err = m.toState(in, store.StateTerminating, "terminate"); err != nil {
			return nil, err
		}
		go m.finishTerminate(context.Background(), in.ID)
		return in, nil
	default:
		return nil, fmt.Errorf("unknown action %q", action)
	}
}

func (m *Manager) toState(in *store.Instance, to, why string) (*store.Instance, error) {
	updated, err := m.st.Transition(in.ID, to, store.EventTransition, why)
	if err != nil {
		return nil, conflictErr(err)
	}
	return updated, nil
}

func (m *Manager) finishStop(ctx context.Context, in *store.Instance) (*store.Instance, error) {
	if err := m.rt.Stop(ctx, in.ID, 30); err != nil {
		m.fail(in.ID, "runtime stop: "+err.Error())
		return m.st.GetInstance(in.ID)
	}
	return m.st.Transition(in.ID, store.StateStopped, store.EventTransition, "stopped")
}

func (m *Manager) finishTerminate(ctx context.Context, id string) {
	unlock := m.lock(id)
	defer unlock()
	// Ignore stop errors: removal proceeds even if the unit already exited.
	_ = m.rt.Stop(ctx, id, 30)
	// Root-volume removal arrives with volumes (Phase 6); containers only now.
	if err := m.rt.Remove(ctx, id); err != nil {
		log.Printf("orbit: terminate(%s): remove: %v", id, err)
	}
	if _, err := m.st.Transition(id, store.StateTerminated, store.EventTransition, "terminated"); err != nil {
		log.Printf("orbit: terminate(%s): %v", id, err)
	}
}

// Reconcile compares desired vs actual state for all non-terminal instances
// and repairs divergence. Called on startup and from the ticker loop.
func (m *Manager) Reconcile(ctx context.Context) {
	list, err := m.st.ListInstances("")
	if err != nil {
		log.Printf("orbit: reconcile list: %v", err)
		return
	}
	for _, in := range list {
		if store.Terminal(in.State) ||
			in.State == store.StateProvisioning || in.State == store.StateTerminating {
			continue // in-flight states are owned by their goroutine
		}
		func() {
			unlock := m.lock(in.ID)
			defer unlock()
			st, err := m.rt.Inspect(ctx, in.ID)
			if err != nil {
				log.Printf("orbit: reconcile(%s): inspect: %v", in.ID, err)
				return
			}
			switch st.State {
			case "running":
				if in.State != store.StateRunning {
					m.st.Transition(in.ID, store.StateRunning, store.EventReconcile, "runtime reports running")
				}
			case "exited":
				to, msg := store.StateStopped, fmt.Sprintf("exited (code %d)", st.ExitCode)
				if st.ExitCode != 0 || st.OOMKilled {
					to = store.StateFailed
					if st.OOMKilled {
						msg += ", OOM killed"
					}
					if st.Error != "" {
						msg += ", " + st.Error
					}
				}
				m.st.Transition(in.ID, to, store.EventReconcile, msg)
			case "missing":
				m.st.Transition(in.ID, store.StateFailed, store.EventReconcile, "runtime object missing (orphaned)")
			}
		}()
	}
}

// Capacity exposes the current budget (GET /v1/capacity).
func (m *Manager) Capacity(ctx context.Context) (capacity.Budget, error) {
	return m.cap.Budget(ctx)
}

// Get returns one instance or ErrNotFound.
func (m *Manager) Get(id string) (*store.Instance, error) {
	in, err := m.st.GetInstance(id)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, ErrNotFound
	}
	return in, nil
}

// List returns instances filtered by state ("" = all).
func (m *Manager) List(state string) ([]*store.Instance, error) {
	return m.st.ListInstances(state)
}

// Events returns lifecycle events for an instance.
func (m *Manager) Events(id string) ([]store.Event, error) {
	if _, err := m.Get(id); err != nil {
		return nil, err
	}
	return m.st.Events(id)
}

// StartReconciler runs Reconcile immediately (startup reconciliation) and
// then on every tick until ctx is cancelled.
func (m *Manager) StartReconciler(ctx context.Context, every time.Duration) {
	m.Reconcile(ctx)
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.Reconcile(ctx)
			}
		}
	}()
}
