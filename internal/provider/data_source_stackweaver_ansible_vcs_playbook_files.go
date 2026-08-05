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

// Native data source — no terraform-provider-tfe equivalent. Discovery helper
// that lists playbook candidate files in a connected VCS repository at a branch,
// each annotated with whether it is already registered as a
// stackweaver_ansible_playbook. Served by the native internal/stackweaver
// client (plain-JSON endpoint), registered only under its primary
// stackweaver_ansible_vcs_playbook_files name (native = no tfe_ alias).
var (
	_ datasource.DataSource              = &dataSourceStackweaverAnsibleVCSPlaybookFiles{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverAnsibleVCSPlaybookFiles{}
)

// NewAnsibleVCSPlaybookFilesDataSource is the factory for the data source.
func NewAnsibleVCSPlaybookFilesDataSource() datasource.DataSource {
	return &dataSourceStackweaverAnsibleVCSPlaybookFiles{}
}

type dataSourceStackweaverAnsibleVCSPlaybookFiles struct {
	config ConfiguredClient
}

type modelDataSourceVCSPlaybookFiles struct {
	Organization    types.String `tfsdk:"organization"`
	VCSConnectionID types.String `tfsdk:"vcs_connection_id"`
	Repository      types.String `tfsdk:"repository"`
	Branch          types.String `tfsdk:"branch"`
	Path            types.String `tfsdk:"path"`
	ID              types.String `tfsdk:"id"`
	Files           types.List   `tfsdk:"files"`
}

// vcsPlaybookFileAttrTypes is the object type of a files element.
var vcsPlaybookFileAttrTypes = map[string]attr.Type{
	"path":          types.StringType,
	"name":          types.StringType,
	"registered":    types.BoolType,
	"playbook_id":   types.StringType,
	"playbook_name": types.StringType,
}

func (d *dataSourceStackweaverAnsibleVCSPlaybookFiles) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_vcs_playbook_files"
}

func (d *dataSourceStackweaverAnsibleVCSPlaybookFiles) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists playbook candidate files in a connected VCS repository at a branch, each annotated with whether it is already registered as a stackweaver_ansible_playbook.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description: "Name of the organization. Defaults to the provider's organization.",
				Optional:    true,
				Computed:    true,
			},
			"vcs_connection_id": schema.StringAttribute{
				Description: "ID of the VCS connection to browse. Must belong to the organization.",
				Required:    true,
			},
			"repository": schema.StringAttribute{
				Description: "Repository in \"owner/repo\" format.",
				Required:    true,
			},
			"branch": schema.StringAttribute{
				Description: "Branch to list files from. Required — the listing and annotation are branch-scoped.",
				Required:    true,
			},
			"path": schema.StringAttribute{
				Description: "Optional repo path prefix; when set, only files under it are returned.",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "Synthesized identifier (organization/connection/repository/branch).",
				Computed:    true,
			},
			"files": schema.ListNestedAttribute{
				Description: "Discovered playbook candidate files.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Description: "Repo-relative file path.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Base filename.",
							Computed:    true,
						},
						"registered": schema.BoolAttribute{
							Description: "Whether a playbook is already registered for this path.",
							Computed:    true,
						},
						"playbook_id": schema.StringAttribute{
							Description: "ID of the registered playbook (present only when registered).",
							Computed:    true,
						},
						"playbook_name": schema.StringAttribute{
							Description: "Name of the registered playbook (present only when registered).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *dataSourceStackweaverAnsibleVCSPlaybookFiles) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dataSourceStackweaverAnsibleVCSPlaybookFiles) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelDataSourceVCSPlaybookFiles
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider must be configured with a hostname and token to read stackweaver_ansible_vcs_playbook_files.")
		return
	}

	var organization string
	resp.Diagnostics.Append(d.config.dataOrDefaultOrganization(ctx, req.Config, &organization)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Organization = types.StringValue(organization)

	svc := stackweaver.NewAnsibleVCSPlaybookFilesService(d.config.Stackweaver)
	options := stackweaver.VCSPlaybookFilesListOptions{
		VCSConnectionID: config.VCSConnectionID.ValueString(),
		Repository:      config.Repository.ValueString(),
		Branch:          config.Branch.ValueString(),
		Path:            config.Path.ValueString(),
	}

	tflog.Debug(ctx, fmt.Sprintf("Read ansible vcs playbook files for %s/%s@%s", organization, options.Repository, options.Branch))
	files, err := svc.List(ctx, organization, options)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			resp.Diagnostics.AddError("VCS connection not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Error listing VCS playbook files", err.Error())
		return
	}

	elems := make([]attr.Value, len(files))
	for i, f := range files {
		obj, diags := types.ObjectValue(vcsPlaybookFileAttrTypes, map[string]attr.Value{
			"path":          types.StringValue(f.Path),
			"name":          types.StringValue(f.Name),
			"registered":    types.BoolValue(f.Registered),
			"playbook_id":   types.StringValue(f.PlaybookID),
			"playbook_name": types.StringValue(f.PlaybookName),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elems[i] = obj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: vcsPlaybookFileAttrTypes}, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Files = list
	config.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s", organization, options.VCSConnectionID, options.Repository, options.Branch))

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
