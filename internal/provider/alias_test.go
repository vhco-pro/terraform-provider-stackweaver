// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// frameworkResourceTypeNames returns the set of type names the framework
// provider registers, as Terraform sees them (ProviderTypeName == "stackweaver").
func frameworkResourceTypeNames(t *testing.T) map[string]bool {
	t.Helper()
	p := &frameworkProvider{}
	names := map[string]bool{}
	for _, factory := range p.Resources(context.Background()) {
		resp := &resource.MetadataResponse{}
		factory().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "stackweaver"}, resp)
		names[resp.TypeName] = true
	}
	return names
}

// frameworkDataSourceTypeNames is the data-source equivalent of the above.
func frameworkDataSourceTypeNames(t *testing.T) map[string]bool {
	t.Helper()
	p := &frameworkProvider{}
	names := map[string]bool{}
	for _, factory := range p.DataSources(context.Background()) {
		resp := &datasource.MetadataResponse{}
		factory().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "stackweaver"}, resp)
		names[resp.TypeName] = true
	}
	return names
}

// TestSDKv2DualPrefixRegistration proves the SDKv2 half exposes each kept
// resource/data source under both prefixes and none of the stripped types.
func TestSDKv2DualPrefixRegistration(t *testing.T) {
	p := Provider()

	present := []string{
		// resource, both prefixes
		"tfe_workspace", "stackweaver_workspace",
		"tfe_agent_pool", "stackweaver_agent_pool",
		"tfe_team", "stackweaver_team",
	}
	for _, name := range present {
		if _, ok := p.ResourcesMap[name]; !ok {
			t.Errorf("SDKv2 resource %q should be registered but is not", name)
		}
	}

	presentDS := []string{
		"tfe_workspace", "stackweaver_workspace",
		"tfe_variable_set", "stackweaver_variable_set",
	}
	for _, name := range presentDS {
		if _, ok := p.DataSourcesMap[name]; !ok {
			t.Errorf("SDKv2 data source %q should be registered but is not", name)
		}
	}

	// Stripped resources must be gone under BOTH prefixes.
	absent := []string{
		"tfe_policy_set", "stackweaver_policy_set",
		"tfe_oauth_client", "stackweaver_oauth_client",
		"tfe_sentinel_policy", "stackweaver_sentinel_policy",
		"tfe_organization", "stackweaver_organization", // partial/blocked
		"tfe_team_member", "stackweaver_team_member", // partial/blocked
	}
	for _, name := range absent {
		if _, ok := p.ResourcesMap[name]; ok {
			t.Errorf("SDKv2 resource %q was stripped but is still registered", name)
		}
	}

	absentDS := []string{
		"tfe_policy_set", "stackweaver_policy_set",
		"tfe_oauth_client", "stackweaver_oauth_client",
		"tfe_github_app_installation", "stackweaver_github_app_installation",
	}
	for _, name := range absentDS {
		if _, ok := p.DataSourcesMap[name]; ok {
			t.Errorf("SDKv2 data source %q was stripped but is still registered", name)
		}
	}
}

// TestFrameworkDualPrefixRegistration proves the plugin-framework half exposes
// each kept resource/data source under both the stackweaver_* primary name and
// the tfe_* alias, and none of the stripped types.
func TestFrameworkDualPrefixRegistration(t *testing.T) {
	resources := frameworkResourceTypeNames(t)
	dataSources := frameworkDataSourceTypeNames(t)

	presentResources := []string{
		"stackweaver_project", "tfe_project",
		"stackweaver_variable", "tfe_variable", // exercises the identity-variant alias
		"stackweaver_terraform_version", "tfe_terraform_version", // was hardcoded upstream
		"stackweaver_workspace_run_task", "tfe_workspace_run_task",
	}
	for _, name := range presentResources {
		if !resources[name] {
			t.Errorf("framework resource %q should be registered but is not", name)
		}
	}

	presentDataSources := []string{
		"stackweaver_project", "tfe_project",
		"stackweaver_outputs", "tfe_outputs",
	}
	for _, name := range presentDataSources {
		if !dataSources[name] {
			t.Errorf("framework data source %q should be registered but is not", name)
		}
	}

	// Stripped framework types must be gone under BOTH prefixes.
	absentResources := []string{
		"stackweaver_ssh_key", "tfe_ssh_key",
		"stackweaver_stack", "tfe_stack",
		"stackweaver_saml_settings", "tfe_saml_settings",
		"stackweaver_opa_version", "tfe_opa_version",
	}
	for _, name := range absentResources {
		if resources[name] {
			t.Errorf("framework resource %q was stripped but is still registered", name)
		}
	}

	absentDataSources := []string{
		"stackweaver_ip_ranges", "tfe_ip_ranges",
		"stackweaver_registry_module", "tfe_registry_module",
	}
	for _, name := range absentDataSources {
		if dataSources[name] {
			t.Errorf("framework data source %q was stripped but is still registered", name)
		}
	}
}

// TestKeptSurfaceCounts guards the exact v0.1 forked surface: 39 resources and
// 22 data sources, each under both prefixes across the two muxed halves.
func TestKeptSurfaceCounts(t *testing.T) {
	p := Provider()
	fwResources := frameworkResourceTypeNames(t)
	fwDataSources := frameworkDataSourceTypeNames(t)

	countTFE := func(names map[string]bool) int {
		n := 0
		for name := range names {
			if len(name) > 4 && name[:4] == "tfe_" {
				n++
			}
		}
		return n
	}
	sdkResourceTFE := 0
	sdkResourceSW := 0
	for name := range p.ResourcesMap {
		switch {
		case len(name) > 4 && name[:4] == "tfe_":
			sdkResourceTFE++
		case len(name) > 12 && name[:12] == "stackweaver_":
			sdkResourceSW++
		}
	}
	sdkDSTFE := 0
	sdkDSSW := 0
	for name := range p.DataSourcesMap {
		switch {
		case len(name) > 4 && name[:4] == "tfe_":
			sdkDSTFE++
		case len(name) > 12 && name[:12] == "stackweaver_":
			sdkDSSW++
		}
	}

	fwResTFE := countTFE(fwResources)
	fwDSTFE := countTFE(fwDataSources)

	totalResources := sdkResourceTFE + fwResTFE
	totalDataSources := sdkDSTFE + fwDSTFE

	if totalResources != 39 {
		t.Errorf("expected 39 kept resources (tfe_ names), got %d (sdk=%d, framework=%d)", totalResources, sdkResourceTFE, fwResTFE)
	}
	if totalDataSources != 22 {
		t.Errorf("expected 22 kept data sources (tfe_ names), got %d (sdk=%d, framework=%d)", totalDataSources, sdkDSTFE, fwDSTFE)
	}

	// Every tfe_ name must have a stackweaver_ twin in the SDKv2 maps.
	if sdkResourceTFE != sdkResourceSW {
		t.Errorf("SDKv2 resource prefix mismatch: %d tfe_ vs %d stackweaver_", sdkResourceTFE, sdkResourceSW)
	}
	if sdkDSTFE != sdkDSSW {
		t.Errorf("SDKv2 data-source prefix mismatch: %d tfe_ vs %d stackweaver_", sdkDSTFE, sdkDSSW)
	}
}
