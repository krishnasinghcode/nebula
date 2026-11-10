package store

import "time"

// Instance states (docs/features/orbit/design/data-and-api.md §2).
const (
	StateProvisioning = "provisioning"
	StateRunning      = "running"
	StateStopping     = "stopping"
	StateStopped      = "stopped"
	StateRebooting    = "rebooting"
	StateFailed       = "failed"
	StateTerminating  = "terminating"
	StateTerminated   = "terminated"
)

// Terminal reports whether the state is final.
func Terminal(s string) bool { return s == StateTerminated }

// transitions lists legal state changes. Anything not listed is a 409 conflict.
var transitions = map[string][]string{
	StateProvisioning: {StateRunning, StateFailed, StateTerminated},
	StateRunning:      {StateStopping, StateRebooting, StateFailed, StateTerminating},
	StateStopping:     {StateStopped, StateFailed, StateTerminating},
	StateStopped:      {StateRunning, StateTerminating},
	StateRebooting:    {StateRunning, StateFailed},
	StateFailed:       {StateStopped, StateTerminating},
	StateTerminating:  {StateTerminated},
	StateTerminated:   {},
}

// CanTransition reports whether from→to is legal.
func CanTransition(from, to string) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// Instance is an Orbit Instance record (control-plane view).
type Instance struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	State           string    `json:"state"`
	ImageID         string    `json:"image_id"`
	TypeID          string    `json:"type_id"`
	KeyID           string    `json:"key_id"`
	UserData        string    `json:"-"`
	PrivateIP       string    `json:"private_ip,omitempty"`
	Ports           []Port    `json:"ports,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	TransitionError string    `json:"transition_error,omitempty"`
	Owner           string    `json:"-"`
	RootVolumeID    string    `json:"-"`
}

// Port is a published port mapping.
type Port struct {
	InstancePort int    `json:"instance_port"`
	HostPort     int    `json:"host_port"` // 0 = auto (resolved in Phase 5)
	Bind         string `json:"bind"`      // "loopback" (default) or "all"
}

// Event is a lifecycle event appended on every transition.
type Event struct {
	ID         int64     `json:"id"`
	InstanceID string    `json:"instance_id"`
	Type       string    `json:"type"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Event types.
const (
	EventCreate     = "create"
	EventTransition = "transition"
	EventError      = "error"
	EventReconcile  = "reconcile"
)
