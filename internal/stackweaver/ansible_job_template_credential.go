// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// AnsibleJobTemplateCredentialsService is the native service for the job
// template multi-credential association (the AWX "one credential per type,
// multiple vaults with distinct vault IDs" set). It is a pure association:
// attach = create, detach = delete, no update. The attach request is plain
// JSON; the response is JSON:API-shaped. Wire contract (relative to /api/v2):
//
//	Attach: POST   /ansible/job-templates/:id/credentials   {"credential_id":"..."}
//	List:   GET    /ansible/job-templates/:id/credentials
//	Detach: DELETE /ansible/job-templates/:id/credentials/:credential_id
//
// It is self-contained: build it with NewAnsibleJobTemplateCredentialsService.
type AnsibleJobTemplateCredentialsService struct {
	client *Client
}

// NewAnsibleJobTemplateCredentialsService constructs the service against an
// existing native Client.
func NewAnsibleJobTemplateCredentialsService(c *Client) *AnsibleJobTemplateCredentialsService {
	return &AnsibleJobTemplateCredentialsService{client: c}
}

// AnsibleTemplateCredential is the native representation of a credential
// attached to a job template, flattened from the JSON:API resource object the
// attach/list endpoints return (type "ansible-credentials").
type AnsibleTemplateCredential struct {
	ID             string
	Name           string
	CredentialType string
	VaultID        string
	Username       string
}

// templateCredentialResource mirrors the JSON:API resource object the backend
// returns (handlers/ansible/template_credentials.go formatTemplateCredential).
type templateCredentialResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name           string `json:"name"`
		CredentialType string `json:"credential-type"`
		VaultID        string `json:"vault-id"`
		Username       string `json:"username"`
	} `json:"attributes"`
}

func (r *templateCredentialResource) toModel() *AnsibleTemplateCredential {
	return &AnsibleTemplateCredential{
		ID:             r.ID,
		Name:           r.Attributes.Name,
		CredentialType: r.Attributes.CredentialType,
		VaultID:        r.Attributes.VaultID,
		Username:       r.Attributes.Username,
	}
}

// attachCredentialRequest is the plain-JSON attach body.
type attachCredentialRequest struct {
	CredentialID string `json:"credential_id"`
}

// Attach attaches credentialID to jobTemplateID's credential set, enforcing the
// AWX one-per-type rule server-side (a conflicting attach returns a 409 APIError).
func (s *AnsibleJobTemplateCredentialsService) Attach(ctx context.Context, jobTemplateID, credentialID string) (*AnsibleTemplateCredential, error) {
	if jobTemplateID == "" {
		return nil, fmt.Errorf("stackweaver: Attach requires a JobTemplateID")
	}
	if credentialID == "" {
		return nil, fmt.Errorf("stackweaver: Attach requires a CredentialID")
	}

	reqBody, err := s.client.plain.Marshal(attachCredentialRequest{CredentialID: credentialID})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/credentials", url.PathEscape(jobTemplateID))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	var resource templateCredentialResource
	if err := s.client.jsonapi.Unmarshal(respBody, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}

// List returns the credentials attached to jobTemplateID.
func (s *AnsibleJobTemplateCredentialsService) List(ctx context.Context, jobTemplateID string) ([]*AnsibleTemplateCredential, error) {
	if jobTemplateID == "" {
		return nil, fmt.Errorf("stackweaver: List requires a JobTemplateID")
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/credentials", url.PathEscape(jobTemplateID))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resources []templateCredentialResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, err
	}

	creds := make([]*AnsibleTemplateCredential, len(resources))
	for i := range resources {
		creds[i] = resources[i].toModel()
	}
	return creds, nil
}

// Read returns the attached credential whose id equals credentialID. There is no
// per-association GET, so it lists the template's credentials and selects by id;
// an absent association surfaces as ErrNotFound.
func (s *AnsibleJobTemplateCredentialsService) Read(ctx context.Context, jobTemplateID, credentialID string) (*AnsibleTemplateCredential, error) {
	if credentialID == "" {
		return nil, fmt.Errorf("stackweaver: Read requires a CredentialID")
	}

	creds, err := s.List(ctx, jobTemplateID)
	if err != nil {
		return nil, err
	}
	for _, cred := range creds {
		if cred.ID == credentialID {
			return cred, nil
		}
	}
	return nil, ErrNotFound
}

// Detach removes credentialID from jobTemplateID's credential set.
func (s *AnsibleJobTemplateCredentialsService) Detach(ctx context.Context, jobTemplateID, credentialID string) error {
	if jobTemplateID == "" {
		return fmt.Errorf("stackweaver: Detach requires a JobTemplateID")
	}
	if credentialID == "" {
		return fmt.Errorf("stackweaver: Detach requires a CredentialID")
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/credentials/%s", url.PathEscape(jobTemplateID), url.PathEscape(credentialID))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}
