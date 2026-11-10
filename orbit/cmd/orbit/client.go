// Package main: API client helpers for the orbit CLI.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"orbit/internal/server"
)

// client talks to the local control plane API.
type client struct {
	base  string
	token string
	http  *http.Client
}

func newClient(url string) (*client, error) {
	if url == "" {
		url = "http://127.0.0.1:7420"
	}
	dir, err := server.DataDir()
	if err != nil {
		return nil, err
	}
	tok, err := os.ReadFile(filepath.Join(dir, "token"))
	if err != nil {
		return nil, fmt.Errorf("read API token (is `orbit serve` running?): %w", err)
	}
	return &client{
		base:  url + "/v1",
		token: string(tok),
		http:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *client) do(method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API %s %s: %d %s", method, path, resp.StatusCode, b)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
