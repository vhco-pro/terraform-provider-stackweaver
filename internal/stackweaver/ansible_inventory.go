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

// inventoryResourceType is the JSON:API `type` member for an inventory, matching
// the backend handler (handlers/ansible/inventories.go formatInventoryResponse).
const inventoryResourceType = "ansible-inventories"

// AnsibleInventoriesService is the native service for Ansible inventories. Unlike
// the reference AnsiblePlaybooksService it is self-contained (constructed via
// NewAnsibleInventoriesService) rather than a field on Client, so registration is
// wired centrally without editing client.go. Wire contract (paths relative to
// /api/v2):
//
//	Create: POST   /organizations/:org/ansible/inventories
//	Read:   GET    /ansible/inventories/:id
//	Update: PATCH  /ansible/inventories/:id
//	Delete: DELETE /ansible/inventories/:id
//	Sync:   POST   /ansible/inventories/:id/actions/sync
//	List:   GET    /organizations/:org/ansible/inventories
//
// The envelope is JSON:API-style with mixed snake/kebab attribute keys
// (inventory-type, source-vars, constructed-*, input-inventory-ids); project and
// vcs_connection are relationships.
type AnsibleInventoriesService struct {
	client *Client
}

// NewAnsibleInventoriesService builds the inventories service around an existing
// native Client.
func NewAnsibleInventoriesService(c *Client) *AnsibleInventoriesService {
	return &AnsibleInventoriesService{client: c}
}

// AnsibleInventory is the native representation of an Ansible inventory, flattened
// from the JSON:API resource object into the shape the Terraform resource
// consumes. Fields the backend never echoes (organization name, project id) are
// intentionally absent — the resource preserves those from prior state/plan.
type AnsibleInventory struct {
	ID                      string
	Name                    string
	Description             string
	Type                    string
	Source                  string
	Variables               map[string]string
	VCSConnectionID         string
	VCSRepository           string
	VCSBranch               string
	InventoryPath           string
	SourceVars              string
	ConstructedLimit        string
	ConstructedCacheTimeout int64
	// InputInventoryIDs is populated only when the response carries the
	// input-inventories member (constructed inventories via GET). It is nil
	// otherwise, so the resource can distinguish "not reported" from "empty".
	InputInventoryIDs       []string
	LastSyncAt              string
	LastSyncStatus          string
	LastSyncError           string
	LastSyncHostsDiscovered int64
	LastSyncLog             string
	CreatedAt               string
	UpdatedAt               string
}

// AnsibleInventoryCreateOptions are the fields accepted when creating an
// inventory. Organization scopes the create endpoint; ProjectID/VCSConnectionID
// are sent as JSON:API relationships, the rest as attributes.
type AnsibleInventoryCreateOptions struct {
	Organization            string
	ProjectID               string
	Name                    string
	Description             string
	Type                    string
	Source                  string
	Variables               map[string]string
	VCSConnectionID         string
	VCSRepository           string
	VCSBranch               string
	InventoryPath           string
	SourceVars              string
	ConstructedLimit        string
	ConstructedCacheTimeout int64
	InputInventoryIDs       []string
}

// AnsibleInventoryUpdateOptions are the fields accepted when updating an
// inventory. A nil pointer leaves the corresponding field unchanged; note that
// inventory-type is intentionally absent — the backend has no update path for it,
// so type is ForceNew.
type AnsibleInventoryUpdateOptions struct {
	Name                    *string
	Description             *string
	Source                  *string
	Variables               map[string]string
	VCSConnectionID         *string
	VCSRepository           *string
	VCSBranch               *string
	InventoryPath           *string
	SourceVars              *string
	ConstructedLimit        *string
	ConstructedCacheTimeout *int64
	InputInventoryIDs       []string
}

// inventoryResource mirrors the JSON:API resource object the backend returns.
type inventoryResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name                    string                 `json:"name"`
		Description             string                 `json:"description"`
		Type                    string                 `json:"inventory-type"`
		Source                  string                 `json:"source"`
		Variables               map[string]interface{} `json:"variables"`
		VCSConnectionID         string                 `json:"vcs_connection_id"`
		VCSRepository           string                 `json:"vcs_repository"`
		VCSBranch               string                 `json:"vcs_branch"`
		InventoryPath           string                 `json:"inventory_path"`
		SourceVars              string                 `json:"source-vars"`
		ConstructedLimit        string                 `json:"constructed-limit"`
		ConstructedCacheTimeout int64                  `json:"constructed-cache-timeout"`
		LastSyncAt              string                 `json:"last-sync-at"`
		LastSyncStatus          string                 `json:"last-sync-status"`
		LastSyncError           string                 `json:"last-sync-error"`
		LastSyncHostsDiscovered int64                  `json:"last-sync-hosts-discovered"`
		LastSyncLog             string                 `json:"last-sync-log"`
		CreatedAt               string                 `json:"created-at"`
		UpdatedAt               string                 `json:"updated-at"`
		InputInventories        []struct {
			ID string `json:"id"`
		} `json:"input-inventories"`
	} `json:"attributes"`
}

// toModel flattens the wire resource into the public AnsibleInventory.
func (r *inventoryResource) toModel() *AnsibleInventory {
	inv := &AnsibleInventory{
		ID:                      r.ID,
		Name:                    r.Attributes.Name,
		Description:             r.Attributes.Description,
		Type:                    r.Attributes.Type,
		Source:                  r.Attributes.Source,
		Variables:               stringifyVars(r.Attributes.Variables),
		VCSConnectionID:         r.Attributes.VCSConnectionID,
		VCSRepository:           r.Attributes.VCSRepository,
		VCSBranch:               r.Attributes.VCSBranch,
		InventoryPath:           r.Attributes.InventoryPath,
		SourceVars:              r.Attributes.SourceVars,
		ConstructedLimit:        r.Attributes.ConstructedLimit,
		ConstructedCacheTimeout: r.Attributes.ConstructedCacheTimeout,
		LastSyncAt:              r.Attributes.LastSyncAt,
		LastSyncStatus:          r.Attributes.LastSyncStatus,
		LastSyncError:           r.Attributes.LastSyncError,
		LastSyncHostsDiscovered: r.Attributes.LastSyncHostsDiscovered,
		LastSyncLog:             r.Attributes.LastSyncLog,
		CreatedAt:               r.Attributes.CreatedAt,
		UpdatedAt:               r.Attributes.UpdatedAt,
	}
	if r.Attributes.InputInventories != nil {
		ids := make([]string, len(r.Attributes.InputInventories))
		for i := range r.Attributes.InputInventories {
			ids[i] = r.Attributes.InputInventories[i].ID
		}
		inv.InputInventoryIDs = ids
	}
	return inv
}

// stringifyVars flattens a jsonb variables object (arbitrary JSON values) into a
// map[string]string: string values pass through verbatim, everything else is
// re-encoded as its JSON literal. A nil input yields a non-nil empty map so the
// resource settles cleanly to {} instead of null.
func stringifyVars(in map[string]interface{}) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if s, ok := v.(string); ok {
			out[k] = s
			continue
		}
		if raw, err := json.Marshal(v); err == nil {
			out[k] = string(raw)
		}
	}
	return out
}

// Create creates a new inventory under options.Organization.
func (s *AnsibleInventoriesService) Create(ctx context.Context, options AnsibleInventoryCreateOptions) (*AnsibleInventory, error) {
	if options.Organization == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an Organization")
	}

	attributes := map[string]any{
		"name": options.Name,
	}
	if options.Description != "" {
		attributes["description"] = options.Description
	}
	if options.Type != "" {
		attributes["inventory-type"] = options.Type
	}
	if options.Source != "" {
		attributes["source"] = options.Source
	}
	if options.Variables != nil {
		attributes["variables"] = options.Variables
	}
	if options.VCSRepository != "" {
		attributes["vcs_repository"] = options.VCSRepository
	}
	if options.VCSBranch != "" {
		attributes["vcs_branch"] = options.VCSBranch
	}
	if options.InventoryPath != "" {
		attributes["inventory_path"] = options.InventoryPath
	}
	if options.SourceVars != "" {
		attributes["source-vars"] = options.SourceVars
	}
	if options.ConstructedLimit != "" {
		attributes["constructed-limit"] = options.ConstructedLimit
	}
	if options.ConstructedCacheTimeout != 0 {
		attributes["constructed-cache-timeout"] = options.ConstructedCacheTimeout
	}
	if len(options.InputInventoryIDs) > 0 {
		attributes["input-inventory-ids"] = options.InputInventoryIDs
	}

	relationships := map[string]jsonAPIRelationship{}
	if options.ProjectID != "" {
		relationships["project"] = jsonAPIRelationship{
			Data: &jsonAPIResourceRef{Type: "projects", ID: options.ProjectID},
		}
	}
	if options.VCSConnectionID != "" {
		relationships["vcs_connection"] = jsonAPIRelationship{
			Data: &jsonAPIResourceRef{Type: "vcs-connections", ID: options.VCSConnectionID},
		}
	}

	reqBody, err := s.client.plain.Marshal(inventoryRequest{
		Data: inventoryRequestData{
			Type:          inventoryResourceType,
			Attributes:    attributes,
			Relationships: relationships,
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/organizations/%s/ansible/inventories", url.PathEscape(options.Organization))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Read returns the inventory identified by id. A missing inventory surfaces as
// ErrNotFound.
func (s *AnsibleInventoriesService) Read(ctx context.Context, id string) (*AnsibleInventory, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}

	path := fmt.Sprintf("/ansible/inventories/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// Update updates the inventory identified by id.
func (s *AnsibleInventoriesService) Update(ctx context.Context, id string, options AnsibleInventoryUpdateOptions) (*AnsibleInventory, error) {
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
	if options.Source != nil {
		attributes["source"] = *options.Source
	}
	if options.Variables != nil {
		attributes["variables"] = options.Variables
	}
	if options.VCSRepository != nil {
		attributes["vcs_repository"] = *options.VCSRepository
	}
	if options.VCSBranch != nil {
		attributes["vcs_branch"] = *options.VCSBranch
	}
	if options.InventoryPath != nil {
		attributes["inventory_path"] = *options.InventoryPath
	}
	if options.SourceVars != nil {
		attributes["source-vars"] = *options.SourceVars
	}
	if options.ConstructedLimit != nil {
		attributes["constructed-limit"] = *options.ConstructedLimit
	}
	if options.ConstructedCacheTimeout != nil {
		attributes["constructed-cache-timeout"] = *options.ConstructedCacheTimeout
	}
	if options.InputInventoryIDs != nil {
		attributes["input-inventory-ids"] = options.InputInventoryIDs
	}

	data := inventoryRequestData{
		Type:       inventoryResourceType,
		Attributes: attributes,
	}
	if options.VCSConnectionID != nil && *options.VCSConnectionID != "" {
		data.Relationships = map[string]jsonAPIRelationship{
			"vcs_connection": {Data: &jsonAPIResourceRef{Type: "vcs-connections", ID: *options.VCSConnectionID}},
		}
	}

	reqBody, err := s.client.plain.Marshal(inventoryRequest{Data: data})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/inventories/%s", url.PathEscape(id))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Delete removes the inventory identified by id.
func (s *AnsibleInventoriesService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}

	path := fmt.Sprintf("/ansible/inventories/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// Sync triggers a refresh of a dynamic/VCS/constructed inventory (POST
// /ansible/inventories/:id/actions/sync).
func (s *AnsibleInventoriesService) Sync(ctx context.Context, id string) (*AnsibleInventory, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Sync requires an id")
	}

	path := fmt.Sprintf("/ansible/inventories/%s/actions/sync", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// decode unmarshals a single-inventory JSON:API response into the public model.
func (s *AnsibleInventoriesService) decode(body []byte) (*AnsibleInventory, error) {
	var resource inventoryResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}

// inventoryRequest is the JSON:API request envelope for create/update.
type inventoryRequest struct {
	Data inventoryRequestData `json:"data"`
}

type inventoryRequestData struct {
	Type          string                         `json:"type"`
	Attributes    map[string]any                 `json:"attributes"`
	Relationships map[string]jsonAPIRelationship `json:"relationships,omitempty"`
}
