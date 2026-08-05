// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native data source — no terraform-provider-tfe equivalent. Read-only
// discovery helper listing candidate YAML files (or, in inventory mode,
// inventory files) inside a repository reachable through a Stackweaver VCS
// connection. A single data source covers both backing endpoints via file_type,
// since they differ only in the server-side extension filter.
var (
	_ datasource.DataSource              = &dataSourceStackweaverVCSYamlFiles{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverVCSYamlFiles{}
)

const (
	vcsFileTypePlaybook  = "playbook"
	vcsFileTypeInventory = "inventory"
)

// NewVCSYamlFilesDataSource is the factory registered in frameworkDataSources.
func NewVCSYamlFilesDataSource() datasource.DataSource {
	return &dataSourceStackweaverVCSYamlFiles{}
}

type dataSourceStackweaverVCSYamlFiles struct {
	config ConfiguredClient
}

// modelStackweaverVCSYamlFiles maps the data source schema to a struct.
type modelStackweaverVCSYamlFiles struct {
	VCSConnectionID types.String `tfsdk:"vcs_connection_id"`
	Owner           types.String `tfsdk:"owner"`
	Repo            types.String `tfsdk:"repo"`
	Ref             types.String `tfsdk:"ref"`
	FileType        types.String `tfsdk:"file_type"`
	ID              types.String `tfsdk:"id"`
	Paths           types.List   `tfsdk:"paths"`
}

// Metadata implements datasource.DataSource.
func (d *dataSourceStackweaverVCSYamlFiles) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vcs_yaml_files"
}

// Schema implements datasource.DataSource.
func (d *dataSourceStackweaverVCSYamlFiles) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists candidate YAML (or inventory) file paths in a repository reachable through a Stackweaver VCS connection.",
		Attributes: map[string]schema.Attribute{
			"vcs_connection_id": schema.StringAttribute{
				Description: "ID of the VCS connection.",
				Required:    true,
			},
			"owner": schema.StringAttribute{
				Description: "Repository owner (org/user/project).",
				Required:    true,
			},
			"repo": schema.StringAttribute{
				Description: "Repository name.",
				Required:    true,
			},
			"ref": schema.StringAttribute{
				Description: "Branch or commit to read. Empty reads the default branch.",
				Optional:    true,
			},
			"file_type": schema.StringAttribute{
				Description: "Which file set to list: \"playbook\" (default → .yaml/.yml) or \"inventory\" (→ .ini/.yaml/.yml/.json).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(vcsFileTypePlaybook, vcsFileTypeInventory),
				},
			},
			"id": schema.StringAttribute{
				Description: "Synthetic identifier — \"<connection_id>/<owner>/<repo>/<file_type>@<ref>\".",
				Computed:    true,
			},
			"paths": schema.ListAttribute{
				Description: "Repo-relative paths of the matching files.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure implements datasource.DataSourceWithConfigure.
func (d *dataSourceStackweaverVCSYamlFiles) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *dataSourceStackweaverVCSYamlFiles) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelStackweaverVCSYamlFiles
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider must be configured with a hostname and token to read stackweaver_vcs_yaml_files.",
		)
		return
	}

	svc := stackweaver.NewVCSService(d.config.Stackweaver)
	connID := config.VCSConnectionID.ValueString()
	owner := config.Owner.ValueString()
	repo := config.Repo.ValueString()
	ref := config.Ref.ValueString()

	fileType := config.FileType.ValueString()
	if fileType == "" {
		fileType = vcsFileTypePlaybook
	}

	tflog.Debug(ctx, fmt.Sprintf("Listing %s files for %s/%s via VCS connection %s (ref=%q)", fileType, owner, repo, connID, ref))

	var paths []string
	var err error
	switch fileType {
	case vcsFileTypeInventory:
		paths, err = svc.ListInventoryFiles(ctx, connID, owner, repo, ref)
	default:
		paths, err = svc.ListYamlFiles(ctx, connID, owner, repo, ref)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error listing files", err.Error())
		return
	}

	pathsList, diags := types.ListValueFrom(ctx, types.StringType, paths)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.FileType = types.StringValue(fileType)
	config.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s@%s", connID, owner, repo, fileType, ref))
	config.Paths = pathsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
