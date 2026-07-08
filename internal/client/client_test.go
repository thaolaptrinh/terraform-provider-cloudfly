// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient(context.Background(), Config{APIToken: "k"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, DefaultBaseURL)
	}
	if c.HTTPClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", c.HTTPClient.Timeout)
	}
}

func TestNewClient_NormalizesBaseURL(t *testing.T) {
	c, err := NewClient(context.Background(), Config{APIToken: "k", BaseURL: "https://example.com/api/"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BaseURL != "https://example.com/api" {
		t.Errorf("BaseURL = %q, want trailing slash trimmed", c.BaseURL)
	}
}

func TestNewClient_MissingAPIToken(t *testing.T) {
	if _, err := NewClient(context.Background(), Config{}); err == nil {
		t.Fatal("expected error when APIToken empty, got nil")
	}
}

func TestDo_SetsAuthAndAcceptHeaders(t *testing.T) {
	var gotAuth, gotAccept, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIToken: "secret", BaseURL: srv.URL})
	_, _ = c.Do(context.Background(), http.MethodGet, "/ping", nil, nil)

	if gotAuth != "Token secret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Token secret")
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if gotContentType != "" {
		t.Errorf("Content-Type = %q, want empty for nil body", gotContentType)
	}
}

func TestDo_SetsContentTypeOnBody(t *testing.T) {
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	_, _ = c.Do(context.Background(), http.MethodPost, "/x", map[string]string{"a": "b"}, nil)

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
}

func TestDo_RetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway) // 502 -> retry
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIToken: "k", BaseURL: srv.URL})
	resp, err := c.Do(context.Background(), http.MethodGet, "/x", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if calls < 3 {
		t.Errorf("calls = %d, expected at least 3 (retries)", calls)
	}
}

func TestAsError_SuccessNil(t *testing.T) {
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}
	if err := AsError(resp); err != nil {
		t.Errorf("expected nil for 2xx, got %v", err)
	}
}

func TestAsError_4xx(t *testing.T) {
	resp := &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(`{"detail":"not found"}`))}
	err := AsError(resp)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	er, ok := err.(*ErrorResponse)
	if !ok {
		t.Fatalf("expected *ErrorResponse, got %T", err)
	}
	if er.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", er.StatusCode)
	}
}
