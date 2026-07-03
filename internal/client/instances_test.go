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

func TestStartInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/start" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"Instance started"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.StartInstance(context.Background(), "i1"); err != nil {
		t.Fatalf("StartInstance error: %v", err)
	}
}

func TestStopInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/stop" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"Instance is stopping"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.StopInstance(context.Background(), "i1"); err != nil {
		t.Fatalf("StopInstance error: %v", err)
	}
}

func TestRebootInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/reboot" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"reboot server is in progress, please wait a moment"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.RebootInstance(context.Background(), "i1"); err != nil {
		t.Fatalf("RebootInstance error: %v", err)
	}
}

func TestRenameInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/rename" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"The server has been rename successfully initialized. Please wait a moment"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.RenameInstance(context.Background(), "i1", "newname"); err != nil {
		t.Fatalf("RenameInstance error: %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/change-password" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"Change instance password successfully"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.ChangePassword(context.Background(), "i1", "newpw"); err != nil {
		t.Fatalf("ChangePassword error: %v", err)
	}
}

func TestUpdateReverseDNS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/update-reverse-dns" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"Successfully updated reverse dns"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.UpdateReverseDNS(context.Background(), "i1", "host.example.com", "1.2.3.4"); err != nil {
		t.Fatalf("UpdateReverseDNS error: %v", err)
	}
}

func TestAddSecurityGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/add-security-group" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"Successfully added security group to server"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.AddSecurityGroup(context.Background(), "i1", "sg-abc123"); err != nil {
		t.Fatalf("AddSecurityGroup error: %v", err)
	}
}

func TestRemoveSecurityGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/remove-security-group" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"Successfully removed security group from server"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.RemoveSecurityGroup(context.Background(), "i1", "sg-abc123"); err != nil {
		t.Fatalf("RemoveSecurityGroup error: %v", err)
	}
}

func TestListSecurityGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/security-groups" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"sg-1","name":"default"},{"id":"sg-2","name":"web"}]`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	sgs, err := c.ListSecurityGroups(context.Background(), "i1")
	if err != nil {
		t.Fatalf("ListSecurityGroups error: %v", err)
	}
	if len(sgs) != 2 || sgs[0].ID != "sg-1" || sgs[1].ID != "sg-2" {
		t.Errorf("unexpected SGs: %+v", sgs)
	}
}

func TestWaitInstanceStopped_PollsUntilShutoff(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			_, _ = w.Write([]byte(`{"id":"i1","status":"ACTIVE"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"i1","status":"SHUTOFF"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if err := c.WaitInstanceStopped(context.Background(), "i1", 1*time.Second, 1*time.Millisecond); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls < 3 {
		t.Errorf("calls=%d, want >=3", calls)
	}
}

func TestWaitInstanceStopped_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"i1","status":"ACTIVE"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	err := c.WaitInstanceStopped(context.Background(), "i1", 50*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestCreateSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/create_snapshot" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detail":"Create snapshot is in progress, please wait a moment"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	err := c.CreateSnapshot(context.Background(), "i1", SnapshotCreate{Name: "snap1"})
	if err != nil {
		t.Fatalf("CreateSnapshot error: %v", err)
	}
}

func TestListSnapshots(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"snap-1","name":"snap1","status":"available","size":1024,"size_in_gb":"1"}]`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	snaps, err := c.ListSnapshots(context.Background(), "i1")
	if err != nil {
		t.Fatalf("ListSnapshots error: %v", err)
	}
	if len(snaps) != 1 || snaps[0].ID != "snap-1" {
		t.Errorf("unexpected: %+v", snaps)
	}
}

func TestGetSnapshot_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"snap-1","name":"snap1","status":"available"}]`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	snap, err := c.GetSnapshot(context.Background(), "i1", "snap-1")
	if err != nil {
		t.Fatalf("GetSnapshot error: %v", err)
	}
	if snap.ID != "snap-1" {
		t.Errorf("unexpected: %+v", snap)
	}
}

func TestGetSnapshot_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.GetSnapshot(context.Background(), "i1", "missing")
	if err == nil {
		t.Fatal("expected error for missing snapshot")
	}
}

func TestGetMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/i1/metrics" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"timestamp":"2026-01-01","value":42.5}]}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	resp, err := c.GetMetrics(context.Background(), "i1", "vcpu", "1h")
	if err != nil {
		t.Fatalf("GetMetrics error: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected non-nil data")
	}
	s := string(resp.Data)
	if s != `{"data":[{"timestamp":"2026-01-01","value":42.5}]}` {
		t.Errorf("unexpected data: %s", s)
	}
}

func TestGetUsageHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"date":"2026-01-01","usage_mb":500}]`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	items, err := c.GetUsageHistory(context.Background(), "i1")
	if err != nil {
		t.Fatalf("GetUsageHistory error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestGetUsageSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"csv_path":"https://example.com/summary.csv"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	resp, err := c.GetUsageSummary(context.Background())
	if err != nil {
		t.Fatalf("GetUsageSummary error: %v", err)
	}
	if resp.CSVPath != "https://example.com/summary.csv" {
		t.Errorf("unexpected csv_path: %s", resp.CSVPath)
	}
}

func TestListBackupSchedules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"instance":"i1","rotation":7,"run_at":"2026-01-01T00:00:00Z","backup_name":"weekly","backup_type":"weekly"}]`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	schedules, err := c.ListBackupSchedules(context.Background(), "i1")
	if err != nil {
		t.Fatalf("ListBackupSchedules error: %v", err)
	}
	if len(schedules) != 1 || schedules[0].BackupType != "weekly" {
		t.Errorf("unexpected: %+v", schedules)
	}
}
