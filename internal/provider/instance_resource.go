// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	GetInstanceOptions(ctx context.Context) ([]client.InstanceOption, error)
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
	ListInterfaces(ctx context.Context, id string) ([]client.InterfaceGroup, error)
	AttachInterface(ctx context.Context, id, networkID string) error
	DetachInterface(ctx context.Context, id, interfaceID string) error
	WaitInstanceStopped(ctx context.Context, id string, timeout, interval time.Duration) error
	EnableIPv6Range(ctx context.Context, id string) error
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
	NetworkIDs       types.List   `tfsdk:"network_ids"`

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
				MarkdownDescription: "CloudFly region, e.g. `CLOUD-HN02`, `HN-Cloud01`, `HCM-CLOUD01`, `CLOUD-DN01`. " +
					"Run `terraform plan` after switching regions: CloudFly's backend rejects region+flavor_type " +
					"combinations that have no matching catalog entry (e.g. `Standard` is not available in `CLOUD-HN02`). " +
					"Use the `cloudfly_instance_options` data source to discover valid combinations.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"flavor_type": schema.StringAttribute{
				MarkdownDescription: "CloudFly flavor group, e.g. `Standard`, `Premium`. Availability is region-specific: " +
					"`HN-Cloud01` currently exposes `Standard` configs, `CLOUD-HN02` exposes `Premium` configs. " +
					"Use the `cloudfly_instance_options` data source to list valid groups per region.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"image_name":             schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"ram":                    schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"vcpus":                  schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"disk":                   schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.AtLeast(20)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"enable_ipv6":            schema.BoolAttribute{Optional: true},
			"enable_private_network": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"auto_backup":            schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"ssh_key_ids":            schema.ListAttribute{ElementType: types.Int64Type, Optional: true, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
			"status":                 schema.StringAttribute{Computed: true},
			"access_ipv4":            schema.StringAttribute{Computed: true},
			"created":                schema.StringAttribute{Computed: true},

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
			"network_ids": schema.ListAttribute{
				MarkdownDescription: "Additional network IDs to attach to the instance. The default public network is managed automatically and excluded from this list.",
				ElementType:         types.StringType,
				Optional:            true,
			},

			"access_ipv6":              schema.StringAttribute{Computed: true},
			"username":                 schema.StringAttribute{Computed: true},
			"task_state":               schema.StringAttribute{Computed: true},
			"backup_server":            schema.StringAttribute{Computed: true},
			"stopped_by_cloudfly":      schema.BoolAttribute{Computed: true},
			"current_month_traffic":    schema.StringAttribute{Computed: true},
			"current_month_traffic_mb": schema.StringAttribute{Computed: true},
			"remain_max_ip_addon":      schema.StringAttribute{Computed: true},
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
		resp.Diagnostics.AddError("Failed to create instance", enrichCreateError(ctx, r.api, createReq.Region, createReq.FlavorType, err))
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
	if err := r.applyUpdate(ctx, &state, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to update instance", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// applyUpdate contains the update decision logic: power state, reboot,
// password, rename, reverse DNS and security-group diff. It mutates plan
// in place. Pure enough to unit-test without a live Terraform request.
func (r *instanceResource) applyUpdate(ctx context.Context, state, plan *InstanceResourceModel) error {
	id := state.ID.ValueString()

	if !state.PowerState.Equal(plan.PowerState) {
		switch plan.PowerState.ValueString() {
		case "running":
			if err := r.api.StartInstance(ctx, id); err != nil {
				return fmt.Errorf("start instance: %w", err)
			}
			if err := r.api.WaitInstanceActive(ctx, id, instanceCreateTimeout, instancePollInterval); err != nil {
				return fmt.Errorf("wait active after start: %w", err)
			}
		case "stopped":
			if err := r.api.StopInstance(ctx, id); err != nil {
				return fmt.Errorf("stop instance: %w", err)
			}
			if err := r.api.WaitInstanceStopped(ctx, id, instanceCreateTimeout, instancePollInterval); err != nil {
				return fmt.Errorf("wait stopped: %w", err)
			}
		default:
			return fmt.Errorf("invalid power_state %q: must be running or stopped", plan.PowerState.ValueString())
		}
	}

	if plan.Reboot.ValueBool() && !state.Reboot.ValueBool() {
		if err := r.api.RebootInstance(ctx, id); err != nil {
			return fmt.Errorf("reboot instance: %w", err)
		}
		if err := r.api.WaitInstanceActive(ctx, id, instanceCreateTimeout, instancePollInterval); err != nil {
			return fmt.Errorf("wait active after reboot: %w", err)
		}
	}

	if !plan.AdminPassword.IsNull() && !plan.AdminPassword.IsUnknown() && plan.AdminPassword.ValueString() != "" {
		if !state.AdminPassword.Equal(plan.AdminPassword) {
			if err := r.api.ChangePassword(ctx, id, plan.AdminPassword.ValueString()); err != nil {
				return fmt.Errorf("change password: %w", err)
			}
		}
	}

	if !state.Name.Equal(plan.Name) {
		if err := r.api.RenameInstance(ctx, id, plan.Name.ValueString()); err != nil {
			return fmt.Errorf("rename instance: %w", err)
		}
	}

	if !state.ReverseDNS.Equal(plan.ReverseDNS) {
		newDNS := plan.ReverseDNS.ValueString()
		if newDNS == "" {
			// API rejects blank reverse_dns; skip when user clears it.
		} else if err := r.api.UpdateReverseDNS(ctx, id, newDNS, state.AccessIPv4.ValueString()); err != nil {
			return fmt.Errorf("update reverse DNS: %w", err)
		}
	}

	if !state.EnableIPv6.Equal(plan.EnableIPv6) {
		if plan.EnableIPv6.ValueBool() && !state.EnableIPv6.ValueBool() {
			if err := r.api.EnableIPv6Range(ctx, id); err != nil {
				return fmt.Errorf("enable ipv6: %w", err)
			}
		}
	}

	if !state.SecurityGroupIDs.Equal(plan.SecurityGroupIDs) {
		if err := r.reconcileSecurityGroups(ctx, id, plan.SecurityGroupIDs); err != nil {
			return err
		}
	}

	if !state.NetworkIDs.Equal(plan.NetworkIDs) {
		if err := r.reconcileNetworks(ctx, id, plan.NetworkIDs); err != nil {
			return err
		}
	}

	inst, err := r.api.GetInstance(ctx, id)
	if err != nil {
		return fmt.Errorf("read instance after update: %w", err)
	}
	instanceToModel(inst, plan)
	return nil
}

// reconcileSecurityGroups brings the instance's attached security groups
// in line with the plan list.
func (r *instanceResource) reconcileSecurityGroups(ctx context.Context, id string, planList types.List) error {
	currentSGs, err := r.api.ListSecurityGroups(ctx, id)
	if err != nil {
		return fmt.Errorf("list security groups: %w", err)
	}

	currentIDs := make(map[string]bool, len(currentSGs))
	for _, sg := range currentSGs {
		currentIDs[sg.ID] = true
	}

	planIDs := make(map[string]bool)
	if !planList.IsNull() && !planList.IsUnknown() {
		var ids []string
		if diags := planList.ElementsAs(ctx, &ids, false); diags.HasError() {
			return fmt.Errorf("decode security_group_ids: %v", diags.Errors())
		}
		for _, sgID := range ids {
			planIDs[sgID] = true
		}
	}

	for _, sg := range currentSGs {
		if !planIDs[sg.ID] {
			if err := r.api.RemoveSecurityGroup(ctx, id, sg.ID); err != nil {
				return fmt.Errorf("remove security group %s: %w", sg.ID, err)
			}
		}
	}

	for sgID := range planIDs {
		if !currentIDs[sgID] {
			if err := r.api.AddSecurityGroup(ctx, id, sgID); err != nil {
				return fmt.Errorf("add security group %s: %w", sgID, err)
			}
		}
	}
	return nil
}

// reconcileNetworks brings the instance's attached networks in line with the
// plan list. Attach uses network_id; detach uses interface_id (one network
// may have multiple interfaces). The default public network is excluded.
func (r *instanceResource) reconcileNetworks(ctx context.Context, id string, planList types.List) error {
	if planList.IsNull() || planList.IsUnknown() {
		return nil
	}

	groups, err := r.api.ListInterfaces(ctx, id)
	if err != nil {
		return fmt.Errorf("list interfaces: %w", err)
	}

	planIDs := make(map[string]bool)
	var planSlice []string
	if diags := planList.ElementsAs(ctx, &planSlice, false); diags.HasError() {
		return fmt.Errorf("decode network_ids: %v", diags.Errors())
	}
	for _, nid := range planSlice {
		planIDs[nid] = true
	}

	currentNetworks := make(map[string]bool)
	currentInterfaces := make(map[string][]string) // networkID -> []interfaceID

	for _, group := range groups {
		for _, item := range group.Data {
			if item.IsDefault && group.IsPublic {
				continue
			}
			currentNetworks[item.NetworkID] = true
			currentInterfaces[item.NetworkID] = append(currentInterfaces[item.NetworkID], item.InterfaceID)
		}
	}

	for _, nid := range planSlice {
		if !currentNetworks[nid] {
			if err := r.api.AttachInterface(ctx, id, nid); err != nil {
				return fmt.Errorf("attach network %s: %w", nid, err)
			}
		}
	}

	for nid := range currentNetworks {
		if !planIDs[nid] {
			for _, ifID := range currentInterfaces[nid] {
				if err := r.api.DetachInterface(ctx, id, ifID); err != nil {
					return fmt.Errorf("detach interface %s: %w", ifID, err)
				}
			}
		}
	}

	return nil
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
	m.BackupServer = types.StringValue(string(inst.BackupServer))
	m.StoppedByCloudfly = types.BoolValue(inst.StoppedByCloudfly)
	m.CurrentMonthTraffic = types.StringValue(string(inst.CurrentMonthTraffic))
	m.CurrentMonthTrafficMB = types.StringValue(string(inst.CurrentMonthTrafficMB))
	m.RemainMaxIPAddon = types.StringValue(string(inst.RemainMaxIPAddon))
}

// flavorGroupUnavailableSnippet is the substring CloudFly's API returns (as
// the `detail` field of a 400 response) when the requested flavor group has
// no catalog entries in the requested region. Detected by enrichCreateError
// to provide a more actionable diagnostic.
const flavorGroupUnavailableSnippet = "flavor group is not available in this region"

// enrichCreateError inspects an error returned by CreateInstance and, when
// it recognises the "flavor group not available in region" failure, queries
// the catalog to produce a diagnostic with the available alternatives. Any
// failure during enrichment (e.g. the catalog call also fails) falls back to
// the original error message verbatim so we never mask the real cause.
func enrichCreateError(ctx context.Context, api InstancesAPI, region, flavorType string, createErr error) string {
	apiErr, ok := createErr.(*client.ErrorResponse)
	if !ok || apiErr.StatusCode != 400 || !strings.Contains(apiErr.Body, flavorGroupUnavailableSnippet) {
		return createErr.Error()
	}

	opts, err := api.GetInstanceOptions(ctx)
	if err != nil {
		return createErr.Error()
	}

	// Build region -> set of flavor group names from the catalog.
	groupsByRegion := map[string]map[string]struct{}{}
	for _, opt := range opts {
		r := opt.Region.Name
		if groupsByRegion[r] == nil {
			groupsByRegion[r] = map[string]struct{}{}
		}
		groupsByRegion[r][opt.FlavorGroup.Name] = struct{}{}
	}

	var (
		requestedGroups []string
		otherRegions    []string
		seenRegion      = map[string]struct{}{}
	)
	if groups, ok := groupsByRegion[region]; ok {
		for g := range groups {
			requestedGroups = append(requestedGroups, g)
		}
		sort.Strings(requestedGroups)
	}
	for r, groups := range groupsByRegion {
		if r == region {
			continue
		}
		if _, ok := groups[flavorType]; !ok {
			continue
		}
		if _, seen := seenRegion[r]; !seen {
			otherRegions = append(otherRegions, r)
			seenRegion[r] = struct{}{}
		}
	}
	sort.Strings(otherRegions)

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", createErr.Error())
	fmt.Fprintf(&b, "Region %q does not have any %q flavor group configs.\n", region, flavorType)
	if len(requestedGroups) > 0 {
		fmt.Fprintf(&b, "Flavor groups available in %q: %s.\n", region, strings.Join(requestedGroups, ", "))
	} else {
		fmt.Fprintf(&b, "No flavor groups were returned for region %q — the region may be unavailable for your account.\n", region)
	}
	if len(otherRegions) > 0 {
		fmt.Fprintf(&b, "Regions where %q IS available: %s.\n", flavorType, strings.Join(otherRegions, ", "))
	}
	b.WriteString("Use the `cloudfly_instance_options` data source to list valid combinations.")
	return b.String()
}
