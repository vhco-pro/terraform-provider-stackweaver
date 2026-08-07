// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native resource - no terraform-provider-tfe equivalent. A single variable
// scoped to one Ansible job template, served by the native internal/stackweaver
// client and registered only under its primary name (no tfe_ alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleJobTemplateVariable{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleJobTemplateVariable{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleJobTemplateVariable{}
)

// NewAnsibleJobTemplateVariableResource is the factory registered in
// frameworkResources.
func NewAnsibleJobTemplateVariableResource() resource.Resource {
	return &resourceStackweaverAnsibleJobTemplateVariable{}
}

type resourceStackweaverAnsibleJobTemplateVariable struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleJobTemplateVariable maps the resource schema to a struct.
type modelStackweaverAnsibleJobTemplateVariable struct {
	ID            types.String `tfsdk:"id"`
	JobTemplateID types.String `tfsdk:"job_template_id"`
	Key           types.String `tfsdk:"key"`
	Value         types.String `tfsdk:"value"`
	Description   types.String `tfsdk:"description"`
	Category      types.String `tfsdk:"category"`
	HCL           types.Bool   `tfsdk:"hcl"`
	Sensitive     types.Bool   `tfsdk:"sensitive"`
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleJobTemplateVariable) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *resourceStackweaverAnsibleJobTemplateVariable) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_job_template_variable"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplateVariable) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a single variable scoped to one Ansible job template (the AWX/TFE analogue of a workspace variable).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the variable (var-… format).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"job_template_id": schema.StringAttribute{
				Description: "ID of the owning job template. Changing this forces a new variable.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Description: "Variable key, unique within the template.",
				Required:    true,
			},
			"value": schema.StringAttribute{
				Description: "Variable value. When sensitive is true, the value is write-only: the API masks it on read, so the configured value is retained in state.",
				Required:    true,
				Sensitive:   true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the variable.",
				Optional:    true,
				Computed:    true,
			},
			"category": schema.StringAttribute{
				Description: "Variable category: \"env\" (default) or \"terraform\".",
				Optional:    true,
				Computed:    true,
			},
			"hcl": schema.BoolAttribute{
				Description: "Whether the value is HCL. Carried for TFE compatibility; not used for Ansible execution.",
				Optional:    true,
				Computed:    true,
			},
			"sensitive": schema.BoolAttribute{
				Description: "Whether the value is sensitive. When true, the server encrypts it at rest and masks it on read.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

// service builds the native job-template-variables service, or returns an error
// when the provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleJobTemplateVariable) service() (*stackweaver.AnsibleJobTemplateVariablesService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_job_template_variable resources")
	}
	return stackweaver.NewAnsibleJobTemplateVariablesService(r.config.Stackweaver), nil
}

// modelFromVariable maps a native variable into the resource model. configuredValue
// is the value the practitioner supplied; it is retained verbatim so a sensitive
// value (masked on read) never appears as drift.
func modelFromVariable(v *stackweaver.AnsibleJobTemplateVariable, configuredValue types.String) modelStackweaverAnsibleJobTemplateVariable {
	value := types.StringValue(v.Value)
	if v.Sensitive || v.Value == stackweaver.SensitiveValueMask {
		// Keep the configured value; the API never returns a usable sensitive value.
		value = configuredValue
	}
	return modelStackweaverAnsibleJobTemplateVariable{
		ID:            types.StringValue(v.ID),
		JobTemplateID: types.StringValue(v.JobTemplateID),
		Key:           types.StringValue(v.Key),
		Value:         value,
		Description:   types.StringValue(v.Description),
		Category:      types.StringValue(v.Category),
		HCL:           types.BoolValue(v.HCL),
		Sensitive:     types.BoolValue(v.Sensitive),
	}
}

// Create implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplateVariable) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleJobTemplateVariable
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	options := stackweaver.AnsibleJobTemplateVariableCreateOptions{
		JobTemplateID: plan.JobTemplateID.ValueString(),
		Key:           plan.Key.ValueString(),
		Value:         plan.Value.ValueString(),
		Description:   plan.Description.ValueString(),
		Category:      plan.Category.ValueString(),
		HCL:           plan.HCL.ValueBool(),
		Sensitive:     plan.Sensitive.ValueBool(),
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible job template variable %s", options.Key))
	variable, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible job template variable", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromVariable(variable, plan.Value))...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplateVariable) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleJobTemplateVariable
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	templateID := state.JobTemplateID.ValueString()
	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Read ansible job template variable %s", id))
	variable, err := svc.Read(ctx, templateID, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible job template variable %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible job template variable", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromVariable(variable, state.Value))...)
}

// Update implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplateVariable) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleJobTemplateVariable
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelStackweaverAnsibleJobTemplateVariable
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	hcl := plan.HCL.ValueBool()
	sensitive := plan.Sensitive.ValueBool()
	options := stackweaver.AnsibleJobTemplateVariableUpdateOptions{
		Key:         plan.Key.ValueString(),
		Value:       plan.Value.ValueString(),
		Description: plan.Description.ValueString(),
		Category:    plan.Category.ValueString(),
		HCL:         &hcl,
		Sensitive:   &sensitive,
	}

	templateID := state.JobTemplateID.ValueString()
	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Update ansible job template variable %s", id))
	variable, err := svc.Update(ctx, templateID, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible job template variable", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, modelFromVariable(variable, plan.Value))...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplateVariable) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleJobTemplateVariable
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	templateID := state.JobTemplateID.ValueString()
	id := state.ID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible job template variable %s", id))
	err = svc.Delete(ctx, templateID, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible job template variable", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState. Because there is no
// single-variable GET, import needs both ids: "<job_template_id>/<variable_id>".
func (r *resourceStackweaverAnsibleJobTemplateVariable) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form <job_template_id>/<variable_id>, got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("job_template_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
