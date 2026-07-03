// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListSSHKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/ssh-keys" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":1,"results":[{"id":42,"name":"mykey","public_key":"ssh-rsa AAAA...","fingerprint":"ab:cd","created_at":"2026-01-01T00:00:00Z"}]}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	keys, err := c.ListSSHKeys(context.Background())
	if err != nil {
		t.Fatalf("ListSSHKeys error: %v", err)
	}
	if len(keys) != 1 || keys[0].ID != 42 || keys[0].Name != "mykey" || keys[0].Fingerprint != "ab:cd" {
		t.Fatalf("unexpected: %+v", keys)
	}
}

func TestListSSHKeys_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"detail":"Invalid Token."}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if _, err := c.ListSSHKeys(context.Background()); err == nil {
		t.Fatal("expected error on 401")
	}
}
