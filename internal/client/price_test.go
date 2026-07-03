// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instances/get_price" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), "\"flavor_type\":\"Standard\"") {
			t.Errorf("body missing flavor_type: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"price_per_month":5.0,"price_per_hour":0.01}`))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	price, err := c.GetPrice(context.Background(), PriceRequest{
		FlavorType: "Standard",
		RAM:        1,
		Disk:       20,
		VCPUs:      1,
		Region:     "HN-Cloud01",
		ImageName:  "CentOS-7.9",
	})
	if err != nil {
		t.Fatalf("GetPrice error: %v", err)
	}
	if price.PricePerMonth != 5.0 || price.PricePerHour != 0.01 {
		t.Fatalf("unexpected: %+v", price)
	}
}

func TestGetPrice_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"detail":"Invalid Token."}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := NewClient(context.Background(), Config{APIKey: "k", BaseURL: srv.URL})
	if _, err := c.GetPrice(context.Background(), PriceRequest{FlavorType: "Standard", RAM: 1, Disk: 20, VCPUs: 1, Region: "HN-Cloud01", ImageName: "CentOS-7.9"}); err == nil {
		t.Fatal("expected error on 401")
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (s == sub || containsAt(s, sub)) }
func containsAt(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
