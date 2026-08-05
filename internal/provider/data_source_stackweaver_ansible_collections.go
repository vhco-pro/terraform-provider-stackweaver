// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native data source — no terraform-provider-tfe equivalent. Discovery helper
// that lists the Ansible Galaxy collections pre-installed on the Stackweaver
// runner image. Served by the native internal/stackweaver client (JSON:API).
// Registered only under its primary stackweaver_ansible_collections name
// (native = no tfe_ alias).
var (
	_ datasource.DataSource              = &dataSourceStackweaverAnsibleCollections{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverAnsibleCollections{}
)

// NewAnsibleCollectionsDataSource is the factory for the data source.
func NewAnsibleCollectionsDataSource() datasource.DataSource {
	return &dataSourceStackweaverAnsibleCollections{}
}

type dataSourceStackweaverAnsibleCollections struct {
	config ConfiguredClient
}

type modelDataSourceAnsibleCollections struct {
	ID          types.String `tfsdk:"id"`
	Collections types.List   `tfsdk:"collections"`
}

// ansibleCollectionAttrTypes is the object type of a collections element.
var ansibleCollectionAttrTypes = map[string]attr.Type{
	"name":        types.StringType,
	"namespace":   types.StringType,
	"version":     types.StringType,
	"description": types.StringType,
	"source":      types.StringType,
}

func (d *dataSourceStackweaverAnsibleCollections) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_collections"
}

func (d *dataSourceStackweaverAnsibleCollections) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the Ansible Galaxy collections pre-installed on the Stackweaver runner image. Galaxy search is not yet available.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthesized constant identifier (pre-installed).",
				Computed:    true,
			},
			"collections": schema.ListNestedAttribute{
				Description: "Pre-installed collections on the runner.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Fully-qualified collection name, e.g. amazon.aws.",
							Computed:    true,
						},
						"namespace": schema.StringAttribute{
							Description: "Collection namespace, e.g. amazon.",
							Computed:    true,
						},
						"version": schema.StringAttribute{
							Description: "Installed version; \"latest\" for pre-installed collections.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Human-readable description.",
							Computed:    true,
						},
						"source": schema.StringAttribute{
							Description: "Origin of the collection; \"pre-installed\" for this data source.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *dataSourceStackweaverAnsibleCollections) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(ConfiguredClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source Configure type",
			fmt.Sprintf("Expected provider.ConfiguredClient, got %T. This is a bug in the provider, so please report it on GitHub.", req.ProviderData),
		)
		return
	}
	d.config = client
}

func (d *dataSourceStackweaverAnsibleCollections) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelDataSourceAnsibleCollections
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider must be configured with a hostname and token to read stackweaver_ansible_collections.")
		return
	}

	svc := stackweaver.NewAnsibleCollectionsService(d.config.Stackweaver)
	tflog.Debug(ctx, "Read pre-installed ansible collections")
	collections, err := svc.ListPreInstalled(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing ansible collections", err.Error())
		return
	}

	elems := make([]attr.Value, len(collections))
	for i, col := range collections {
		obj, diags := types.ObjectValue(ansibleCollectionAttrTypes, map[string]attr.Value{
			"name":        types.StringValue(col.Name),
			"namespace":   types.StringValue(col.Namespace),
			"version":     types.StringValue(col.Version),
			"description": types.StringValue(col.Description),
			"source":      types.StringValue(col.Source),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elems[i] = obj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: ansibleCollectionAttrTypes}, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Collections = list
	config.ID = types.StringValue("pre-installed")

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
