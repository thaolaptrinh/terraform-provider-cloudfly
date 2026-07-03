// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCreateInstance_PollsSearchByName(t *testing.T) {
	// POST returns only {detail}; then GET /instances?search= finds the instance by display_name.
	var postCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/instances":
			atomic.AddInt32(&postCalls, 1)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"detail":"The server has been successfully initialized. Please wait a moment"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/instances":
			if r.URL.Query().Get("search") != "myinst" {
				t.Errorf("expected search=myinst, got %q", r.URL.Query().Get("search"))
			}
			_, _ = w.Write([]byte(`{"count":1,"results":[{"id":"inst-123","display_name":"myinst","status":"BUILDING"}]}`))
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL)
		}
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	id, err := c.CreateInstance(context.Background(), InstanceCreate{
		Name: "myinst", FlavorType: "Standard", Region: "HN-Cloud01", ImageName: "CentOS-7.9",
		RAM: 1, Disk: 20, VCPUs: 1,
	})
	if err != nil {
		t.Fatalf("CreateInstance error: %v", err)
	}
	if id != "inst-123" {
		t.Errorf("id = %q, want inst-123", id)
	}
}

func TestGetInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i9" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"i9","display_name":"dn","name":"n","status":"ACTIVE","region":{"id":"HN-Cloud01","name":"HN-Cloud01","description":"HN"},"accessIPv4":"1.2.3.4","created":"2026-01-01T00:00:00Z"}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	inst, err := c.GetInstance(context.Background(), "i9")
	if err != nil {
		t.Fatalf("GetInstance error: %v", err)
	}
	if inst.ID != "i9" || inst.Status != "ACTIVE" || inst.AccessIPv4 != "1.2.3.4" {
		t.Errorf("unexpected: %+v", inst)
	}
}

func TestDeleteInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1" || r.Method != http.MethodDelete {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"detail":"Delete instance is in progress, please wait a moment"}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.DeleteInstance(context.Background(), "i1"); err != nil {
		t.Fatalf("DeleteInstance error: %v", err)
	}
}

func TestWaitInstanceActive_PollsUntilActive(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			_, _ = w.Write([]byte(`{"id":"i1","status":"BUILDING"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"i1","status":"ACTIVE"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.WaitInstanceActive(context.Background(), "i1", 1*time.Second, 1*time.Millisecond); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls < 3 {
		t.Errorf("calls=%d, want >=3", calls)
	}
}

func TestWaitInstanceActive_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"i1","status":"BUILDING"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	err := c.WaitInstanceActive(context.Background(), "i1", 50*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestWaitInstanceDeleted_PollsUntil404(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"i1","status":"ACTIVE"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not found"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.WaitInstanceDeleted(context.Background(), "i1", 1*time.Second, 1*time.Millisecond); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls < 2 {
		t.Errorf("calls=%d, want >=2", calls)
	}
}
