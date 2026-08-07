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

// Native data source - no terraform-provider-tfe equivalent. Read-only
// discovery helper listing the repositories reachable through a Stackweaver VCS
// connection, served by the native internal/stackweaver client (not go-tfe).
var (
	_ datasource.DataSource              = &dataSourceStackweaverVCSRepositories{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverVCSRepositories{}
)

// NewVCSRepositoriesDataSource is the factory registered in frameworkDataSources.
func NewVCSRepositoriesDataSource() datasource.DataSource {
	return &dataSourceStackweaverVCSRepositories{}
}

type dataSourceStackweaverVCSRepositories struct {
	config ConfiguredClient
}

// modelStackweaverVCSRepositories maps the data source schema to a struct.
type modelStackweaverVCSRepositories struct {
	VCSConnectionID types.String `tfsdk:"vcs_connection_id"`
	Project         types.String `tfsdk:"project"`
	ID              types.String `tfsdk:"id"`
	Repositories    types.List   `tfsdk:"repositories"`
}

// modelVCSRepositoryElem is one element of the computed repositories list.
type modelVCSRepositoryElem struct {
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	FullName      types.String `tfsdk:"full_name"`
	Description   types.String `tfsdk:"description"`
	Private       types.Bool   `tfsdk:"private"`
	DefaultBranch types.String `tfsdk:"default_branch"`
	URL           types.String `tfsdk:"url"`
	CloneURL      types.String `tfsdk:"clone_url"`
	SSHURL        types.String `tfsdk:"ssh_url"`
}

// vcsRepositoryElemAttrTypes is the attribute-type map for a repositories[] element.
func vcsRepositoryElemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":             types.Int64Type,
		"name":           types.StringType,
		"full_name":      types.StringType,
		"description":    types.StringType,
		"private":        types.BoolType,
		"default_branch": types.StringType,
		"url":            types.StringType,
		"clone_url":      types.StringType,
		"ssh_url":        types.StringType,
	}
}

// Metadata implements datasource.DataSource.
func (d *dataSourceStackweaverVCSRepositories) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vcs_repositories"
}

// Schema implements datasource.DataSource.
func (d *dataSourceStackweaverVCSRepositories) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the repositories reachable through a Stackweaver VCS connection.",
		Attributes: map[string]schema.Attribute{
			"vcs_connection_id": schema.StringAttribute{
				Description: "ID of the VCS connection to enumerate.",
				Required:    true,
			},
			"project": schema.StringAttribute{
				Description: "Azure DevOps project scope. Ignored by providers without a project layer.",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "Synthetic identifier - the VCS connection ID.",
				Computed:    true,
			},
			"repositories": schema.ListNestedAttribute{
				Description: "The repositories reachable through the connection.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.Int64Attribute{Description: "Provider numeric repository ID.", Computed: true},
						"name":           schema.StringAttribute{Description: "Short repository name.", Computed: true},
						"full_name":      schema.StringAttribute{Description: "Full repository name (\"owner/repo\").", Computed: true},
						"description":    schema.StringAttribute{Description: "Repository description.", Computed: true},
						"private":        schema.BoolAttribute{Description: "Whether the repository is private.", Computed: true},
						"default_branch": schema.StringAttribute{Description: "Default branch name.", Computed: true},
						"url":            schema.StringAttribute{Description: "Web URL of the repository.", Computed: true},
						"clone_url":      schema.StringAttribute{Description: "HTTPS clone URL.", Computed: true},
						"ssh_url":        schema.StringAttribute{Description: "SSH clone URL.", Computed: true},
					},
				},
			},
		},
	}
}

// Configure implements datasource.DataSourceWithConfigure.
func (d *dataSourceStackweaverVCSRepositories) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read implements datasource.DataSource.
func (d *dataSourceStackweaverVCSRepositories) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelStackweaverVCSRepositories
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider must be configured with a hostname and token to read stackweaver_vcs_repositories.",
		)
		return
	}

	svc := stackweaver.NewVCSService(d.config.Stackweaver)
	connID := config.VCSConnectionID.ValueString()

	tflog.Debug(ctx, fmt.Sprintf("Listing repositories for VCS connection %s", connID))
	repos, err := svc.ListRepositories(ctx, stackweaver.VCSRepositoryListOptions{
		ConnectionID: connID,
		Project:      config.Project.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing repositories", err.Error())
		return
	}

	elems := make([]modelVCSRepositoryElem, 0, len(repos))
	for _, r := range repos {
		elems = append(elems, modelVCSRepositoryElem{
			ID:            types.Int64Value(r.ID),
			Name:          types.StringValue(r.Name),
			FullName:      types.StringValue(r.FullName),
			Description:   types.StringValue(r.Description),
			Private:       types.BoolValue(r.Private),
			DefaultBranch: types.StringValue(r.DefaultBranch),
			URL:           types.StringValue(r.URL),
			CloneURL:      types.StringValue(r.CloneURL),
			SSHURL:        types.StringValue(r.SSHURL),
		})
	}

	reposList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: vcsRepositoryElemAttrTypes()}, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.ID = types.StringValue(connID)
	config.Repositories = reposList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
