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

// Native resource — no terraform-provider-tfe equivalent. Attaches a dynamic
// inventory source to an existing Ansible inventory. Served by the native
// internal/stackweaver client and registered only under its primary
// "stackweaver_ansible_inventory_source" name (no tfe_ alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleInventorySource{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleInventorySource{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleInventorySource{}
)

// NewAnsibleInventorySourceResource is the factory registered in
// frameworkResources.
func NewAnsibleInventorySourceResource() resource.Resource {
	return &resourceStackweaverAnsibleInventorySource{}
}

type resourceStackweaverAnsibleInventorySource struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleInventorySource maps the resource schema to a struct.
type modelStackweaverAnsibleInventorySource struct {
	ID                      types.String `tfsdk:"id"`
	InventoryID             types.String `tfsdk:"inventory_id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	SourceType              types.String `tfsdk:"source_type"`
	CredentialID            types.String `tfsdk:"credential_id"`
	Config                  types.String `tfsdk:"config"`
	UpdateOnLaunch          types.Bool   `tfsdk:"update_on_launch"`
	UpdateCacheTimeout      types.Int64  `tfsdk:"update_cache_timeout"`
	Overwrite               types.Bool   `tfsdk:"overwrite"`
	OverwriteVars           types.Bool   `tfsdk:"overwrite_vars"`
	Verbosity               types.Int64  `tfsdk:"verbosity"`
	GroupByInstanceID       types.Bool   `tfsdk:"group_by_instance_id"`
	GroupByRegion           types.Bool   `tfsdk:"group_by_region"`
	GroupByAvailabilityZone types.Bool   `tfsdk:"group_by_availability_zone"`
	GroupByTag              types.String `tfsdk:"group_by_tag"`
	HostnameVar             types.String `tfsdk:"hostname_var"`
	InstanceFilters         types.String `tfsdk:"instance_filters"`
	SyncSchedule            types.String `tfsdk:"sync_schedule"`
	Status                  types.String `tfsdk:"status"`
	LastSyncAt              types.String `tfsdk:"last_sync_at"`
	LastSyncError           types.String `tfsdk:"last_sync_error"`
	LastSyncLog             types.String `tfsdk:"last_sync_log"`
	HostsCount              types.Int64  `tfsdk:"hosts_count"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
}

// applyServer overlays every server-returned field onto the model. config is
// stored normalized; credential_id maps an absent relationship to null.
func (m *modelStackweaverAnsibleInventorySource) applyServer(s *stackweaver.AnsibleInventorySource) {
	m.ID = types.StringValue(s.ID)
	m.InventoryID = types.StringValue(s.InventoryID)
	m.Name = types.StringValue(s.Name)
	m.Description = types.StringValue(s.Description)
	m.SourceType = types.StringValue(s.Type)
	if s.CredentialID != "" {
		m.CredentialID = types.StringValue(s.CredentialID)
	} else {
		m.CredentialID = types.StringNull()
	}
	m.Config = types.StringValue(s.Config)
	m.UpdateOnLaunch = types.BoolValue(s.UpdateOnLaunch)
	m.UpdateCacheTimeout = types.Int64Value(s.UpdateCacheTimeout)
	m.Overwrite = types.BoolValue(s.Overwrite)
	m.OverwriteVars = types.BoolValue(s.OverwriteVars)
	m.Verbosity = types.Int64Value(s.Verbosity)
	m.GroupByInstanceID = types.BoolValue(s.GroupByInstanceID)
	m.GroupByRegion = types.BoolValue(s.GroupByRegion)
	m.GroupByAvailabilityZone = types.BoolValue(s.GroupByAvailabilityZone)
	m.GroupByTag = types.StringValue(s.GroupByTag)
	m.HostnameVar = types.StringValue(s.HostnameVar)
	m.InstanceFilters = types.StringValue(s.InstanceFilters)
	m.SyncSchedule = types.StringValue(s.SyncSchedule)
	m.Status = types.StringValue(s.Status)
	m.LastSyncAt = types.StringValue(s.LastSyncAt)
	m.LastSyncError = types.StringValue(s.LastSyncError)
	m.LastSyncLog = types.StringValue(s.LastSyncLog)
	m.HostsCount = types.Int64Value(s.HostsCount)
	m.Enabled = types.BoolValue(s.Enabled)
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleInventorySource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *resourceStackweaverAnsibleInventorySource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_inventory_source"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleInventorySource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a dynamic inventory source (AWS, Azure, GCP, VMware, or custom) to an existing Ansible inventory. A sync populates the inventory's hosts and groups.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the inventory source.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inventory_id": schema.StringAttribute{
				Description: "ID of the owning inventory. Changing this forces a new source.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the inventory source (1-255 chars).",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the source.",
				Optional:    true,
				Computed:    true,
			},
			"source_type": schema.StringAttribute{
				Description: "Source type: one of aws, azure, gcp, vmware, custom. Immutable — changing it forces a new source.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"credential_id": schema.StringAttribute{
				Description: "ID of the cloud credential used for discovery. Omit (or set empty) to use workload-identity/OIDC.",
				Optional:    true,
			},
			"config": schema.StringAttribute{
				Description: "Provider-specific configuration as a JSON string (use jsonencode). Defaults to an empty object.",
				Optional:    true,
				Computed:    true,
			},
			"update_on_launch": schema.BoolAttribute{
				Description: "Sync the source before each job run. Defaults to true.",
				Optional:    true,
				Computed:    true,
			},
			"update_cache_timeout": schema.Int64Attribute{
				Description: "Seconds a prior sync stays fresh for update-on-launch (0 = always sync). Defaults to 0.",
				Optional:    true,
				Computed:    true,
			},
			"overwrite": schema.BoolAttribute{
				Description: "Remove source-owned hosts/groups the provider no longer reports. Defaults to false.",
				Optional:    true,
				Computed:    true,
			},
			"overwrite_vars": schema.BoolAttribute{
				Description: "Replace host vars wholesale on sync (false = merge). Defaults to false.",
				Optional:    true,
				Computed:    true,
			},
			"verbosity": schema.Int64Attribute{
				Description: "Verbosity 0-4, adds -v..-vvvv to ansible-inventory. Defaults to 0.",
				Optional:    true,
				Computed:    true,
			},
			"group_by_instance_id": schema.BoolAttribute{
				Description: "Group discovered hosts by instance ID. Defaults to false.",
				Optional:    true,
				Computed:    true,
			},
			"group_by_region": schema.BoolAttribute{
				Description: "Group discovered hosts by region. Defaults to true.",
				Optional:    true,
				Computed:    true,
			},
			"group_by_availability_zone": schema.BoolAttribute{
				Description: "Group discovered hosts by availability zone. Defaults to false.",
				Optional:    true,
				Computed:    true,
			},
			"group_by_tag": schema.StringAttribute{
				Description: "Tag key to group discovered hosts by (e.g. \"Environment\").",
				Optional:    true,
				Computed:    true,
			},
			"hostname_var": schema.StringAttribute{
				Description: "Which variable becomes the Ansible hostname. Defaults to \"public_ip\".",
				Optional:    true,
				Computed:    true,
			},
			"instance_filters": schema.StringAttribute{
				Description: "JSON array of provider-specific instance filters.",
				Optional:    true,
				Computed:    true,
			},
			"sync_schedule": schema.StringAttribute{
				Description: "Cron expression for scheduled sync (e.g. \"0 */6 * * *\").",
				Optional:    true,
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the source is enabled. Disabled sources reject sync. Defaults to true.",
				Optional:    true,
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Sync status: never_synced, syncing, successful, or failed.",
				Computed:    true,
			},
			"last_sync_at": schema.StringAttribute{
				Description: "Timestamp of the most recent sync.",
				Computed:    true,
			},
			"last_sync_error": schema.StringAttribute{
				Description: "Error from the most recent sync, if any.",
				Computed:    true,
			},
			"last_sync_log": schema.StringAttribute{
				Description: "Stderr/warnings captured from the most recent sync.",
				Computed:    true,
			},
			"hosts_count": schema.Int64Attribute{
				Description: "Number of hosts discovered by the most recent sync.",
				Computed:    true,
			},
		},
	}
}

// service returns the native inventory-sources service, or an error diagnostic
// when the provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleInventorySource) service() (*stackweaver.AnsibleInventorySourcesService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_inventory_source resources")
	}
	return stackweaver.NewAnsibleInventorySourcesService(r.config.Stackweaver), nil
}

// boolPtr returns a pointer to the bool value, or nil when the value is
// null/unknown (so an unset field falls through to the server default).
func boolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// int64Ptr returns a pointer to the int64 value, or nil when null/unknown.
func int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

// Create implements resource.Resource.
func (r *resourceStackweaverAnsibleInventorySource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleInventorySource
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	options := stackweaver.AnsibleInventorySourceCreateOptions{
		InventoryID:             plan.InventoryID.ValueString(),
		Name:                    plan.Name.ValueString(),
		Description:             plan.Description.ValueString(),
		Type:                    plan.SourceType.ValueString(),
		CredentialID:            plan.CredentialID.ValueString(),
		Config:                  plan.Config.ValueString(),
		SyncSchedule:            plan.SyncSchedule.ValueString(),
		GroupByTag:              plan.GroupByTag.ValueString(),
		HostnameVar:             plan.HostnameVar.ValueString(),
		InstanceFilters:         plan.InstanceFilters.ValueString(),
		UpdateOnLaunch:          boolPtr(plan.UpdateOnLaunch),
		UpdateCacheTimeout:      int64Ptr(plan.UpdateCacheTimeout),
		Overwrite:               boolPtr(plan.Overwrite),
		OverwriteVars:           boolPtr(plan.OverwriteVars),
		Verbosity:               int64Ptr(plan.Verbosity),
		GroupByInstanceID:       boolPtr(plan.GroupByInstanceID),
		GroupByRegion:           boolPtr(plan.GroupByRegion),
		GroupByAvailabilityZone: boolPtr(plan.GroupByAvailabilityZone),
		Enabled:                 boolPtr(plan.Enabled),
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible inventory source %s", options.Name))
	source, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible inventory source", err.Error())
		return
	}

	plan.applyServer(source)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsibleInventorySource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleInventorySource
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
	tflog.Debug(ctx, fmt.Sprintf("Read ansible inventory source %s", id))
	source, err := svc.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible inventory source %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible inventory source", err.Error())
		return
	}

	state.applyServer(source)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update implements resource.Resource.
func (r *resourceStackweaverAnsibleInventorySource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleInventorySource
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelStackweaverAnsibleInventorySource
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	options := stackweaver.AnsibleInventorySourceUpdateOptions{
		Name:                    plan.Name.ValueString(),
		Description:             plan.Description.ValueString(),
		Config:                  plan.Config.ValueString(),
		SyncSchedule:            plan.SyncSchedule.ValueString(),
		GroupByTag:              plan.GroupByTag.ValueString(),
		HostnameVar:             plan.HostnameVar.ValueString(),
		InstanceFilters:         plan.InstanceFilters.ValueString(),
		UpdateOnLaunch:          boolPtr(plan.UpdateOnLaunch),
		UpdateCacheTimeout:      int64Ptr(plan.UpdateCacheTimeout),
		Overwrite:               boolPtr(plan.Overwrite),
		OverwriteVars:           boolPtr(plan.OverwriteVars),
		Verbosity:               int64Ptr(plan.Verbosity),
		GroupByInstanceID:       boolPtr(plan.GroupByInstanceID),
		GroupByRegion:           boolPtr(plan.GroupByRegion),
		GroupByAvailabilityZone: boolPtr(plan.GroupByAvailabilityZone),
		Enabled:                 boolPtr(plan.Enabled),
	}
	// Empty/null credential_id clears the credential (switch to OIDC).
	if plan.CredentialID.IsNull() || plan.CredentialID.ValueString() == "" {
		options.ClearCredential = true
	} else {
		options.CredentialID = plan.CredentialID.ValueString()
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Update ansible inventory source %s", id))
	source, err := svc.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible inventory source", err.Error())
		return
	}

	plan.applyServer(source)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsibleInventorySource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleInventorySource
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
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible inventory source %s", id))
	err = svc.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible inventory source", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState (import by id).
func (r *resourceStackweaverAnsibleInventorySource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
