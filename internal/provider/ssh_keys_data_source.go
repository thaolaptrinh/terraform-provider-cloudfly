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

// SSHKeysAPI is the client subset this data source needs. Defined here so
// unit tests can inject a fake. *client.Client satisfies this interface.
type SSHKeysAPI interface {
	ListSSHKeys(ctx context.Context) ([]client.SSHKey, error)
}

type sshKeysDataSource struct {
	api SSHKeysAPI
}

type SSHKeysModel struct {
	SSHKeys []SSHKeyModel `tfsdk:"ssh_keys"`
}

type SSHKeyModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func NewSSHKeysDataSource() datasource.DataSource { return &sshKeysDataSource{} }

func (d *sshKeysDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_ssh_keys"
}

func (d *sshKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all SSH keys in your CloudFly account.",
		Attributes: map[string]schema.Attribute{
			"ssh_keys": schema.ListNestedAttribute{
				MarkdownDescription: "List of SSH keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"public_key":  schema.StringAttribute{Computed: true},
						"fingerprint": schema.StringAttribute{Computed: true},
						"created_at":  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *sshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *sshKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.api.ListSSHKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list SSH keys", err.Error())
		return
	}
	state := SSHKeysModel{SSHKeys: toSSHKeyModels(keys)}
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// toSSHKeyModels is a pure mapper, tested directly without framework harness.
func toSSHKeyModels(in []client.SSHKey) []SSHKeyModel {
	out := make([]SSHKeyModel, 0, len(in))
	for _, k := range in {
		out = append(out, SSHKeyModel{
			ID:          types.Int64Value(int64(k.ID)),
			Name:        types.StringValue(k.Name),
			PublicKey:   types.StringValue(k.PublicKey),
			Fingerprint: types.StringValue(k.Fingerprint),
			CreatedAt:   types.StringValue(k.CreatedAt),
		})
	}
	return out
}
