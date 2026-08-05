// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native data source — no terraform-provider-tfe equivalent. Reads the
// effective ad hoc module allowlist for an organization (its configured list,
// or the built-in AWX default). Served by the native internal/stackweaver
// client (JSON:API single object). Registered only under its primary
// stackweaver_ansible_adhoc_modules name (native = no tfe_ alias).
var (
	_ datasource.DataSource              = &dataSourceStackweaverAnsibleAdHocModules{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverAnsibleAdHocModules{}
)

// NewAnsibleAdHocModulesDataSource is the factory for the data source.
func NewAnsibleAdHocModulesDataSource() datasource.DataSource {
	return &dataSourceStackweaverAnsibleAdHocModules{}
}

type dataSourceStackweaverAnsibleAdHocModules struct {
	config ConfiguredClient
}

type modelDataSourceAdHocModules struct {
	Organization types.String `tfsdk:"organization"`
	ID           types.String `tfsdk:"id"`
	Modules      types.List   `tfsdk:"modules"`
}

func (d *dataSourceStackweaverAnsibleAdHocModules) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_adhoc_modules"
}

func (d *dataSourceStackweaverAnsibleAdHocModules) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the effective ad hoc module allowlist for an organization: the modules an AWX-style Run Command execution is permitted to use.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description: "Name of the organization. Defaults to the provider's organization.",
				Optional:    true,
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The organization's ID.",
				Computed:    true,
			},
			"modules": schema.ListAttribute{
				Description: "The effective ad hoc module allowlist.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *dataSourceStackweaverAnsibleAdHocModules) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dataSourceStackweaverAnsibleAdHocModules) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelDataSourceAdHocModules
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider must be configured with a hostname and token to read stackweaver_ansible_adhoc_modules.")
		return
	}

	var organization string
	resp.Diagnostics.Append(d.config.dataOrDefaultOrganization(ctx, req.Config, &organization)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Organization = types.StringValue(organization)

	svc := stackweaver.NewAnsibleAdHocModulesService(d.config.Stackweaver)
	tflog.Debug(ctx, fmt.Sprintf("Read ansible adhoc modules for %s", organization))
	result, err := svc.List(ctx, organization)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			resp.Diagnostics.AddError("Organization not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Error reading ansible adhoc modules", err.Error())
		return
	}

	elems := make([]attr.Value, len(result.Modules))
	for i, m := range result.Modules {
		elems[i] = types.StringValue(m)
	}
	list, diags := types.ListValue(types.StringType, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Modules = list
	config.ID = types.StringValue(result.OrganizationID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
