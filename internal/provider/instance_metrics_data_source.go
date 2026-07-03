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

type MetricsAPI interface {
	GetMetrics(ctx context.Context, instanceID, metricType, startTime string) (*client.MetricsResponse, error)
}

type instanceMetricsDataSource struct {
	api MetricsAPI
}

type InstanceMetricsModel struct {
	InstanceID types.String `tfsdk:"instance_id"`
	MetricType types.String `tfsdk:"metric_type"`
	StartTime  types.String `tfsdk:"start_time"`
	Result     types.String `tfsdk:"result"`
}

func NewInstanceMetricsDataSource() datasource.DataSource { return &instanceMetricsDataSource{} }

func (d *instanceMetricsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_instance_metrics"
}

func (d *instanceMetricsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets metrics for a CloudFly instance.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{Required: true, MarkdownDescription: "Instance ID."},
			"metric_type": schema.StringAttribute{Required: true, MarkdownDescription: "Metric type. Valid values: `vcpu`, `memory`, `disk`, `interface`, `packet`."},
			"start_time":  schema.StringAttribute{Required: true, MarkdownDescription: "Time range. Valid values: `1h`, `2h`, `1d`, `7d`, `30d`."},
			"result":      schema.StringAttribute{Computed: true, MarkdownDescription: "Metrics data returned by the API (JSON)."},
		},
	}
}

func (d *instanceMetricsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *instanceMetricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config InstanceMetricsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	metrics, err := d.api.GetMetrics(ctx, config.InstanceID.ValueString(), config.MetricType.ValueString(), config.StartTime.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get instance metrics", err.Error())
		return
	}

	jsonBytes, err := json.Marshal(metrics.Data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal metrics data", err.Error())
		return
	}

	config.Result = types.StringValue(string(jsonBytes))
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
