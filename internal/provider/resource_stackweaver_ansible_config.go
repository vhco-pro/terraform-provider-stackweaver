// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource — no terraform-provider-tfe equivalent. Manages the
// ansible.cfg content for a scope. There is one config per scope entity
// (singleton), upserted via PUT: create = first PUT, update = subsequent PUT.
// The scope selector (organization or project_id) is ForceNew — changing it
// targets a different singleton. Only org and project scopes have REST routes;
// workspace scope is Computed-only (see the spec divergences).
var (
	_ resource.Resource                     = &resourceStackweaverAnsibleConfig{}
	_ resource.ResourceWithConfigure        = &resourceStackweaverAnsibleConfig{}
	_ resource.ResourceWithConfigValidators = &resourceStackweaverAnsibleConfig{}
)

// NewAnsibleConfigResource is the factory registered in frameworkResources.
func NewAnsibleConfigResource() resource.Resource {
	return &resourceStackweaverAnsibleConfig{}
}

type resourceStackweaverAnsibleConfig struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleConfig maps the resource schema to a struct.
type modelStackweaverAnsibleConfig struct {
	ID            types.String `tfsdk:"id"`
	Organization  types.String `tfsdk:"organization"`
	ProjectID     types.String `tfsdk:"project_id"`
	ConfigContent types.String `tfsdk:"config_content"`
	Scope         types.String `tfsdk:"scope"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

// applyServer overlays the server-returned fields onto the model. The scope
// selector (organization / project_id) is kept from configuration since the API
// returns UUIDs, not the org name.
func (m *modelStackweaverAnsibleConfig) applyServer(c *stackweaver.AnsibleConfig) {
	m.ID = types.StringValue(c.ID)
	m.ConfigContent = types.StringValue(c.ConfigContent)
	m.Scope = types.StringValue(c.Scope)
	m.CreatedAt = types.StringValue(c.CreatedAt)
	m.UpdatedAt = types.StringValue(c.UpdatedAt)
	if c.WorkspaceID != "" {
		m.WorkspaceID = types.StringValue(c.WorkspaceID)
	} else {
		m.WorkspaceID = types.StringNull()
	}
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleConfig) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Metadata implements resource.Resource.
func (r *resourceStackweaverAnsibleConfig) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_config"
}

// ConfigValidators enforces that exactly one scope selector is set.
func (r *resourceStackweaverAnsibleConfig) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("organization"),
			path.MatchRoot("project_id"),
		),
	}
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleConfig) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the ansible.cfg content for a scope (organization or project). There is exactly one config per scope entity; runs resolve the most-specific config (Workspace > Project > Organization).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the config.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "Name of the organization for an org-scoped config. Exactly one of organization or project_id must be set. Changing it forces a new config.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "ID of the project for a project-scoped config. Exactly one of organization or project_id must be set. Changing it forces a new config.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config_content": schema.StringAttribute{
				Description: "The raw ansible.cfg content. Updated in place via the same PUT upsert.",
				Required:    true,
			},
			"scope": schema.StringAttribute{
				Description: "Server-returned scope of the config: organization, project, or workspace.",
				Computed:    true,
			},
			"workspace_id": schema.StringAttribute{
				Description: "Workspace ID when the config is workspace-scoped. Computed only — there is no workspace route to set it via this resource.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the config was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the config was last updated.",
				Computed:    true,
			},
		},
	}
}

// service returns the native ansible-config service, or an error diagnostic when
// the provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleConfig) service() (*stackweaver.AnsibleConfigService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_config resources")
	}
	return stackweaver.NewAnsibleConfigService(r.config.Stackweaver), nil
}

// Create implements resource.Resource (first PUT upsert).
func (r *resourceStackweaverAnsibleConfig) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleConfig
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	content := plan.ConfigContent.ValueString()
	var cfg *stackweaver.AnsibleConfig
	if !plan.Organization.IsNull() {
		tflog.Debug(ctx, fmt.Sprintf("Upsert org ansible config for %s", plan.Organization.ValueString()))
		cfg, err = svc.UpsertByOrganization(ctx, plan.Organization.ValueString(), content)
	} else {
		tflog.Debug(ctx, fmt.Sprintf("Upsert project ansible config for %s", plan.ProjectID.ValueString()))
		cfg, err = svc.UpsertByProject(ctx, plan.ProjectID.ValueString(), content)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible config", err.Error())
		return
	}

	plan.applyServer(cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsibleConfig) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleConfig
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	var cfg *stackweaver.AnsibleConfig
	if !state.Organization.IsNull() {
		cfg, err = svc.GetByOrganization(ctx, state.Organization.ValueString())
	} else {
		cfg, err = svc.GetByProject(ctx, state.ProjectID.ValueString())
	}
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, "Ansible config no longer exists")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible config", err.Error())
		return
	}

	state.applyServer(cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update implements resource.Resource (subsequent PUT upsert).
func (r *resourceStackweaverAnsibleConfig) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleConfig
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	content := plan.ConfigContent.ValueString()
	var cfg *stackweaver.AnsibleConfig
	if !plan.Organization.IsNull() {
		cfg, err = svc.UpsertByOrganization(ctx, plan.Organization.ValueString(), content)
	} else {
		cfg, err = svc.UpsertByProject(ctx, plan.ProjectID.ValueString(), content)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible config", err.Error())
		return
	}

	plan.applyServer(cfg)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsibleConfig) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleConfig
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	if !state.Organization.IsNull() {
		err = svc.DeleteByOrganization(ctx, state.Organization.ValueString())
	} else {
		err = svc.DeleteByProject(ctx, state.ProjectID.ValueString())
	}
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible config", err.Error())
		return
	}
}
