// Copyright IBM Corp. 2018, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"os"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-tfe/internal/client"
)

const defaultSSLSkipVerify = false

var (
	errMissingOrganization = errors.New("no organization was specified on the resource or provider")
)

// ConfiguredClient wraps the tfe.Client the provider uses, plus the default
// organization name to be used by resources that need an organization but don't
// specify one.
type ConfiguredClient struct {
	Client       *tfe.Client
	Organization string
}

func (c ConfiguredClient) schemaOrDefaultOrganization(resource *schema.ResourceData) (string, error) {
	return c.schemaOrDefaultOrganizationKey(resource, "organization")
}

func (c ConfiguredClient) schemaOrDefaultOrganizationKey(resource *schema.ResourceData, key string) (string, error) {
	schemaOrg, got := resource.GetOk(key)
	if got {
		return schemaOrg.(string), nil
	}
	if c.Organization == "" {
		return "", errMissingOrganization
	}
	return c.Organization, nil
}

// MeetsMinRemoteTFEVersion checks if the TFE version meets the minimum required version.
func (c ConfiguredClient) MeetsMinRemoteTFEVersion(minVersion string) (bool, error) {
	return meetsMinTFEVersion(c.Client.IsCloud(), c.RemoteTFEVersion(), minVersion)
}

func meetsMinTFEVersion(isCloud bool, currentVersion, minVersion string) (bool, error) {
	if isCloud {
		return true, nil
	}

	meets, err := checkTFEVersion(currentVersion, minVersion)
	if err != nil {
		return false, err
	}

	return meets, nil
}

// RemoteTFEVersion returns the TFE version.
func (c ConfiguredClient) RemoteTFEVersion() string {
	if v := c.Client.RemoteTFENumericVersion(); v != "" {
		return v
	}
	return c.Client.RemoteTFEVersion()
}

// ctx is used as default context.Context when making TFE calls.
var ctx = context.Background()

// Provider returns a schema.Provider
func Provider() *schema.Provider {
	return &schema.Provider{
		// Note that defaults and fallbacks which are usually handled by DefaultFunc here are
		// instead handled when fetching a HCP Terraform and Terraform Enterprise client in getClient(). This is because the this
		// provider is actually two muxed providers which must respect the same logic for fetching
		// those values in each schema.
		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["hostname"],
			},

			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["token"],
			},

			"ssl_skip_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["ssl_skip_verify"],
			},

			"organization": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["organization"],
			},
		},

		// Each supported (kept) SDKv2 data source is registered under BOTH its
		// primary "stackweaver_*" name and a "tfe_*" alias via
		// addStackweaverAliases (see alias.go). Unsupported upstream data
		// sources are unregistered here (their source files stay in the tree so
		// upstream syncs still diff cleanly — see plan.md §2-3).
		DataSourcesMap: addStackweaverAliases(map[string]*schema.Resource{
			"tfe_agent_pool":              dataSourceTFEAgentPool(),
			"tfe_organization_membership": dataSourceTFEOrganizationMembership(),
			"tfe_team":                    dataSourceTFETeam(),
			"tfe_teams":                   dataSourceTFETeams(),
			"tfe_team_access":             dataSourceTFETeamAccess(),
			"tfe_team_project_access":     dataSourceTFETeamProjectAccess(),
			"tfe_workspace":               dataSourceTFEWorkspace(),
			"tfe_workspace_ids":           dataSourceTFEWorkspaceIDs(),
			"tfe_variable_set":            dataSourceTFEVariableSet(),
			"tfe_organization_members":    dataSourceTFEOrganizationMembers(),
			// IMPORTANT:
			// New data sources should be defined in provider_next.go using
			// the [Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).
		}),

		// Each supported (kept) SDKv2 resource is registered under BOTH its
		// primary "stackweaver_*" name and a "tfe_*" alias via
		// addStackweaverAliases (see alias.go).
		ResourcesMap: addStackweaverAliases(map[string]*schema.Resource{
			"tfe_agent_pool":                     resourceTFEAgentPool(),
			"tfe_agent_pool_allowed_projects":    resourceTFEAgentPoolAllowedProjects(),
			"tfe_agent_pool_allowed_workspaces":  resourceTFEAgentPoolAllowedWorkspaces(),
			"tfe_agent_pool_excluded_workspaces": resourceTFEAgentPoolExcludedWorkspaces(),
			"tfe_agent_token":                    resourceTFEAgentToken(),
			"tfe_organization_membership":        resourceTFEOrganizationMembership(),
			"tfe_organization_token":             resourceTFEOrganizationToken(),
			"tfe_project_variable_set":           resourceTFEProjectVariableSet(),
			"tfe_run_trigger":                    resourceTFERunTrigger(),
			"tfe_team":                           resourceTFETeam(),
			"tfe_team_access":                    resourceTFETeamAccess(),
			"tfe_team_organization_member":       resourceTFETeamOrganizationMember(),
			"tfe_team_organization_members":      resourceTFETeamOrganizationMembers(),
			"tfe_team_project_access":            resourceTFETeamProjectAccess(),
			"tfe_team_members":                   resourceTFETeamMembers(),
			"tfe_workspace":                      resourceTFEWorkspace(),
			"tfe_variable_set":                   resourceTFEVariableSet(),
			"tfe_workspace_run":                  resourceTFEWorkspaceRun(),
			"tfe_workspace_variable_set":         resourceTFEWorkspaceVariableSet(),
			// IMPORTANT:
			// New resources should be defined in provider_next.go using
			// the [Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).
		}),
		ConfigureContextFunc: configure(),
	}
}

func configure() schema.ConfigureContextFunc {
	return func(ctx context.Context, rd *schema.ResourceData) (any, diag.Diagnostics) {
		providerOrganization := rd.Get("organization").(string)
		if providerOrganization == "" {
			providerOrganization = os.Getenv("TFE_ORGANIZATION")
		}

		providerClient, err := configureClient(rd)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		var diagnosticWarnings diag.Diagnostics = nil

		if providerClient.SendAuthenticationWarning() {
			diagnosticWarnings = diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "Authentication method has limited TFE provider permissions",
					Detail:   "When running in HCP Terraform or Terraform Enterprise, the current authentication method may not have sufficient permissions for all TFE provider operations. Data sources and plans may work, but resource create, update, or delete operations can fail. To avoid this, authenticate using the provider token argument or the TFE_TOKEN environment variable.",
				},
			}
		}

		return ConfiguredClient{
			providerClient.TfeClient,
			providerOrganization,
		}, diagnosticWarnings
	}
}

func configureClient(d *schema.ResourceData) (*client.ProviderClient, error) {
	hostname := d.Get("hostname").(string)
	token := d.Get("token").(string)
	insecure := d.Get("ssl_skip_verify").(bool)

	return client.GetClient(hostname, token, insecure)
}

var descriptions = map[string]string{
	"hostname": "The Stackweaver hostname to connect to. Defaults to app.stackweaver.io. Can also be set with the TFE_HOSTNAME environment variable.",
	"token": "The token used to authenticate with Stackweaver. We recommend omitting\n" +
		"the token which can be set as credentials in the CLI config file. Can also be set with the TFE_TOKEN environment variable.",
	"ssl_skip_verify": "Whether or not to skip certificate verifications.",
	"organization": "The organization to apply to a resource if one is not defined on\n" +
		"the resource itself",
}
