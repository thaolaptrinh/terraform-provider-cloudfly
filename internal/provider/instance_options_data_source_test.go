package provider

import (
	"context"
	"testing"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

type fakeInstanceOptionsAPI struct {
	opts []client.InstanceOption
	err  error
}

func (f *fakeInstanceOptionsAPI) GetInstanceOptions(_ context.Context) ([]client.InstanceOption, error) {
	return f.opts, f.err
}

func TestToOptionModels(t *testing.T) {
	in := []client.InstanceOption{{
		Name:        "n",
		Description: "d",
		Prices:      client.OptionPrices{PricePerMonth: 1.5, PricePerHour: 0.02},
		Region:      client.OptionRef{Name: "r", Description: "rd"},
		MemoryMB:    1024,
		VCPUs:       2,
		RootGB:      40,
		FlavorGroup: client.FlavorGroup{Name: "Standard", Description: "sd", MaxIPAddon: 6},
	}}
	got := toOptionModels(in)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Prices.PricePerMonth.ValueFloat64() != 1.5 || got[0].FlavorGroup.MaxIPAddon.ValueInt64() != 6 {
		t.Fatalf("nested wrong: %+v", got[0])
	}
}

func TestToOptionModels_Empty(t *testing.T) {
	if got := toOptionModels(nil); len(got) != 0 {
		t.Fatalf("expected empty")
	}
}
