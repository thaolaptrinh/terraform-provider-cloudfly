// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetInstanceOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/get_options" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"configs":[{"name":"20GB SSD","description":"Small","prices":{"price_per_month":5.0,"price_per_hour":0.01},"region":{"name":"HN-Cloud01","description":"HN"},"memory_mb":1024,"vcpus":1,"root_gb":20,"flavor_group":{"name":"Standard","description":"Std","max_ip_addon":6}}]}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	opts, err := c.GetInstanceOptions(context.Background())
	if err != nil {
		t.Fatalf("GetInstanceOptions error: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("got %d options, want 1", len(opts))
	}
	o := opts[0]
	if o.Name != "20GB SSD" || o.MemoryMB != 1024 || o.VCPUs != 1 || o.RootGB != 20 {
		t.Errorf("scalar fields wrong: %+v", o)
	}
	if o.Prices.PricePerMonth != 5.0 || o.Prices.PricePerHour != 0.01 {
		t.Errorf("prices wrong: %+v", o.Prices)
	}
	if o.Region.Name != "HN-Cloud01" || o.FlavorGroup.Name != "Standard" || o.FlavorGroup.MaxIPAddon != 6 {
		t.Errorf("nested wrong: region=%+v flavor=%+v", o.Region, o.FlavorGroup)
	}
}

func TestGetInstanceOptions_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"Invalid Token."}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if _, err := c.GetInstanceOptions(context.Background()); err == nil {
		t.Fatal("expected error on 401")
	}
}
