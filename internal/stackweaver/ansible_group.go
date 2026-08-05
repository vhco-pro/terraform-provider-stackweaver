// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// groupResourceType is the JSON:API `type` member for a group, matching the
// backend handler (handlers/ansible/groups.go formatGroupResponse).
const groupResourceType = "ansible-groups"

// AnsibleGroupsService is the native service for groups inside an Ansible
// inventory. Self-contained (constructed via NewAnsibleGroupsService) so
// registration is wired centrally without editing client.go. Wire contract (paths
// relative to /api/v2):
//
//	Create: POST   /ansible/inventories/:inventory_id/groups
//	Read:   GET    /ansible/groups/:id
//	Update: PATCH  /ansible/groups/:id
//	Delete: DELETE /ansible/groups/:id
//
// The envelope is JSON:API-style with snake_case attribute keys; the parent
// inventory is taken from the create URL path and the nested-group parent is a
// `parent` relationship.
type AnsibleGroupsService struct {
	client *Client
}

// NewAnsibleGroupsService builds the groups service around an existing native
// Client.
func NewAnsibleGroupsService(c *Client) *AnsibleGroupsService {
	return &AnsibleGroupsService{client: c}
}

// AnsibleGroup is the native representation of an inventory group, flattened from
// the JSON:API resource object into the shape the Terraform resource consumes.
type AnsibleGroup struct {
	ID          string
	InventoryID string
	ParentID    string
	Name        string
	Description string
	Variables   map[string]string
	CreatedAt   string
	UpdatedAt   string
}

// AnsibleGroupCreateOptions are the fields accepted when creating a group.
// InventoryID scopes the create endpoint (URL path); ParentID is sent as a
// JSON:API relationship, the rest as attributes.
type AnsibleGroupCreateOptions struct {
	InventoryID string
	Name        string
	Description string
	Variables   map[string]string
	ParentID    string
}

// AnsibleGroupUpdateOptions are the fields accepted when updating a group. A nil
// pointer leaves the corresponding field unchanged.
type AnsibleGroupUpdateOptions struct {
	Name        *string
	Description *string
	Variables   map[string]string
	ParentID    *string
}

// groupResource mirrors the JSON:API resource object the backend returns.
type groupResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Variables   map[string]interface{} `json:"variables"`
		CreatedAt   string                 `json:"created-at"`
		UpdatedAt   string                 `json:"updated-at"`
	} `json:"attributes"`
	Relationships struct {
		Inventory jsonAPIRelationship `json:"inventory"`
		Parent    jsonAPIRelationship `json:"parent"`
	} `json:"relationships"`
}

// toModel flattens the wire resource into the public AnsibleGroup.
func (r *groupResource) toModel() *AnsibleGroup {
	g := &AnsibleGroup{
		ID:          r.ID,
		Name:        r.Attributes.Name,
		Description: r.Attributes.Description,
		Variables:   stringifyVars(r.Attributes.Variables),
		CreatedAt:   r.Attributes.CreatedAt,
		UpdatedAt:   r.Attributes.UpdatedAt,
	}
	if r.Relationships.Inventory.Data != nil {
		g.InventoryID = r.Relationships.Inventory.Data.ID
	}
	if r.Relationships.Parent.Data != nil {
		g.ParentID = r.Relationships.Parent.Data.ID
	}
	return g
}

// Create creates a new group under options.InventoryID.
func (s *AnsibleGroupsService) Create(ctx context.Context, options AnsibleGroupCreateOptions) (*AnsibleGroup, error) {
	if options.InventoryID == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an InventoryID")
	}

	attributes := map[string]any{
		"name": options.Name,
	}
	if options.Description != "" {
		attributes["description"] = options.Description
	}
	if options.Variables != nil {
		attributes["variables"] = options.Variables
	}

	data := groupRequestData{
		Type:       groupResourceType,
		Attributes: attributes,
	}
	if options.ParentID != "" {
		data.Relationships = map[string]jsonAPIRelationship{
			"parent": {Data: &jsonAPIResourceRef{Type: groupResourceType, ID: options.ParentID}},
		}
	}

	reqBody, err := s.client.plain.Marshal(groupRequest{Data: data})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/inventories/%s/groups", url.PathEscape(options.InventoryID))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Read returns the group identified by id. A missing group surfaces as
// ErrNotFound.
func (s *AnsibleGroupsService) Read(ctx context.Context, id string) (*AnsibleGroup, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}

	path := fmt.Sprintf("/ansible/groups/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// Update updates the group identified by id. Note the backend never clears an
// existing parent link (its handler hardcodes clearParent=false and exposes no
// signal to detach), so a set parent can be re-pointed but not removed in place.
func (s *AnsibleGroupsService) Update(ctx context.Context, id string, options AnsibleGroupUpdateOptions) (*AnsibleGroup, error) {
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
	if options.Variables != nil {
		attributes["variables"] = options.Variables
	}

	data := groupRequestData{
		Type:       groupResourceType,
		Attributes: attributes,
	}
	if options.ParentID != nil && *options.ParentID != "" {
		data.Relationships = map[string]jsonAPIRelationship{
			"parent": {Data: &jsonAPIResourceRef{Type: groupResourceType, ID: *options.ParentID}},
		}
	}

	reqBody, err := s.client.plain.Marshal(groupRequest{Data: data})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/groups/%s", url.PathEscape(id))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Delete removes the group identified by id.
func (s *AnsibleGroupsService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}

	path := fmt.Sprintf("/ansible/groups/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// decode unmarshals a single-group JSON:API response into the public model.
func (s *AnsibleGroupsService) decode(body []byte) (*AnsibleGroup, error) {
	var resource groupResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}

// groupRequest is the JSON:API request envelope for create/update.
type groupRequest struct {
	Data groupRequestData `json:"data"`
}

type groupRequestData struct {
	Type          string                         `json:"type"`
	Attributes    map[string]any                 `json:"attributes"`
	Relationships map[string]jsonAPIRelationship `json:"relationships,omitempty"`
}
