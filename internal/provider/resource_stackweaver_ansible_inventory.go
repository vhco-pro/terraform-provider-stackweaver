// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource — no terraform-provider-tfe equivalent. Served by the native
// internal/stackweaver client and registered only under its primary
// "stackweaver_ansible_inventory" name (native resources get NO tfe_ alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleInventory{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleInventory{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleInventory{}
)

// NewAnsibleInventoryResource is the factory registered in frameworkResources.
func NewAnsibleInventoryResource() resource.Resource {
	return &resourceStackweaverAnsibleInventory{}
}

type resourceStackweaverAnsibleInventory struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleInventory maps the resource schema to a struct.
type modelStackweaverAnsibleInventory struct {
	ID                      types.String `tfsdk:"id"`
	Organization            types.String `tfsdk:"organization"`
	ProjectID               types.String `tfsdk:"project_id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Type                    types.String `tfsdk:"type"`
	Variables               types.Map    `tfsdk:"variables"`
	Source                  types.String `tfsdk:"source"`
	VCSConnectionID         types.String `tfsdk:"vcs_connection_id"`
	VCSRepository           types.String `tfsdk:"vcs_repository"`
	VCSBranch               types.String `tfsdk:"vcs_branch"`
	InventoryPath           types.String `tfsdk:"inventory_path"`
	SourceVars              types.String `tfsdk:"source_vars"`
	ConstructedLimit        types.String `tfsdk:"constructed_limit"`
	ConstructedCacheTimeout types.Int64  `tfsdk:"constructed_cache_timeout"`
	InputInventoryIDs       types.List   `tfsdk:"input_inventory_ids"`
	LastSyncAt              types.String `tfsdk:"last_sync_at"`
	LastSyncStatus          types.String `tfsdk:"last_sync_status"`
	LastSyncError           types.String `tfsdk:"last_sync_error"`
	LastSyncHostsDiscovered types.Int64  `tfsdk:"last_sync_hosts_discovered"`
	LastSyncLog             types.String `tfsdk:"last_sync_log"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

// modelFromInventory overlays the server-derived fields of inv onto base, which
// carries the attributes the backend never echoes (organization name, project id,
// and — for non-constructed inventories — input_inventory_ids), preserving them
// across reads without producing drift.
func modelFromInventory(ctx context.Context, inv *stackweaver.AnsibleInventory, base modelStackweaverAnsibleInventory) (modelStackweaverAnsibleInventory, diag.Diagnostics) {
	var diags diag.Diagnostics

	vars, d := types.MapValueFrom(ctx, types.StringType, inv.Variables)
	diags.Append(d...)

	m := modelStackweaverAnsibleInventory{
		ID:                      types.StringValue(inv.ID),
		Organization:            base.Organization,
		ProjectID:               base.ProjectID,
		Name:                    types.StringValue(inv.Name),
		Description:             types.StringValue(inv.Description),
		Type:                    types.StringValue(inv.Type),
		Variables:               vars,
		Source:                  types.StringValue(inv.Source),
		VCSConnectionID:         types.StringValue(inv.VCSConnectionID),
		VCSRepository:           types.StringValue(inv.VCSRepository),
		VCSBranch:               types.StringValue(inv.VCSBranch),
		InventoryPath:           types.StringValue(inv.InventoryPath),
		SourceVars:              types.StringValue(inv.SourceVars),
		ConstructedLimit:        types.StringValue(inv.ConstructedLimit),
		ConstructedCacheTimeout: types.Int64Value(inv.ConstructedCacheTimeout),
		InputInventoryIDs:       base.InputInventoryIDs,
		LastSyncAt:              types.StringValue(inv.LastSyncAt),
		LastSyncStatus:          types.StringValue(inv.LastSyncStatus),
		LastSyncError:           types.StringValue(inv.LastSyncError),
		LastSyncHostsDiscovered: types.Int64Value(inv.LastSyncHostsDiscovered),
		LastSyncLog:             types.StringValue(inv.LastSyncLog),
		CreatedAt:               types.StringValue(inv.CreatedAt),
		UpdatedAt:               types.StringValue(inv.UpdatedAt),
	}

	// The backend reports input-inventories only for constructed inventories
	// (via GET). When present it is authoritative; otherwise base (plan/state)
	// is preserved above.
	if inv.InputInventoryIDs != nil {
		ids, d := types.ListValueFrom(ctx, types.StringType, inv.InputInventoryIDs)
		diags.Append(d...)
		m.InputInventoryIDs = ids
	}

	return m, diags
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleInventory) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *resourceStackweaverAnsibleInventory) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_inventory"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleInventory) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Ansible inventory: a named collection of hosts and groups, either organization- or project-scoped, of one of four types (static, dynamic, vcs, constructed).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the inventory.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "Name of the owning organization. Changing this forces a new inventory.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "ID of the owning project. Null = organization-scoped. Changing this forces a new inventory.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the inventory, unique within the organization.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the inventory.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Inventory type: \"static\" (default), \"dynamic\", \"vcs\", or \"constructed\". Changing this forces a new inventory.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"variables": schema.MapAttribute{
				Description: "Global inventory variables. May carry secret values (not encrypted at rest).",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"source": schema.StringAttribute{
				Description: "Plugin config / legacy VCS URL (deprecated for VCS inventories).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vcs_connection_id": schema.StringAttribute{
				Description: "ID of the GitHub-App VCS connection backing a vcs inventory.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vcs_repository": schema.StringAttribute{
				Description: "Full name of the repository (\"owner/repo\") for a vcs inventory.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vcs_branch": schema.StringAttribute{
				Description: "Branch to pull for a vcs inventory. Defaults to \"main\".",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inventory_path": schema.StringAttribute{
				Description: "Path to the inventory file within the repository.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_vars": schema.StringAttribute{
				Description: "Constructed inventory: YAML compose/groups/keyed_groups rules.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"constructed_limit": schema.StringAttribute{
				Description: "Constructed inventory: host limit expression.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"constructed_cache_timeout": schema.Int64Attribute{
				Description: "Constructed inventory: rebuild cache TTL in seconds (0 = always rebuild).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"input_inventory_ids": schema.ListAttribute{
				Description: "Constructed inventory: ordered input inventory ids.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"last_sync_at": schema.StringAttribute{
				Description: "Timestamp of the most recent sync.",
				Computed:    true,
			},
			"last_sync_status": schema.StringAttribute{
				Description: "Status of the most recent sync (syncing | successful | failed).",
				Computed:    true,
			},
			"last_sync_error": schema.StringAttribute{
				Description: "Error message from the most recent sync, if any.",
				Computed:    true,
			},
			"last_sync_hosts_discovered": schema.Int64Attribute{
				Description: "Number of hosts discovered during the most recent sync.",
				Computed:    true,
			},
			"last_sync_log": schema.StringAttribute{
				Description: "ansible-inventory stderr/warnings from the most recent sync.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp the inventory was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp the inventory was last updated.",
				Computed:    true,
			},
		},
	}
}

// service returns the native inventories service, or an error diagnostic when the
// provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleInventory) service() (*stackweaver.AnsibleInventoriesService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_inventory resources")
	}
	return stackweaver.NewAnsibleInventoriesService(r.config.Stackweaver), nil
}

// mapVariables extracts a types.Map into a map[string]string, or nil when the map
// is null/unknown (so the server applies its default).
func mapVariables(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	vars := map[string]string{}
	diags := m.ElementsAs(ctx, &vars, false)
	return vars, diags
}

// listStrings extracts a types.List into a []string, or nil when null/unknown.
func listStrings(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := l.ElementsAs(ctx, &out, false)
	return out, diags
}

// Create implements resource.Resource.
func (r *resourceStackweaverAnsibleInventory) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleInventory
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	vars, d := mapVariables(ctx, plan.Variables)
	resp.Diagnostics.Append(d...)
	inputs, d := listStrings(ctx, plan.InputInventoryIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := stackweaver.AnsibleInventoryCreateOptions{
		Organization:            plan.Organization.ValueString(),
		ProjectID:               plan.ProjectID.ValueString(),
		Name:                    plan.Name.ValueString(),
		Description:             plan.Description.ValueString(),
		Type:                    plan.Type.ValueString(),
		Source:                  plan.Source.ValueString(),
		Variables:               vars,
		VCSConnectionID:         plan.VCSConnectionID.ValueString(),
		VCSRepository:           plan.VCSRepository.ValueString(),
		VCSBranch:               plan.VCSBranch.ValueString(),
		InventoryPath:           plan.InventoryPath.ValueString(),
		SourceVars:              plan.SourceVars.ValueString(),
		ConstructedLimit:        plan.ConstructedLimit.ValueString(),
		ConstructedCacheTimeout: plan.ConstructedCacheTimeout.ValueInt64(),
		InputInventoryIDs:       inputs,
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible inventory %s", options.Name))
	inventory, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible inventory", err.Error())
		return
	}

	model, d := modelFromInventory(ctx, inventory, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsibleInventory) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleInventory
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Read ansible inventory %s", id))
	inventory, err := svc.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible inventory %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible inventory", err.Error())
		return
	}

	model, d := modelFromInventory(ctx, inventory, state)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Update implements resource.Resource.
func (r *resourceStackweaverAnsibleInventory) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleInventory
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelStackweaverAnsibleInventory
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	vars, d := mapVariables(ctx, plan.Variables)
	resp.Diagnostics.Append(d...)
	inputs, d := listStrings(ctx, plan.InputInventoryIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	source := plan.Source.ValueString()
	vcsConnectionID := plan.VCSConnectionID.ValueString()
	vcsRepository := plan.VCSRepository.ValueString()
	vcsBranch := plan.VCSBranch.ValueString()
	inventoryPath := plan.InventoryPath.ValueString()
	sourceVars := plan.SourceVars.ValueString()
	constructedLimit := plan.ConstructedLimit.ValueString()
	constructedCacheTimeout := plan.ConstructedCacheTimeout.ValueInt64()

	options := stackweaver.AnsibleInventoryUpdateOptions{
		Name:                    &name,
		Description:             &description,
		Source:                  &source,
		Variables:               vars,
		VCSConnectionID:         &vcsConnectionID,
		VCSRepository:           &vcsRepository,
		VCSBranch:               &vcsBranch,
		InventoryPath:           &inventoryPath,
		SourceVars:              &sourceVars,
		ConstructedLimit:        &constructedLimit,
		ConstructedCacheTimeout: &constructedCacheTimeout,
		InputInventoryIDs:       inputs,
	}

	tflog.Debug(ctx, fmt.Sprintf("Update ansible inventory %s", id))
	inventory, err := svc.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible inventory", err.Error())
		return
	}

	model, d := modelFromInventory(ctx, inventory, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsibleInventory) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleInventory
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible inventory %s", id))
	err = svc.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible inventory", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState (import by id). The
// organization name and project id are not recoverable from the API and remain
// null after import until set in configuration.
func (r *resourceStackweaverAnsibleInventory) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
