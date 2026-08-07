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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native relationship resource - no terraform-provider-tfe equivalent. It binds a
// notification template to exactly one target (job template OR workflow) with the
// per-trigger flags. Create/delete only: every attribute is ForceNew (no update).
// Registered only under "stackweaver_ansible_notification_attachment" (no alias).
var (
	_ resource.Resource                     = &resourceStackweaverAnsibleNotificationAttachment{}
	_ resource.ResourceWithConfigure        = &resourceStackweaverAnsibleNotificationAttachment{}
	_ resource.ResourceWithConfigValidators = &resourceStackweaverAnsibleNotificationAttachment{}
	_ resource.ResourceWithImportState      = &resourceStackweaverAnsibleNotificationAttachment{}
)

// NewAnsibleNotificationAttachmentResource is the factory registered in
// frameworkResources.
func NewAnsibleNotificationAttachmentResource() resource.Resource {
	return &resourceStackweaverAnsibleNotificationAttachment{}
}

type resourceStackweaverAnsibleNotificationAttachment struct {
	config ConfiguredClient
}

type modelStackweaverAnsibleNotificationAttachment struct {
	ID                     types.String `tfsdk:"id"`
	NotificationTemplateID types.String `tfsdk:"notification_template_id"`
	JobTemplateID          types.String `tfsdk:"job_template_id"`
	WorkflowID             types.String `tfsdk:"workflow_id"`
	OnStarted              types.Bool   `tfsdk:"on_started"`
	OnSuccess              types.Bool   `tfsdk:"on_success"`
	OnFailure              types.Bool   `tfsdk:"on_failure"`
}

func (r *resourceStackweaverAnsibleNotificationAttachment) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceStackweaverAnsibleNotificationAttachment) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_notification_attachment"
}

func (r *resourceStackweaverAnsibleNotificationAttachment) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Binds an Ansible notification template to exactly one target (a job template or a workflow) with per-trigger flags. Create/delete only - every attribute forces replacement. The organization is taken from the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the attachment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"notification_template_id": schema.StringAttribute{
				Description: "ID of the notification template (channel) being attached. Changing this forces a new attachment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"job_template_id": schema.StringAttribute{
				Description: "ID of the job template to attach to. Exactly one of job_template_id or workflow_id must be set. Changing this forces a new attachment.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"workflow_id": schema.StringAttribute{
				Description: "ID of the workflow to attach to. Exactly one of job_template_id or workflow_id must be set. Changing this forces a new attachment.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"on_started": schema.BoolAttribute{
				Description: "Fire the channel when the target starts. Defaults to false. Changing this forces a new attachment.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"on_success": schema.BoolAttribute{
				Description: "Fire the channel on success. Defaults to true. Changing this forces a new attachment.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"on_failure": schema.BoolAttribute{
				Description: "Fire the channel on failure. Defaults to true. Changing this forces a new attachment.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// ConfigValidators enforces the exactly-one-target rule at plan time.
func (r *resourceStackweaverAnsibleNotificationAttachment) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("job_template_id"),
			path.MatchRoot("workflow_id"),
		),
	}
}

func (r *resourceStackweaverAnsibleNotificationAttachment) client() (*stackweaver.Client, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_notification_attachment resources")
	}
	return r.config.Stackweaver, nil
}

func (r *resourceStackweaverAnsibleNotificationAttachment) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleNotificationAttachment
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
		resp.Diagnostics.AddError("Missing organization", "An organization must be configured on the provider to create a stackweaver_ansible_notification_attachment.")
		return
	}

	options := stackweaver.AnsibleNotificationAttachmentCreateOptions{
		Organization:           r.config.Organization,
		NotificationTemplateID: plan.NotificationTemplateID.ValueString(),
		JobTemplateID:          plan.JobTemplateID.ValueString(),
		WorkflowID:             plan.WorkflowID.ValueString(),
		OnStarted:              plan.OnStarted.ValueBool(),
		OnSuccess:              plan.OnSuccess.ValueBool(),
		OnFailure:              plan.OnFailure.ValueBool(),
	}

	tflog.Debug(ctx, "Create ansible notification attachment")
	svc := stackweaver.NewAnsibleNotificationAttachmentsService(client)
	attachment, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible notification attachment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.modelFromAttachment(attachment, plan))...)
}

func (r *resourceStackweaverAnsibleNotificationAttachment) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleNotificationAttachment
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Workflow-target attachments have no listing route - there is no server read
	// path, so keep prior state (can't detect out-of-band drift).
	if state.JobTemplateID.IsNull() || state.JobTemplateID.ValueString() == "" {
		return
	}

	client, err := r.client()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Read ansible notification attachment %s (via job-template notifications)", id))
	svc := stackweaver.NewAnsibleNotificationAttachmentsService(client)
	attachment, err := svc.ReadByJobTemplate(ctx, state.JobTemplateID.ValueString(), id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible notification attachment %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible notification attachment", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, r.modelFromAttachment(attachment, state))...)
}

// Update never runs (all attributes ForceNew), but the interface requires it.
func (r *resourceStackweaverAnsibleNotificationAttachment) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleNotificationAttachment
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceStackweaverAnsibleNotificationAttachment) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleNotificationAttachment
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
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible notification attachment %s", id))
	svc := stackweaver.NewAnsibleNotificationAttachmentsService(client)
	err = svc.Delete(ctx, r.config.Organization, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible notification attachment", err.Error())
		return
	}
}

func (r *resourceStackweaverAnsibleNotificationAttachment) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelFromAttachment builds state from a native attachment, preserving the
// planned/prior target ids (the create response echoes them; the read response
// carries them in relationships).
func (r *resourceStackweaverAnsibleNotificationAttachment) modelFromAttachment(a *stackweaver.AnsibleNotificationAttachment, prior modelStackweaverAnsibleNotificationAttachment) modelStackweaverAnsibleNotificationAttachment {
	m := modelStackweaverAnsibleNotificationAttachment{
		ID:                     types.StringValue(a.ID),
		NotificationTemplateID: types.StringValue(a.NotificationTemplateID),
		OnStarted:              types.BoolValue(a.OnStarted),
		OnSuccess:              types.BoolValue(a.OnSuccess),
		OnFailure:              types.BoolValue(a.OnFailure),
		JobTemplateID:          prior.JobTemplateID,
		WorkflowID:             prior.WorkflowID,
	}
	if a.JobTemplateID != "" {
		m.JobTemplateID = types.StringValue(a.JobTemplateID)
	}
	if a.WorkflowID != "" {
		m.WorkflowID = types.StringValue(a.WorkflowID)
	}
	return m
}
