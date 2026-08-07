// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource - no terraform-provider-tfe equivalent; modeled on
// tfe_workspace_run. A lifecycle TRIGGER: create launches a job from a template
// and (by default) waits for a terminal status; update is a no-op (launch inputs
// are ForceNew); delete removes the resource from state without undoing the run.
// Registered only under "stackweaver_ansible_job" (no tfe_ alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleJob{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleJob{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleJob{}
)

// jobPollInterval is the delay between job status polls while waiting for a
// terminal status. The overall wait is bounded by the operation's context
// deadline (the Terraform apply timeout), so it never blocks forever.
const jobPollInterval = 3 * time.Second

// NewAnsibleJobResource is the factory registered in frameworkResources.
func NewAnsibleJobResource() resource.Resource {
	return &resourceStackweaverAnsibleJob{}
}

type resourceStackweaverAnsibleJob struct {
	config ConfiguredClient
}

type modelStackweaverAnsibleJob struct {
	ID                types.String `tfsdk:"id"`
	JobTemplateID     types.String `tfsdk:"job_template_id"`
	ExtraVars         types.String `tfsdk:"extra_vars"`
	WaitForCompletion types.Bool   `tfsdk:"wait_for_completion"`
	Status            types.String `tfsdk:"status"`
	StartedAt         types.String `tfsdk:"started_at"`
	FinishedAt        types.String `tfsdk:"finished_at"`
	ExitCode          types.Int64  `tfsdk:"exit_code"`
	HostsOk           types.Int64  `tfsdk:"hosts_ok"`
	HostsChanged      types.Int64  `tfsdk:"hosts_changed"`
	HostsFailed       types.Int64  `tfsdk:"hosts_failed"`
	HostsUnreachable  types.Int64  `tfsdk:"hosts_unreachable"`
}

func (r *resourceStackweaverAnsibleJob) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceStackweaverAnsibleJob) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_job"
}

func (r *resourceStackweaverAnsibleJob) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Launches an Ansible job from a job template. This is a lifecycle trigger (modeled on tfe_workspace_run): it records a point-in-time execution, not reconciled config. Changing any launch input performs a new launch (ForceNew); a no-op re-plan does not re-launch.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The launched job's identifier.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"job_template_id": schema.StringAttribute{
				Description:   "Job template to launch from. A new value performs a new launch (forces replacement).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"extra_vars": schema.StringAttribute{
				Description:   "Extra-vars override as a JSON object string. Changing it performs a new launch (forces replacement). Note: on the template-launch endpoint only extra_vars is honored today; limit/tags/inventory overrides are not yet backed.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"wait_for_completion": schema.BoolAttribute{
				Description: "Whether to poll until the job reaches a terminal status (default true). When false, apply returns immediately with a pending/running status.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"status": schema.StringAttribute{
				Description: "Job status: successful, failed, canceled, or error after waiting; pending/running when not waiting.",
				Computed:    true,
			},
			"started_at": schema.StringAttribute{
				Description: "When the job started (RFC3339).",
				Computed:    true,
			},
			"finished_at": schema.StringAttribute{
				Description: "When the job finished (RFC3339).",
				Computed:    true,
			},
			"exit_code": schema.Int64Attribute{
				Description: "ansible-playbook exit code.",
				Computed:    true,
			},
			"hosts_ok": schema.Int64Attribute{
				Description: "Number of hosts with ok results.",
				Computed:    true,
			},
			"hosts_changed": schema.Int64Attribute{
				Description: "Number of hosts with changed results.",
				Computed:    true,
			},
			"hosts_failed": schema.Int64Attribute{
				Description: "Number of hosts with failed results.",
				Computed:    true,
			},
			"hosts_unreachable": schema.Int64Attribute{
				Description: "Number of unreachable hosts.",
				Computed:    true,
			},
		},
	}
}

func (r *resourceStackweaverAnsibleJob) client() (*stackweaver.Client, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_job resources")
	}
	return r.config.Stackweaver, nil
}

func (r *resourceStackweaverAnsibleJob) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleJob
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	extraVars, err := jsonStringToMap(plan.ExtraVars.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid extra_vars", fmt.Sprintf("extra_vars must be a JSON object: %s", err))
		return
	}

	svc := stackweaver.NewAnsibleJobsService(client)
	tflog.Debug(ctx, fmt.Sprintf("Launch ansible job from template %s", plan.JobTemplateID.ValueString()))
	job, err := svc.Launch(ctx, plan.JobTemplateID.ValueString(), extraVars)
	if err != nil {
		resp.Diagnostics.AddError("Error launching ansible job", err.Error())
		return
	}

	if plan.WaitForCompletion.ValueBool() {
		job, err = r.waitForTerminal(ctx, svc, job.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for ansible job to complete", fmt.Sprintf("job %s: %s", job.ID, err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.stateFromJob(job, plan))...)
}

func (r *resourceStackweaverAnsibleJob) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleJob
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
	tflog.Debug(ctx, fmt.Sprintf("Read ansible job %s", id))
	svc := stackweaver.NewAnsibleJobsService(client)
	job, err := svc.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible job %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible job", err.Error())
		return
	}

	// A recorded terminal job is treated as stable (trigger semantics): the
	// launch is never re-run on a refresh.
	resp.Diagnostics.Append(resp.State.Set(ctx, r.stateFromJob(job, state))...)
}

// Update is a no-op for the trigger. The only non-ForceNew attribute is
// wait_for_completion, which does not affect the already-launched job; changes to
// any launch input are handled by replacement, not this path.
func (r *resourceStackweaverAnsibleJob) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleJob
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state modelStackweaverAnsibleJob
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Carry the recorded execution forward; only wait_for_completion may differ.
	state.WaitForCompletion = plan.WaitForCompletion
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Delete is a no-op: a completed job is immutable history. Destroy removes the
// resource from state without undoing the run.
func (r *resourceStackweaverAnsibleJob) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *resourceStackweaverAnsibleJob) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// waitForTerminal polls the job until it reaches a terminal status or ctx is done.
func (r *resourceStackweaverAnsibleJob) waitForTerminal(ctx context.Context, svc *stackweaver.AnsibleJobsService, id string) (*stackweaver.AnsibleJob, error) {
	for {
		job, err := svc.Read(ctx, id)
		if err != nil {
			return &stackweaver.AnsibleJob{ID: id}, err
		}
		if stackweaver.IsTerminalJobStatus(job.Status) {
			return job, nil
		}
		select {
		case <-ctx.Done():
			return job, ctx.Err()
		case <-time.After(jobPollInterval):
		}
	}
}

// stateFromJob maps a launched job into resource state, echoing the configured
// job_template_id/extra_vars/wait_for_completion from prior (they are ForceNew /
// provider-side and not reconciled).
func (r *resourceStackweaverAnsibleJob) stateFromJob(job *stackweaver.AnsibleJob, prior modelStackweaverAnsibleJob) modelStackweaverAnsibleJob {
	m := modelStackweaverAnsibleJob{
		ID:                types.StringValue(job.ID),
		JobTemplateID:     prior.JobTemplateID,
		ExtraVars:         planJSONState(prior.ExtraVars, job.ExtraVars),
		WaitForCompletion: prior.WaitForCompletion,
		Status:            types.StringValue(job.Status),
		StartedAt:         stringOrNull(job.StartedAt),
		FinishedAt:        stringOrNull(job.FinishedAt),
		HostsOk:           types.Int64Value(int64(job.HostsOk)),
		HostsChanged:      types.Int64Value(int64(job.HostsChanged)),
		HostsFailed:       types.Int64Value(int64(job.HostsFailed)),
		HostsUnreachable:  types.Int64Value(int64(job.HostsUnreachable)),
	}
	// job_template_id is echoed from prior when known; the job read also carries
	// it via the template relationship (use it when prior is unset, e.g. import).
	if prior.JobTemplateID.IsNull() || prior.JobTemplateID.IsUnknown() {
		m.JobTemplateID = stringOrNull(job.JobTemplateID)
	}
	if job.ExitCode != nil {
		m.ExitCode = types.Int64Value(int64(*job.ExitCode))
	} else {
		m.ExitCode = types.Int64Null()
	}
	// On import prior.WaitForCompletion is null; default it to true.
	if m.WaitForCompletion.IsNull() || m.WaitForCompletion.IsUnknown() {
		m.WaitForCompletion = types.BoolValue(true)
	}
	return m
}
