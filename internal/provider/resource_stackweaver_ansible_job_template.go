// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource — no terraform-provider-tfe equivalent. Served by the native
// internal/stackweaver client, registered only under its primary
// "stackweaver_ansible_job_template" name (native resources get NO tfe_ alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleJobTemplate{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleJobTemplate{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleJobTemplate{}
)

// NewAnsibleJobTemplateResource is the factory registered in frameworkResources.
func NewAnsibleJobTemplateResource() resource.Resource {
	return &resourceStackweaverAnsibleJobTemplate{}
}

type resourceStackweaverAnsibleJobTemplate struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleJobTemplate maps the resource schema to a struct.
type modelStackweaverAnsibleJobTemplate struct {
	ID                types.String `tfsdk:"id"`
	Organization      types.String `tfsdk:"organization"`
	ProjectID         types.String `tfsdk:"project_id"`
	PlaybookID        types.String `tfsdk:"playbook_id"`
	InventoryID       types.String `tfsdk:"inventory_id"`
	CredentialID      types.String `tfsdk:"credential_id"`
	AgentPoolID       types.String `tfsdk:"agent_pool_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ExtraVars         types.Map    `tfsdk:"extra_vars"`
	Limit             types.String `tfsdk:"limit"`
	Tags              types.String `tfsdk:"tags"`
	SkipTags          types.String `tfsdk:"skip_tags"`
	Verbosity         types.Int64  `tfsdk:"verbosity"`
	Forks             types.Int64  `tfsdk:"forks"`
	BecomeEnabled     types.Bool   `tfsdk:"become_enabled"`
	DiffMode          types.Bool   `tfsdk:"diff_mode"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	TimeoutSeconds    types.Int64  `tfsdk:"timeout_seconds"`
	AllowSimultaneous types.Bool   `tfsdk:"allow_simultaneous"`
	RetentionDays     types.Int64  `tfsdk:"retention_days"`
	JobSliceCount     types.Int64  `tfsdk:"job_slice_count"`
	ScheduleEnabled   types.Bool   `tfsdk:"schedule_enabled"`
	ScheduleCron      types.String `tfsdk:"schedule_cron"`
	AllowCallbacks    types.Bool   `tfsdk:"allow_callbacks"`
	LaunchOnWebhook   types.Bool   `tfsdk:"launch_on_webhook"`
	HostConfigKey     types.String `tfsdk:"host_config_key"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

// modelFromJobTemplate maps a native job template into the resource model. The
// organization is carried from the caller because the API does not echo it.
func modelFromJobTemplate(ctx context.Context, t *stackweaver.AnsibleJobTemplate, organization string) (modelStackweaverAnsibleJobTemplate, error) {
	extraVars, err := extraVarsToMap(ctx, t.ExtraVars)
	if err != nil {
		return modelStackweaverAnsibleJobTemplate{}, err
	}

	m := modelStackweaverAnsibleJobTemplate{
		ID:                types.StringValue(t.ID),
		Organization:      types.StringValue(organization),
		ProjectID:         types.StringValue(t.ProjectID),
		PlaybookID:        types.StringValue(t.PlaybookID),
		InventoryID:       types.StringValue(t.InventoryID),
		Name:              types.StringValue(t.Name),
		Description:       types.StringValue(t.Description),
		ExtraVars:         extraVars,
		Limit:             types.StringValue(t.Limit),
		Tags:              types.StringValue(t.Tags),
		SkipTags:          types.StringValue(t.SkipTags),
		Verbosity:         types.Int64Value(int64(t.Verbosity)),
		Forks:             types.Int64Value(int64(t.Forks)),
		BecomeEnabled:     types.BoolValue(t.BecomeEnabled),
		DiffMode:          types.BoolValue(t.DiffMode),
		Enabled:           types.BoolValue(t.Enabled),
		TimeoutSeconds:    types.Int64Value(int64(t.TimeoutSeconds)),
		AllowSimultaneous: types.BoolValue(t.AllowSimultaneous),
		JobSliceCount:     types.Int64Value(int64(t.JobSliceCount)),
		ScheduleEnabled:   types.BoolValue(t.ScheduleEnabled),
		ScheduleCron:      types.StringValue(t.ScheduleCron),
		AllowCallbacks:    types.BoolValue(t.AllowCallbacks),
		LaunchOnWebhook:   types.BoolValue(t.LaunchOnWebhook),
		HostConfigKey:     types.StringValue(t.HostConfigKey),
		CreatedAt:         types.StringValue(t.CreatedAt),
		UpdatedAt:         types.StringValue(t.UpdatedAt),
	}

	// credential_id and agent_pool_id are optional relationships; keep them null
	// when the server reports no linkage so an unset config does not drift.
	if t.CredentialID != "" {
		m.CredentialID = types.StringValue(t.CredentialID)
	} else {
		m.CredentialID = types.StringNull()
	}
	if t.AgentPoolID != "" {
		m.AgentPoolID = types.StringValue(t.AgentPoolID)
	} else {
		m.AgentPoolID = types.StringNull()
	}

	// retention_days is a nullable override (nil = inherit the org default).
	if t.RetentionDays != nil {
		m.RetentionDays = types.Int64Value(int64(*t.RetentionDays))
	} else {
		m.RetentionDays = types.Int64Null()
	}

	return m, nil
}

// extraVarsToMap converts the native extra-vars map into a Terraform string map,
// stringifying non-string scalar values. A nil/empty map becomes an empty (not
// null) map so an omitted, server-defaulted "{}" settles without drift.
func extraVarsToMap(ctx context.Context, raw map[string]any) (types.Map, error) {
	elems := make(map[string]attr.Value, len(raw))
	for k, v := range raw {
		if v == nil {
			elems[k] = types.StringValue("")
			continue
		}
		if s, ok := v.(string); ok {
			elems[k] = types.StringValue(s)
			continue
		}
		elems[k] = types.StringValue(fmt.Sprintf("%v", v))
	}
	m, diags := types.MapValue(types.StringType, elems)
	if diags.HasError() {
		return types.MapNull(types.StringType), fmt.Errorf("failed to build extra_vars map: %v", diags)
	}
	return m, nil
}

// mapToExtraVars converts a Terraform string map into the native extra-vars map.
// A null/unknown map yields nil so create/update omit the attribute and let the
// server apply its "{}" default.
func mapToExtraVars(ctx context.Context, m types.Map) (map[string]any, error) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	strs := make(map[string]string, len(m.Elements()))
	if diags := m.ElementsAs(ctx, &strs, false); diags.HasError() {
		return nil, fmt.Errorf("failed to read extra_vars map: %v", diags)
	}
	out := make(map[string]any, len(strs))
	for k, v := range strs {
		out[k] = v
	}
	return out, nil
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleJobTemplate) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *resourceStackweaverAnsibleJobTemplate) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_job_template"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplate) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers an Ansible job template: the central, AWX-style reusable run configuration binding a playbook, inventory, and (optionally) a credential together with the execution knobs and launch triggers a launched job inherits.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the job template.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "Name of the organization. Create is org-scoped. Changing this forces a new job template.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "ID of the owning project. Defaults to the organization's first project when omitted. Changing this forces a new job template.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"playbook_id": schema.StringAttribute{
				Description: "ID of the playbook to run.",
				Required:    true,
			},
			"inventory_id": schema.StringAttribute{
				Description: "ID of the inventory to run against.",
				Required:    true,
			},
			"credential_id": schema.StringAttribute{
				Description: "ID of the legacy single machine credential. Multi-credential attachment is managed by stackweaver_ansible_job_template_credential.",
				Optional:    true,
			},
			"agent_pool_id": schema.StringAttribute{
				Description: "ID of the agent pool to run on. Must belong to the same organization.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the job template, unique within the project.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the job template.",
				Optional:    true,
				Computed:    true,
			},
			"extra_vars": schema.MapAttribute{
				Description: "Extra variables passed to the playbook run.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"limit": schema.StringAttribute{
				Description: "Host pattern limit (--limit).",
				Optional:    true,
				Computed:    true,
			},
			"tags": schema.StringAttribute{
				Description: "Only run plays and tasks tagged with these values (--tags).",
				Optional:    true,
				Computed:    true,
			},
			"skip_tags": schema.StringAttribute{
				Description: "Skip plays and tasks tagged with these values (--skip-tags).",
				Optional:    true,
				Computed:    true,
			},
			"verbosity": schema.Int64Attribute{
				Description: "Ansible verbosity level, 0-4.",
				Optional:    true,
				Computed:    true,
			},
			"forks": schema.Int64Attribute{
				Description: "Number of parallel forks. Defaults to 5; 0 is coerced to 5.",
				Optional:    true,
				Computed:    true,
			},
			"become_enabled": schema.BoolAttribute{
				Description: "Enable privilege escalation (become / sudo).",
				Optional:    true,
				Computed:    true,
			},
			"diff_mode": schema.BoolAttribute{
				Description: "Run in diff mode (--diff).",
				Optional:    true,
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the template is enabled. Defaults to true. (API projection of the inverted model field Disabled.)",
				Optional:    true,
				Computed:    true,
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "Run timeout in seconds. 0 means no timeout.",
				Optional:    true,
				Computed:    true,
			},
			"allow_simultaneous": schema.BoolAttribute{
				Description: "Allow simultaneous runs of this template (AWX concurrent-run semantics).",
				Optional:    true,
				Computed:    true,
			},
			"retention_days": schema.Int64Attribute{
				Description: "Per-template job retention override. Omit to inherit the organization setting; 0 keeps jobs forever.",
				Optional:    true,
			},
			"job_slice_count": schema.Int64Attribute{
				Description: "Number of job slices. Defaults to 1; a value greater than 1 slices the launch.",
				Optional:    true,
				Computed:    true,
			},
			"schedule_enabled": schema.BoolAttribute{
				Description: "Whether the template's schedule is enabled.",
				Optional:    true,
				Computed:    true,
			},
			"schedule_cron": schema.StringAttribute{
				Description: "Cron expression for the template's schedule.",
				Optional:    true,
				Computed:    true,
			},
			"allow_callbacks": schema.BoolAttribute{
				Description: "Enable provisioning callbacks. Update-only: it is not accepted on create and is applied by a follow-up update.",
				Optional:    true,
				Computed:    true,
			},
			"launch_on_webhook": schema.BoolAttribute{
				Description: "Launch on webhook. Update-only: it is not accepted on create and is applied by a follow-up update.",
				Optional:    true,
				Computed:    true,
			},
			"host_config_key": schema.StringAttribute{
				Description: "Host config key for provisioning callbacks. Read-only.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the job template was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the job template was last updated.",
				Computed:    true,
			},
		},
	}
}

// service builds the native job-templates service, or returns an error when the
// provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleJobTemplate) service() (*stackweaver.AnsibleJobTemplatesService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_job_template resources")
	}
	return stackweaver.NewAnsibleJobTemplatesService(r.config.Stackweaver), nil
}

// int64PtrFromPlan returns a *int for a known, non-null Int64; otherwise nil.
func int64PtrFromPlan(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := int(v.ValueInt64())
	return &i
}

// boolPtrFromPlan returns a *bool for a known, non-null Bool; otherwise nil.
func boolPtrFromPlan(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// Create implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplate) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleJobTemplate
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	extraVars, err := mapToExtraVars(ctx, plan.ExtraVars)
	if err != nil {
		resp.Diagnostics.AddError("Invalid extra_vars", err.Error())
		return
	}

	options := stackweaver.AnsibleJobTemplateCreateOptions{
		Organization:      plan.Organization.ValueString(),
		ProjectID:         plan.ProjectID.ValueString(),
		PlaybookID:        plan.PlaybookID.ValueString(),
		InventoryID:       plan.InventoryID.ValueString(),
		CredentialID:      plan.CredentialID.ValueString(),
		AgentPoolID:       plan.AgentPoolID.ValueString(),
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		ExtraVars:         extraVars,
		Limit:             plan.Limit.ValueString(),
		Tags:              plan.Tags.ValueString(),
		SkipTags:          plan.SkipTags.ValueString(),
		Verbosity:         int(plan.Verbosity.ValueInt64()),
		Forks:             int64PtrFromPlan(plan.Forks),
		BecomeEnabled:     plan.BecomeEnabled.ValueBool(),
		DiffMode:          plan.DiffMode.ValueBool(),
		ScheduleEnabled:   plan.ScheduleEnabled.ValueBool(),
		ScheduleCron:      plan.ScheduleCron.ValueString(),
		Enabled:           boolPtrFromPlan(plan.Enabled),
		TimeoutSeconds:    int(plan.TimeoutSeconds.ValueInt64()),
		AllowSimultaneous: plan.AllowSimultaneous.ValueBool(),
		RetentionDays:     int64PtrFromPlan(plan.RetentionDays),
		JobSliceCount:     int64PtrFromPlan(plan.JobSliceCount),
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible job template %s", options.Name))
	template, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible job template", err.Error())
		return
	}

	// allow_callbacks and launch_on_webhook are not accepted on create; apply
	// them with a follow-up update when the config sets them.
	allowCallbacks := boolPtrFromPlan(plan.AllowCallbacks)
	launchOnWebhook := boolPtrFromPlan(plan.LaunchOnWebhook)
	if allowCallbacks != nil || launchOnWebhook != nil {
		tflog.Debug(ctx, fmt.Sprintf("Applying update-only launch triggers on ansible job template %s", template.ID))
		template, err = svc.Update(ctx, template.ID, stackweaver.AnsibleJobTemplateUpdateOptions{
			AllowCallbacks:  allowCallbacks,
			LaunchOnWebhook: launchOnWebhook,
		})
		if err != nil {
			resp.Diagnostics.AddError("Error applying launch triggers to ansible job template", err.Error())
			return
		}
	}

	model, err := modelFromJobTemplate(ctx, template, plan.Organization.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error mapping ansible job template", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplate) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleJobTemplate
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
	tflog.Debug(ctx, fmt.Sprintf("Read ansible job template %s", id))
	template, err := svc.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible job template %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible job template", err.Error())
		return
	}

	// The API does not echo the organization; carry it from prior state.
	model, err := modelFromJobTemplate(ctx, template, state.Organization.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error mapping ansible job template", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Update implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplate) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleJobTemplate
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelStackweaverAnsibleJobTemplate
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	extraVars, err := mapToExtraVars(ctx, plan.ExtraVars)
	if err != nil {
		resp.Diagnostics.AddError("Invalid extra_vars", err.Error())
		return
	}

	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	limit := plan.Limit.ValueString()
	tags := plan.Tags.ValueString()
	skipTags := plan.SkipTags.ValueString()
	scheduleCron := plan.ScheduleCron.ValueString()
	verbosity := int(plan.Verbosity.ValueInt64())
	timeoutSeconds := int(plan.TimeoutSeconds.ValueInt64())
	becomeEnabled := plan.BecomeEnabled.ValueBool()
	diffMode := plan.DiffMode.ValueBool()
	scheduleEnabled := plan.ScheduleEnabled.ValueBool()
	allowSimultaneous := plan.AllowSimultaneous.ValueBool()

	options := stackweaver.AnsibleJobTemplateUpdateOptions{
		PlaybookID:        plan.PlaybookID.ValueString(),
		InventoryID:       plan.InventoryID.ValueString(),
		CredentialID:      plan.CredentialID.ValueString(),
		AgentPoolID:       plan.AgentPoolID.ValueString(),
		Name:              &name,
		Description:       &description,
		Limit:             &limit,
		Tags:              &tags,
		SkipTags:          &skipTags,
		Verbosity:         &verbosity,
		Forks:             int64PtrFromPlan(plan.Forks),
		BecomeEnabled:     &becomeEnabled,
		DiffMode:          &diffMode,
		ScheduleEnabled:   &scheduleEnabled,
		ScheduleCron:      &scheduleCron,
		Enabled:           boolPtrFromPlan(plan.Enabled),
		TimeoutSeconds:    &timeoutSeconds,
		AllowSimultaneous: &allowSimultaneous,
		RetentionDays:     int64PtrFromPlan(plan.RetentionDays),
		JobSliceCount:     int64PtrFromPlan(plan.JobSliceCount),
		AllowCallbacks:    boolPtrFromPlan(plan.AllowCallbacks),
		LaunchOnWebhook:   boolPtrFromPlan(plan.LaunchOnWebhook),
	}
	if extraVars != nil {
		options.ExtraVars = &extraVars
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Update ansible job template %s", id))
	template, err := svc.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible job template", err.Error())
		return
	}

	model, err := modelFromJobTemplate(ctx, template, plan.Organization.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error mapping ansible job template", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplate) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleJobTemplate
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
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible job template %s", id))
	err = svc.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible job template", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState (import by id).
func (r *resourceStackweaverAnsibleJobTemplate) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
