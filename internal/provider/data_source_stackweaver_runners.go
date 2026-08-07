// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native data source - no terraform-provider-tfe equivalent. Read-only
// observability view over an organization's self-hosted runner fleet, served by
// the native internal/stackweaver client (not go-tfe). There is deliberately no
// stackweaver_runner resource: Terraform never creates or owns a runner.
var (
	_ datasource.DataSource              = &dataSourceStackweaverRunners{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverRunners{}
)

// NewRunnersDataSource is the factory registered in frameworkDataSources.
func NewRunnersDataSource() datasource.DataSource {
	return &dataSourceStackweaverRunners{}
}

type dataSourceStackweaverRunners struct {
	config ConfiguredClient
}

// modelStackweaverRunners maps the data source schema to a struct.
type modelStackweaverRunners struct {
	Organization types.String `tfsdk:"organization"`
	AgentPoolID  types.String `tfsdk:"agent_pool_id"`
	RunnerType   types.String `tfsdk:"runner_type"`
	Status       types.String `tfsdk:"status"`
	ID           types.String `tfsdk:"id"`
	Runners      types.List   `tfsdk:"runners"`
	Stats        types.Object `tfsdk:"stats"`
}

// modelRunnerElem is one element of the computed runners list.
type modelRunnerElem struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	AgentPoolID      types.String `tfsdk:"agent_pool_id"`
	RunnerType       types.String `tfsdk:"runner_type"`
	Status           types.String `tfsdk:"status"`
	Hostname         types.String `tfsdk:"hostname"`
	OSType           types.String `tfsdk:"os_type"`
	AgentVersion     types.String `tfsdk:"agent_version"`
	Labels           types.List   `tfsdk:"labels"`
	TerraformVersion types.String `tfsdk:"terraform_version"`
	AnsibleVersion   types.String `tfsdk:"ansible_version"`
	LastHeartbeatAt  types.String `tfsdk:"last_heartbeat_at"`
}

// modelRunnerStats is the computed stats summary object.
type modelRunnerStats struct {
	Total   types.Int64 `tfsdk:"total"`
	Online  types.Int64 `tfsdk:"online"`
	Offline types.Int64 `tfsdk:"offline"`
}

// runnerElemAttrTypes is the attribute-type map for a runners[] element object.
func runnerElemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                types.StringType,
		"name":              types.StringType,
		"agent_pool_id":     types.StringType,
		"runner_type":       types.StringType,
		"status":            types.StringType,
		"hostname":          types.StringType,
		"os_type":           types.StringType,
		"agent_version":     types.StringType,
		"labels":            types.ListType{ElemType: types.StringType},
		"terraform_version": types.StringType,
		"ansible_version":   types.StringType,
		"last_heartbeat_at": types.StringType,
	}
}

// runnerStatsAttrTypes is the attribute-type map for the stats object.
func runnerStatsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"total":   types.Int64Type,
		"online":  types.Int64Type,
		"offline": types.Int64Type,
	}
}

// Metadata implements datasource.DataSource. Native data sources take the
// primary stackweaver_* type name with no tfe_ alias.
func (d *dataSourceStackweaverRunners) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runners"
}

// Schema implements datasource.DataSource.
func (d *dataSourceStackweaverRunners) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the self-hosted runner fleet in an organization, optionally filtered by agent pool, type, or status, together with a total/online/offline summary.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description: "Name of the organization. Defaults to the provider's default organization.",
				Optional:    true,
				Computed:    true,
			},
			"agent_pool_id": schema.StringAttribute{
				Description: "Filter the fleet to a single agent pool.",
				Optional:    true,
			},
			"runner_type": schema.StringAttribute{
				Description: "Filter the fleet by runner type (\"terraform\", \"ansible\", or \"combined\").",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "Filter the fleet by reported status (\"online\", \"offline\", \"busy\", or \"error\").",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Description: "Synthetic identifier - the organization name.",
				Computed:    true,
			},
			"runners": schema.ListNestedAttribute{
				Description: "The runner fleet matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                schema.StringAttribute{Description: "Server-assigned runner ID.", Computed: true},
						"name":              schema.StringAttribute{Description: "Runner name, unique within the organization.", Computed: true},
						"agent_pool_id":     schema.StringAttribute{Description: "ID of the agent pool the runner belongs to.", Computed: true},
						"runner_type":       schema.StringAttribute{Description: "Runner type.", Computed: true},
						"status":            schema.StringAttribute{Description: "Reported health status.", Computed: true},
						"hostname":          schema.StringAttribute{Description: "Agent-reported hostname.", Computed: true},
						"os_type":           schema.StringAttribute{Description: "Agent-reported OS type.", Computed: true},
						"agent_version":     schema.StringAttribute{Description: "Agent-reported version.", Computed: true},
						"labels":            schema.ListAttribute{Description: "Agent-reported labels.", Computed: true, ElementType: types.StringType},
						"terraform_version": schema.StringAttribute{Description: "Terraform capability version.", Computed: true},
						"ansible_version":   schema.StringAttribute{Description: "Ansible capability version.", Computed: true},
						"last_heartbeat_at": schema.StringAttribute{Description: "Timestamp of the last heartbeat (empty when never seen).", Computed: true},
					},
				},
			},
			"stats": schema.SingleNestedAttribute{
				Description: "Fleet summary counts.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"total":   schema.Int64Attribute{Description: "Total number of registered runners.", Computed: true},
					"online":  schema.Int64Attribute{Description: "Number of online runners.", Computed: true},
					"offline": schema.Int64Attribute{Description: "Number of offline runners.", Computed: true},
				},
			},
		},
	}
}

// Configure implements datasource.DataSourceWithConfigure.
func (d *dataSourceStackweaverRunners) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(ConfiguredClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source Configure type",
			fmt.Sprintf("Expected provider.ConfiguredClient, got %T. This is a bug in the provider, so please report it on GitHub.", req.ProviderData),
		)
		return
	}
	d.config = client
}

// Read implements datasource.DataSource.
func (d *dataSourceStackweaverRunners) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelStackweaverRunners
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider must be configured with a hostname and token to read stackweaver_runners.",
		)
		return
	}

	var organization string
	resp.Diagnostics.Append(d.config.dataOrDefaultOrganization(ctx, req.Config, &organization)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc := stackweaver.NewRunnersService(d.config.Stackweaver)

	tflog.Debug(ctx, fmt.Sprintf("Reading runner fleet for organization %s", organization))
	runners, err := svc.List(ctx, stackweaver.RunnerListOptions{
		Organization: organization,
		AgentPoolID:  config.AgentPoolID.ValueString(),
		RunnerType:   config.RunnerType.ValueString(),
		Status:       config.Status.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing runners", err.Error())
		return
	}

	stats, err := svc.Stats(ctx, organization)
	if err != nil {
		resp.Diagnostics.AddError("Error reading runner stats", err.Error())
		return
	}

	elems := make([]modelRunnerElem, 0, len(runners))
	for _, r := range runners {
		labels, diags := types.ListValueFrom(ctx, types.StringType, r.Labels)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elems = append(elems, modelRunnerElem{
			ID:               types.StringValue(r.ID),
			Name:             types.StringValue(r.Name),
			AgentPoolID:      types.StringValue(r.AgentPoolID),
			RunnerType:       types.StringValue(r.RunnerType),
			Status:           types.StringValue(r.Status),
			Hostname:         types.StringValue(r.Hostname),
			OSType:           types.StringValue(r.OSType),
			AgentVersion:     types.StringValue(r.AgentVersion),
			Labels:           labels,
			TerraformVersion: types.StringValue(r.TerraformVersion),
			AnsibleVersion:   types.StringValue(r.AnsibleVersion),
			LastHeartbeatAt:  types.StringValue(r.LastHeartbeatAt),
		})
	}

	runnersList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: runnerElemAttrTypes()}, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	statsObj, diags := types.ObjectValueFrom(ctx, runnerStatsAttrTypes(), modelRunnerStats{
		Total:   types.Int64Value(int64(stats.Total)),
		Online:  types.Int64Value(int64(stats.Online)),
		Offline: types.Int64Value(int64(stats.Offline)),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.Organization = types.StringValue(organization)
	config.ID = types.StringValue(organization)
	config.Runners = runnersList
	config.Stats = statsObj

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
