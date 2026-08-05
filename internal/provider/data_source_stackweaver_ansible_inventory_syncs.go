// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

// Native data source — no terraform-provider-tfe equivalent. History helper
// that lists the sync runs of one Ansible inventory (newest first). Served by
// the native internal/stackweaver client (JSON:API, dash-cased keys), the list
// omits the large captured output by design. Registered only under its primary
// stackweaver_ansible_inventory_syncs name (native = no tfe_ alias).
var (
	_ datasource.DataSource              = &dataSourceStackweaverAnsibleInventorySyncs{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverAnsibleInventorySyncs{}
)

// NewAnsibleInventorySyncsDataSource is the factory for the data source.
func NewAnsibleInventorySyncsDataSource() datasource.DataSource {
	return &dataSourceStackweaverAnsibleInventorySyncs{}
}

type dataSourceStackweaverAnsibleInventorySyncs struct {
	config ConfiguredClient
}

type modelDataSourceInventorySyncs struct {
	InventoryID types.String `tfsdk:"inventory_id"`
	ID          types.String `tfsdk:"id"`
	Syncs       types.List   `tfsdk:"syncs"`
}

// inventorySyncAttrTypes is the object type of a syncs element.
var inventorySyncAttrTypes = map[string]attr.Type{
	"id":                types.StringType,
	"status":            types.StringType,
	"triggered_by":      types.StringType,
	"hosts_discovered":  types.Int64Type,
	"groups_discovered": types.Int64Type,
	"source_name":       types.StringType,
	"error":             types.StringType,
	"started_at":        types.StringType,
	"finished_at":       types.StringType,
	"created_at":        types.StringType,
}

func (d *dataSourceStackweaverAnsibleInventorySyncs) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ansible_inventory_syncs"
}

func (d *dataSourceStackweaverAnsibleInventorySyncs) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the sync-run history (AWX's inventory update jobs) of one Ansible inventory, newest first.",
		Attributes: map[string]schema.Attribute{
			"inventory_id": schema.StringAttribute{
				Description: "ID of the inventory whose sync history to list.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Set to the requested inventory_id.",
				Computed:    true,
			},
			"syncs": schema.ListNestedAttribute{
				Description: "Sync runs, newest first. The captured output is intentionally omitted.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Sync run ID.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Lifecycle status: pending | running | successful | failed.",
							Computed:    true,
						},
						"triggered_by": schema.StringAttribute{
							Description: "What started the run: manual | schedule | launch | workflow | webhook.",
							Computed:    true,
						},
						"hosts_discovered": schema.Int64Attribute{
							Description: "Number of hosts discovered by the run.",
							Computed:    true,
						},
						"groups_discovered": schema.Int64Attribute{
							Description: "Number of groups discovered by the run.",
							Computed:    true,
						},
						"source_name": schema.StringAttribute{
							Description: "Name of the dynamic source, when the run is a source sync.",
							Computed:    true,
						},
						"error": schema.StringAttribute{
							Description: "Failure detail; empty on success.",
							Computed:    true,
						},
						"started_at": schema.StringAttribute{
							Description: "RFC3339 start time; null until the run starts.",
							Computed:    true,
						},
						"finished_at": schema.StringAttribute{
							Description: "RFC3339 finish time; null until the run finishes.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "RFC3339 creation time.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *dataSourceStackweaverAnsibleInventorySyncs) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dataSourceStackweaverAnsibleInventorySyncs) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelDataSourceInventorySyncs
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider must be configured with a hostname and token to read stackweaver_ansible_inventory_syncs.")
		return
	}

	inventoryID := config.InventoryID.ValueString()
	svc := stackweaver.NewAnsibleInventorySyncsService(d.config.Stackweaver)

	// Page through the history so the data source surfaces the full run set.
	const pageSize = 100
	var syncs []*stackweaver.InventorySync
	for offset := 0; ; offset += pageSize {
		tflog.Debug(ctx, fmt.Sprintf("Read ansible inventory syncs for %s (offset %d)", inventoryID, offset))
		page, total, err := svc.ListByInventory(ctx, inventoryID, stackweaver.InventorySyncListOptions{Limit: pageSize, Offset: offset})
		if err != nil {
			if errors.Is(err, stackweaver.ErrNotFound) {
				resp.Diagnostics.AddError("Inventory not found", err.Error())
				return
			}
			resp.Diagnostics.AddError("Error listing inventory syncs", err.Error())
			return
		}
		syncs = append(syncs, page...)
		if len(page) == 0 || len(syncs) >= total {
			break
		}
	}

	elems := make([]attr.Value, len(syncs))
	for i, s := range syncs {
		obj, diags := types.ObjectValue(inventorySyncAttrTypes, map[string]attr.Value{
			"id":                types.StringValue(s.ID),
			"status":            types.StringValue(s.Status),
			"triggered_by":      types.StringValue(s.TriggeredBy),
			"hosts_discovered":  types.Int64Value(int64(s.HostsDiscovered)),
			"groups_discovered": types.Int64Value(int64(s.GroupsDiscovered)),
			"source_name":       types.StringValue(s.SourceName),
			"error":             types.StringValue(s.Error),
			"started_at":        types.StringValue(s.StartedAt),
			"finished_at":       types.StringValue(s.FinishedAt),
			"created_at":        types.StringValue(s.CreatedAt),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elems[i] = obj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: inventorySyncAttrTypes}, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Syncs = list
	config.ID = types.StringValue(inventoryID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
