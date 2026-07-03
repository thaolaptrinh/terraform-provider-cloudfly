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
		ID:                    "i9",
		Status:                "ACTIVE",
		AccessIPv4:            "1.2.3.4",
		AccessIPv6:            "2001:db8::1",
		Created:               "2026-01-01",
		Name:                  "myinst",
		Region:                client.RegionRef{Name: "HN-Cloud01"},
		Flavor:                client.Flavor{MemoryMB: 1024, VCPUs: 1, RootGB: 20, FlavorGroup: client.FlavorGroup{Name: "Standard"}},
		Image:                 client.Image{Name: "CentOS-7.9"},
		Username:              "root",
		TaskState:             "spawning",
		BackupServer:          "active",
		StoppedByCloudfly:     false,
		CurrentMonthTraffic:   "100",
		CurrentMonthTrafficMB: "100",
		RemainMaxIPAddon:      "5",
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
	if m.PowerState.ValueString() != "running" {
		t.Fatalf("power_state should be running for ACTIVE, got %s", m.PowerState.ValueString())
	}
	if m.AccessIPv6.ValueString() != "2001:db8::1" {
		t.Fatalf("access_ipv6 wrong: %s", m.AccessIPv6.ValueString())
	}
	if m.Username.ValueString() != "root" {
		t.Fatalf("username wrong: %s", m.Username.ValueString())
	}
	if m.TaskState.ValueString() != "spawning" {
		t.Fatalf("task_state wrong: %s", m.TaskState.ValueString())
	}
	if m.BackupServer.ValueString() != "active" {
		t.Fatalf("backup_server wrong: %s", m.BackupServer.ValueString())
	}
	if m.RemainMaxIPAddon.ValueString() != "5" {
		t.Fatalf("remain_max_ip_addon wrong: %s", m.RemainMaxIPAddon.ValueString())
	}
}

func TestInstanceToModel_PowerStateShutoff(t *testing.T) {
	m := &InstanceResourceModel{}
	instanceToModel(&client.Instance{
		ID:     "i10",
		Status: "SHUTOFF",
		Region: client.RegionRef{Name: "HN-Cloud01"},
		Flavor: client.Flavor{FlavorGroup: client.FlavorGroup{Name: "Standard"}},
		Image:  client.Image{Name: "CentOS-7.9"},
	}, m)
	if m.PowerState.ValueString() != "stopped" {
		t.Fatalf("power_state should be stopped for SHUTOFF, got %s", m.PowerState.ValueString())
	}
}

func TestInstanceToModel_PowerStateStopped(t *testing.T) {
	m := &InstanceResourceModel{}
	instanceToModel(&client.Instance{
		ID:     "i11",
		Status: "STOPPED",
		Region: client.RegionRef{Name: "HN-Cloud01"},
		Flavor: client.Flavor{FlavorGroup: client.FlavorGroup{Name: "Standard"}},
		Image:  client.Image{Name: "CentOS-7.9"},
	}, m)
	if m.PowerState.ValueString() != "stopped" {
		t.Fatalf("power_state should be stopped for STOPPED, got %s", m.PowerState.ValueString())
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
