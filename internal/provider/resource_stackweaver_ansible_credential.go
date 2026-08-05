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

// Native resource — no terraform-provider-tfe equivalent. Served by the native
// internal/stackweaver client and registered only under its primary
// "stackweaver_ansible_credential" name (native resources get NO tfe_ alias).
//
// Secret material is write-only: the API accepts it on write but never echoes
// it, so Read never overwrites the secret attributes (they are kept from prior
// state), and presence is tracked via the computed has_* booleans.
var (
	_ resource.Resource                = &resourceStackweaverAnsibleCredential{}
	_ resource.ResourceWithConfigure   = &resourceStackweaverAnsibleCredential{}
	_ resource.ResourceWithImportState = &resourceStackweaverAnsibleCredential{}
)

// NewAnsibleCredentialResource is the factory registered in frameworkResources.
func NewAnsibleCredentialResource() resource.Resource {
	return &resourceStackweaverAnsibleCredential{}
}

type resourceStackweaverAnsibleCredential struct {
	config ConfiguredClient
}

// modelStackweaverAnsibleCredential maps the resource schema to a struct.
type modelStackweaverAnsibleCredential struct {
	ID             types.String `tfsdk:"id"`
	Organization   types.String `tfsdk:"organization"`
	ProjectID      types.String `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	CredentialType types.String `tfsdk:"credential_type"`
	Username       types.String `tfsdk:"username"`
	AzureTenantID  types.String `tfsdk:"azure_tenant_id"`
	AzureClientID  types.String `tfsdk:"azure_client_id"`
	SSHPort        types.Int64  `tfsdk:"ssh_port"`
	SSHBecomeUser  types.String `tfsdk:"ssh_become_user"`

	// Write-only secrets — never read back from the API.
	SSHPrivateKey      types.String `tfsdk:"ssh_private_key"`
	SSHPassphrase      types.String `tfsdk:"ssh_passphrase"`
	Password           types.String `tfsdk:"password"`
	VaultPassword      types.String `tfsdk:"vault_password"`
	BecomePassword     types.String `tfsdk:"become_password"`
	AWSAccessKeyID     types.String `tfsdk:"aws_access_key_id"`
	AWSSecretAccessKey types.String `tfsdk:"aws_secret_access_key"`
	AzureClientSecret  types.String `tfsdk:"azure_client_secret"`
	GCPServiceAccount  types.String `tfsdk:"gcp_service_account"`

	// Presence readbacks for the four secrets the API reports on.
	HasSSHPrivateKey  types.Bool `tfsdk:"has_ssh_private_key"`
	HasPassword       types.Bool `tfsdk:"has_password"`
	HasVaultPassword  types.Bool `tfsdk:"has_vault_password"`
	HasBecomePassword types.Bool `tfsdk:"has_become_password"`
}

// applyReadable overlays the server-returned readable fields onto the model,
// leaving the write-only secrets, organization, and project_id untouched (the
// API cannot return those).
func (m *modelStackweaverAnsibleCredential) applyReadable(c *stackweaver.AnsibleCredential) {
	m.ID = types.StringValue(c.ID)
	m.Name = types.StringValue(c.Name)
	m.Description = types.StringValue(c.Description)
	m.CredentialType = types.StringValue(c.Type)
	m.Username = types.StringValue(c.Username)
	m.AzureTenantID = types.StringValue(c.AzureTenantID)
	m.AzureClientID = types.StringValue(c.AzureClientID)
	m.SSHPort = types.Int64Value(c.SSHPort)
	m.SSHBecomeUser = types.StringValue(c.SSHBecomeUser)
	m.HasSSHPrivateKey = types.BoolValue(c.HasSSHPrivateKey)
	m.HasPassword = types.BoolValue(c.HasPassword)
	m.HasVaultPassword = types.BoolValue(c.HasVaultPassword)
	m.HasBecomePassword = types.BoolValue(c.HasBecomePassword)
}

// Configure implements resource.ResourceWithConfigure.
func (r *resourceStackweaverAnsibleCredential) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *resourceStackweaverAnsibleCredential) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_credential"
}

// Schema implements resource.Resource.
func (r *resourceStackweaverAnsibleCredential) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Ansible credential: a named, typed secret bundle scoped to an organization (optionally a project) for host access, SCM access, vault decryption, or cloud-inventory auth. Secret material is write-only — accepted on write and never read back.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Service-generated identifier for the credential.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization": schema.StringAttribute{
				Description: "Name of the owning organization. Changing this forces a new credential.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "ID of the project to narrow the credential to. Omit for an org-scoped credential (resolves to the organization's default project). Not returned by the API, so it is tracked from configuration only.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the credential, unique within the organization.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the credential.",
				Optional:    true,
				Computed:    true,
			},
			"credential_type": schema.StringAttribute{
				Description: "Credential type: one of ssh, scm, vault, machine-ssh, aws, azure, gcp, vmware. Immutable — changing it forces a new credential.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "Username for host/SCM access (readable).",
				Optional:    true,
				Computed:    true,
			},
			"azure_tenant_id": schema.StringAttribute{
				Description: "Azure tenant ID (readable, not a secret).",
				Optional:    true,
				Computed:    true,
			},
			"azure_client_id": schema.StringAttribute{
				Description: "Azure client ID (readable, not a secret).",
				Optional:    true,
				Computed:    true,
			},
			"ssh_port": schema.Int64Attribute{
				Description: "SSH port for host access. Defaults to 22.",
				Optional:    true,
				Computed:    true,
			},
			"ssh_become_user": schema.StringAttribute{
				Description: "User to become for privilege escalation. Defaults to \"root\".",
				Optional:    true,
				Computed:    true,
			},
			"ssh_private_key": schema.StringAttribute{
				Description: "SSH private key (write-only — never read back).",
				Optional:    true,
				Sensitive:   true,
			},
			"ssh_passphrase": schema.StringAttribute{
				Description: "Passphrase for the SSH private key (write-only).",
				Optional:    true,
				Sensitive:   true,
			},
			"password": schema.StringAttribute{
				Description: "Password for host/SCM access (write-only).",
				Optional:    true,
				Sensitive:   true,
			},
			"vault_password": schema.StringAttribute{
				Description: "Ansible Vault password (write-only).",
				Optional:    true,
				Sensitive:   true,
			},
			"become_password": schema.StringAttribute{
				Description: "Privilege-escalation (sudo) password (write-only).",
				Optional:    true,
				Sensitive:   true,
			},
			"aws_access_key_id": schema.StringAttribute{
				Description: "AWS access key ID (write-only — no presence readback).",
				Optional:    true,
				Sensitive:   true,
			},
			"aws_secret_access_key": schema.StringAttribute{
				Description: "AWS secret access key (write-only — no presence readback).",
				Optional:    true,
				Sensitive:   true,
			},
			"azure_client_secret": schema.StringAttribute{
				Description: "Azure client secret (write-only — no presence readback).",
				Optional:    true,
				Sensitive:   true,
			},
			"gcp_service_account": schema.StringAttribute{
				Description: "GCP service account JSON (write-only — no presence readback).",
				Optional:    true,
				Sensitive:   true,
			},
			"has_ssh_private_key": schema.BoolAttribute{
				Description: "Whether an SSH private key is stored.",
				Computed:    true,
			},
			"has_password": schema.BoolAttribute{
				Description: "Whether a password is stored.",
				Computed:    true,
			},
			"has_vault_password": schema.BoolAttribute{
				Description: "Whether a vault password is stored.",
				Computed:    true,
			},
			"has_become_password": schema.BoolAttribute{
				Description: "Whether a become password is stored.",
				Computed:    true,
			},
		},
	}
}

// service returns the native credentials service, or an error diagnostic when
// the provider was configured without a hostname/token.
func (r *resourceStackweaverAnsibleCredential) service() (*stackweaver.AnsibleCredentialsService, error) {
	if r.config.Stackweaver == nil {
		return nil, errors.New("the provider must be configured with a hostname and token to manage stackweaver_ansible_credential resources")
	}
	return stackweaver.NewAnsibleCredentialsService(r.config.Stackweaver), nil
}

// Create implements resource.Resource.
func (r *resourceStackweaverAnsibleCredential) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelStackweaverAnsibleCredential
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.service()
	if err != nil {
		resp.Diagnostics.AddError("Provider not configured", err.Error())
		return
	}

	options := stackweaver.AnsibleCredentialCreateOptions{
		Organization:       plan.Organization.ValueString(),
		ProjectID:          plan.ProjectID.ValueString(),
		Name:               plan.Name.ValueString(),
		Description:        plan.Description.ValueString(),
		Type:               plan.CredentialType.ValueString(),
		Username:           plan.Username.ValueString(),
		AzureTenantID:      plan.AzureTenantID.ValueString(),
		AzureClientID:      plan.AzureClientID.ValueString(),
		SSHPort:            plan.SSHPort.ValueInt64(),
		SSHBecomeUser:      plan.SSHBecomeUser.ValueString(),
		SSHPrivateKey:      plan.SSHPrivateKey.ValueString(),
		SSHPassphrase:      plan.SSHPassphrase.ValueString(),
		Password:           plan.Password.ValueString(),
		VaultPassword:      plan.VaultPassword.ValueString(),
		BecomePassword:     plan.BecomePassword.ValueString(),
		AWSAccessKeyID:     plan.AWSAccessKeyID.ValueString(),
		AWSSecretAccessKey: plan.AWSSecretAccessKey.ValueString(),
		AzureClientSecret:  plan.AzureClientSecret.ValueString(),
		GCPServiceAccount:  plan.GCPServiceAccount.ValueString(),
	}

	tflog.Debug(ctx, fmt.Sprintf("Create ansible credential %s", options.Name))
	cred, err := svc.Create(ctx, options)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ansible credential", err.Error())
		return
	}

	plan.applyReadable(cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read implements resource.Resource.
func (r *resourceStackweaverAnsibleCredential) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelStackweaverAnsibleCredential
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
	tflog.Debug(ctx, fmt.Sprintf("Read ansible credential %s", id))
	cred, err := svc.Read(ctx, id)
	if err != nil {
		if errors.Is(err, stackweaver.ErrNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Ansible credential %s no longer exists", id))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ansible credential", err.Error())
		return
	}

	// Overlay readable fields; secrets/organization/project_id are preserved
	// from state because the API does not return them.
	state.applyReadable(cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update implements resource.Resource.
func (r *resourceStackweaverAnsibleCredential) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelStackweaverAnsibleCredential
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state modelStackweaverAnsibleCredential
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
	options := stackweaver.AnsibleCredentialUpdateOptions{
		Name:               plan.Name.ValueString(),
		Description:        plan.Description.ValueString(),
		Username:           plan.Username.ValueString(),
		AzureTenantID:      plan.AzureTenantID.ValueString(),
		AzureClientID:      plan.AzureClientID.ValueString(),
		SSHPort:            plan.SSHPort.ValueInt64(),
		SSHBecomeUser:      plan.SSHBecomeUser.ValueString(),
		ProjectID:          plan.ProjectID.ValueString(),
		SSHPrivateKey:      plan.SSHPrivateKey.ValueString(),
		SSHPassphrase:      plan.SSHPassphrase.ValueString(),
		Password:           plan.Password.ValueString(),
		VaultPassword:      plan.VaultPassword.ValueString(),
		BecomePassword:     plan.BecomePassword.ValueString(),
		AWSAccessKeyID:     plan.AWSAccessKeyID.ValueString(),
		AWSSecretAccessKey: plan.AWSSecretAccessKey.ValueString(),
		AzureClientSecret:  plan.AzureClientSecret.ValueString(),
		GCPServiceAccount:  plan.GCPServiceAccount.ValueString(),
	}

	tflog.Debug(ctx, fmt.Sprintf("Update ansible credential %s", id))
	cred, err := svc.Update(ctx, id, options)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ansible credential", err.Error())
		return
	}

	plan.applyReadable(cred)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete implements resource.Resource.
func (r *resourceStackweaverAnsibleCredential) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelStackweaverAnsibleCredential
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
	tflog.Debug(ctx, fmt.Sprintf("Delete ansible credential %s", id))
	err = svc.Delete(ctx, id)
	if err != nil && !errors.Is(err, stackweaver.ErrNotFound) {
		resp.Diagnostics.AddError("Error deleting ansible credential", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState (import by id). Secret
// attributes cannot be imported (the API never returns them); set them in
// configuration after import.
func (r *resourceStackweaverAnsibleCredential) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
