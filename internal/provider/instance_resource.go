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
			"name": schema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
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

func (r *instanceResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes use RequiresReplace; framework should never call Update.
	resp.Diagnostics.AddError("Update not supported", "cloudfly_instance replaces on every change; Update should not be reached")
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
func instanceToModel(inst *client.Instance, m *InstanceResourceModel) {
	m.ID = types.StringValue(inst.ID)
	m.Status = types.StringValue(inst.Status)
	m.AccessIPv4 = types.StringValue(inst.AccessIPv4)
	m.Created = types.StringValue(inst.Created)
}
