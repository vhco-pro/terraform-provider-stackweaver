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

// Native data source — no terraform-provider-tfe equivalent. Audit/debug helper
// that lists recent VCS webhook deliveries recorded for an organization. Served
// by the native internal/stackweaver client (JSON:API, dash-cased keys); the raw
// webhook payload is never exposed. Registered only under its primary
// stackweaver_webhook_events name (native = no tfe_ alias).
var (
	_ datasource.DataSource              = &dataSourceStackweaverWebhookEvents{}
	_ datasource.DataSourceWithConfigure = &dataSourceStackweaverWebhookEvents{}
)

// NewWebhookEventsDataSource is the factory for the data source.
func NewWebhookEventsDataSource() datasource.DataSource {
	return &dataSourceStackweaverWebhookEvents{}
}

type dataSourceStackweaverWebhookEvents struct {
	config ConfiguredClient
}

type modelDataSourceWebhookEvents struct {
	Organization types.String `tfsdk:"organization"`
	ID           types.String `tfsdk:"id"`
	Events       types.List   `tfsdk:"events"`
}

// webhookEventAttrTypes is the object type of an events element.
var webhookEventAttrTypes = map[string]attr.Type{
	"id":            types.StringType,
	"event_type":    types.StringType,
	"provider":      types.StringType,
	"repository":    types.StringType,
	"branch":        types.StringType,
	"commit":        types.StringType,
	"status":        types.StringType,
	"response_code": types.Int64Type,
	"message":       types.StringType,
	"delivered_at":  types.StringType,
	"processed_at":  types.StringType,
}

func (d *dataSourceStackweaverWebhookEvents) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_events"
}

func (d *dataSourceStackweaverWebhookEvents) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists recent VCS webhook deliveries recorded for an organization, newest first. The raw webhook payload is never exposed.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description: "Name of the organization. Defaults to the provider's organization.",
				Optional:    true,
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "Set to the organization name.",
				Computed:    true,
			},
			"events": schema.ListNestedAttribute{
				Description: "Recent webhook deliveries, newest first.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Delivery ID.",
							Computed:    true,
						},
						"event_type": schema.StringAttribute{
							Description: "Event type: push | pull_request | ping | installation | ...",
							Computed:    true,
						},
						"provider": schema.StringAttribute{
							Description: "VCS provider: github | gitlab | ...",
							Computed:    true,
						},
						"repository": schema.StringAttribute{
							Description: "Repository in \"owner/repo\" format, when applicable.",
							Computed:    true,
						},
						"branch": schema.StringAttribute{
							Description: "Branch, when applicable.",
							Computed:    true,
						},
						"commit": schema.StringAttribute{
							Description: "Commit SHA, when applicable.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Delivery status: success | failed | ignored.",
							Computed:    true,
						},
						"response_code": schema.Int64Attribute{
							Description: "HTTP code Stackweaver returned.",
							Computed:    true,
						},
						"message": schema.StringAttribute{
							Description: "Additional info / error message.",
							Computed:    true,
						},
						"delivered_at": schema.StringAttribute{
							Description: "RFC3339 receipt time.",
							Computed:    true,
						},
						"processed_at": schema.StringAttribute{
							Description: "RFC3339 processing time; null until processed.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *dataSourceStackweaverWebhookEvents) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dataSourceStackweaverWebhookEvents) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelDataSourceWebhookEvents
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.config.Stackweaver == nil {
		resp.Diagnostics.AddError("Provider not configured", "The provider must be configured with a hostname and token to read stackweaver_webhook_events.")
		return
	}

	var organization string
	resp.Diagnostics.Append(d.config.dataOrDefaultOrganization(ctx, req.Config, &organization)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Organization = types.StringValue(organization)

	svc := stackweaver.NewWebhookEventsService(d.config.Stackweaver)

	// Page through the delivery log so the data source surfaces the full set.
	const pageSize = 100
	var events []*stackweaver.WebhookEvent
	for pageNumber := 1; ; pageNumber++ {
		tflog.Debug(ctx, fmt.Sprintf("Read webhook events for %s (page %d)", organization, pageNumber))
		page, total, err := svc.ListByOrganization(ctx, organization, stackweaver.WebhookEventListOptions{PageSize: pageSize, PageNumber: pageNumber})
		if err != nil {
			if errors.Is(err, stackweaver.ErrNotFound) {
				resp.Diagnostics.AddError("Organization not found", err.Error())
				return
			}
			resp.Diagnostics.AddError("Error listing webhook events", err.Error())
			return
		}
		events = append(events, page...)
		if len(page) == 0 || len(events) >= total {
			break
		}
	}

	elems := make([]attr.Value, len(events))
	for i, e := range events {
		obj, diags := types.ObjectValue(webhookEventAttrTypes, map[string]attr.Value{
			"id":            types.StringValue(e.ID),
			"event_type":    types.StringValue(e.EventType),
			"provider":      types.StringValue(e.Provider),
			"repository":    types.StringValue(e.Repository),
			"branch":        types.StringValue(e.Branch),
			"commit":        types.StringValue(e.Commit),
			"status":        types.StringValue(e.Status),
			"response_code": types.Int64Value(int64(e.ResponseCode)),
			"message":       types.StringValue(e.Message),
			"delivered_at":  types.StringValue(e.DeliveredAt),
			"processed_at":  types.StringValue(e.ProcessedAt),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elems[i] = obj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: webhookEventAttrTypes}, elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Events = list
	config.ID = types.StringValue(organization)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
