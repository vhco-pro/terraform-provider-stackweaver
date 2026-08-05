// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// credentialResourceType is the JSON:API `type` member for an Ansible
// credential, matching the backend handler
// (handlers/ansible/credentials.go formatCredentialResponse).
const credentialResourceType = "ansible-credentials"

// AnsibleCredentialsService is the native service for Ansible credentials. The
// endpoint uses the JSON:API envelope with kebab-case attribute keys and a
// `project` request relationship (the response echoes only `organization`).
// Wire contract (all paths relative to /api/v2):
//
//	Create: POST   /organizations/:org/ansible/credentials
//	Read:   GET    /ansible/credentials/:id
//	Update: PATCH  /ansible/credentials/:id
//	Delete: DELETE /ansible/credentials/:id
//
// This service is self-contained (built inline via NewAnsibleCredentialsService)
// so it does not depend on a field on Client.
//
// NOTE: this resource is the intended future home of tfe_ssh_key — a thin
// TFE-compatible face over the `ssh` credential type can be layered on later.
type AnsibleCredentialsService struct {
	client *Client
}

// NewAnsibleCredentialsService builds a credentials service over c.
func NewAnsibleCredentialsService(c *Client) *AnsibleCredentialsService {
	return &AnsibleCredentialsService{client: c}
}

// AnsibleCredential is the native representation of a credential, flattened from
// the JSON:API resource object. Only the readable (non-secret) attributes and
// the four `has-*` presence booleans are ever returned by the API; secret
// material is write-only and never echoed.
type AnsibleCredential struct {
	ID                string
	OrganizationID    string
	Name              string
	Description       string
	Type              string
	Username          string
	AzureTenantID     string
	AzureClientID     string
	SSHPort           int64
	SSHBecomeUser     string
	HasSSHPrivateKey  bool
	HasPassword       bool
	HasVaultPassword  bool
	HasBecomePassword bool
}

// AnsibleCredentialCreateOptions are the fields accepted when creating a
// credential. Organization scopes the create endpoint; ProjectID is sent as a
// JSON:API relationship; the secrets are write-only attributes.
type AnsibleCredentialCreateOptions struct {
	Organization  string
	ProjectID     string
	Name          string
	Description   string
	Type          string
	Username      string
	AzureTenantID string
	AzureClientID string
	SSHPort       int64
	SSHBecomeUser string
	// Write-only secrets (accepted on write, never returned on read).
	SSHPrivateKey      string
	SSHPassphrase      string
	Password           string
	VaultPassword      string
	BecomePassword     string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AzureClientSecret  string
	GCPServiceAccount  string
}

// AnsibleCredentialUpdateOptions are the fields accepted when updating a
// credential. ProjectID is sent as a relationship only when non-empty; a secret
// is sent only when non-empty (an empty value leaves the stored secret
// unchanged — the API cannot echo it, so it is config-driven). credential-type
// is not accepted by the update handler and is therefore ForceNew.
type AnsibleCredentialUpdateOptions struct {
	Name               string
	Description        string
	Username           string
	AzureTenantID      string
	AzureClientID      string
	SSHPort            int64
	SSHBecomeUser      string
	ProjectID          string
	SSHPrivateKey      string
	SSHPassphrase      string
	Password           string
	VaultPassword      string
	BecomePassword     string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AzureClientSecret  string
	GCPServiceAccount  string
}

// credentialResource mirrors the JSON:API resource object the backend returns.
type credentialResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name              string `json:"name"`
		Description       string `json:"description"`
		CredentialType    string `json:"credential-type"`
		Username          string `json:"username"`
		AzureTenantID     string `json:"azure-tenant-id"`
		AzureClientID     string `json:"azure-client-id"`
		SSHPort           int64  `json:"ssh-port"`
		SSHBecomeUser     string `json:"ssh-become-user"`
		HasSSHPrivateKey  bool   `json:"has-ssh-private-key"`
		HasPassword       bool   `json:"has-password"`
		HasVaultPassword  bool   `json:"has-vault-password"`
		HasBecomePassword bool   `json:"has-become-password"`
	} `json:"attributes"`
	Relationships struct {
		Organization jsonAPIRelationship `json:"organization"`
	} `json:"relationships"`
}

// toModel flattens the wire resource into the public AnsibleCredential.
func (r *credentialResource) toModel() *AnsibleCredential {
	cred := &AnsibleCredential{
		ID:                r.ID,
		Name:              r.Attributes.Name,
		Description:       r.Attributes.Description,
		Type:              r.Attributes.CredentialType,
		Username:          r.Attributes.Username,
		AzureTenantID:     r.Attributes.AzureTenantID,
		AzureClientID:     r.Attributes.AzureClientID,
		SSHPort:           r.Attributes.SSHPort,
		SSHBecomeUser:     r.Attributes.SSHBecomeUser,
		HasSSHPrivateKey:  r.Attributes.HasSSHPrivateKey,
		HasPassword:       r.Attributes.HasPassword,
		HasVaultPassword:  r.Attributes.HasVaultPassword,
		HasBecomePassword: r.Attributes.HasBecomePassword,
	}
	if r.Relationships.Organization.Data != nil {
		cred.OrganizationID = r.Relationships.Organization.Data.ID
	}
	return cred
}

// Create creates a new credential under options.Organization.
func (s *AnsibleCredentialsService) Create(ctx context.Context, options AnsibleCredentialCreateOptions) (*AnsibleCredential, error) {
	if options.Organization == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an Organization")
	}

	attributes := map[string]any{
		"name":            options.Name,
		"credential-type": options.Type,
	}
	setIfNotEmpty(attributes, "description", options.Description)
	setIfNotEmpty(attributes, "username", options.Username)
	setIfNotEmpty(attributes, "azure-tenant-id", options.AzureTenantID)
	setIfNotEmpty(attributes, "azure-client-id", options.AzureClientID)
	setIfNotEmpty(attributes, "ssh-private-key", options.SSHPrivateKey)
	setIfNotEmpty(attributes, "ssh-passphrase", options.SSHPassphrase)
	setIfNotEmpty(attributes, "password", options.Password)
	setIfNotEmpty(attributes, "vault-password", options.VaultPassword)
	setIfNotEmpty(attributes, "become-password", options.BecomePassword)
	setIfNotEmpty(attributes, "aws-access-key-id", options.AWSAccessKeyID)
	setIfNotEmpty(attributes, "aws-secret-access-key", options.AWSSecretAccessKey)
	setIfNotEmpty(attributes, "azure-client-secret", options.AzureClientSecret)
	setIfNotEmpty(attributes, "gcp-service-account", options.GCPServiceAccount)
	if options.SSHPort > 0 {
		attributes["ssh-port"] = options.SSHPort
	}
	setIfNotEmpty(attributes, "ssh-become-user", options.SSHBecomeUser)

	data := jsonAPIRequestData{
		Type:       credentialResourceType,
		Attributes: attributes,
	}
	if options.ProjectID != "" {
		data.Relationships = map[string]jsonAPIRelationship{
			"project": {Data: &jsonAPIResourceRef{Type: "projects", ID: options.ProjectID}},
		}
	}

	reqBody, err := s.client.plain.Marshal(jsonAPIRequest{Data: data})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/organizations/%s/ansible/credentials", url.PathEscape(options.Organization))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Read returns the credential identified by id. A missing credential surfaces as
// ErrNotFound.
func (s *AnsibleCredentialsService) Read(ctx context.Context, id string) (*AnsibleCredential, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}

	path := fmt.Sprintf("/ansible/credentials/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// Update updates the credential identified by id.
func (s *AnsibleCredentialsService) Update(ctx context.Context, id string, options AnsibleCredentialUpdateOptions) (*AnsibleCredential, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Update requires an id")
	}

	attributes := map[string]any{
		"name":            options.Name,
		"description":     options.Description,
		"username":        options.Username,
		"azure-tenant-id": options.AzureTenantID,
		"azure-client-id": options.AzureClientID,
		"ssh-become-user": options.SSHBecomeUser,
	}
	if options.SSHPort > 0 {
		attributes["ssh-port"] = options.SSHPort
	}
	setIfNotEmpty(attributes, "ssh-private-key", options.SSHPrivateKey)
	setIfNotEmpty(attributes, "ssh-passphrase", options.SSHPassphrase)
	setIfNotEmpty(attributes, "password", options.Password)
	setIfNotEmpty(attributes, "vault-password", options.VaultPassword)
	setIfNotEmpty(attributes, "become-password", options.BecomePassword)
	setIfNotEmpty(attributes, "aws-access-key-id", options.AWSAccessKeyID)
	setIfNotEmpty(attributes, "aws-secret-access-key", options.AWSSecretAccessKey)
	setIfNotEmpty(attributes, "azure-client-secret", options.AzureClientSecret)
	setIfNotEmpty(attributes, "gcp-service-account", options.GCPServiceAccount)

	data := jsonAPIRequestData{
		Type:       credentialResourceType,
		Attributes: attributes,
	}
	if options.ProjectID != "" {
		data.Relationships = map[string]jsonAPIRelationship{
			"project": {Data: &jsonAPIResourceRef{Type: "projects", ID: options.ProjectID}},
		}
	}

	reqBody, err := s.client.plain.Marshal(jsonAPIRequest{Data: data})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/credentials/%s", url.PathEscape(id))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Delete removes the credential identified by id. A 409 (still referenced by a
// job template, job, or inventory source) surfaces as an *APIError.
func (s *AnsibleCredentialsService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}

	path := fmt.Sprintf("/ansible/credentials/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// decode unmarshals a single-credential JSON:API response into the public model.
func (s *AnsibleCredentialsService) decode(body []byte) (*AnsibleCredential, error) {
	var resource credentialResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}
