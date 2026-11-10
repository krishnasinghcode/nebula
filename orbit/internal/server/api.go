package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"orbit/internal/manager"
	"orbit/internal/store"
)

// API holds the control-plane dependencies for the v1 routes.
type API struct {
	Manager *manager.Manager
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
	created, existing, err := a.Manager.Create(r.Context(), in)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
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
	}
	if req.TypeID == "" {
		problems = append(problems, "type_id is required")
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
		in, err := a.Manager.Action(r.Context(), r.PathValue("id"), action)
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
