// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

func TestToSSHKeyModels(t *testing.T) {
	in := []client.SSHKey{{ID: 7, Name: "k", PublicKey: "pk", Fingerprint: "fp", CreatedAt: "2026"}}
	got := toSSHKeyModels(in)
	if len(got) != 1 || got[0].ID.ValueInt64() != 7 || got[0].Name.ValueString() != "k" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestToSSHKeyModels_Empty(t *testing.T) {
	if got := toSSHKeyModels(nil); len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}
