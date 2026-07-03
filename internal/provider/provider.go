package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

var _ provider.Provider = &CloudFlyProvider{}

type CloudFlyProvider struct {
	version string
}

type CloudFlyProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

func (p *CloudFlyProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cloudfly"
	resp.Version = p.version
}

func (p *CloudFlyProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The CloudFly provider is used to interact with resources supported by CloudFly.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "CloudFly API key. May also be set via the `CLOUDFLY_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "CloudFly API base URL. May also be set via the `CLOUDFLY_BASE_URL` environment variable. Defaults to `https://api.cloudfly.vn/backend/api`.",
				Optional:            true,
			},
		},
	}
}

func (p *CloudFlyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CloudFlyProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfgKey := stringValue(data.APIKey)
	envKey := os.Getenv("CLOUDFLY_API_KEY")
	cfgBase := stringValue(data.BaseURL)
	envBase := os.Getenv("CLOUDFLY_BASE_URL")

	c, err := buildClient(ctx, cfgKey, cfgBase, envKey, envBase)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to configure CloudFly client",
			fmt.Sprintf("A valid api_key is required. Set the `api_key` argument or the CLOUDFLY_API_KEY environment variable. Underlying error: %v", err),
		)
		return
	}

	tflog.Info(ctx, "Configured CloudFly client", map[string]interface{}{
		"base_url": c.BaseURL,
	})
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *CloudFlyProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *CloudFlyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CloudFlyProvider{version: version}
	}
}

// buildClient resolves config + env (config takes priority) and constructs the client.
func buildClient(ctx context.Context, cfgKey, cfgBase, envKey, envBase string) (*client.Client, error) {
	apiKey := firstNonEmpty(cfgKey, envKey)
	if apiKey == "" {
		return nil, fmt.Errorf("api_key not provided in config or CLOUDFLY_API_KEY env var")
	}
	baseURL := firstNonEmpty(cfgBase, envBase)
	return client.NewClient(ctx, client.Config{APIKey: apiKey, BaseURL: baseURL})
}

func stringValue(s types.String) string {
	if s.IsNull() || s.IsUnknown() {
		return ""
	}
	return s.ValueString()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
