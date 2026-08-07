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

// Native resource - no terraform-provider-tfe equivalent. Registered only under
// its primary "stackweaver_ansible_notification_template" name (no tfe_ alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleNotificationTemplate{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleNotificationTemplate{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleNotificationTemplate{}
)

// NewAnsibleNotificationTemplateResource is the factory registered in
// frameworkResources.
func NewAnsibleNotificationTemplateResource() resource.Resource {
	return &resourceStackweaverAnsibleNotificationTemplate{}
}

type resourceStackweaverAnsibleNotificationTemplate struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleNotificationTemplate maps the resource schema. config is
// a JSON string (jsonb); secret is write-only and never read back.
type modelStackweaverAnsibleNotificationTemplate struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Config      types.String `tfsdk:"config"`
	Secret      types.String `tfsdk:"secret"`
}

func (r *resourceStackweaverAnsibleNotificationTemplate) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceStackweaverAnsibleNotificationTemplate) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_notification_template"
}

func (r *resourceStackweaverAnsibleNotificationTemplate) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An org-scoped Ansible notification channel (webhook, email, or Teams) that can be attached to job templates and workflows. The organization is taken from the provider configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the notification template.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the notification template, unique within the organization.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Delivery channel: \"webhook\", \"email\", or \"teams\". Changing this forces a new template.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the template.",
				Optional:    true,
				Computed:    true,
			},
			"config": schema.StringAttribute{
				Description: "Channel-specific non-secret settings as a JSON object string (e.g. webhook url/method/headers, SMTP host/port, Teams url).",
				Optional:    true,
				Computed:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The channel's single sensitive value (webhook basic-auth password / SMTP password). Write-only: it is sent on create/update but never read back from the API.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *resourceStackweaverAnsibleNotificationTemplate) client() (*stackweaver.Client, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_notification_template resources")
	}
	return r.config.Stackweaver, nil
}

func (r *resourceStackweaverAnsibleNotificationTemplate) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleNotificationTemplate
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
		resp.Diagnostics.AddError("Missing organization", "An organization must be configured on the provider to create a stackweaver_ansible_notification_template.")
		return
	}

	config, err := jsonStringToMap(plan.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid config", fmt.Sprintf("config must be a JSON object: %s", err))
		return
	}

	options := stackweaver.AnsibleNotificationTemplateCreateOptions{
		Organization: r.config.Organization,
		Name:         plan.Name.ValueString(),
		Description:  plan.Description.ValueString(),
		Type:         plan.Type.ValueString(),
		Config:       config,
	}
	if !plan.Secret.IsNull() {
		secret := plan.Secret.ValueString()
		options.Secret = &secret
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible notification template %s", options.Name))
	svc := stackweaver.NewAnsibleNotificationTemplatesService(client)
	template, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible notification template", err.Error())
		return
	}

	state := r.modelFromTemplate(template, plan.Config, plan.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceStackweaverAnsibleNotificationTemplate) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleNotificationTemplate
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
	tflog.Debug(ctx, fmt.Sprintf("Read ansible notification template %s (via org list)", id))
	svc := stackweaver.NewAnsibleNotificationTemplatesService(client)
	template, err := svc.Read(ctx, r.config.Organization, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible notification template %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible notification template", err.Error())
		return
	}

	// secret is write-only: preserve the prior state value (never returned).
	newState := modelStackweaverAnsibleNotificationTemplate{
		ID:          types.StringValue(template.ID),
		Name:        types.StringValue(template.Name),
		Description: types.StringValue(template.Description),
		Type:        types.StringValue(template.Type),
		Config:      refreshJSONState(state.Config, template.Config),
		Secret:      state.Secret,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *resourceStackweaverAnsibleNotificationTemplate) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleNotificationTemplate
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state modelStackweaverAnsibleNotificationTemplate
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
	options := stackweaver.AnsibleNotificationTemplateUpdateOptions{
		Name:        &name,
		Description: &description,
		Config:      config,
	}
	if !plan.Secret.IsNull() {
		secret := plan.Secret.ValueString()
		options.Secret = &secret
	}

	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Update ansible notification template %s", id))
	svc := stackweaver.NewAnsibleNotificationTemplatesService(client)
	template, err := svc.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible notification template", err.Error())
		return
	}

	newState := r.modelFromTemplate(template, plan.Config, plan.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *resourceStackweaverAnsibleNotificationTemplate) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleNotificationTemplate
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
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible notification template %s", id))
	svc := stackweaver.NewAnsibleNotificationTemplatesService(client)
	err = svc.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible notification template", err.Error())
		return
	}
}

func (r *resourceStackweaverAnsibleNotificationTemplate) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelFromTemplate builds resource state after a write, echoing the configured
// config/secret so the result matches the plan exactly.
func (r *resourceStackweaverAnsibleNotificationTemplate) modelFromTemplate(t *stackweaver.AnsibleNotificationTemplate, planConfig, planSecret types.String) modelStackweaverAnsibleNotificationTemplate {
	return modelStackweaverAnsibleNotificationTemplate{
		ID:          types.StringValue(t.ID),
		Name:        types.StringValue(t.Name),
		Description: types.StringValue(t.Description),
		Type:        types.StringValue(t.Type),
		Config:      planJSONState(planConfig, t.Config),
		Secret:      planSecret,
	}
}
