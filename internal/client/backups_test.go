// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateBackupSchedule(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"detail":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	err := c.CreateBackupSchedule(context.Background(), "i1", BackupScheduleCreate{Name: "daily", BackupType: "auto"})
	if err != nil {
		t.Fatalf("CreateBackupSchedule error: %v", err)
	}
	if method != http.MethodPost || path != "/instances/i1/create_backup_schedule" {
		t.Errorf("method=%s path=%s, want POST /instances/i1/create_backup_schedule", method, path)
	}
}

func TestCreateBackupScheduleError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"bad request"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	err := c.CreateBackupSchedule(context.Background(), "i1", BackupScheduleCreate{BackupType: "invalid"})
	if err == nil {
		t.Fatal("expected error from 400, got nil")
	}
}

func TestGetBackupSchedule(t *testing.T) {
	var listPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		listPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":42,"instance":"i1","rotation":7,"run_at":"03:00","backup_name":"daily","backup_type":"auto"}]`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	schedule, err := c.GetBackupSchedule(context.Background(), "i1", "42")
	if err != nil {
		t.Fatalf("GetBackupSchedule error: %v", err)
	}
	if schedule.ID != 42 || schedule.BackupType != "auto" {
		t.Errorf("unexpected: %+v", schedule)
	}
	if listPath != "/instances/i1/backup-server" {
		t.Errorf("listPath=%s, want /instances/i1/backup-server", listPath)
	}
}

func TestGetBackupScheduleNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"instance":"i1","backup_type":"weekly"}]`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	_, err := c.GetBackupSchedule(context.Background(), "i1", "999")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestGetBackupScheduleInvalidID(t *testing.T) {
	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: "http://localhost"})
	_, err := c.GetBackupSchedule(context.Background(), "i1", "abc")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestDeleteBackupSchedule(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"detail":"deleted"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	err := c.DeleteBackupSchedule(context.Background(), 42)
	if err != nil {
		t.Fatalf("DeleteBackupSchedule error: %v", err)
	}
	if method != http.MethodDelete || path != "/instances/backup-servers/42" {
		t.Errorf("method=%s path=%s, want DELETE /instances/backup-servers/42", method, path)
	}
}

func TestDeleteBackupScheduleError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not found"}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	err := c.DeleteBackupSchedule(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error from 404, got nil")
	}
}
