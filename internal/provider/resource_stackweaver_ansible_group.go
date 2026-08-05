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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource — no terraform-provider-tfe equivalent. Registered only under
// its primary "stackweaver_ansible_group" name (native resources get NO tfe_
// alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleGroup{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleGroup{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleGroup{}
)

// NewAnsibleGroupResource is the factory registered in frameworkResources.
func NewAnsibleGroupResource() resource.Resource {
	return &resourceStackweaverAnsibleGroup{}
}

type resourceStackweaverAnsibleGroup struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleGroup maps the resource schema to a struct.
type modelStackweaverAnsibleGroup struct {
	ID          types.String `tfsdk:"id"`
	InventoryID types.String `tfsdk:"inventory_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Variables   types.Map    `tfsdk:"variables"`
	ParentID    types.String `tfsdk:"parent_id"`
	SourceID    types.String `tfsdk:"source_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// modelFromGroup maps a native group into the resource model. source_id is never
// echoed by the backend (dynamic-source provenance), so it always reads back null
// for a Terraform-managed group; parent_id reads back null when the group is
// top-level.
func modelFromGroup(ctx context.Context, g *stackweaver.AnsibleGroup) (modelStackweaverAnsibleGroup, diag.Diagnostics) {
	vars, diags := types.MapValueFrom(ctx, types.StringType, g.Variables)
	m := modelStackweaverAnsibleGroup{
		ID:          types.StringValue(g.ID),
		InventoryID: types.StringValue(g.InventoryID),
		Name:        types.StringValue(g.Name),
		Description: types.StringValue(g.Description),
		Variables:   vars,
		ParentID:    types.StringNull(),
		SourceID:    types.StringNull(),
		CreatedAt:   types.StringValue(g.CreatedAt),
		UpdatedAt:   types.StringValue(g.UpdatedAt),
	}
	if g.ParentID != "" {
		m.ParentID = types.StringValue(g.ParentID)
	}
	return m, diags
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleGroup) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *resourceStackweaverAnsibleGroup) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_group"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleGroup) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a group within a stackweaver_ansible_inventory: a named grouping of hosts with group-level variables, optionally nested under a parent group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inventory_id": schema.StringAttribute{
				Description: "ID of the owning inventory. Changing this forces a new group.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the group, unique within the inventory.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the group.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"variables": schema.MapAttribute{
				Description: "Group-specific variables. May carry secret values (not encrypted at rest).",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"parent_id": schema.StringAttribute{
				Description: "ID of the parent group for nested groups. Note: the backend cannot detach an existing parent in place.",
				Optional:    true,
			},
			"source_id": schema.StringAttribute{
				Description: "Dynamic-source owner (null = manually managed). Server/sync-set.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp the group was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp the group was last updated.",
				Computed:    true,
			},
		},
	}
}

// service returns the native groups service, or an error diagnostic when the
// provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleGroup) service() (*stackweaver.AnsibleGroupsService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_group resources")
	}
	return stackweaver.NewAnsibleGroupsService(r.config.Stackweaver), nil
}

// Create implements resource.Resource.
func (r *resourceStackweaverAnsibleGroup) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleGroup
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
	if resp.Diagnostics.HasError() {
		return
	}

	options := stackweaver.AnsibleGroupCreateOptions{
		InventoryID: plan.InventoryID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Variables:   vars,
		ParentID:    plan.ParentID.ValueString(),
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible group %s", options.Name))
	group, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible group", err.Error())
		return
	}

	model, d := modelFromGroup(ctx, group)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsibleGroup) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleGroup
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
	tflog.Debug(ctx, fmt.Sprintf("Read ansible group %s", id))
	group, err := svc.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible group %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible group", err.Error())
		return
	}

	model, d := modelFromGroup(ctx, group)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Update implements resource.Resource.
func (r *resourceStackweaverAnsibleGroup) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleGroup
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelStackweaverAnsibleGroup
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
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	parentID := plan.ParentID.ValueString()

	options := stackweaver.AnsibleGroupUpdateOptions{
		Name:        &name,
		Description: &description,
		Variables:   vars,
		ParentID:    &parentID,
	}

	tflog.Debug(ctx, fmt.Sprintf("Update ansible group %s", id))
	group, err := svc.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible group", err.Error())
		return
	}

	model, d := modelFromGroup(ctx, group)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsibleGroup) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleGroup
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
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible group %s", id))
	err = svc.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible group", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState (import by id).
func (r *resourceStackweaverAnsibleGroup) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
