// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

func strList(t *testing.T, vals ...string) types.List {
	t.Helper()
	elems := make([]interface{}, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	l, diags := types.ListValueFrom(context.Background(), types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("strList: %v", diags)
	}
	return l
}

func tfString(v types.String) tftypes.Value {
	if v.IsNull() || v.IsUnknown() {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, v.ValueString())
}

func tfInt64(v types.Int64) tftypes.Value {
	if v.IsNull() || v.IsUnknown() {
		return tftypes.NewValue(tftypes.Number, nil)
	}
	return tftypes.NewValue(tftypes.Number, big.NewFloat(float64(v.ValueInt64())))
}

// --- Snapshot helpers ---

var snapshotTfType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id": tftypes.String, "instance_id": tftypes.String,
		"name": tftypes.String, "description": tftypes.String,
		"status": tftypes.String, "size": tftypes.Number,
		"size_in_gb": tftypes.String, "type": tftypes.String,
		"os_distro": tftypes.String, "created_at": tftypes.String,
	},
}

func newSnapshotPlan(m SnapshotResourceModel) tfsdk.Plan {
	return tfsdk.Plan{Raw: tftypes.NewValue(snapshotTfType, map[string]tftypes.Value{
		"id": tfString(m.ID), "instance_id": tfString(m.InstanceID),
		"name": tfString(m.Name), "description": tfString(m.Description),
		"status": tfString(m.Status), "size": tfInt64(m.Size),
		"size_in_gb": tfString(m.SizeInGB), "type": tfString(m.Type),
		"os_distro": tfString(m.OSDistro), "created_at": tfString(m.CreatedAt),
	})}
}

func newSnapshotState(m SnapshotResourceModel) tfsdk.State {
	return tfsdk.State{Raw: tftypes.NewValue(snapshotTfType, map[string]tftypes.Value{
		"id": tfString(m.ID), "instance_id": tfString(m.InstanceID),
		"name": tfString(m.Name), "description": tfString(m.Description),
		"status": tfString(m.Status), "size": tfInt64(m.Size),
		"size_in_gb": tfString(m.SizeInGB), "type": tfString(m.Type),
		"os_distro": tfString(m.OSDistro), "created_at": tfString(m.CreatedAt),
	})}
}

func snapshotCreateReq(m SnapshotResourceModel) resource.CreateRequest {
	return resource.CreateRequest{Plan: newSnapshotPlan(m)}
}

func snapshotReadReq(m SnapshotResourceModel) resource.ReadRequest {
	return resource.ReadRequest{State: newSnapshotState(m)}
}

func snapshotDeleteReq(m SnapshotResourceModel) resource.DeleteRequest {
	return resource.DeleteRequest{State: newSnapshotState(m)}
}

func snapshotImportReq(id string) resource.ImportStateRequest {
	return resource.ImportStateRequest{ID: id}
}

// --- Backup schedule helpers ---

var backupScheduleTfType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id": tftypes.String, "instance_id": tftypes.String,
		"name": tftypes.String, "backup_type": tftypes.String,
		"rotation": tftypes.Number, "run_at": tftypes.String,
	},
}

func newBackupSchedulePlan(m BackupScheduleResourceModel) tfsdk.Plan {
	return tfsdk.Plan{Raw: tftypes.NewValue(backupScheduleTfType, map[string]tftypes.Value{
		"id": tfString(m.ID), "instance_id": tfString(m.InstanceID),
		"name": tfString(m.Name), "backup_type": tfString(m.BackupType),
		"rotation": tfInt64(m.Rotation), "run_at": tfString(m.RunAt),
	})}
}

func newBackupScheduleState(m BackupScheduleResourceModel) tfsdk.State {
	return tfsdk.State{Raw: tftypes.NewValue(backupScheduleTfType, map[string]tftypes.Value{
		"id": tfString(m.ID), "instance_id": tfString(m.InstanceID),
		"name": tfString(m.Name), "backup_type": tfString(m.BackupType),
		"rotation": tfInt64(m.Rotation), "run_at": tfString(m.RunAt),
	})}
}

func backupScheduleCreateReq(m BackupScheduleResourceModel) resource.CreateRequest {
	return resource.CreateRequest{Plan: newBackupSchedulePlan(m)}
}

func backupScheduleReadReq(m BackupScheduleResourceModel) resource.ReadRequest {
	return resource.ReadRequest{State: newBackupScheduleState(m)}
}

func backupScheduleDeleteReq(m BackupScheduleResourceModel) resource.DeleteRequest {
	return resource.DeleteRequest{State: newBackupScheduleState(m)}
}

func backupScheduleImportReq(id string) resource.ImportStateRequest {
	return resource.ImportStateRequest{ID: id}
}

// --- Data source helpers ---

func metricsConfigReq(m InstanceMetricsModel) datasource.ReadRequest {
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"instance_id": tftypes.String, "metric_type": tftypes.String,
			"start_time": tftypes.String, "result": tftypes.String,
		},
	}
	return datasource.ReadRequest{
		Config: tfsdk.Config{Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"instance_id": tfString(m.InstanceID),
			"metric_type": tfString(m.MetricType),
			"start_time":  tfString(m.StartTime),
			"result":      tfString(m.Result),
		})},
	}
}

func usageConfigReq(m InstanceUsageModel) datasource.ReadRequest {
	objType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"instance_id": tftypes.String, "items": tftypes.String,
		},
	}
	return datasource.ReadRequest{
		Config: tfsdk.Config{Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"instance_id": tfString(m.InstanceID),
			"items":       tfString(m.Items),
		})},
	}
}
