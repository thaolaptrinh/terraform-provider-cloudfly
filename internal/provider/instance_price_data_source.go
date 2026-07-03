package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

// InstancePriceAPI is the client subset this data source needs.
type InstancePriceAPI interface {
	GetPrice(ctx context.Context, req client.PriceRequest) (*client.Price, error)
}

type instancePriceDataSource struct {
	api InstancePriceAPI
}

type InstancePriceModel struct {
	FlavorType    types.String  `tfsdk:"flavor_type"`
	RAM           types.Int64   `tfsdk:"ram"`
	Disk          types.Int64   `tfsdk:"disk"`
	VCPUs         types.Int64   `tfsdk:"vcpus"`
	Region        types.String  `tfsdk:"region"`
	ImageName     types.String  `tfsdk:"image_name"`
	PricePerMonth types.Float64 `tfsdk:"price_per_month"`
	PricePerHour  types.Float64 `tfsdk:"price_per_hour"`
}

func NewInstancePriceDataSource() datasource.DataSource { return &instancePriceDataSource{} }

func (d *instancePriceDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_instance_price"
}

func (d *instancePriceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Queries the CloudFly API for the price of an instance configuration. Note: this data source issues a POST on every read/refresh.",
		Attributes: map[string]schema.Attribute{
			"flavor_type": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.OneOf("Standard", "Premium")},
			},
			"ram": schema.Int64Attribute{
				Required:   true,
				Validators: []validator.Int64{int64validator.AtLeast(1)},
			},
			"disk": schema.Int64Attribute{
				Required:   true,
				Validators: []validator.Int64{int64validator.AtLeast(20)},
			},
			"vcpus": schema.Int64Attribute{
				Required:   true,
				Validators: []validator.Int64{int64validator.AtLeast(1)},
			},
			"region":          schema.StringAttribute{Required: true},
			"image_name":      schema.StringAttribute{Required: true},
			"price_per_month": schema.Float64Attribute{Computed: true},
			"price_per_hour":  schema.Float64Attribute{Computed: true},
		},
	}
}

func (d *instancePriceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *instancePriceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config InstancePriceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	priceReq := client.PriceRequest{
		FlavorType: config.FlavorType.ValueString(),
		RAM:        int(config.RAM.ValueInt64()),
		Disk:       int(config.Disk.ValueInt64()),
		VCPUs:      int(config.VCPUs.ValueInt64()),
		Region:     config.Region.ValueString(),
		ImageName:  config.ImageName.ValueString(),
	}
	price, err := d.api.GetPrice(ctx, priceReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get price", err.Error())
		return
	}
	config.PricePerMonth = types.Float64Value(price.PricePerMonth)
	config.PricePerHour = types.Float64Value(price.PricePerHour)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
