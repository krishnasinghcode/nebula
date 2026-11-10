package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"orbit/internal/runtime"
)

func TestHealthReportsRuntime(t *testing.T) {
	// Healthy adapter -> ok
	h := New("tok", runtime.NewFake())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Authorization", "Bearer tok")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health: got %d, want 200", rec.Code)
	}

	// Nil adapter -> ok without runtime check
	h = New("tok", nil)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Authorization", "Bearer tok")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health nil adapter: got %d, want 200", rec.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	f := runtime.NewFake()
	h := New("secrettoken1234567890abcdef", f)

	// No token -> 401
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: got %d, want 401", rec.Code)
	}

	// Wrong token -> 401
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", rec.Code)
	}

	// Unknown route with valid token -> 404
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/instances", nil)
	req.Header.Set("Authorization", "Bearer secrettoken1234567890abcdef")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown route: got %d, want 404", rec.Code)
	}

	// Health with valid token -> 200
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Authorization", "Bearer secrettoken1234567890abcdef")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("health: got %d, want 200", rec.Code)
	}
}

func TestLoadTokenPersists(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir()) // Windows
	t.Setenv("HOME", t.TempDir())        // Unix

	tok1, err := LoadToken()
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if len(tok1) < 32 {
		t.Fatalf("token too short: %d", len(tok1))
	}
	tok2, err := LoadToken()
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if tok1 != tok2 {
		t.Error("token regenerated on second load; want stable")
	}
}
