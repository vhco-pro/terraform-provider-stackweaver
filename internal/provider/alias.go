// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This file is the single intentional seam vs. upstream terraform-provider-tfe:
// it exposes every supported resource/data source under BOTH a primary
// "stackweaver_*" name and a "tfe_*" alias (drop-in migration path). The
// individual resource_tfe_*.go / data_source_*.go files stay byte-identical to
// upstream so the sync agent's targeted diff of them stays clean (plan.md §2).
//
//   - SDKv2 (provider.go): addStackweaverAliases duplicates each kept map key.
//   - Framework (provider_next.go): the real factories already produce
//     "stackweaver_*" (ProviderTypeName == "stackweaver"); the alias*Factories
//     below wrap each kept factory in a thin decorator that emits the "tfe_*"
//     name and delegates every other method to the inner resource/data source.

// ---------------------------------------------------------------------------
// SDKv2 dual-key registration
// ---------------------------------------------------------------------------

// addStackweaverAliases returns a new map in which every "tfe_<name>" key from
// the input is present under both "tfe_<name>" and "stackweaver_<name>",
// pointing at the same *schema.Resource factory result. Non-"tfe_"-prefixed
// keys (there should be none) are passed through unchanged.
func addStackweaverAliases(m map[string]*schema.Resource) map[string]*schema.Resource {
	out := make(map[string]*schema.Resource, len(m)*2)
	for name, r := range m {
		out[name] = r
		if suffix, ok := strings.CutPrefix(name, "tfe_"); ok {
			out["stackweaver_"+suffix] = r
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Framework alias factories (the "tfe_*" wrappers)
// ---------------------------------------------------------------------------

// aliasResourceFactories returns one tfe_*-emitting alias factory per kept
// framework resource. Keep this list in lock-step with
// frameworkProvider.frameworkResources.
func aliasResourceFactories() []func() resource.Resource {
	return []func() resource.Resource{
		aliasResourceFactory(NewAuditTrailTokenResource, "tfe_audit_trail_token"),
		aliasResourceFactory(NewOrganizationDefaultSettings, "tfe_organization_default_settings"),
		aliasResourceFactory(NewOrganizationRunTaskGlobalSettingsResource, "tfe_organization_run_task_global_settings"),
		aliasResourceFactory(NewOrganizationRunTaskResource, "tfe_organization_run_task"),
		aliasResourceFactory(NewProjectResource, "tfe_project"),
		aliasResourceFactory(NewRegistryGPGKeyResource, "tfe_registry_gpg_key"),
		aliasResourceFactory(NewRegistryProviderResource, "tfe_registry_provider"),
		aliasResourceFactory(NewResourceVariable, "tfe_variable"),
		aliasResourceFactory(NewResourceWorkspaceSettings, "tfe_workspace_settings"),
		aliasResourceFactory(NewTeamNotificationConfigurationResource, "tfe_team_notification_configuration"),
		aliasResourceFactory(NewWorkspaceRunTaskResource, "tfe_workspace_run_task"),
		aliasResourceFactory(NewNotificationConfigurationResource, "tfe_notification_configuration"),
		aliasResourceFactory(NewProjectNotificationConfigurationResource, "tfe_project_notification_configuration"),
		aliasResourceFactory(NewTeamTokenResource, "tfe_team_token"),
		aliasResourceFactory(NewProjectSettingsResource, "tfe_project_settings"),
		aliasResourceFactory(NewTerraformVersionResource, "tfe_terraform_version"),
		aliasResourceFactory(NewAWSOIDCConfigurationResource, "tfe_aws_oidc_configuration"),
		aliasResourceFactory(NewGCPOIDCConfigurationResource, "tfe_gcp_oidc_configuration"),
		aliasResourceFactory(NewAzureOIDCConfigurationResource, "tfe_azure_oidc_configuration"),
		aliasResourceFactory(NewVaultOIDCConfigurationResource, "tfe_vault_oidc_configuration"),
	}
}

// aliasDataSourceFactories returns one tfe_*-emitting alias factory per kept
// framework data source. Keep this list in lock-step with
// frameworkProvider.frameworkDataSources.
func aliasDataSourceFactories() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		aliasDataSourceFactory(NewCurrentUserDataSource, "tfe_current_user"),
		aliasDataSourceFactory(NewOrganizationRunTaskDataSource, "tfe_organization_run_task"),
		aliasDataSourceFactory(NewOrganizationRunTaskGlobalSettingsDataSource, "tfe_organization_run_task_global_settings"),
		aliasDataSourceFactory(NewOutputsDataSource, "tfe_outputs"),
		aliasDataSourceFactory(NewProjectDataSource, "tfe_project"),
		aliasDataSourceFactory(NewProjectsDataSource, "tfe_projects"),
		aliasDataSourceFactory(NewRegistryGPGKeyDataSource, "tfe_registry_gpg_key"),
		aliasDataSourceFactory(NewRegistryGPGKeysDataSource, "tfe_registry_gpg_keys"),
		aliasDataSourceFactory(NewRegistryProviderDataSource, "tfe_registry_provider"),
		aliasDataSourceFactory(NewRegistryProvidersDataSource, "tfe_registry_providers"),
		aliasDataSourceFactory(NewVariablesDataSource, "tfe_variables"),
		aliasDataSourceFactory(NewWorkspaceRunTaskDataSource, "tfe_workspace_run_task"),
	}
}

// aliasResourceFactory wraps a framework resource factory so the returned
// resource reports the tfe_* type name while delegating all behavior to a fresh
// inner resource. A new inner is constructed per call (Terraform news a resource
// per instance). The wrapper variant is chosen once, up front, so that the
// optional ResourceWithIdentity capability is advertised only when the inner
// actually implements it (the identity schema is queried eagerly at provider
// load, so it must not be faked - see aliasResourceWithIdentity).
func aliasResourceFactory(inner func() resource.Resource, tfeName string) func() resource.Resource {
	_, hasIdentity := inner().(resource.ResourceWithIdentity)
	if hasIdentity {
		return func() resource.Resource {
			return &aliasResourceWithIdentity{aliasResource{inner: inner(), typeName: tfeName}}
		}
	}
	return func() resource.Resource {
		return &aliasResource{inner: inner(), typeName: tfeName}
	}
}

// aliasDataSourceFactory wraps a framework data source factory so the returned
// data source reports the tfe_* type name while delegating all behavior to a
// fresh inner data source.
func aliasDataSourceFactory(inner func() datasource.DataSource, tfeName string) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &aliasDataSource{inner: inner(), typeName: tfeName}
	}
}

// ---------------------------------------------------------------------------
// aliasResource - framework resource decorator
// ---------------------------------------------------------------------------

// aliasResource decorates an inner resource.Resource, overriding only its
// reported type name (to the tfe_* alias) and delegating every other method to
// the inner. It implements the optional resource interfaces whose presence is
// either universally safe to advertise (a no-op / nil fallback is
// behaviour-identical to a resource that does not implement it) or handled via
// a faithful fallback (ImportState). The one capability that must NOT be faked -
// ResourceWithIdentity, whose schema is queried eagerly at provider load - is
// gated in aliasResourceFactory and added by aliasResourceWithIdentity.
type aliasResource struct {
	inner    resource.Resource
	typeName string // the full tfe_* alias name, e.g. "tfe_project"
}

var (
	_ resource.Resource                     = (*aliasResource)(nil)
	_ resource.ResourceWithConfigure        = (*aliasResource)(nil)
	_ resource.ResourceWithImportState      = (*aliasResource)(nil)
	_ resource.ResourceWithModifyPlan       = (*aliasResource)(nil)
	_ resource.ResourceWithValidateConfig   = (*aliasResource)(nil)
	_ resource.ResourceWithConfigValidators = (*aliasResource)(nil)
	_ resource.ResourceWithUpgradeState     = (*aliasResource)(nil)
	_ resource.ResourceWithMoveState        = (*aliasResource)(nil)
)

// Metadata delegates to the inner first (preserving fields such as
// ResourceBehavior) and then overrides only the type name with the tfe_* alias.
func (a *aliasResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	a.inner.Metadata(ctx, req, resp)
	resp.TypeName = a.typeName
}

func (a *aliasResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	a.inner.Schema(ctx, req, resp)
}

func (a *aliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	a.inner.Create(ctx, req, resp)
}

func (a *aliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	a.inner.Read(ctx, req, resp)
}

func (a *aliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	a.inner.Update(ctx, req, resp)
}

func (a *aliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	a.inner.Delete(ctx, req, resp)
}

func (a *aliasResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if inner, ok := a.inner.(resource.ResourceWithConfigure); ok {
		inner.Configure(ctx, req, resp)
	}
}

func (a *aliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if inner, ok := a.inner.(resource.ResourceWithImportState); ok {
		inner.ImportState(ctx, req, resp)
		return
	}
	resp.Diagnostics.AddError(
		"Resource does not support import",
		fmt.Sprintf("The %q resource does not support import.", a.typeName),
	)
}

func (a *aliasResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if inner, ok := a.inner.(resource.ResourceWithModifyPlan); ok {
		inner.ModifyPlan(ctx, req, resp)
	}
}

func (a *aliasResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if inner, ok := a.inner.(resource.ResourceWithValidateConfig); ok {
		inner.ValidateConfig(ctx, req, resp)
	}
}

func (a *aliasResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	if inner, ok := a.inner.(resource.ResourceWithConfigValidators); ok {
		return inner.ConfigValidators(ctx)
	}
	return nil
}

func (a *aliasResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	if inner, ok := a.inner.(resource.ResourceWithUpgradeState); ok {
		return inner.UpgradeState(ctx)
	}
	return nil
}

func (a *aliasResource) MoveState(ctx context.Context) []resource.StateMover {
	if inner, ok := a.inner.(resource.ResourceWithMoveState); ok {
		return inner.MoveState(ctx)
	}
	return nil
}

// aliasResourceWithIdentity is the aliasResource variant used when the inner
// resource implements managed-resource identity. It is only constructed for
// such resources (see aliasResourceFactory), so IdentitySchema always has a real
// inner to delegate to and identity is never advertised for a resource that
// lacks it.
type aliasResourceWithIdentity struct {
	aliasResource
}

var (
	_ resource.ResourceWithIdentity        = (*aliasResourceWithIdentity)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*aliasResourceWithIdentity)(nil)
)

func (a *aliasResourceWithIdentity) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	// Guaranteed to succeed: the identity variant is only built for inners that
	// implement ResourceWithIdentity.
	a.inner.(resource.ResourceWithIdentity).IdentitySchema(ctx, req, resp)
}

func (a *aliasResourceWithIdentity) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	if inner, ok := a.inner.(resource.ResourceWithUpgradeIdentity); ok {
		return inner.UpgradeIdentity(ctx)
	}
	return nil
}

// ---------------------------------------------------------------------------
// aliasDataSource - framework data source decorator
// ---------------------------------------------------------------------------

// aliasDataSource decorates an inner datasource.DataSource, overriding only its
// reported type name (to the tfe_* alias) and delegating every other method to
// the inner. Data sources have no eagerly-advertised optional capabilities, so a
// single decorator covers Configure / ConfigValidators / ValidateConfig with
// safe fallbacks.
type aliasDataSource struct {
	inner    datasource.DataSource
	typeName string // the full tfe_* alias name, e.g. "tfe_project"
}

var (
	_ datasource.DataSource                     = (*aliasDataSource)(nil)
	_ datasource.DataSourceWithConfigure        = (*aliasDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*aliasDataSource)(nil)
	_ datasource.DataSourceWithValidateConfig   = (*aliasDataSource)(nil)
)

func (a *aliasDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	a.inner.Metadata(ctx, req, resp)
	resp.TypeName = a.typeName
}

func (a *aliasDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	a.inner.Schema(ctx, req, resp)
}

func (a *aliasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	a.inner.Read(ctx, req, resp)
}

func (a *aliasDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if inner, ok := a.inner.(datasource.DataSourceWithConfigure); ok {
		inner.Configure(ctx, req, resp)
	}
}

func (a *aliasDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	if inner, ok := a.inner.(datasource.DataSourceWithConfigValidators); ok {
		return inner.ConfigValidators(ctx)
	}
	return nil
}

func (a *aliasDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	if inner, ok := a.inner.(datasource.DataSourceWithValidateConfig); ok {
		inner.ValidateConfig(ctx, req, resp)
	}
}
