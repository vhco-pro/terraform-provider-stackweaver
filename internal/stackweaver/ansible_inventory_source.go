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

// inventorySourceResourceType is the JSON:API `type` member for an inventory
// source, matching the backend handler
// (handlers/ansible/inventory_sources.go formatInventorySourceResponse).
const inventorySourceResourceType = "inventory-sources"

// AnsibleInventorySourcesService is the native service for dynamic inventory
// sources. The endpoint uses the JSON:API envelope with kebab-case attribute
// keys. Note the asymmetry: credential_id is a request *attribute*
// (`credential-id`) but a response *relationship* (`credential.data.id`). Wire
// contract (all paths relative to /api/v2):
//
//	Create: POST   /ansible/inventories/:id/sources   (inventory id from the path)
//	Read:   GET    /ansible/inventory-sources/:source_id
//	Update: PATCH  /ansible/inventory-sources/:source_id
//	Delete: DELETE /ansible/inventory-sources/:source_id
//	Sync:   POST   /ansible/inventory-sources/:source_id/actions/sync
//
// This service is self-contained (built inline via
// NewAnsibleInventorySourcesService).
type AnsibleInventorySourcesService struct {
	client *Client
}

// NewAnsibleInventorySourcesService builds an inventory-sources service over c.
func NewAnsibleInventorySourcesService(c *Client) *AnsibleInventorySourcesService {
	return &AnsibleInventorySourcesService{client: c}
}

// AnsibleInventorySource is the native representation of an inventory source,
// flattened from the JSON:API resource object.
type AnsibleInventorySource struct {
	ID                      string
	InventoryID             string
	Name                    string
	Description             string
	Type                    string
	CredentialID            string
	Config                  string
	UpdateOnLaunch          bool
	UpdateCacheTimeout      int64
	Overwrite               bool
	OverwriteVars           bool
	Verbosity               int64
	GroupByInstanceID       bool
	GroupByRegion           bool
	GroupByAvailabilityZone bool
	GroupByTag              string
	HostnameVar             string
	InstanceFilters         string
	SyncSchedule            string
	Status                  string
	LastSyncAt              string
	LastSyncError           string
	LastSyncLog             string
	HostsCount              int64
	Enabled                 bool
}

// AnsibleInventorySourceCreateOptions are the fields accepted when creating a
// source. InventoryID scopes the create path (it is not in the body). Bool/int
// sync-behavior fields are pointers so an unset field falls through to the
// server default instead of being forced to its zero value.
type AnsibleInventorySourceCreateOptions struct {
	InventoryID             string
	Name                    string
	Description             string
	Type                    string
	CredentialID            string
	Config                  string
	SyncSchedule            string
	GroupByTag              string
	HostnameVar             string
	InstanceFilters         string
	UpdateOnLaunch          *bool
	UpdateCacheTimeout      *int64
	Overwrite               *bool
	OverwriteVars           *bool
	Verbosity               *int64
	GroupByInstanceID       *bool
	GroupByRegion           *bool
	GroupByAvailabilityZone *bool
	Enabled                 *bool
}

// AnsibleInventorySourceUpdateOptions are the fields accepted when updating a
// source. ClearCredential sends `credential-id: ""` to detach the credential
// (switch to OIDC); otherwise CredentialID is set when non-empty. source-type
// is not applied by the update handler and is therefore ForceNew.
type AnsibleInventorySourceUpdateOptions struct {
	Name                    string
	Description             string
	Config                  string
	CredentialID            string
	ClearCredential         bool
	SyncSchedule            string
	GroupByTag              string
	HostnameVar             string
	InstanceFilters         string
	UpdateOnLaunch          *bool
	UpdateCacheTimeout      *int64
	Overwrite               *bool
	OverwriteVars           *bool
	Verbosity               *int64
	GroupByInstanceID       *bool
	GroupByRegion           *bool
	GroupByAvailabilityZone *bool
	Enabled                 *bool
}

// inventorySourceResource mirrors the JSON:API resource object the backend
// returns.
type inventorySourceResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name                    string          `json:"name"`
		Description             string          `json:"description"`
		SourceType              string          `json:"source-type"`
		Config                  json.RawMessage `json:"config"`
		UpdateOnLaunch          bool            `json:"update-on-launch"`
		UpdateCacheTimeout      int64           `json:"update-cache-timeout"`
		Overwrite               bool            `json:"overwrite"`
		OverwriteVars           bool            `json:"overwrite-vars"`
		Verbosity               int64           `json:"verbosity"`
		GroupByInstanceID       bool            `json:"group-by-instance-id"`
		GroupByRegion           bool            `json:"group-by-region"`
		GroupByAvailabilityZone bool            `json:"group-by-availability-zone"`
		GroupByTag              string          `json:"group-by-tag"`
		HostnameVar             string          `json:"hostname-var"`
		InstanceFilters         string          `json:"instance-filters"`
		SyncSchedule            string          `json:"sync-schedule"`
		Status                  string          `json:"status"`
		LastSyncAt              *string         `json:"last-sync-at"`
		LastSyncError           string          `json:"last-sync-error"`
		LastSyncLog             string          `json:"last-sync-log"`
		HostsCount              int64           `json:"hosts-count"`
		Enabled                 bool            `json:"enabled"`
	} `json:"attributes"`
	Relationships struct {
		Inventory  jsonAPIRelationship `json:"inventory"`
		Credential jsonAPIRelationship `json:"credential"`
	} `json:"relationships"`
}

// toModel flattens the wire resource into the public AnsibleInventorySource. The
// config JSON is normalized (compacted, keys sorted) so key-order/whitespace
// changes from the server do not surface as a perpetual diff.
func (r *inventorySourceResource) toModel() *AnsibleInventorySource {
	src := &AnsibleInventorySource{
		ID:                      r.ID,
		Name:                    r.Attributes.Name,
		Description:             r.Attributes.Description,
		Type:                    r.Attributes.SourceType,
		Config:                  normalizeJSON(r.Attributes.Config),
		UpdateOnLaunch:          r.Attributes.UpdateOnLaunch,
		UpdateCacheTimeout:      r.Attributes.UpdateCacheTimeout,
		Overwrite:               r.Attributes.Overwrite,
		OverwriteVars:           r.Attributes.OverwriteVars,
		Verbosity:               r.Attributes.Verbosity,
		GroupByInstanceID:       r.Attributes.GroupByInstanceID,
		GroupByRegion:           r.Attributes.GroupByRegion,
		GroupByAvailabilityZone: r.Attributes.GroupByAvailabilityZone,
		GroupByTag:              r.Attributes.GroupByTag,
		HostnameVar:             r.Attributes.HostnameVar,
		InstanceFilters:         r.Attributes.InstanceFilters,
		SyncSchedule:            r.Attributes.SyncSchedule,
		Status:                  r.Attributes.Status,
		LastSyncError:           r.Attributes.LastSyncError,
		LastSyncLog:             r.Attributes.LastSyncLog,
		HostsCount:              r.Attributes.HostsCount,
		Enabled:                 r.Attributes.Enabled,
	}
	if r.Attributes.LastSyncAt != nil {
		src.LastSyncAt = *r.Attributes.LastSyncAt
	}
	if r.Relationships.Inventory.Data != nil {
		src.InventoryID = r.Relationships.Inventory.Data.ID
	}
	if r.Relationships.Credential.Data != nil {
		src.CredentialID = r.Relationships.Credential.Data.ID
	}
	return src
}

// normalizeJSON compacts raw and sorts object keys (Go marshals map keys in
// sorted order) so equivalent JSON always renders identically. It returns "{}"
// for empty/invalid input.
func normalizeJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "{}"
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(out)
}

// Create creates a new source under the inventory options.InventoryID.
func (s *AnsibleInventorySourcesService) Create(ctx context.Context, options AnsibleInventorySourceCreateOptions) (*AnsibleInventorySource, error) {
	if options.InventoryID == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an InventoryID")
	}

	attributes := map[string]any{
		"name":        options.Name,
		"source-type": options.Type,
	}
	setIfNotEmpty(attributes, "description", options.Description)
	setIfNotEmpty(attributes, "credential-id", options.CredentialID)
	setIfNotEmpty(attributes, "sync-schedule", options.SyncSchedule)
	setIfNotEmpty(attributes, "group-by-tag", options.GroupByTag)
	setIfNotEmpty(attributes, "hostname-var", options.HostnameVar)
	setIfNotEmpty(attributes, "instance-filters", options.InstanceFilters)
	if options.Config != "" {
		attributes["config"] = json.RawMessage(options.Config)
	}
	setBoolPtr(attributes, "update-on-launch", options.UpdateOnLaunch)
	setInt64Ptr(attributes, "update-cache-timeout", options.UpdateCacheTimeout)
	setBoolPtr(attributes, "overwrite", options.Overwrite)
	setBoolPtr(attributes, "overwrite-vars", options.OverwriteVars)
	setInt64Ptr(attributes, "verbosity", options.Verbosity)
	setBoolPtr(attributes, "group-by-instance-id", options.GroupByInstanceID)
	setBoolPtr(attributes, "group-by-region", options.GroupByRegion)
	setBoolPtr(attributes, "group-by-availability-zone", options.GroupByAvailabilityZone)
	setBoolPtr(attributes, "enabled", options.Enabled)

	reqBody, err := s.client.plain.Marshal(jsonAPIRequest{
		Data: jsonAPIRequestData{
			Type:       inventorySourceResourceType,
			Attributes: attributes,
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/inventories/%s/sources", url.PathEscape(options.InventoryID))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Read returns the source identified by id. A missing source surfaces as
// ErrNotFound.
func (s *AnsibleInventorySourcesService) Read(ctx context.Context, id string) (*AnsibleInventorySource, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}

	path := fmt.Sprintf("/ansible/inventory-sources/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// Update updates the source identified by id.
func (s *AnsibleInventorySourcesService) Update(ctx context.Context, id string, options AnsibleInventorySourceUpdateOptions) (*AnsibleInventorySource, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Update requires an id")
	}

	attributes := map[string]any{
		"name":             options.Name,
		"description":      options.Description,
		"group-by-tag":     options.GroupByTag,
		"hostname-var":     options.HostnameVar,
		"instance-filters": options.InstanceFilters,
		"sync-schedule":    options.SyncSchedule,
	}
	if options.Config != "" {
		attributes["config"] = json.RawMessage(options.Config)
	}
	// credential-id: empty string clears the credential (switch to OIDC).
	if options.ClearCredential {
		attributes["credential-id"] = ""
	} else if options.CredentialID != "" {
		attributes["credential-id"] = options.CredentialID
	}
	setBoolPtr(attributes, "update-on-launch", options.UpdateOnLaunch)
	setInt64Ptr(attributes, "update-cache-timeout", options.UpdateCacheTimeout)
	setBoolPtr(attributes, "overwrite", options.Overwrite)
	setBoolPtr(attributes, "overwrite-vars", options.OverwriteVars)
	setInt64Ptr(attributes, "verbosity", options.Verbosity)
	setBoolPtr(attributes, "group-by-instance-id", options.GroupByInstanceID)
	setBoolPtr(attributes, "group-by-region", options.GroupByRegion)
	setBoolPtr(attributes, "group-by-availability-zone", options.GroupByAvailabilityZone)
	setBoolPtr(attributes, "enabled", options.Enabled)

	reqBody, err := s.client.plain.Marshal(jsonAPIRequest{
		Data: jsonAPIRequestData{
			Type:       inventorySourceResourceType,
			Attributes: attributes,
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/inventory-sources/%s", url.PathEscape(id))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Delete removes the source identified by id.
func (s *AnsibleInventorySourcesService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}

	path := fmt.Sprintf("/ansible/inventory-sources/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// Sync triggers an ansible-inventory sync of the source identified by id (POST
// /ansible/inventory-sources/:source_id/actions/sync). It returns the source
// with its status advanced to "syncing". A disabled source is rejected (400).
func (s *AnsibleInventorySourcesService) Sync(ctx context.Context, id string) (*AnsibleInventorySource, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Sync requires an id")
	}

	path := fmt.Sprintf("/ansible/inventory-sources/%s/actions/sync", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// decode unmarshals a single-source JSON:API response into the public model.
func (s *AnsibleInventorySourcesService) decode(body []byte) (*AnsibleInventorySource, error) {
	var resource inventorySourceResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}

// setBoolPtr adds key=*val to m when val is non-nil.
func setBoolPtr(m map[string]any, key string, val *bool) {
	if val != nil {
		m[key] = *val
	}
}

// setInt64Ptr adds key=*val to m when val is non-nil.
func setInt64Ptr(m map[string]any, key string, val *int64) {
	if val != nil {
		m[key] = *val
	}
}
