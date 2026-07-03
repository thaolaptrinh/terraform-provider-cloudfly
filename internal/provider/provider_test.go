// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudfly": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("CLOUDFLY_API_KEY") == "" {
		t.Skip("CLOUDFLY_API_KEY not set; skipping acceptance test")
	}
}

func TestBuildClient_FromConfig(t *testing.T) {
	c, err := buildClient(context.Background(), "cfg-key", "", "env-key", "env-url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.APIKey != "cfg-key" {
		t.Errorf("APIKey = %q, want cfg-key (config wins)", c.APIKey)
	}
}

func TestBuildClient_EnvFallback(t *testing.T) {
	c, err := buildClient(context.Background(), "", "", "env-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.APIKey != "env-key" {
		t.Errorf("APIKey = %q, want env-key fallback", c.APIKey)
	}
}

func TestBuildClient_MissingKey(t *testing.T) {
	if _, err := buildClient(context.Background(), "", "", "", ""); err == nil {
		t.Fatal("expected error when no api_key source, got nil")
	}
}

func TestBuildClient_EnvBaseURLFallback(t *testing.T) {
	c, err := buildClient(context.Background(), "k", "", "k", "https://env.example/api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BaseURL != "https://env.example/api" {
		t.Errorf("BaseURL = %q, want env fallback", c.BaseURL)
	}
}
