package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"orbit/internal/capacity"
	"orbit/internal/catalogue"
	"orbit/internal/manager"
	"orbit/internal/store"

	"golang.org/x/crypto/ssh"
)

// keyRe: key names share the instance-name shape.
var keyRe = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// API holds the control-plane dependencies for the v1 routes.
type API struct {
	Manager *manager.Manager
	Keys    *store.Store
}

// Register mounts the v1 API routes on mux.
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/instances", a.createInstance)
	mux.HandleFunc("GET /v1/instances", a.listInstances)
	mux.HandleFunc("GET /v1/instances/{id}", a.describeInstance)
	mux.HandleFunc("POST /v1/instances/{id}/start", a.action("start"))
	mux.HandleFunc("POST /v1/instances/{id}/stop", a.action("stop"))
	mux.HandleFunc("POST /v1/instances/{id}/reboot", a.action("reboot"))
	mux.HandleFunc("POST /v1/instances/{id}/terminate", a.action("terminate"))
	mux.HandleFunc("GET /v1/instances/{id}/events", a.instanceEvents)
	mux.HandleFunc("PUT /v1/instances/{id}/ports", a.updatePorts)

	mux.HandleFunc("GET /v1/images", a.listImages)
	mux.HandleFunc("GET /v1/instance-types", a.listTypes)
	mux.HandleFunc("GET /v1/capacity", a.capacity)

	mux.HandleFunc("POST /v1/keys", a.createKey)
	mux.HandleFunc("GET /v1/keys", a.listKeys)
	mux.HandleFunc("GET /v1/keys/{name}", a.getKey)
	mux.HandleFunc("DELETE /v1/keys/{name}", a.deleteKey)
}

// listImages serves the read-only curated catalogue.
func (a *API) listImages(w http.ResponseWriter, _ *http.Request) {
	writeJSONBody(w, http.StatusOK, catalogue.Images)
}

func (a *API) listTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSONBody(w, http.StatusOK, capacity.DefaultTypes)
}

func (a *API) capacity(w http.ResponseWriter, r *http.Request) {
	b, err := a.Manager.Capacity(r.Context())
	if err != nil {
		errorJSON(w, http.StatusServiceUnavailable, "runtime_error",
			"cannot read host capacity: "+err.Error(), nil)
		return
	}
	// 5.4: be honest about the shared bridge — instance-to-instance traffic
	// on orbit-net is unrestricted by design (security.md §5).
	payload := struct {
		capacity.Budget
		Network map[string]string `json:"network"`
	}{b, map[string]string{
		"name":   capacity.NetName,
		"subnet": b.Subnet,
		"note":   "all instances share this bridge and can reach each other by private IP",
	}}
	writeJSONBody(w, http.StatusOK, payload)
}

type updatePortsReq struct {
	Ports []store.Port `json:"ports"`
}

func (a *API) updatePorts(w http.ResponseWriter, r *http.Request) {
	var req updatePortsReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, "validation_failed", "invalid JSON body: "+err.Error(), nil)
		return
	}
	in, err := a.Manager.UpdatePorts(r.Context(), r.PathValue("id"), req.Ports)
	var ce *store.ConflictError
	switch {
	case err == nil:
		writeJSONBody(w, http.StatusOK, in)
	case errors.Is(err, manager.ErrNotFound):
		errorJSON(w, http.StatusNotFound, "not_found", "instance not found", nil)
	case errors.As(err, &ce):
		errorJSON(w, http.StatusUnprocessableEntity, "validation_failed", ce.Message, nil)
	default:
		errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}
}

type createKeyReq struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

func (a *API) createKey(w http.ResponseWriter, r *http.Request) {
	var req createKeyReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, "validation_failed", "invalid JSON body: "+err.Error(), nil)
		return
	}
	if !keyRe.MatchString(req.Name) {
		errorJSON(w, http.StatusUnprocessableEntity, "validation_failed",
			"name must match ^[a-z][a-z0-9-]{2,30}$", nil)
		return
	}
	fp, err := fingerprint(req.PublicKey)
	if err != nil {
		errorJSON(w, http.StatusUnprocessableEntity, "validation_failed",
			"public_key is not a valid OpenSSH public key: "+err.Error(), nil)
		return
	}
	key := &store.SSHKey{Name: req.Name, PublicKey: req.PublicKey, Fingerprint: fp}
	if err := a.Keys.CreateSSHKey(key); err != nil {
		errorJSON(w, http.StatusConflict, "conflict", "key name already registered", nil)
		return
	}
	writeJSONBody(w, http.StatusCreated, key)
}

func (a *API) listKeys(w http.ResponseWriter, _ *http.Request) {
	keys, err := a.Keys.ListSSHKeys()
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
		return
	}
	if keys == nil {
		keys = []store.SSHKey{}
	}
	writeJSONBody(w, http.StatusOK, keys)
}

func (a *API) getKey(w http.ResponseWriter, r *http.Request) {
	key, err := a.Keys.GetSSHKey(r.PathValue("name"))
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
		return
	}
	if key == nil {
		errorJSON(w, http.StatusNotFound, "not_found", "key not found", nil)
		return
	}
	writeJSONBody(w, http.StatusOK, key)
}

func (a *API) deleteKey(w http.ResponseWriter, r *http.Request) {
	err := a.Keys.DeleteSSHKey(r.PathValue("name"))
	var ce *store.ConflictError
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.As(err, &ce):
		errorJSON(w, http.StatusConflict, "conflict", ce.Message, nil)
	default:
		errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}
}

// fingerprint computes the SHA256 fingerprint of an OpenSSH public key.
func fingerprint(pubkey string) (string, error) {
	parsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubkey))
	if err != nil {
		return "", err
	}
	return ssh.FingerprintSHA256(parsed), nil
}

type createInstanceReq struct {
	Name             string `json:"name"`
	ImageID          string `json:"image_id"`
	TypeID           string `json:"type_id"`
	KeyID            string `json:"key_id"`
	UserData         string `json:"user_data"`
	Ports            []Port `json:"ports"`
	IdempotencyToken string `json:"idempotency_token"`
}

// Port mirrors store.Port in API JSON.
type Port struct {
	InstancePort int    `json:"instance_port"`
	HostPort     int    `json:"host_port"`
	Bind         string `json:"bind"`
}

func (a *API) createInstance(w http.ResponseWriter, r *http.Request) {
	var req createInstanceReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, "validation_failed", "invalid JSON body: "+err.Error(), nil)
		return
	}
	if verr := validateCreate(req); verr != "" {
		errorJSON(w, http.StatusUnprocessableEntity, "validation_failed", verr, nil)
		return
	}
	ports := make([]store.Port, len(req.Ports))
	for i, p := range req.Ports {
		ports[i] = store.Port(p)
	}
	in := &store.Instance{
		Name: req.Name, ImageID: req.ImageID, TypeID: req.TypeID, KeyID: req.KeyID,
		UserData: req.UserData, Ports: ports, Owner: "local",
	}
	// Admission errors -> structured 422s; other errors -> 503/500.
	created, existing, err := a.Manager.Create(r.Context(), in)
	if err != nil {
		var ice *capacity.InsufficientCapacity
		var qe *store.ConflictError
		switch {
		case errors.As(err, &ice):
			errorJSON(w, http.StatusUnprocessableEntity, "insufficient_capacity", err.Error(), map[string]any{
				"requested": ice.Requested, "available": ice.Available,
			})
		case errors.As(err, &qe):
			errorJSON(w, http.StatusUnprocessableEntity, "quota_exceeded", err.Error(), nil)
		default:
			errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
		}
		return
	}
	code := http.StatusAccepted // 202: async provisioning
	if existing {
		code = http.StatusOK // idempotent replay
	}
	writeJSONBody(w, code, created)
}

func validateCreate(req createInstanceReq) string {
	var problems []string
	if !nameRe.MatchString(req.Name) {
		problems = append(problems, "name must match ^[a-z][a-z0-9-]{2,30}$")
	}
	if req.ImageID == "" {
		problems = append(problems, "image_id is required")
	} else if catalogue.Get(req.ImageID) == nil {
		problems = append(problems, "unknown image_id (see GET /v1/images)")
	}
	if req.TypeID == "" {
		problems = append(problems, "type_id is required")
	} else if capacity.GetType(req.TypeID) == nil {
		problems = append(problems, "unknown type_id (see GET /v1/instance-types)")
	}
	if req.KeyID == "" {
		problems = append(problems, "key_id is required")
	}
	if len(req.UserData) > 16*1024 {
		problems = append(problems, "user_data exceeds 16 KiB")
	}
	if req.UserData != "" && !strings.HasPrefix(req.UserData, "#!") {
		problems = append(problems, "user_data must begin with a shebang")
	}
	return strings.Join(problems, "; ")
}

func (a *API) listInstances(w http.ResponseWriter, r *http.Request) {
	list, err := a.Manager.List(r.URL.Query().Get("state"))
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
		return
	}
	if list == nil {
		list = []*store.Instance{}
	}
	writeJSONBody(w, http.StatusOK, list)
}

func (a *API) describeInstance(w http.ResponseWriter, r *http.Request) {
	in, err := a.Manager.Get(r.PathValue("id"))
	if errors.Is(err, manager.ErrNotFound) {
		errorJSON(w, http.StatusNotFound, "not_found", "instance not found", nil)
		return
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
		return
	}
	events, _ := a.Manager.Events(in.ID)
	writeJSONBody(w, http.StatusOK, map[string]any{"instance": in, "events": events})
}

func (a *API) action(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var in *store.Instance
		switch action {
		case "terminate":
			// Optional body: {"keep_root_volume": true} keeps the named data volume.
			var body struct {
				KeepRootVolume bool `json:"keep_root_volume"`
			}
			if r.Body != nil {
				_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body)
			}
			in, err = a.Manager.Terminate(r.Context(), r.PathValue("id"), body.KeepRootVolume)
		default:
			in, err = a.Manager.Action(r.Context(), r.PathValue("id"), action)
		}
		switch {
		case errors.Is(err, manager.ErrNotFound):
			errorJSON(w, http.StatusNotFound, "not_found", "instance not found", nil)
		case errors.Is(err, manager.ErrConflict):
			errorJSON(w, http.StatusConflict, "conflict",
				"instance is mid-transition or action not permitted in current state", nil)
		case err != nil:
			errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
		default:
			writeJSONBody(w, http.StatusAccepted, in)
		}
	}
}

func (a *API) instanceEvents(w http.ResponseWriter, r *http.Request) {
	events, err := a.Manager.Events(r.PathValue("id"))
	if errors.Is(err, manager.ErrNotFound) {
		errorJSON(w, http.StatusNotFound, "not_found", "instance not found", nil)
		return
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
		return
	}
	if events == nil {
		events = []store.Event{}
	}
	writeJSONBody(w, http.StatusOK, events)
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func errorJSON(w http.ResponseWriter, code int, errCode, msg string, details any) {
	writeJSONBody(w, code, map[string]any{"error": apiError{Code: errCode, Message: msg, Details: details}})
}

func writeJSONBody(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
