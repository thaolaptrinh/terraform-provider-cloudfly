// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

const (
	backupScheduleCreateTimeout = 2 * time.Minute
	backupSchedulePollInterval  = 5 * time.Second
)

type BackupScheduleAPI interface {
	CreateBackupSchedule(ctx context.Context, instanceID string, req client.BackupScheduleCreate) error
	ListBackupSchedules(ctx context.Context, instanceID string) ([]client.BackupSchedule, error)
	GetBackupSchedule(ctx context.Context, instanceID, scheduleID string) (*client.BackupSchedule, error)
	DeleteBackupSchedule(ctx context.Context, scheduleID int64) error
}

type backupScheduleResource struct {
	api BackupScheduleAPI
}

type BackupScheduleResourceModel struct {
	ID         types.String `tfsdk:"id"`
	InstanceID types.String `tfsdk:"instance_id"`
	Name       types.String `tfsdk:"name"`
	BackupType types.String `tfsdk:"backup_type"`
	Rotation   types.Int64  `tfsdk:"rotation"`
	RunAt      types.String `tfsdk:"run_at"`
}

func NewBackupScheduleResource() resource.Resource { return &backupScheduleResource{} }

func (r *backupScheduleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "cloudfly_backup_schedule"
}

func (r *backupScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CloudFly instance backup schedule.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"instance_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":        schema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"backup_type": schema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"rotation":    schema.Int64Attribute{Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"run_at":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *backupScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected ProviderData type", "expected *client.Client")
		return
	}
	r.api = c
}

func (r *backupScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BackupScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instID := plan.InstanceID.ValueString()
	backupType := plan.BackupType.ValueString()
	if backupType == "" {
		backupType = "weekly"
	}

	createReq := client.BackupScheduleCreate{
		Name:       plan.Name.ValueString(),
		BackupType: backupType,
	}

	if err := r.api.CreateBackupSchedule(ctx, instID, createReq); err != nil {
		resp.Diagnostics.AddError("Failed to create backup schedule", err.Error())
		return
	}

	found, err := waitForBackupSchedule(ctx, r.api, instID, backupType, plan.Name.ValueString(), backupScheduleCreateTimeout, backupSchedulePollInterval)
	if err != nil {
		resp.Diagnostics.AddError("Backup schedule did not appear", err.Error())
		return
	}

	backupScheduleToModel(found, &plan)
	plan.BackupType = types.StringValue(backupType)
	if plan.Name.ValueString() != "" {
		plan.Name = types.StringValue(plan.Name.ValueString())
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func waitForBackupSchedule(ctx context.Context, api BackupScheduleAPI, instID, backupType, name string, timeout, interval time.Duration) (*client.BackupSchedule, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		schedules, err := api.ListBackupSchedules(ctx, instID)
		if err != nil {
			return nil, fmt.Errorf("list backup schedules: %w", err)
		}
		for i := range schedules {
			if schedules[i].Instance == instID && schedules[i].BackupType == backupType {
				if name == "" || strings.Contains(schedules[i].BackupName, name) {
					return &schedules[i], nil
				}
			}
		}
		time.Sleep(interval)
	}
	return nil, fmt.Errorf("backup schedule (type=%q, name=%q) not found on instance %q within %s", backupType, name, instID, timeout)
}

func (r *backupScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BackupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule, err := r.api.GetBackupSchedule(ctx, state.InstanceID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read backup schedule", err.Error())
		return
	}
	backupScheduleToModel(schedule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *backupScheduleResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "All backup schedule attributes use RequiresReplace; Update should not be reached")
}

func (r *backupScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BackupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseScheduleID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid schedule id", err.Error())
		return
	}
	if err := r.api.DeleteBackupSchedule(ctx, id); err != nil {
		resp.Diagnostics.AddError("Failed to delete backup schedule", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *backupScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func backupScheduleToModel(bs *client.BackupSchedule, m *BackupScheduleResourceModel) {
	m.ID = types.StringValue(fmt.Sprintf("%d", bs.ID))
	m.Rotation = types.Int64Value(bs.Rotation)
	m.RunAt = types.StringValue(bs.RunAt)
	if bs.Instance != "" {
		m.InstanceID = types.StringValue(bs.Instance)
	}
	if bs.BackupName != "" {
		m.Name = types.StringValue(bs.BackupName)
	}
	if bs.BackupType != "" {
		m.BackupType = types.StringValue(bs.BackupType)
	}
}

func parseScheduleID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid schedule id %q: %w", s, err)
	}
	return id, nil
}
