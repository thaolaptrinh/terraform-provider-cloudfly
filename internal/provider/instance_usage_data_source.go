// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

type UsageAPI interface {
	GetUsageHistory(ctx context.Context, instanceID string) ([]client.UsageItem, error)
}

type instanceUsageDataSource struct {
	api UsageAPI
}

type InstanceUsageModel struct {
	InstanceID types.String `tfsdk:"instance_id"`
	Items      types.String `tfsdk:"items"`
}

func NewInstanceUsageDataSource() datasource.DataSource { return &instanceUsageDataSource{} }

func (d *instanceUsageDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_instance_usage"
}

func (d *instanceUsageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets usage history for a CloudFly instance.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{Required: true, MarkdownDescription: "Instance ID."},
			"items":       schema.StringAttribute{Computed: true, MarkdownDescription: "Usage history items (JSON)."},
		},
	}
}

func (d *instanceUsageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *instanceUsageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config InstanceUsageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := readUsage(ctx, d.api, &config); err != nil {
		resp.Diagnostics.AddError("Failed to get usage history", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func readUsage(ctx context.Context, api UsageAPI, m *InstanceUsageModel) error {
	items, err := api.GetUsageHistory(ctx, m.InstanceID.ValueString())
	if err != nil {
		return err
	}

	jsonBytes, err := json.Marshal(items)
	if err != nil {
		return err
	}

	m.Items = types.StringValue(string(jsonBytes))
	return nil
}
