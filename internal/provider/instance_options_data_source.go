// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

// InstanceOptionsAPI is the client subset this data source needs. Defined
// here so unit tests can inject a fake. *client.Client satisfies this interface.
type InstanceOptionsAPI interface {
	GetInstanceOptions(ctx context.Context) ([]client.InstanceOption, error)
}

type instanceOptionsDataSource struct {
	api InstanceOptionsAPI
}

type InstanceOptionsModel struct {
	Options []InstanceOptionModel `tfsdk:"options"`
}

type InstanceOptionModel struct {
	Name        types.String      `tfsdk:"name"`
	Description types.String      `tfsdk:"description"`
	Prices      OptionPricesModel `tfsdk:"prices"`
	Region      OptionRefModel    `tfsdk:"region"`
	MemoryMB    types.Int64       `tfsdk:"memory_mb"`
	VCPUs       types.Int64       `tfsdk:"vcpus"`
	RootGB      types.Int64       `tfsdk:"root_gb"`
	FlavorGroup FlavorGroupModel  `tfsdk:"flavor_group"`
}

type OptionPricesModel struct {
	PricePerMonth types.Float64 `tfsdk:"price_per_month"`
	PricePerHour  types.Float64 `tfsdk:"price_per_hour"`
}

type OptionRefModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type FlavorGroupModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	MaxIPAddon  types.Int64  `tfsdk:"max_ip_addon"`
}

func NewInstanceOptionsDataSource() datasource.DataSource { return &instanceOptionsDataSource{} }

func (d *instanceOptionsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_instance_options"
}

func (d *instanceOptionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists available CloudFly instance options (flavors) across regions.",
		Attributes: map[string]schema.Attribute{
			"options": schema.ListNestedAttribute{
				MarkdownDescription: "List of instance options.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"prices": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"price_per_month": schema.Float64Attribute{Computed: true},
								"price_per_hour":  schema.Float64Attribute{Computed: true},
							},
						},
						"region": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"name":        schema.StringAttribute{Computed: true},
								"description": schema.StringAttribute{Computed: true},
							},
						},
						"memory_mb": schema.Int64Attribute{Computed: true},
						"vcpus":     schema.Int64Attribute{Computed: true},
						"root_gb":   schema.Int64Attribute{Computed: true},
						"flavor_group": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"name":         schema.StringAttribute{Computed: true},
								"description":  schema.StringAttribute{Computed: true},
								"max_ip_addon": schema.Int64Attribute{Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

func (d *instanceOptionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *instanceOptionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	opts, err := d.api.GetInstanceOptions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list instance options", err.Error())
		return
	}
	state := InstanceOptionsModel{Options: toOptionModels(opts)}
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// toOptionModels is a pure mapper, tested directly without framework harness.
func toOptionModels(in []client.InstanceOption) []InstanceOptionModel {
	out := make([]InstanceOptionModel, 0, len(in))
	for _, o := range in {
		out = append(out, InstanceOptionModel{
			Name:        types.StringValue(o.Name),
			Description: types.StringValue(o.Description),
			Prices: OptionPricesModel{
				PricePerMonth: types.Float64Value(o.Prices.PricePerMonth),
				PricePerHour:  types.Float64Value(o.Prices.PricePerHour),
			},
			Region: OptionRefModel{
				Name:        types.StringValue(o.Region.Name),
				Description: types.StringValue(o.Region.Description),
			},
			MemoryMB: types.Int64Value(int64(o.MemoryMB)),
			VCPUs:    types.Int64Value(int64(o.VCPUs)),
			RootGB:   types.Int64Value(int64(o.RootGB)),
			FlavorGroup: FlavorGroupModel{
				Name:        types.StringValue(o.FlavorGroup.Name),
				Description: types.StringValue(o.FlavorGroup.Description),
				MaxIPAddon:  types.Int64Value(int64(o.FlavorGroup.MaxIPAddon)),
			},
		})
	}
	return out
}
