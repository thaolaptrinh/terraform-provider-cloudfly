// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRegions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cloud-regions" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":2,"results":[{"id":"r1","name":"HN-Cloud01","description":"HN"},{"id":"r2","name":"CLOUD-HN02","description":"HN2"}]}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	regions, err := c.ListRegions(context.Background())
	if err != nil {
		t.Fatalf("ListRegions error: %v", err)
	}
	if len(regions) != 2 || regions[0].ID != "r1" || regions[0].Name != "HN-Cloud01" {
		t.Fatalf("unexpected: %+v", regions)
	}
}

func TestListRegions_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Invalid Token."}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	if _, err := c.ListRegions(context.Background()); err == nil {
		t.Fatal("expected error on 401")
	}
}
