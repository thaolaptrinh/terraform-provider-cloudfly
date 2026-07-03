// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

func TestSnapshotToModel(t *testing.T) {
	snap := &client.Snapshot{
		ID:           "snap-1",
		Name:         "my-snap",
		Status:       "available",
		Size:         1024,
		SizeInGB:     client.FlexString("1"),
		Type:         "snapshot",
		OSDistro:     "ubuntu",
		CreatedAt:    "2026-01-01",
		InstanceUUID: "inst-1",
		Description:  "test",
	}
	m := &SnapshotResourceModel{}
	snapshotToModel(snap, m)

	want := &SnapshotResourceModel{}
	want.ID = types.StringValue("snap-1")
	want.Status = types.StringValue("available")
	want.Size = types.Int64Value(1024)
	want.SizeInGB = types.StringValue("1")
	want.Type = types.StringValue("snapshot")
	want.OSDistro = types.StringValue("ubuntu")
	want.CreatedAt = types.StringValue("2026-01-01")
	want.InstanceID = types.StringValue("inst-1")
	want.Description = types.StringValue("test")

	if diff := cmp.Diff(want, m); diff != "" {
		t.Fatalf("snapshotToModel mismatch (-want +got):\n%s", diff)
	}
}
