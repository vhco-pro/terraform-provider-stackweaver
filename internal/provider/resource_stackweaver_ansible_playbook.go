// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource — no terraform-provider-tfe equivalent. It is served by the
// native internal/stackweaver client, not go-tfe, and is registered only under
// its primary "stackweaver_ansible_playbook" name (native resources get NO tfe_
// alias — see provider_next.go frameworkResources).
var (
	_ resource.Resource                = &resourceStackweaverAnsiblePlaybook{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsiblePlaybook{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsiblePlaybook{}
)

// NewAnsiblePlaybookResource is the factory registered in frameworkResources.
func NewAnsiblePlaybookResource() resource.Resource {
	return &resourceStackweaverAnsiblePlaybook{}
}

type resourceStackweaverAnsiblePlaybook struct {
	config ConfiguredClient
}

// modelStackweaverAnsiblePlaybook maps the resource schema to a struct.
type modelStackweaverAnsiblePlaybook struct {
	ID              types.String `tfsdk:"id"`
	ProjectID       types.String `tfsdk:"project_id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	VCSConnectionID types.String `tfsdk:"vcs_connection_id"`
	VCSRepository   types.String `tfsdk:"vcs_repository"`
	VCSBranch       types.String `tfsdk:"vcs_branch"`
	PlaybookPath    types.String `tfsdk:"playbook_path"`
	SourceMode      types.String `tfsdk:"source_mode"`
	LastSyncAt      types.String `tfsdk:"last_sync_at"`
	LastSyncStatus  types.String `tfsdk:"last_sync_status"`
	LastSyncCommit  types.String `tfsdk:"last_sync_commit"`
	CachedCommit    types.String `tfsdk:"cached_commit"`
	CachedAt        types.String `tfsdk:"cached_at"`
}

// modelFromPlaybook maps a native playbook into the resource model.
func modelFromPlaybook(p *stackweaver.AnsiblePlaybook) modelStackweaverAnsiblePlaybook {
	return modelStackweaverAnsiblePlaybook{
		ID:              types.StringValue(p.ID),
		ProjectID:       types.StringValue(p.ProjectID),
		Name:            types.StringValue(p.Name),
		Description:     types.StringValue(p.Description),
		VCSConnectionID: types.StringValue(p.VCSConnectionID),
		VCSRepository:   types.StringValue(p.VCSRepository),
		VCSBranch:       types.StringValue(p.VCSBranch),
		PlaybookPath:    types.StringValue(p.PlaybookPath),
		SourceMode:      types.StringValue(p.SourceMode),
		LastSyncAt:      types.StringValue(p.LastSyncAt),
		LastSyncStatus:  types.StringValue(p.LastSyncStatus),
		LastSyncCommit:  types.StringValue(p.LastSyncCommit),
		CachedCommit:    types.StringValue(p.CachedCommit),
		CachedAt:        types.StringValue(p.CachedAt),
	}
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsiblePlaybook) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(ConfiguredClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource Configure type",
			fmt.Sprintf("Expected provider.ConfiguredClient, got %T. This is a bug in the provider, so please report it on GitHub.", req.ProviderData),
		)
		return
	}
	r.config = client
}

// Metadata implements resource.Resource. Native resources take the primary
// stackweaver_* type name with no tfe_ alias.
func (r *resourceStackweaverAnsiblePlaybook) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_playbook"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsiblePlaybook) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers an Ansible playbook: a named pointer, within a project, at a playbook file in a VCS repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the playbook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "ID of the owning project. Changing this forces a new playbook.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the playbook, unique within the project.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the playbook.",
				Optional:    true,
				Computed:    true,
			},
			"vcs_connection_id": schema.StringAttribute{
				Description: "ID of the VCS connection to pull the playbook repository from.",
				Optional:    true,
				Computed:    true,
			},
			"vcs_repository": schema.StringAttribute{
				Description: "Full name of the repository (\"owner/repo\").",
				Optional:    true,
				Computed:    true,
			},
			"vcs_branch": schema.StringAttribute{
				Description: "Branch to pull. Defaults to \"main\".",
				Optional:    true,
				Computed:    true,
			},
			"playbook_path": schema.StringAttribute{
				Description: "Path to the playbook file within the repository. Defaults to \"site.yml\".",
				Optional:    true,
				Computed:    true,
			},
			"source_mode": schema.StringAttribute{
				Description: "Where a job sources the playbook content from: \"cached\" (default) or \"fresh\".",
				Optional:    true,
				Computed:    true,
			},
			"last_sync_at": schema.StringAttribute{
				Description: "Timestamp of the most recent VCS sync.",
				Computed:    true,
			},
			"last_sync_status": schema.StringAttribute{
				Description: "Status of the most recent VCS sync.",
				Computed:    true,
			},
			"last_sync_commit": schema.StringAttribute{
				Description: "Git commit SHA of the most recent VCS sync.",
				Computed:    true,
			},
			"cached_commit": schema.StringAttribute{
				Description: "Git commit SHA of the cached playbook snapshot.",
				Computed:    true,
			},
			"cached_at": schema.StringAttribute{
				Description: "Timestamp of the cached playbook snapshot.",
				Computed:    true,
			},
		},
	}
}

// client returns the native Stackweaver client, or an error diagnostic when the
// provider was configured without a hostname/token.
func (r *resourceStackweaverAnsiblePlaybook) client() (*stackweaver.Client, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_playbook resources")
	}
	return r.config.Stackweaver, nil
}

// Create implements resource.Resource.
func (r *resourceStackweaverAnsiblePlaybook) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsiblePlaybook
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	if r.config.Organization == "" {
		resp.Diagnostics.AddError(
			"Missing organization",
			"An organization must be configured on the provider to create a stackweaver_ansible_playbook.",
		)
		return
	}

	options := stackweaver.AnsiblePlaybookCreateOptions{
		Organization:    r.config.Organization,
		ProjectID:       plan.ProjectID.ValueString(),
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueString(),
		VCSConnectionID: plan.VCSConnectionID.ValueString(),
		VCSRepository:   plan.VCSRepository.ValueString(),
		VCSBranch:       plan.VCSBranch.ValueString(),
		PlaybookPath:    plan.PlaybookPath.ValueString(),
		SourceMode:      plan.SourceMode.ValueString(),
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible playbook %s", options.Name))
	playbook, err := client.AnsiblePlaybooks.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible playbook", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromPlaybook(playbook))...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsiblePlaybook) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsiblePlaybook
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Read ansible playbook %s", id))
	playbook, err := client.AnsiblePlaybooks.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible playbook %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible playbook", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromPlaybook(playbook))...)
}

// Update implements resource.Resource.
func (r *resourceStackweaverAnsiblePlaybook) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsiblePlaybook
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelStackweaverAnsiblePlaybook
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	id := state.ID.ValueString()
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	vcsConnectionID := plan.VCSConnectionID.ValueString()
	vcsRepository := plan.VCSRepository.ValueString()
	vcsBranch := plan.VCSBranch.ValueString()
	playbookPath := plan.PlaybookPath.ValueString()
	sourceMode := plan.SourceMode.ValueString()

	options := stackweaver.AnsiblePlaybookUpdateOptions{
		Name:            &name,
		Description:     &description,
		VCSConnectionID: &vcsConnectionID,
		VCSRepository:   &vcsRepository,
		VCSBranch:       &vcsBranch,
		PlaybookPath:    &playbookPath,
		SourceMode:      &sourceMode,
	}

	tflog.Debug(ctx, fmt.Sprintf("Update ansible playbook %s", id))
	playbook, err := client.AnsiblePlaybooks.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible playbook", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromPlaybook(playbook))...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsiblePlaybook) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsiblePlaybook
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible playbook %s", id))
	err = client.AnsiblePlaybooks.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible playbook", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState (import by id).
func (r *resourceStackweaverAnsiblePlaybook) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
