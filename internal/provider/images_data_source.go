// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

type ImagesAPI interface {
	ListImages(ctx context.Context) ([]client.Image, error)
}

type imagesDataSource struct {
	api ImagesAPI
}

type ImagesModel struct {
	Images types.List `tfsdk:"images"`
}

var imageObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	},
}

func NewImagesDataSource() datasource.DataSource { return &imagesDataSource{} }

func (d *imagesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "cloudfly_images"
}

func (d *imagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all available CloudFly instance images.",
		Attributes: map[string]schema.Attribute{
			"images": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of available images.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Image ID (UUID)."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Image display name."},
					},
				},
			},
		},
	}
}

func (d *imagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *imagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ImagesModel
	images, err := d.api.ListImages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list images", err.Error())
		return
	}

	objs := make([]attr.Value, 0, len(images))
	for _, img := range images {
		obj, diags := types.ObjectValue(imageObjectType.AttrTypes, map[string]attr.Value{
			"id":   types.StringValue(img.ID),
			"name": types.StringValue(img.Name),
		})
		resp.Diagnostics.Append(diags...)
		objs = append(objs, obj)
	}
	listValue, diags := types.ListValue(imageObjectType, objs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Images = listValue
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
