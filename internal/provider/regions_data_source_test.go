package provider

import (
	"context"
	"testing"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

type fakeRegionsAPI struct {
	regions []client.Region
	err     error
}

func (f *fakeRegionsAPI) ListRegions(ctx context.Context) ([]client.Region, error) {
	return f.regions, f.err
}

func TestToRegionModels(t *testing.T) {
	in := []client.Region{{ID: "r1", Name: "HN", Description: "d"}}
	got := toRegionModels(in)
	if len(got) != 1 || got[0].ID.ValueString() != "r1" || got[0].Name.ValueString() != "HN" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestToRegionModels_Empty(t *testing.T) {
	got := toRegionModels(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}
