// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

const (
	instanceCreateTimeout = 10 * time.Minute
	instanceDeleteTimeout = 10 * time.Minute
	instancePollInterval  = 10 * time.Second
)

// InstancesAPI is the client subset this resource needs.
type InstancesAPI interface {
	CreateInstance(ctx context.Context, req client.InstanceCreate) (string, error)
	GetInstance(ctx context.Context, id string) (*client.Instance, error)
	DeleteInstance(ctx context.Context, id string) error
	WaitInstanceActive(ctx context.Context, id string, timeout, interval time.Duration) error
	WaitInstanceDeleted(ctx context.Context, id string, timeout, interval time.Duration) error

	StartInstance(ctx context.Context, id string) error
	StopInstance(ctx context.Context, id string) error
	RebootInstance(ctx context.Context, id string) error
	RenameInstance(ctx context.Context, id string, name string) error
	ChangePassword(ctx context.Context, id string, password string) error
	UpdateReverseDNS(ctx context.Context, id string, dns string, ip string) error
	AddSecurityGroup(ctx context.Context, id string, sgID string) error
	RemoveSecurityGroup(ctx context.Context, id string, sgID string) error
	ListSecurityGroups(ctx context.Context, id string) ([]client.SecurityGroup, error)
	WaitInstanceStopped(ctx context.Context, id string, timeout, interval time.Duration) error
}

type instanceResource struct {
	api InstancesAPI
}

type InstanceResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Region           types.String `tfsdk:"region"`
	FlavorType       types.String `tfsdk:"flavor_type"`
	ImageName        types.String `tfsdk:"image_name"`
	RAM              types.Int64  `tfsdk:"ram"`
	VCPUs            types.Int64  `tfsdk:"vcpus"`
	Disk             types.Int64  `tfsdk:"disk"`
	EnableIPv6       types.Bool   `tfsdk:"enable_ipv6"`
	EnablePrivateNet types.Bool   `tfsdk:"enable_private_network"`
	AutoBackup       types.Bool   `tfsdk:"auto_backup"`
	SSHKeyIDs        types.List   `tfsdk:"ssh_key_ids"`
	Status           types.String `tfsdk:"status"`
	AccessIPv4       types.String `tfsdk:"access_ipv4"`
	Created          types.String `tfsdk:"created"`

	PowerState       types.String `tfsdk:"power_state"`
	Reboot           types.Bool   `tfsdk:"reboot"`
	AdminPassword    types.String `tfsdk:"admin_password"`
	ReverseDNS       types.String `tfsdk:"reverse_dns"`
	SecurityGroupIDs types.List   `tfsdk:"security_group_ids"`

	AccessIPv6            types.String `tfsdk:"access_ipv6"`
	Username              types.String `tfsdk:"username"`
	TaskState             types.String `tfsdk:"task_state"`
	BackupServer          types.String `tfsdk:"backup_server"`
	StoppedByCloudfly     types.Bool   `tfsdk:"stopped_by_cloudfly"`
	CurrentMonthTraffic   types.String `tfsdk:"current_month_traffic"`
	CurrentMonthTrafficMB types.String `tfsdk:"current_month_traffic_mb"`
	RemainMaxIPAddon      types.String `tfsdk:"remain_max_ip_addon"`
}

func NewInstanceResource() resource.Resource { return &instanceResource{} }

func (r *instanceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "cloudfly_instance"
}

func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CloudFly compute instance.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name": schema.StringAttribute{Optional: true},
			"region": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("CLOUD-HN02", "HN-Cloud01")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"flavor_type": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("Standard", "Premium")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"image_name":             schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"ram":                    schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"vcpus":                  schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"disk":                   schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.AtLeast(20)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"enable_ipv6":            schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"enable_private_network": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"auto_backup":            schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"ssh_key_ids":            schema.ListAttribute{ElementType: types.Int64Type, Optional: true, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
			"status":                 schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"access_ipv4":            schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"created":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},

			"power_state": schema.StringAttribute{
				MarkdownDescription: "Desired power state of the instance. Valid values: `running`, `stopped`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"reboot": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` to reboot the instance. Resets to `null` after the reboot completes.",
				Optional:            true,
			},
			"admin_password": schema.StringAttribute{
				MarkdownDescription: "New administrator password for the instance.",
				Optional:            true,
				Sensitive:           true,
			},
			"reverse_dns": schema.StringAttribute{
				MarkdownDescription: "Reverse DNS entry for the instance's primary IPv4 address.",
				Optional:            true,
			},
			"security_group_ids": schema.ListAttribute{
				MarkdownDescription: "Security group IDs to assign to the instance.",
				ElementType:         types.StringType,
				Optional:            true,
			},

			"access_ipv6":              schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"username":                 schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"task_state":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"backup_server":            schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"stopped_by_cloudfly":      schema.BoolAttribute{Computed: true},
			"current_month_traffic":    schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"current_month_traffic_mb": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"remain_max_ip_addon":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *instanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createReq, diags := instanceCreateFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.api.CreateInstance(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}
	if err := r.api.WaitInstanceActive(ctx, id, instanceCreateTimeout, instancePollInterval); err != nil {
		resp.Diagnostics.AddError("Instance did not become active", err.Error())
		return
	}
	inst, err := r.api.GetInstance(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance after create", err.Error())
		return
	}
	instanceToModel(inst, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	inst, err := r.api.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance", err.Error())
		return
	}
	instanceToModel(inst, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()

	if !state.PowerState.Equal(plan.PowerState) {
		switch plan.PowerState.ValueString() {
		case "running":
			if err := r.api.StartInstance(ctx, id); err != nil {
				resp.Diagnostics.AddError("Failed to start instance", err.Error())
				return
			}
			if err := r.api.WaitInstanceActive(ctx, id, instanceCreateTimeout, instancePollInterval); err != nil {
				resp.Diagnostics.AddError("Instance did not become active after start", err.Error())
				return
			}
		case "stopped":
			if err := r.api.StopInstance(ctx, id); err != nil {
				resp.Diagnostics.AddError("Failed to stop instance", err.Error())
				return
			}
			if err := r.api.WaitInstanceStopped(ctx, id, instanceCreateTimeout, instancePollInterval); err != nil {
				resp.Diagnostics.AddError("Instance did not stop in time", err.Error())
				return
			}
		}
	}

	if plan.Reboot.ValueBool() && !state.Reboot.ValueBool() {
		if err := r.api.RebootInstance(ctx, id); err != nil {
			resp.Diagnostics.AddError("Failed to reboot instance", err.Error())
			return
		}
		if err := r.api.WaitInstanceActive(ctx, id, instanceCreateTimeout, instancePollInterval); err != nil {
			resp.Diagnostics.AddError("Instance did not become active after reboot", err.Error())
			return
		}
	}

	if !plan.AdminPassword.IsNull() && !plan.AdminPassword.IsUnknown() && plan.AdminPassword.ValueString() != "" {
		if !state.AdminPassword.Equal(plan.AdminPassword) {
			if err := r.api.ChangePassword(ctx, id, plan.AdminPassword.ValueString()); err != nil {
				resp.Diagnostics.AddError("Failed to change password", err.Error())
				return
			}
		}
	}

	if !state.Name.Equal(plan.Name) {
		if err := r.api.RenameInstance(ctx, id, plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename instance", err.Error())
			return
		}
	}

	if !state.ReverseDNS.Equal(plan.ReverseDNS) {
		if err := r.api.UpdateReverseDNS(ctx, id, plan.ReverseDNS.ValueString(), state.AccessIPv4.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to update reverse DNS", err.Error())
			return
		}
	}

	if !state.SecurityGroupIDs.Equal(plan.SecurityGroupIDs) {
		currentSGs, err := r.api.ListSecurityGroups(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list security groups", err.Error())
			return
		}

		currentIDs := make(map[string]bool)
		for _, sg := range currentSGs {
			currentIDs[sg.ID] = true
		}

		planIDs := make(map[string]bool)
		if !plan.SecurityGroupIDs.IsNull() && !plan.SecurityGroupIDs.IsUnknown() {
			var planList []string
			diags := plan.SecurityGroupIDs.ElementsAs(ctx, &planList, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			for _, sgID := range planList {
				planIDs[sgID] = true
			}
		}

		for _, sg := range currentSGs {
			if !planIDs[sg.ID] {
				if err := r.api.RemoveSecurityGroup(ctx, id, sg.ID); err != nil {
					resp.Diagnostics.AddError("Failed to remove security group", err.Error())
					return
				}
			}
		}

		for sgID := range planIDs {
			if !currentIDs[sgID] {
				if err := r.api.AddSecurityGroup(ctx, id, sgID); err != nil {
					resp.Diagnostics.AddError("Failed to add security group", err.Error())
					return
				}
			}
		}
	}

	inst, err := r.api.GetInstance(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance after update", err.Error())
		return
	}
	instanceToModel(inst, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.api.DeleteInstance(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
	if err := r.api.WaitInstanceDeleted(ctx, state.ID.ValueString(), instanceDeleteTimeout, instancePollInterval); err != nil {
		resp.Diagnostics.AddError("Instance was not deleted in time", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// instanceCreateFromModel maps model to client.InstanceCreate (pure helper, tested directly).
func instanceCreateFromModel(ctx context.Context, m InstanceResourceModel) (client.InstanceCreate, diag.Diagnostics) {
	var diags diag.Diagnostics
	var sshKeyIDs []int
	if !m.SSHKeyIDs.IsNull() && !m.SSHKeyIDs.IsUnknown() {
		diags.Append(m.SSHKeyIDs.ElementsAs(ctx, &sshKeyIDs, false)...)
	}
	return client.InstanceCreate{
		Name:          m.Name.ValueString(),
		FlavorType:    m.FlavorType.ValueString(),
		Region:        m.Region.ValueString(),
		ImageName:     m.ImageName.ValueString(),
		RAM:           int(m.RAM.ValueInt64()),
		Disk:          int(m.Disk.ValueInt64()),
		VCPUs:         int(m.VCPUs.ValueInt64()),
		EnableIPv6:    m.EnableIPv6.ValueBool(),
		EnablePrivNet: m.EnablePrivateNet.ValueBool(),
		AutoBackup:    m.AutoBackup.ValueBool(),
		SSHKeyIDs:     sshKeyIDs,
	}, diags
}

// instanceToModel maps a client.Instance into an existing model (pure helper).
// It populates every field the API returns so that Read (including import
// refresh) reconstructs a complete state. Input fields (region, ram, etc.)
// are derived from the nested flavor/image/region objects because the detail
// endpoint echoes them there rather than as top-level scalars.
func instanceToModel(inst *client.Instance, m *InstanceResourceModel) {
	m.ID = types.StringValue(inst.ID)
	m.Status = types.StringValue(inst.Status)
	m.AccessIPv4 = types.StringValue(inst.AccessIPv4)
	m.Created = types.StringValue(inst.Created)
	if inst.Name != "" {
		m.Name = types.StringValue(inst.Name)
	} else if inst.DisplayName != "" {
		m.Name = types.StringValue(inst.DisplayName)
	}
	if inst.Region.Name != "" {
		m.Region = types.StringValue(inst.Region.Name)
	}
	if inst.Flavor.FlavorGroup.Name != "" {
		m.FlavorType = types.StringValue(inst.Flavor.FlavorGroup.Name)
	}
	if inst.Image.Name != "" {
		m.ImageName = types.StringValue(inst.Image.Name)
	}
	if inst.Flavor.MemoryMB > 0 {
		m.RAM = types.Int64Value(int64(inst.Flavor.MemoryMB / 1024))
	}
	if inst.Flavor.VCPUs > 0 {
		m.VCPUs = types.Int64Value(int64(inst.Flavor.VCPUs))
	}
	if inst.Flavor.RootGB > 0 {
		m.Disk = types.Int64Value(int64(inst.Flavor.RootGB))
	}

	switch inst.Status {
	case "ACTIVE":
		m.PowerState = types.StringValue("running")
	case "SHUTOFF", "STOPPED":
		m.PowerState = types.StringValue("stopped")
	default:
		m.PowerState = types.StringValue("stopped")
	}

	m.AccessIPv6 = types.StringValue(inst.AccessIPv6)
	m.Username = types.StringValue(inst.Username)
	m.TaskState = types.StringValue(inst.TaskState)
	m.BackupServer = types.StringValue(inst.BackupServer)
	m.StoppedByCloudfly = types.BoolValue(inst.StoppedByCloudfly)
	m.CurrentMonthTraffic = types.StringValue(inst.CurrentMonthTraffic)
	m.CurrentMonthTrafficMB = types.StringValue(inst.CurrentMonthTrafficMB)
	m.RemainMaxIPAddon = types.StringValue(inst.RemainMaxIPAddon)
}
