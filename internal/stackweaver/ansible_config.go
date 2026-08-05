// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ansibleConfigResourceType is the JSON:API `type` member for an ansible config.
// As of monorepo #608 the endpoint speaks the standard JSON:API envelope
// (handlers/ansible_config.go buildAnsibleConfigResponse) rather than the older
// flat shape: request { data: { type, attributes: { "config-content" } } },
// response { data: { id, type, attributes: { scope, config-content, ... },
// relationships: { organization|project|workspace } } }.
const ansibleConfigResourceType = "ansible-configs"

// AnsibleConfigService is the native service for the per-scope ansible.cfg
// singleton. There is exactly one config per scope entity; PUT is upsert
// (create-or-update, no separate POST). Only org and project scopes have REST
// routes — workspace scope is echoed but not manageable. Wire contract (all
// paths relative to /api/v2):
//
//	Get:    GET    /organizations/:name/ansible-config  |  /projects/:id/ansible-config
//	Upsert: PUT    /organizations/:name/ansible-config  |  /projects/:id/ansible-config
//	Delete: DELETE /organizations/:name/ansible-config  |  /projects/:id/ansible-config
//
// This service is self-contained (built inline via NewAnsibleConfigService).
type AnsibleConfigService struct {
	client *Client
}

// NewAnsibleConfigService builds an ansible-config service over c.
func NewAnsibleConfigService(c *Client) *AnsibleConfigService {
	return &AnsibleConfigService{client: c}
}

// AnsibleConfig is the native representation of a scope's ansible.cfg, flattened
// from the JSON:API resource object.
type AnsibleConfig struct {
	ID             string
	Scope          string
	OrganizationID string
	ProjectID      string
	WorkspaceID    string
	ConfigContent  string
	CreatedAt      string
	UpdatedAt      string
}

// ansibleConfigResource mirrors the JSON:API resource object the backend
// returns.
type ansibleConfigResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Scope         string `json:"scope"`
		ConfigContent string `json:"config-content"`
		CreatedAt     string `json:"created-at"`
		UpdatedAt     string `json:"updated-at"`
	} `json:"attributes"`
	Relationships struct {
		Organization jsonAPIRelationship `json:"organization"`
		Project      jsonAPIRelationship `json:"project"`
		Workspace    jsonAPIRelationship `json:"workspace"`
	} `json:"relationships"`
}

// toModel flattens the wire resource into the public AnsibleConfig.
func (r *ansibleConfigResource) toModel() *AnsibleConfig {
	cfg := &AnsibleConfig{
		ID:            r.ID,
		Scope:         r.Attributes.Scope,
		ConfigContent: r.Attributes.ConfigContent,
		CreatedAt:     r.Attributes.CreatedAt,
		UpdatedAt:     r.Attributes.UpdatedAt,
	}
	if r.Relationships.Organization.Data != nil {
		cfg.OrganizationID = r.Relationships.Organization.Data.ID
	}
	if r.Relationships.Project.Data != nil {
		cfg.ProjectID = r.Relationships.Project.Data.ID
	}
	if r.Relationships.Workspace.Data != nil {
		cfg.WorkspaceID = r.Relationships.Workspace.Data.ID
	}
	return cfg
}

// GetByOrganization returns the org-scoped config. A missing config (or org)
// surfaces as ErrNotFound.
func (s *AnsibleConfigService) GetByOrganization(ctx context.Context, orgName string) (*AnsibleConfig, error) {
	if orgName == "" {
		return nil, fmt.Errorf("stackweaver: GetByOrganization requires an organization")
	}
	path := fmt.Sprintf("/organizations/%s/ansible-config", url.PathEscape(orgName))
	return s.get(ctx, path)
}

// GetByProject returns the project-scoped config.
func (s *AnsibleConfigService) GetByProject(ctx context.Context, projectID string) (*AnsibleConfig, error) {
	if projectID == "" {
		return nil, fmt.Errorf("stackweaver: GetByProject requires a project id")
	}
	path := fmt.Sprintf("/projects/%s/ansible-config", url.PathEscape(projectID))
	return s.get(ctx, path)
}

// UpsertByOrganization creates or updates the org-scoped config.
func (s *AnsibleConfigService) UpsertByOrganization(ctx context.Context, orgName, configContent string) (*AnsibleConfig, error) {
	if orgName == "" {
		return nil, fmt.Errorf("stackweaver: UpsertByOrganization requires an organization")
	}
	path := fmt.Sprintf("/organizations/%s/ansible-config", url.PathEscape(orgName))
	return s.upsert(ctx, path, configContent)
}

// UpsertByProject creates or updates the project-scoped config.
func (s *AnsibleConfigService) UpsertByProject(ctx context.Context, projectID, configContent string) (*AnsibleConfig, error) {
	if projectID == "" {
		return nil, fmt.Errorf("stackweaver: UpsertByProject requires a project id")
	}
	path := fmt.Sprintf("/projects/%s/ansible-config", url.PathEscape(projectID))
	return s.upsert(ctx, path, configContent)
}

// DeleteByOrganization removes the org-scoped config.
func (s *AnsibleConfigService) DeleteByOrganization(ctx context.Context, orgName string) error {
	if orgName == "" {
		return fmt.Errorf("stackweaver: DeleteByOrganization requires an organization")
	}
	path := fmt.Sprintf("/organizations/%s/ansible-config", url.PathEscape(orgName))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// DeleteByProject removes the project-scoped config.
func (s *AnsibleConfigService) DeleteByProject(ctx context.Context, projectID string) error {
	if projectID == "" {
		return fmt.Errorf("stackweaver: DeleteByProject requires a project id")
	}
	path := fmt.Sprintf("/projects/%s/ansible-config", url.PathEscape(projectID))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// get performs a GET against the given scope path and decodes the response.
func (s *AnsibleConfigService) get(ctx context.Context, path string) (*AnsibleConfig, error) {
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return s.decode(body)
}

// upsert PUTs the config content to the given scope path and decodes the
// response.
func (s *AnsibleConfigService) upsert(ctx context.Context, path, configContent string) (*AnsibleConfig, error) {
	reqBody, err := s.client.plain.Marshal(jsonAPIRequest{
		Data: jsonAPIRequestData{
			Type:       ansibleConfigResourceType,
			Attributes: map[string]any{"config-content": configContent},
		},
	})
	if err != nil {
		return nil, err
	}
	body, err := s.client.do(ctx, http.MethodPut, path, reqBody)
	if err != nil {
		return nil, err
	}
	return s.decode(body)
}

// decode unmarshals a single-config JSON:API response into the public model.
func (s *AnsibleConfigService) decode(body []byte) (*AnsibleConfig, error) {
	var resource ansibleConfigResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}
