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

// Native data source — no terraform-provider-tfe equivalent. Read-only
// discovery helper listing the branches of one repository reachable through a
// Stackweaver VCS connection, served by the native internal/stackweaver client.
var (
	_ datasource.DataSource              = &dataSourceStackweaverVCSRepositoryBranches{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverVCSRepositoryBranches{}
)

// NewVCSRepositoryBranchesDataSource is the factory registered in frameworkDataSources.
func NewVCSRepositoryBranchesDataSource() datasource.DataSource {
	return &dataSourceStackweaverVCSRepositoryBranches{}
}

type dataSourceStackweaverVCSRepositoryBranches struct {
	config ConfiguredClient
}

// modelStackweaverVCSRepositoryBranches maps the data source schema to a struct.
type modelStackweaverVCSRepositoryBranches struct {
	VCSConnectionID types.String `tfsdk:"vcs_connection_id"`
	Owner           types.String `tfsdk:"owner"`
	Repo            types.String `tfsdk:"repo"`
	ID              types.String `tfsdk:"id"`
	Branches        types.List   `tfsdk:"branches"`
}

// modelVCSBranchElem is one element of the computed branches list.
type modelVCSBranchElem struct {
	Name      types.String `tfsdk:"name"`
	CommitSHA types.String `tfsdk:"commit_sha"`
	Protected types.Bool   `tfsdk:"protected"`
}

// vcsBranchElemAttrTypes is the attribute-type map for a branches[] element.
func vcsBranchElemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":       types.StringType,
		"commit_sha": types.StringType,
		"protected":  types.BoolType,
	}
}

// Metadata implements datasource.DataSource.
func (d *dataSourceStackweaverVCSRepositoryBranches) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vcs_repository_branches"
}

// Schema implements datasource.DataSource.
func (d *dataSourceStackweaverVCSRepositoryBranches) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the branches of a repository reachable through a Stackweaver VCS connection.",
		Attributes: map[string]schema.Attribute{
			"vcs_connection_id": schema.StringAttribute{
				Description: "ID of the VCS connection.",
				Required:    true,
			},
			"owner": schema.StringAttribute{
				Description: "Repository owner (org/user/project — the left half of \"owner/repo\").",
				Required:    true,
			},
			"repo": schema.StringAttribute{
				Description: "Repository name (the right half of \"owner/repo\").",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Synthetic identifier — \"<connection_id>/<owner>/<repo>\".",
				Computed:    true,
			},
			"branches": schema.ListNestedAttribute{
				Description: "The branches of the repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":       schema.StringAttribute{Description: "Branch name.", Computed: true},
						"commit_sha": schema.StringAttribute{Description: "Head commit SHA (flattened from commit.sha).", Computed: true},
						"protected":  schema.BoolAttribute{Description: "Whether the branch is protected.", Computed: true},
					},
				},
			},
		},
	}
}

// Configure implements datasource.DataSourceWithConfigure.
func (d *dataSourceStackweaverVCSRepositoryBranches) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *dataSourceStackweaverVCSRepositoryBranches) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelStackweaverVCSRepositoryBranches
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider must be configured with a hostname and token to read stackweaver_vcs_repository_branches.",
		)
		return
	}

	svc := stackweaver.NewVCSService(d.config.Stackweaver)
	connID := config.VCSConnectionID.ValueString()
	owner := config.Owner.ValueString()
	repo := config.Repo.ValueString()

	tflog.Debug(ctx, fmt.Sprintf("Listing branches for %s/%s via VCS connection %s", owner, repo, connID))
	branches, err := svc.ListBranches(ctx, connID, owner, repo)
	if err != nil {
		resp.Diagnostics.AddError("Error listing branches", err.Error())
		return
	}

	elems := make([]modelVCSBranchElem, 0, len(branches))
	for _, b := range branches {
		elems = append(elems, modelVCSBranchElem{
			Name:      types.StringValue(b.Name),
			CommitSHA: types.StringValue(b.CommitSHA),
			Protected: types.BoolValue(b.Protected),
		})
	}

	branchesList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: vcsBranchElemAttrTypes()}, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", connID, owner, repo))
	config.Branches = branchesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
