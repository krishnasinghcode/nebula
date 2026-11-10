// Package server implements the orbit control plane HTTP API.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"orbit/internal/runtime"
)

// DataDir returns the orbit data directory (created if missing).
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".orbit")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// LoadToken returns the API token, generating one on first run.
func LoadToken() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "token")
	if b, err := os.ReadFile(path); err == nil && len(b) >= 32 {
		return string(b), nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

// New builds the control plane API server. rt is the execution-environment
// adapter (nil = skip runtime health reporting). Loopback-only listener is
// the caller's responsibility (see Serve).
func New(token string, rt runtime.Adapter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		status, detail := "ok", ""
		if rt != nil {
			if err := rt.Ping(r.Context()); err != nil {
				status, detail = "degraded", "runtime unreachable: "+err.Error()
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": status, "runtime": detail})
	})
	// Unknown routes still require auth before a 404 leaks route info.
	return withAuth(token, mux)
}

// Serve listens on 127.0.0.1:port (loopback only) and serves the API.
func Serve(port int, handler http.Handler) error {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	return http.ListenAndServe(addr, handler)
}

func withAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "unauthorized: missing or invalid bearer token",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
