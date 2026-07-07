// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

func TestEnrichCreateError_NonFlavorGroupError(t *testing.T) {
	mock := &mockInstancesAPI{}
	// Generic 500 — must NOT trigger enrichment (must return original error verbatim).
	orig := &client.ErrorResponse{StatusCode: 500, Body: `{"detail":"oops"}`}
	got := enrichCreateError(context.Background(), mock, "CLOUD-HN02", "Standard", orig)
	if got != orig.Error() {
		t.Fatalf("expected passthrough of non-flavor-group error, got %q", got)
	}
	if mock.getInstanceOptionsCalls != 0 {
		t.Fatalf("catalog should not be queried for unrelated errors, got %d calls", mock.getInstanceOptionsCalls)
	}
}

func TestEnrichCreateError_NonErrorResponsePassthrough(t *testing.T) {
	mock := &mockInstancesAPI{}
	// Even if the snippet appears in some random error, only API 400s are enriched.
	orig := strings.NewReader("not an error response")
	_ = orig
	got := enrichCreateError(context.Background(), mock, "CLOUD-HN02", "Standard", errBoom{})
	if got != "boom" {
		t.Fatalf("expected passthrough of non-ErrorResponse, got %q", got)
	}
}

func TestEnrichCreateError_FlavorGroupMissing_ListsAlternatives(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstanceOptions: []client.InstanceOption{
			{Region: client.OptionRef{Name: "CLOUD-HN02"}, FlavorGroup: client.FlavorGroup{Name: "Premium"}},
			{Region: client.OptionRef{Name: "HN-Cloud01"}, FlavorGroup: client.FlavorGroup{Name: "Standard"}},
			{Region: client.OptionRef{Name: "HCM-CLOUD01"}, FlavorGroup: client.FlavorGroup{Name: "Standard"}},
		},
	}
	orig := &client.ErrorResponse{
		StatusCode: 400,
		Body:       `{"detail":"The selected flavor group is not available in this region."}`,
	}
	got := enrichCreateError(context.Background(), mock, "CLOUD-HN02", "Standard", orig)

	for _, want := range []string{
		"flavor group is not available",
		`Region "CLOUD-HN02" does not have any "Standard"`,
		`Flavor groups available in "CLOUD-HN02": Premium`,
		`Regions where "Standard" IS available: HCM-CLOUD01, HN-Cloud01`,
		"cloudfly_instance_options",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("enriched error missing %q\nGot:\n%s", want, got)
		}
	}
	if mock.getInstanceOptionsCalls != 1 {
		t.Fatalf("expected exactly 1 catalog call, got %d", mock.getInstanceOptionsCalls)
	}
}

func TestEnrichCreateError_RegionUnknownToCatalog(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstanceOptions: []client.InstanceOption{
			{Region: client.OptionRef{Name: "CLOUD-HN02"}, FlavorGroup: client.FlavorGroup{Name: "Premium"}},
		},
	}
	orig := &client.ErrorResponse{StatusCode: 400, Body: `{"detail":"flavor group is not available in this region"}`}
	got := enrichCreateError(context.Background(), mock, "Unknown-Region", "Standard", orig)

	if !strings.Contains(got, "No flavor groups were returned for region") {
		t.Fatalf("expected 'no flavor groups' fallback message, got:\n%s", got)
	}
}

func TestEnrichCreateError_CatalogFails_FallsBackToOriginal(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstanceOptionsErr: errBoom{},
	}
	orig := &client.ErrorResponse{StatusCode: 400, Body: `{"detail":"flavor group is not available in this region"}`}
	got := enrichCreateError(context.Background(), mock, "CLOUD-HN02", "Standard", orig)
	if got != orig.Error() {
		t.Fatalf("catalog failure must fall back to original error; got %q", got)
	}
}

// errBoom is a trivial error type used to simulate non-ErrorResponse failures.
type errBoom struct{}

func (errBoom) Error() string { return "boom" }
