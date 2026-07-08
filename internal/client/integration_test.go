// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"os"
	"testing"
)

// TestIntegration_CloudRegions verifies the client authenticates against the live
// CloudFly API. Skipped unless CLOUDFLY_API_TOKEN is set — never runs in default CI.
func TestIntegration_CloudRegions(t *testing.T) {
	key := os.Getenv("CLOUDFLY_API_TOKEN")
	if key == "" {
		t.Skip("CLOUDFLY_API_TOKEN not set")
	}
	c, err := NewClient(context.Background(), Config{APIToken: key})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	resp, err := c.Do(context.Background(), http.MethodGet, "/cloud-regions", nil, nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
