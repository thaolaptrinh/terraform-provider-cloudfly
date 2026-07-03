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

type UsageSummaryAPI interface {
	GetUsageSummary(ctx context.Context) (*client.UsageSummaryResponse, error)
}

type usageSummaryDataSource struct {
	api UsageSummaryAPI
}

type UsageSummaryModel struct {
	CSVPath types.String `tfsdk:"csv_path"`
}

func NewUsageSummaryDataSource() datasource.DataSource { return &usageSummaryDataSource{} }

func (d *usageSummaryDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_usage_summary"
}

func (d *usageSummaryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Exports a usage summary CSV for all instances.",
		Attributes: map[string]schema.Attribute{
			"csv_path": schema.StringAttribute{Computed: true, MarkdownDescription: "URL to the generated CSV file."},
		},
	}
}

func (d *usageSummaryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *usageSummaryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state UsageSummaryModel
	summary, err := d.api.GetUsageSummary(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get usage summary", err.Error())
		return
	}
	state.CSVPath = types.StringValue(summary.CSVPath)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
