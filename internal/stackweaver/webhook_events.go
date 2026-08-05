// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// WebhookEventsService lists the recent VCS webhook deliveries recorded for an
// organization (the inbound push / pull_request / ping / installation delivery
// log). It backs the stackweaver_webhook_events data source.
//
// Self-contained service: the data source constructs it via
// NewWebhookEventsService(cc.Stackweaver). The endpoint returns a JSON:API
// envelope ({"data":[{"type":"webhook-events",...}],"meta":{"pagination":{...}}})
// with dash-cased attribute keys; the raw payload is never exposed. Wire
// contract (path relative to /api/v2):
//
//	ListByOrganization: GET /organizations/:org/webhook-events?page[size]=&page[number]=
type WebhookEventsService struct {
	client *Client
}

// NewWebhookEventsService constructs the service against c.
func NewWebhookEventsService(c *Client) *WebhookEventsService {
	return &WebhookEventsService{client: c}
}

// WebhookEvent is one delivery, flattened from the JSON:API resource object.
type WebhookEvent struct {
	ID           string
	EventType    string
	Provider     string
	Repository   string
	Branch       string
	Commit       string
	Status       string
	ResponseCode int
	Message      string
	DeliveredAt  string
	ProcessedAt  string
}

// WebhookEventListOptions are the pagination options for ListByOrganization.
type WebhookEventListOptions struct {
	PageSize   int
	PageNumber int
}

// webhookEventResource mirrors the JSON:API resource object (dash-cased keys).
type webhookEventResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		EventType    string `json:"event-type"`
		Provider     string `json:"provider"`
		Repository   string `json:"repository"`
		Branch       string `json:"branch"`
		Commit       string `json:"commit"`
		Status       string `json:"status"`
		ResponseCode int    `json:"response-code"`
		Message      string `json:"message"`
		DeliveredAt  string `json:"delivered-at"`
		ProcessedAt  string `json:"processed-at"`
	} `json:"attributes"`
}

func (r *webhookEventResource) toModel() *WebhookEvent {
	return &WebhookEvent{
		ID:           r.ID,
		EventType:    r.Attributes.EventType,
		Provider:     r.Attributes.Provider,
		Repository:   r.Attributes.Repository,
		Branch:       r.Attributes.Branch,
		Commit:       r.Attributes.Commit,
		Status:       r.Attributes.Status,
		ResponseCode: r.Attributes.ResponseCode,
		Message:      r.Attributes.Message,
		DeliveredAt:  r.Attributes.DeliveredAt,
		ProcessedAt:  r.Attributes.ProcessedAt,
	}
}

// ListByOrganization returns one page of an org's webhook deliveries (newest
// first) plus meta.pagination.total-count, the full count across all pages.
func (s *WebhookEventsService) ListByOrganization(ctx context.Context, org string, options WebhookEventListOptions) ([]*WebhookEvent, int, error) {
	if org == "" {
		return nil, 0, fmt.Errorf("stackweaver: ListByOrganization requires an organization")
	}

	query := url.Values{}
	if options.PageSize > 0 {
		query.Set("page[size]", fmt.Sprintf("%d", options.PageSize))
	}
	if options.PageNumber > 0 {
		query.Set("page[number]", fmt.Sprintf("%d", options.PageNumber))
	}

	path := fmt.Sprintf("/organizations/%s/webhook-events", url.PathEscape(org))
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}

	var resources []webhookEventResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, 0, err
	}

	// The JSON:API codec only unwraps "data"; pull the pagination total separately.
	var meta struct {
		Meta struct {
			Pagination struct {
				TotalCount int `json:"total-count"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	_ = json.Unmarshal(body, &meta)

	events := make([]*WebhookEvent, len(resources))
	for i := range resources {
		events[i] = resources[i].toModel()
	}
	return events, meta.Meta.Pagination.TotalCount, nil
}
