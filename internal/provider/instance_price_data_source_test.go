// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

type fakeInstancePriceAPI struct {
	price *client.Price
	err   error
}

func (f *fakeInstancePriceAPI) GetPrice(_ context.Context, req client.PriceRequest) (*client.Price, error) {
	return f.price, f.err
}

func TestFakeInstancePriceAPI(t *testing.T) {
	f := &fakeInstancePriceAPI{price: &client.Price{PricePerMonth: 9.9, PricePerHour: 0.03}}
	p, err := f.GetPrice(context.Background(), client.PriceRequest{FlavorType: "Standard"})
	if err != nil || p.PricePerMonth != 9.9 {
		t.Fatalf("unexpected: %v %+v", err, p)
	}
}
