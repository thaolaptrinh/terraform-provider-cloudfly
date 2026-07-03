// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

type BackupSchedulesAPI interface {
	ListBackupSchedules(ctx context.Context, instanceID string) ([]client.BackupSchedule, error)
}

type backupSchedulesDataSource struct {
	api BackupSchedulesAPI
}

type BackupSchedulesModel struct {
	InstanceID types.String `tfsdk:"instance_id"`
	Schedules  types.List   `tfsdk:"schedules"`
}

var backupScheduleObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":          types.Int64Type,
		"instance":    types.StringType,
		"rotation":    types.Int64Type,
		"run_at":      types.StringType,
		"backup_name": types.StringType,
		"backup_type": types.StringType,
	},
}

func NewBackupSchedulesDataSource() datasource.DataSource { return &backupSchedulesDataSource{} }

func (d *backupSchedulesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_backup_schedules"
}

func (d *backupSchedulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists backup schedules for a CloudFly instance.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{Required: true, MarkdownDescription: "Instance ID."},
			"schedules": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of backup schedules.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{Computed: true},
						"instance":    schema.StringAttribute{Computed: true},
						"rotation":    schema.Int64Attribute{Computed: true},
						"run_at":      schema.StringAttribute{Computed: true},
						"backup_name": schema.StringAttribute{Computed: true},
						"backup_type": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *backupSchedulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected ProviderData type", "expected *client.Client")
		return
	}
	d.api = c
}

func (d *backupSchedulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config BackupSchedulesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedules, err := d.api.ListBackupSchedules(ctx, config.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backup schedules", err.Error())
		return
	}

	listValue, diags := schedulesToList(schedules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.Schedules = listValue
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// schedulesToList converts a slice of client.BackupSchedule into a
// framework-compatible types.List of nested objects. Pure helper, tested
// directly without a live datasource.Read.
func schedulesToList(schedules []client.BackupSchedule) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	scheduleObjs := make([]attr.Value, 0, len(schedules))
	for _, s := range schedules {
		obj, d := types.ObjectValue(backupScheduleObjectType.AttrTypes, map[string]attr.Value{
			"id":          types.Int64Value(s.ID),
			"instance":    types.StringValue(s.Instance),
			"rotation":    types.Int64Value(s.Rotation),
			"run_at":      types.StringValue(s.RunAt),
			"backup_name": types.StringValue(s.BackupName),
			"backup_type": types.StringValue(s.BackupType),
		})
		diags.Append(d...)
		scheduleObjs = append(scheduleObjs, obj)
	}
	listValue, d := types.ListValue(backupScheduleObjectType, scheduleObjs)
	diags.Append(d...)
	return listValue, diags
}
