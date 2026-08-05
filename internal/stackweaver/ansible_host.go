// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// hostResourceType is the JSON:API `type` member for a host, matching the backend
// handler (handlers/ansible/hosts.go formatHostResponse).
const hostResourceType = "ansible-hosts"

// AnsibleHostsService is the native service for hosts inside an Ansible inventory.
// Self-contained (constructed via NewAnsibleHostsService) so registration is wired
// centrally without editing client.go. Wire contract (paths relative to /api/v2):
//
//	Create: POST   /ansible/inventories/:inventory_id/hosts
//	Read:   GET    /ansible/hosts/:id
//	Update: PATCH  /ansible/hosts/:id
//	Delete: DELETE /ansible/hosts/:id
//
// The envelope is JSON:API-style with snake_case attribute keys and no
// relationships block; the parent inventory is taken from the create URL path.
type AnsibleHostsService struct {
	client *Client
}

// NewAnsibleHostsService builds the hosts service around an existing native
// Client.
func NewAnsibleHostsService(c *Client) *AnsibleHostsService {
	return &AnsibleHostsService{client: c}
}

// AnsibleHost is the native representation of an inventory host, flattened from
// the JSON:API resource object into the shape the Terraform resource consumes.
type AnsibleHost struct {
	ID          string
	InventoryID string
	Name        string
	Description string
	Hostname    string
	Port        int64
	Variables   map[string]string
	Enabled     bool
	CreatedAt   string
	UpdatedAt   string
}

// AnsibleHostCreateOptions are the fields accepted when creating a host.
// InventoryID scopes the create endpoint (URL path), the rest are attributes.
type AnsibleHostCreateOptions struct {
	InventoryID string
	Name        string
	Description string
	Hostname    string
	Port        int64
	Variables   map[string]string
	// Enabled is a pointer so an omitted value defers to the server default
	// (true) rather than forcing false.
	Enabled *bool
}

// AnsibleHostUpdateOptions are the fields accepted when updating a host. A nil
// pointer leaves the corresponding field unchanged.
type AnsibleHostUpdateOptions struct {
	Name        *string
	Description *string
	Hostname    *string
	Port        *int64
	Variables   map[string]string
	Enabled     *bool
}

// hostResource mirrors the JSON:API resource object the backend returns.
type hostResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Hostname    string                 `json:"hostname"`
		Port        int64                  `json:"port"`
		Variables   map[string]interface{} `json:"variables"`
		Enabled     bool                   `json:"enabled"`
		CreatedAt   string                 `json:"created-at"`
		UpdatedAt   string                 `json:"updated-at"`
	} `json:"attributes"`
	Relationships struct {
		Inventory jsonAPIRelationship `json:"inventory"`
	} `json:"relationships"`
}

// toModel flattens the wire resource into the public AnsibleHost.
func (r *hostResource) toModel() *AnsibleHost {
	h := &AnsibleHost{
		ID:          r.ID,
		Name:        r.Attributes.Name,
		Description: r.Attributes.Description,
		Hostname:    r.Attributes.Hostname,
		Port:        r.Attributes.Port,
		Variables:   stringifyVars(r.Attributes.Variables),
		Enabled:     r.Attributes.Enabled,
		CreatedAt:   r.Attributes.CreatedAt,
		UpdatedAt:   r.Attributes.UpdatedAt,
	}
	if r.Relationships.Inventory.Data != nil {
		h.InventoryID = r.Relationships.Inventory.Data.ID
	}
	return h
}

// Create creates a new host under options.InventoryID.
func (s *AnsibleHostsService) Create(ctx context.Context, options AnsibleHostCreateOptions) (*AnsibleHost, error) {
	if options.InventoryID == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an InventoryID")
	}

	attributes := map[string]any{
		"name": options.Name,
	}
	if options.Description != "" {
		attributes["description"] = options.Description
	}
	if options.Hostname != "" {
		attributes["hostname"] = options.Hostname
	}
	if options.Port != 0 {
		attributes["port"] = options.Port
	}
	if options.Variables != nil {
		attributes["variables"] = options.Variables
	}
	if options.Enabled != nil {
		attributes["enabled"] = *options.Enabled
	}

	reqBody, err := s.client.jsonapi.Marshal(hostResourceType, attributes)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/inventories/%s/hosts", url.PathEscape(options.InventoryID))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Read returns the host identified by id. A missing host surfaces as ErrNotFound.
func (s *AnsibleHostsService) Read(ctx context.Context, id string) (*AnsibleHost, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}

	path := fmt.Sprintf("/ansible/hosts/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// Update updates the host identified by id.
func (s *AnsibleHostsService) Update(ctx context.Context, id string, options AnsibleHostUpdateOptions) (*AnsibleHost, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Update requires an id")
	}

	attributes := map[string]any{}
	if options.Name != nil {
		attributes["name"] = *options.Name
	}
	if options.Description != nil {
		attributes["description"] = *options.Description
	}
	if options.Hostname != nil {
		attributes["hostname"] = *options.Hostname
	}
	if options.Port != nil {
		attributes["port"] = *options.Port
	}
	if options.Variables != nil {
		attributes["variables"] = options.Variables
	}
	if options.Enabled != nil {
		attributes["enabled"] = *options.Enabled
	}

	reqBody, err := s.client.jsonapi.Marshal(hostResourceType, attributes)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/hosts/%s", url.PathEscape(id))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Delete removes the host identified by id.
func (s *AnsibleHostsService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}

	path := fmt.Sprintf("/ansible/hosts/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// decode unmarshals a single-host JSON:API response into the public model.
func (s *AnsibleHostsService) decode(body []byte) (*AnsibleHost, error) {
	var resource hostResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}
