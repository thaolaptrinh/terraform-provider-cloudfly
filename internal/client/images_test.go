// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListImages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":2,"results":[{"id":"img-1","name":"CentOS-7.9"},{"id":"img-2","name":"Ubuntu-22.04"}]}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	images, err := c.ListImages(context.Background())
	if err != nil {
		t.Fatalf("ListImages error: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].ID != "img-1" || images[0].Name != "CentOS-7.9" {
		t.Errorf("unexpected image[0]: %+v", images[0])
	}
	if images[1].ID != "img-2" || images[1].Name != "Ubuntu-22.04" {
		t.Errorf("unexpected image[1]: %+v", images[1])
	}
}

func TestListImages_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	images, err := c.ListImages(context.Background())
	if err != nil {
		t.Fatalf("ListImages error: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("expected 0 images, got %d", len(images))
	}
}

func TestListImages_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"unauthorized"}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	_, err := c.ListImages(context.Background())
	if err == nil {
		t.Fatal("expected error from 401, got nil")
	}
}
