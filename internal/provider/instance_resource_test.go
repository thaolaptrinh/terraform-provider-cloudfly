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
	instanceToModel(&client.Instance{ID: "i9", Status: "ACTIVE", AccessIPv4: "1.2.3.4", Created: "2026-01-01"}, m)
	if m.ID.ValueString() != "i9" || m.Status.ValueString() != "ACTIVE" || m.AccessIPv4.ValueString() != "1.2.3.4" {
		t.Fatalf("unexpected: %+v", m)
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
