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

// AnsibleInventorySyncsService lists the sync-run history (AWX's "inventory
// update jobs") for one Ansible inventory. It backs the
// stackweaver_ansible_inventory_syncs data source.
//
// Self-contained service: the data source constructs it via
// NewAnsibleInventorySyncsService(cc.Stackweaver). Its list endpoint is a
// JSON:API-shaped envelope ({"data":[...],"meta":{"total"}}) with dash-cased
// attribute keys. Wire contract (path relative to /api/v2):
//
//	ListByInventory: GET /ansible/inventories/:id/syncs?limit=&offset=
type AnsibleInventorySyncsService struct {
	client *Client
}

// NewAnsibleInventorySyncsService constructs the service against c.
func NewAnsibleInventorySyncsService(c *Client) *AnsibleInventorySyncsService {
	return &AnsibleInventorySyncsService{client: c}
}

// InventorySync is one sync run, flattened from the JSON:API resource object.
// The list endpoint intentionally omits the large captured Output.
type InventorySync struct {
	ID               string
	Status           string
	TriggeredBy      string
	HostsDiscovered  int
	GroupsDiscovered int
	SourceName       string
	Error            string
	StartedAt        string
	FinishedAt       string
	CreatedAt        string
}

// InventorySyncListOptions are the pagination options for ListByInventory.
type InventorySyncListOptions struct {
	Limit  int
	Offset int
}

// inventorySyncResource mirrors the JSON:API resource object (dash-cased keys).
type inventorySyncResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Status           string `json:"status"`
		TriggeredBy      string `json:"triggered-by"`
		HostsDiscovered  int    `json:"hosts-discovered"`
		GroupsDiscovered int    `json:"groups-discovered"`
		SourceName       string `json:"source-name"`
		Error            string `json:"error"`
		StartedAt        string `json:"started-at"`
		FinishedAt       string `json:"finished-at"`
		CreatedAt        string `json:"created-at"`
	} `json:"attributes"`
}

func (r *inventorySyncResource) toModel() *InventorySync {
	return &InventorySync{
		ID:               r.ID,
		Status:           r.Attributes.Status,
		TriggeredBy:      r.Attributes.TriggeredBy,
		HostsDiscovered:  r.Attributes.HostsDiscovered,
		GroupsDiscovered: r.Attributes.GroupsDiscovered,
		SourceName:       r.Attributes.SourceName,
		Error:            r.Attributes.Error,
		StartedAt:        r.Attributes.StartedAt,
		FinishedAt:       r.Attributes.FinishedAt,
		CreatedAt:        r.Attributes.CreatedAt,
	}
}

// ListByInventory returns one page of the inventory's sync history (newest
// first) plus meta.total, the full count across all pages.
func (s *AnsibleInventorySyncsService) ListByInventory(ctx context.Context, inventoryID string, options InventorySyncListOptions) ([]*InventorySync, int, error) {
	if inventoryID == "" {
		return nil, 0, fmt.Errorf("stackweaver: ListByInventory requires an inventoryID")
	}

	query := url.Values{}
	if options.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", options.Limit))
	}
	if options.Offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", options.Offset))
	}

	path := fmt.Sprintf("/ansible/inventories/%s/syncs", url.PathEscape(inventoryID))
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}

	var resources []inventorySyncResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, 0, err
	}

	// The JSON:API codec only unwraps "data"; pull meta.total separately.
	var meta struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_ = json.Unmarshal(body, &meta)

	syncs := make([]*InventorySync, len(resources))
	for i := range resources {
		syncs[i] = resources[i].toModel()
	}
	return syncs, meta.Meta.Total, nil
}
