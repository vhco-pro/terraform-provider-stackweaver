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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource - no terraform-provider-tfe equivalent. A declarative cron
// schedule that periodically triggers an Ansible target. Registered only under
// "stackweaver_ansible_schedule" (no tfe_ alias).
var (
	_ resource.Resource                     = &resourceStackweaverAnsibleSchedule{}
	_ resource.ResourceWithConfigure        = &resourceStackweaverAnsibleSchedule{}
	_ resource.ResourceWithConfigValidators = &resourceStackweaverAnsibleSchedule{}
	_ resource.ResourceWithImportState      = &resourceStackweaverAnsibleSchedule{}
)

// NewAnsibleScheduleResource is the factory registered in frameworkResources.
func NewAnsibleScheduleResource() resource.Resource {
	return &resourceStackweaverAnsibleSchedule{}
}

type resourceStackweaverAnsibleSchedule struct {
	config ConfiguredClient
}

type modelStackweaverAnsibleSchedule struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Type              types.String `tfsdk:"type"`
	JobTemplateID     types.String `tfsdk:"job_template_id"`
	InventorySourceID types.String `tfsdk:"inventory_source_id"`
	PlaybookID        types.String `tfsdk:"playbook_id"`
	WorkflowID        types.String `tfsdk:"workflow_id"`
	CronExpression    types.String `tfsdk:"cron_expression"`
	Timezone          types.String `tfsdk:"timezone"`
	StartDateTime     types.String `tfsdk:"start_date_time"`
	EndDateTime       types.String `tfsdk:"end_date_time"`
	Config            types.String `tfsdk:"config"`
	Status            types.String `tfsdk:"status"`
	NextRunAt         types.String `tfsdk:"next_run_at"`
	LastRunAt         types.String `tfsdk:"last_run_at"`
	LastRunStatus     types.String `tfsdk:"last_run_status"`
	LastJobID         types.String `tfsdk:"last_job_id"`
	RunCount          types.Int64  `tfsdk:"run_count"`
}

func (r *resourceStackweaverAnsibleSchedule) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceStackweaverAnsibleSchedule) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_schedule"
}

func (r *resourceStackweaverAnsibleSchedule) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	forceNewString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "A declarative cron schedule that periodically triggers an Ansible target (job template, inventory-source sync, playbook sync, or workflow). The organization is taken from the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Service-generated identifier for the schedule.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "Display name of the schedule.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the schedule.",
				Optional:    true,
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description:   "What the schedule triggers: \"job_template\", \"inventory_source\", \"playbook_sync\", or \"workflow\". Determines which target id is required. Changing this forces a new schedule.",
				Required:      true,
				PlanModifiers: forceNewString,
			},
			"job_template_id": schema.StringAttribute{
				Description:   "Target job template (set iff type = job_template). Changing this forces a new schedule.",
				Optional:      true,
				PlanModifiers: forceNewString,
			},
			"inventory_source_id": schema.StringAttribute{
				Description:   "Target inventory source (set iff type = inventory_source). Changing this forces a new schedule.",
				Optional:      true,
				PlanModifiers: forceNewString,
			},
			"playbook_id": schema.StringAttribute{
				Description:   "Target playbook (set iff type = playbook_sync). Changing this forces a new schedule.",
				Optional:      true,
				PlanModifiers: forceNewString,
			},
			"workflow_id": schema.StringAttribute{
				Description:   "Target workflow (set iff type = workflow). Changing this forces a new schedule.",
				Optional:      true,
				PlanModifiers: forceNewString,
			},
			"cron_expression": schema.StringAttribute{
				Description: "Standard 5-field cron expression (minute hour day month weekday).",
				Required:    true,
			},
			"timezone": schema.StringAttribute{
				Description: "IANA timezone for the cron expression (e.g. America/New_York). Defaults to UTC.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("UTC"),
			},
			"start_date_time": schema.StringAttribute{
				Description:   "RFC3339 instant after which the schedule becomes active. Changing this forces a new schedule (not updatable in place).",
				Optional:      true,
				PlanModifiers: forceNewString,
			},
			"end_date_time": schema.StringAttribute{
				Description:   "RFC3339 instant after which the schedule stops. Changing this forces a new schedule (not updatable in place).",
				Optional:      true,
				PlanModifiers: forceNewString,
			},
			"config": schema.StringAttribute{
				Description: "Extra configuration as a JSON object string (e.g. extra_vars override for job-template schedules).",
				Optional:    true,
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Whether the schedule is enabled or disabled. Managed server-side (via the enable/disable actions); read-only here.",
				Computed:    true,
			},
			"next_run_at": schema.StringAttribute{
				Description: "Next calculated run time.",
				Computed:    true,
			},
			"last_run_at": schema.StringAttribute{
				Description: "Most recent execution time.",
				Computed:    true,
			},
			"last_run_status": schema.StringAttribute{
				Description: "Status of the most recent execution.",
				Computed:    true,
			},
			"last_job_id": schema.StringAttribute{
				Description: "ID of the most recent job the schedule created.",
				Computed:    true,
			},
			"run_count": schema.Int64Attribute{
				Description: "Total number of executions.",
				Computed:    true,
			},
		},
	}
}

// ConfigValidators enforces that exactly one target id is set.
func (r *resourceStackweaverAnsibleSchedule) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("job_template_id"),
			path.MatchRoot("inventory_source_id"),
			path.MatchRoot("playbook_id"),
			path.MatchRoot("workflow_id"),
		),
	}
}

func (r *resourceStackweaverAnsibleSchedule) client() (*stackweaver.Client, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_schedule resources")
	}
	return r.config.Stackweaver, nil
}

func (r *resourceStackweaverAnsibleSchedule) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleSchedule
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
		resp.Diagnostics.AddError("Missing organization", "An organization must be configured on the provider to create a stackweaver_ansible_schedule.")
		return
	}

	config, err := jsonStringToMap(plan.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid config", fmt.Sprintf("config must be a JSON object: %s", err))
		return
	}

	options := stackweaver.AnsibleScheduleCreateOptions{
		Organization:      r.config.Organization,
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		Type:              plan.Type.ValueString(),
		JobTemplateID:     plan.JobTemplateID.ValueString(),
		InventorySourceID: plan.InventorySourceID.ValueString(),
		PlaybookID:        plan.PlaybookID.ValueString(),
		WorkflowID:        plan.WorkflowID.ValueString(),
		CronExpression:    plan.CronExpression.ValueString(),
		Timezone:          plan.Timezone.ValueString(),
		StartDateTime:     plan.StartDateTime.ValueString(),
		EndDateTime:       plan.EndDateTime.ValueString(),
		Config:            config,
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible schedule %s", options.Name))
	svc := stackweaver.NewAnsibleSchedulesService(client)
	schedule, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible schedule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.stateFromSchedule(schedule, plan, true))...)
}

func (r *resourceStackweaverAnsibleSchedule) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleSchedule
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
	tflog.Debug(ctx, fmt.Sprintf("Read ansible schedule %s", id))
	svc := stackweaver.NewAnsibleSchedulesService(client)
	schedule, err := svc.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible schedule %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible schedule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.stateFromSchedule(schedule, state, false))...)
}

func (r *resourceStackweaverAnsibleSchedule) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleSchedule
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state modelStackweaverAnsibleSchedule
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	config, err := jsonStringToMap(plan.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid config", fmt.Sprintf("config must be a JSON object: %s", err))
		return
	}

	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	cron := plan.CronExpression.ValueString()
	timezone := plan.Timezone.ValueString()
	options := stackweaver.AnsibleScheduleUpdateOptions{
		Name:           &name,
		Description:    &description,
		CronExpression: &cron,
		Timezone:       &timezone,
		Config:         config,
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Update ansible schedule %s", id))
	svc := stackweaver.NewAnsibleSchedulesService(client)
	schedule, err := svc.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible schedule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.stateFromSchedule(schedule, plan, true))...)
}

func (r *resourceStackweaverAnsibleSchedule) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleSchedule
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
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible schedule %s", id))
	svc := stackweaver.NewAnsibleSchedulesService(client)
	err = svc.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible schedule", err.Error())
		return
	}
}

func (r *resourceStackweaverAnsibleSchedule) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// stateFromSchedule maps a native schedule into resource state. When echoPlan is
// true (after a write) the configured config/target/date-window values are echoed
// from prior so the result matches the plan exactly; otherwise (refresh) config
// is reconciled semantically and the user-set target/date fields are preserved to
// avoid format/echo drift (they are ForceNew and never change in place).
func (r *resourceStackweaverAnsibleSchedule) stateFromSchedule(s *stackweaver.AnsibleSchedule, prior modelStackweaverAnsibleSchedule, echoPlan bool) modelStackweaverAnsibleSchedule {
	m := modelStackweaverAnsibleSchedule{
		ID:             types.StringValue(s.ID),
		Name:           types.StringValue(s.Name),
		Description:    types.StringValue(s.Description),
		Type:           types.StringValue(s.Type),
		CronExpression: types.StringValue(s.CronExpression),
		Timezone:       types.StringValue(s.Timezone),
		Status:         stringOrNull(s.Status),
		NextRunAt:      stringOrNull(s.NextRunAt),
		LastRunAt:      stringOrNull(s.LastRunAt),
		LastRunStatus:  stringOrNull(s.LastRunStatus),
		LastJobID:      stringOrNull(s.LastJobID),
		RunCount:       types.Int64Value(int64(s.RunCount)),
		// Target ids and the date window are ForceNew and not always echoed by the
		// raw-model read; preserve the configured values.
		JobTemplateID:     prior.JobTemplateID,
		InventorySourceID: prior.InventorySourceID,
		PlaybookID:        prior.PlaybookID,
		WorkflowID:        prior.WorkflowID,
		StartDateTime:     prior.StartDateTime,
		EndDateTime:       prior.EndDateTime,
	}
	if echoPlan {
		m.Config = planJSONState(prior.Config, s.Config)
	} else {
		m.Config = refreshJSONState(prior.Config, s.Config)
	}
	return m
}

// stringOrNull returns a null string for "" (computed fields the server omits),
// else the value.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
