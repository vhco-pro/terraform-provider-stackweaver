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

// Native relationship resource — no terraform-provider-tfe equivalent. Attaches
// one Ansible credential to a job template's multi-credential set. A pure
// association: create = attach, delete = detach, no update. Served by the native
// internal/stackweaver client and registered only under its primary name (no
// tfe_ alias).
var (
	_ resource.Resource                = &resourceStackweaverAnsibleJobTemplateCredential{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleJobTemplateCredential{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleJobTemplateCredential{}
)

// NewAnsibleJobTemplateCredentialResource is the factory registered in
// frameworkResources.
func NewAnsibleJobTemplateCredentialResource() resource.Resource {
	return &resourceStackweaverAnsibleJobTemplateCredential{}
}

type resourceStackweaverAnsibleJobTemplateCredential struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleJobTemplateCredential maps the resource schema to a struct.
type modelStackweaverAnsibleJobTemplateCredential struct {
	ID             types.String `tfsdk:"id"`
	JobTemplateID  types.String `tfsdk:"job_template_id"`
	CredentialID   types.String `tfsdk:"credential_id"`
	CredentialType types.String `tfsdk:"credential_type"`
	CredentialName types.String `tfsdk:"credential_name"`
}

// compositeID synthesizes the resource id "<job_template_id>/<credential_id>";
// the join table exposes no standalone row id.
func compositeID(jobTemplateID, credentialID string) string {
	return jobTemplateID + "/" + credentialID
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleJobTemplateCredential) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *resourceStackweaverAnsibleJobTemplateCredential) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_job_template_credential"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleJobTemplateCredential) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches one Ansible credential to a job template's multi-credential set (the AWX one-credential-per-type rule). A pure association with no in-place update: any change re-attaches.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Synthetic composite identifier, <job_template_id>/<credential_id>.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"job_template_id": schema.StringAttribute{
				Description: "ID of the job template. Changing this re-attaches the credential.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"credential_id": schema.StringAttribute{
				Description: "ID of the credential to attach. Must belong to the template's organization. Changing this re-attaches.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"credential_type": schema.StringAttribute{
				Description: "Type of the attached credential (the per-type uniqueness key).",
				Computed:    true,
			},
			"credential_name": schema.StringAttribute{
				Description: "Name of the attached credential.",
				Computed:    true,
			},
		},
	}
}

// service builds the native job-template-credentials service, or returns an
// error when the provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleJobTemplateCredential) service() (*stackweaver.AnsibleJobTemplateCredentialsService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_job_template_credential resources")
	}
	return stackweaver.NewAnsibleJobTemplateCredentialsService(r.config.Stackweaver), nil
}

// Create implements resource.Resource (attach).
func (r *resourceStackweaverAnsibleJobTemplateCredential) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleJobTemplateCredential
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	templateID := plan.JobTemplateID.ValueString()
	credentialID := plan.CredentialID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Attach credential %s to ansible job template %s", credentialID, templateID))
	cred, err := svc.Attach(ctx, templateID, credentialID)
	if err != nil {
		resp.Diagnostics.AddError("Error attaching credential to ansible job template", err.Error())
		return
	}

	plan.ID = types.StringValue(compositeID(templateID, credentialID))
	plan.CredentialType = types.StringValue(cred.CredentialType)
	plan.CredentialName = types.StringValue(cred.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read implements resource.Resource (membership inferred from the list).
func (r *resourceStackweaverAnsibleJobTemplateCredential) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleJobTemplateCredential
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
	credentialID := state.CredentialID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Read credential %s on ansible job template %s", credentialID, templateID))
	cred, err := svc.Read(ctx, templateID, credentialID)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Credential %s no longer attached to ansible job template %s", credentialID, templateID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible job template credential", err.Error())
		return
	}

	state.ID = types.StringValue(compositeID(templateID, credentialID))
	state.CredentialType = types.StringValue(cred.CredentialType)
	state.CredentialName = types.StringValue(cred.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update implements resource.Resource. Both keys are ForceNew, so no in-place
// update is ever requested; this satisfies the interface as a no-op.
func (r *resourceStackweaverAnsibleJobTemplateCredential) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleJobTemplateCredential
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete implements resource.Resource (detach).
func (r *resourceStackweaverAnsibleJobTemplateCredential) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleJobTemplateCredential
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
	credentialID := state.CredentialID.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("Detach credential %s from ansible job template %s", credentialID, templateID))
	err = svc.Detach(ctx, templateID, credentialID)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error detaching credential from ansible job template", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState. Import expects the
// composite id "<job_template_id>/<credential_id>".
func (r *resourceStackweaverAnsibleJobTemplateCredential) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form <job_template_id>/<credential_id>, got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("job_template_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("credential_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), compositeID(parts[0], parts[1]))...)
}
