package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"orbit/internal/manager"
	"orbit/internal/runtime"
	"orbit/internal/store"
)

func testAPI(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	api := &API{Manager: manager.New(st, runtime.NewFake())}
	return New("tok", runtime.NewFake(), api)
}

func do(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func validCreate() map[string]any {
	return map[string]any{
		"name": "web-1", "image_id": "ubuntu-24.04", "type_id": "orbit.micro", "key_id": "dev",
	}
}

func TestCreateInstanceContract(t *testing.T) {
	h := testAPI(t)

	// 202 Accepted, state provisioning
	rec := do(t, h, "POST", "/v1/instances", validCreate())
	if rec.Code != http.StatusAccepted {
		t.Fatalf("create: got %d (%s), want 202", rec.Code, rec.Body.String())
	}
	var created struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	json.Unmarshal(rec.Body.Bytes(), &created)
	if created.ID == "" || created.State != "provisioning" {
		t.Errorf("unexpected create response: %+v", created)
	}

	// Idempotent replay -> 200 with same id
	rec = do(t, h, "POST", "/v1/instances", validCreate())
	if rec.Code != http.StatusOK {
		t.Errorf("idempotent replay: got %d, want 200", rec.Code)
	}
	var replayed struct {
		ID string `json:"id"`
	}
	json.Unmarshal(rec.Body.Bytes(), &replayed)
	if replayed.ID != created.ID {
		t.Errorf("replay id %s, want %s", replayed.ID, created.ID)
	}

	// Validation failures -> 422
	bad := validCreate()
	bad["name"] = "Bad_Name!"
	if rec = do(t, h, "POST", "/v1/instances", bad); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad name: got %d, want 422", rec.Code)
	}
	noImg := validCreate()
	delete(noImg, "image_id")
	if rec = do(t, h, "POST", "/v1/instances", noImg); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("missing image: got %d, want 422", rec.Code)
	}
}

func TestDescribeAndList(t *testing.T) {
	h := testAPI(t)
	rec := do(t, h, "GET", "/v1/instances", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}

	rec = do(t, h, "GET", "/v1/instances/orb-nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("describe missing: %d, want 404", rec.Code)
	}
}

func TestActionConflictAndNotFound(t *testing.T) {
	h := testAPI(t)
	rec := do(t, h, "POST", "/v1/instances/orb-nope/start", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("action on missing: %d, want 404", rec.Code)
	}

	// Create, stop it, then stop again -> 409
	rec = do(t, h, "POST", "/v1/instances", validCreate())
	var created struct {
		ID string `json:"id"`
	}
	json.Unmarshal(rec.Body.Bytes(), &created)

	// wait for running via polling describe
	for i := 0; i < 100; i++ {
		rec = do(t, h, "GET", "/v1/instances/"+created.ID, nil)
		if bytes.Contains(rec.Body.Bytes(), []byte(`"state":"running"`)) {
			break
		}
	}
	rec = do(t, h, "POST", "/v1/instances/"+created.ID+"/stop", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("stop: %d (%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, h, "POST", "/v1/instances/"+created.ID+"/stop", nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("double stop: %d, want 409", rec.Code)
	}
}

func TestAPIRequiresAuth(t *testing.T) {
	h := testAPI(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/instances", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated create: %d, want 401", rec.Code)
	}
}
