package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

// RegionsAPI is the client subset this data source needs. Defined here so
// unit tests can inject a fake. *client.Client satisfies this interface.
type RegionsAPI interface {
	ListRegions(ctx context.Context) ([]client.Region, error)
}

type regionsDataSource struct {
	api RegionsAPI
}

type RegionsModel struct {
	Regions []RegionModel `tfsdk:"regions"`
}

type RegionModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func NewRegionsDataSource() datasource.DataSource { return &regionsDataSource{} }

func (d *regionsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_regions"
}

func (d *regionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all available CloudFly regions.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "List of regions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *regionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *regionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	regions, err := d.api.ListRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list regions", err.Error())
		return
	}
	state := RegionsModel{Regions: toRegionModels(regions)}
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// toRegionModels is a pure mapper, tested directly without framework harness.
func toRegionModels(in []client.Region) []RegionModel {
	out := make([]RegionModel, 0, len(in))
	for _, r := range in {
		out = append(out, RegionModel{
			ID:          types.StringValue(r.ID),
			Name:        types.StringValue(r.Name),
			Description: types.StringValue(r.Description),
		})
	}
	return out
}
