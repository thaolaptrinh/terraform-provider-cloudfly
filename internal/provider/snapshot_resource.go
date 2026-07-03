// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
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
	snapshotCreateTimeout = 10 * time.Minute
	snapshotPollInterval  = 10 * time.Second
)

// SnapshotAPI is the client subset this resource needs.
type SnapshotAPI interface {
	CreateSnapshot(ctx context.Context, instanceID string, req client.SnapshotCreate) error
	GetSnapshot(ctx context.Context, instanceID, snapshotID string) (*client.Snapshot, error)
	ListSnapshots(ctx context.Context, instanceID string) ([]client.Snapshot, error)
}

type snapshotResource struct {
	api SnapshotAPI
}

type SnapshotResourceModel struct {
	ID          types.String `tfsdk:"id"`
	InstanceID  types.String `tfsdk:"instance_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
	Size        types.Int64  `tfsdk:"size"`
	SizeInGB    types.String `tfsdk:"size_in_gb"`
	Type        types.String `tfsdk:"type"`
	OSDistro    types.String `tfsdk:"os_distro"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewSnapshotResource() resource.Resource { return &snapshotResource{} }

func (r *snapshotResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "cloudfly_snapshot"
}

func (r *snapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CloudFly instance snapshot.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"instance_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":        schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"description": schema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"status":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"size":        schema.Int64Attribute{Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"size_in_gb":  schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"type":        schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"os_distro":   schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"created_at":  schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *snapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *snapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SnapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instID := plan.InstanceID.ValueString()
	snapName := plan.Name.ValueString()
	desc := plan.Description.ValueString()

	if err := r.api.CreateSnapshot(ctx, instID, client.SnapshotCreate{Name: snapName, Description: desc}); err != nil {
		resp.Diagnostics.AddError("Failed to create snapshot", err.Error())
		return
	}

	deadline := time.Now().Add(snapshotCreateTimeout)
	var found *client.Snapshot
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Context cancelled", ctx.Err().Error())
			return
		default:
		}
		snaps, err := r.api.ListSnapshots(ctx, instID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list snapshots after create", err.Error())
			return
		}
		for i := range snaps {
			if snaps[i].Name == snapName {
				found = &snaps[i]
				break
			}
		}
		if found != nil {
			break
		}
		time.Sleep(snapshotPollInterval)
	}

	if found == nil {
		resp.Diagnostics.AddError("Snapshot did not appear", fmt.Sprintf("Snapshot %q was not found after creation", snapName))
		return
	}

	snapshotToModel(found, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SnapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snap, err := r.api.GetSnapshot(ctx, state.InstanceID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read snapshot", err.Error())
		return
	}
	snapshotToModel(snap, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *snapshotResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "All snapshot attributes use RequiresReplace; Update should not be reached")
}

func (r *snapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *snapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func snapshotToModel(snap *client.Snapshot, m *SnapshotResourceModel) {
	m.ID = types.StringValue(snap.ID)
	m.Name = types.StringValue(snap.Name)
	m.Status = types.StringValue(snap.Status)
	m.Size = types.Int64Value(snap.Size)
	m.SizeInGB = types.StringValue(snap.SizeInGB)
	m.Type = types.StringValue(snap.Type)
	m.OSDistro = types.StringValue(snap.OSDistro)
	m.CreatedAt = types.StringValue(snap.CreatedAt)
	if snap.InstanceUUID != "" {
		m.InstanceID = types.StringValue(snap.InstanceUUID)
	}
	if snap.Description != "" {
		m.Description = types.StringValue(snap.Description)
	}
}
