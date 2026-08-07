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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource - no terraform-provider-tfe equivalent. Registered only under
// its primary "stackweaver_ansible_host" name (native resources get NO tfe_
// alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleHost{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleHost{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleHost{}
)

// NewAnsibleHostResource is the factory registered in frameworkResources.
func NewAnsibleHostResource() resource.Resource {
	return &resourceStackweaverAnsibleHost{}
}

type resourceStackweaverAnsibleHost struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleHost maps the resource schema to a struct.
type modelStackweaverAnsibleHost struct {
	ID          types.String `tfsdk:"id"`
	InventoryID types.String `tfsdk:"inventory_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Hostname    types.String `tfsdk:"hostname"`
	Port        types.Int64  `tfsdk:"port"`
	Variables   types.Map    `tfsdk:"variables"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	SourceID    types.String `tfsdk:"source_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// modelFromHost maps a native host into the resource model. source_id is never
// echoed by the backend (it marks dynamic-source provenance), so it always reads
// back null for a Terraform-managed host.
func modelFromHost(ctx context.Context, h *stackweaver.AnsibleHost) (modelStackweaverAnsibleHost, diag.Diagnostics) {
	vars, diags := types.MapValueFrom(ctx, types.StringType, h.Variables)
	return modelStackweaverAnsibleHost{
		ID:          types.StringValue(h.ID),
		InventoryID: types.StringValue(h.InventoryID),
		Name:        types.StringValue(h.Name),
		Description: types.StringValue(h.Description),
		Hostname:    types.StringValue(h.Hostname),
		Port:        types.Int64Value(h.Port),
		Variables:   vars,
		Enabled:     types.BoolValue(h.Enabled),
		SourceID:    types.StringNull(),
		CreatedAt:   types.StringValue(h.CreatedAt),
		UpdatedAt:   types.StringValue(h.UpdatedAt),
	}, diags
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleHost) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *resourceStackweaverAnsibleHost) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_host"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleHost) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a single host within a stackweaver_ansible_inventory: a named target with an optional distinct hostname/IP, SSH port, per-host variables, and an enabled flag.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the host.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inventory_id": schema.StringAttribute{
				Description: "ID of the owning inventory. Changing this forces a new host.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the host, unique within the inventory.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the host.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Description: "Actual hostname/IP if different from name.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port": schema.Int64Attribute{
				Description: "SSH port. Defaults to 22.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"variables": schema.MapAttribute{
				Description: "Host-specific variables. May carry secret values (not encrypted at rest).",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the host is included at run time. Defaults to true.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"source_id": schema.StringAttribute{
				Description: "Dynamic-source owner (null = manually managed). Server/sync-set.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp the host was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp the host was last updated.",
				Computed:    true,
			},
		},
	}
}

// service returns the native hosts service, or an error diagnostic when the
// provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleHost) service() (*stackweaver.AnsibleHostsService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_host resources")
	}
	return stackweaver.NewAnsibleHostsService(r.config.Stackweaver), nil
}

// Create implements resource.Resource.
func (r *resourceStackweaverAnsibleHost) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleHost
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

	options := stackweaver.AnsibleHostCreateOptions{
		InventoryID: plan.InventoryID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Hostname:    plan.Hostname.ValueString(),
		Port:        plan.Port.ValueInt64(),
		Variables:   vars,
		Enabled:     boolPointer(plan.Enabled),
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible host %s", options.Name))
	host, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible host", err.Error())
		return
	}

	model, d := modelFromHost(ctx, host)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsibleHost) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleHost
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
	tflog.Debug(ctx, fmt.Sprintf("Read ansible host %s", id))
	host, err := svc.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible host %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible host", err.Error())
		return
	}

	model, d := modelFromHost(ctx, host)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Update implements resource.Resource.
func (r *resourceStackweaverAnsibleHost) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleHost
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelStackweaverAnsibleHost
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
	hostname := plan.Hostname.ValueString()
	port := plan.Port.ValueInt64()

	options := stackweaver.AnsibleHostUpdateOptions{
		Name:        &name,
		Description: &description,
		Hostname:    &hostname,
		Port:        &port,
		Variables:   vars,
		Enabled:     boolPointer(plan.Enabled),
	}

	tflog.Debug(ctx, fmt.Sprintf("Update ansible host %s", id))
	host, err := svc.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible host", err.Error())
		return
	}

	model, d := modelFromHost(ctx, host)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsibleHost) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleHost
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
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible host %s", id))
	err = svc.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible host", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState (import by id).
func (r *resourceStackweaverAnsibleHost) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// boolPointer returns a *bool for a types.Bool, or nil when null/unknown so the
// server applies its default (enabled = true).
func boolPointer(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}
