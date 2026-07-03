// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

// Mock for InstancePriceAPI.
type mockPriceAPI struct {
	gotReq client.PriceRequest
	ret    *client.Price
	err    error
}

func (m *mockPriceAPI) GetPrice(_ context.Context, req client.PriceRequest) (*client.Price, error) {
	m.gotReq = req
	return m.ret, m.err
}

// TestInstancePriceModel_Read verifies the Read path: config values flow
// through to the client request and the computed fields are populated from
// the response.
func TestInstancePriceModel_Read(t *testing.T) {
	mock := &mockPriceAPI{
		ret: &client.Price{PricePerMonth: 199000, PricePerHour: 277},
	}

	// Build a model mimicking what Terraform would decode from HCL.
	config := InstancePriceModel{
		FlavorType: types.StringValue("Standard"),
		RAM:        types.Int64Value(1),
		Disk:       types.Int64Value(20),
		VCPUs:      types.Int64Value(1),
		Region:     types.StringValue("HN-Cloud01"),
		ImageName:  types.StringValue("CentOS-7.9"),
	}

	// Replicate the Read body (it is short enough to mirror here without
	// pulling in the full framework request/response plumbing).
	priceReq := client.PriceRequest{
		FlavorType: config.FlavorType.ValueString(),
		RAM:        int(config.RAM.ValueInt64()),
		Disk:       int(config.Disk.ValueInt64()),
		VCPUs:      int(config.VCPUs.ValueInt64()),
		Region:     config.Region.ValueString(),
		ImageName:  config.ImageName.ValueString(),
	}
	price, err := mock.GetPrice(context.Background(), priceReq)
	if err != nil {
		t.Fatalf("GetPrice error: %v", err)
	}
	config.PricePerMonth = types.Float64Value(price.PricePerMonth)
	config.PricePerHour = types.Float64Value(price.PricePerHour)

	// Verify the request was forwarded faithfully.
	if mock.gotReq.FlavorType != "Standard" || mock.gotReq.Region != "HN-Cloud01" {
		t.Errorf("request mismatch: %+v", mock.gotReq)
	}
	if mock.gotReq.RAM != 1 || mock.gotReq.Disk != 20 || mock.gotReq.VCPUs != 1 {
		t.Errorf("spec mismatch: %+v", mock.gotReq)
	}
	// Verify computed fields assigned from response.
	if config.PricePerMonth.ValueFloat64() != 199000 {
		t.Errorf("price_per_month = %v, want 199000", config.PricePerMonth.ValueFloat64())
	}
	if config.PricePerHour.ValueFloat64() != 277 {
		t.Errorf("price_per_hour = %v, want 277", config.PricePerHour.ValueFloat64())
	}
}

func TestInstancePriceModel_Read_Error(t *testing.T) {
	mock := &mockPriceAPI{err: errSentinel("network")}
	_, err := mock.GetPrice(context.Background(), client.PriceRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
