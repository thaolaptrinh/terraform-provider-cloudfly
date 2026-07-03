// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

func TestInstanceToModel(t *testing.T) {
	m := &InstanceResourceModel{}
	instanceToModel(&client.Instance{
		ID:         "i9",
		Status:     "ACTIVE",
		AccessIPv4: "1.2.3.4",
		Created:    "2026-01-01",
		Name:       "myinst",
		Region:     client.RegionRef{Name: "HN-Cloud01"},
		Flavor:     client.Flavor{MemoryMB: 1024, VCPUs: 1, RootGB: 20, FlavorGroup: client.FlavorGroup{Name: "Standard"}},
		Image:      client.Image{Name: "CentOS-7.9"},
	}, m)
	if m.ID.ValueString() != "i9" || m.Status.ValueString() != "ACTIVE" || m.AccessIPv4.ValueString() != "1.2.3.4" {
		t.Fatalf("computed wrong: %+v", m)
	}
	if m.Region.ValueString() != "HN-Cloud01" || m.FlavorType.ValueString() != "Standard" || m.ImageName.ValueString() != "CentOS-7.9" {
		t.Fatalf("derived inputs wrong: %+v", m)
	}
	if m.RAM.ValueInt64() != 1 || m.VCPUs.ValueInt64() != 1 || m.Disk.ValueInt64() != 20 {
		t.Fatalf("derived specs wrong: %+v", m)
	}
}

func TestInstanceCreateFromModel(t *testing.T) {
	m := InstanceResourceModel{}
	// Leave SSHKeyIDs null to skip the ElementsAs path.
	req, diags := instanceCreateFromModel(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if req.SSHKeyIDs != nil {
		t.Errorf("expected nil SSHKeyIDs for null list, got %v", req.SSHKeyIDs)
	}
}
