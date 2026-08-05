// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// playbookResourceType is the JSON:API `type` member for a playbook, matching
// the backend handler (handlers/ansible/playbooks.go formatPlaybookResponse).
const playbookResourceType = "ansible-playbooks"

// AnsiblePlaybooksService is the reference native service — the template every
// other native resource family follows. ansible_playbook is a JSON:API resource
// (its project and vcs-connection are relationships), so its methods use the
// JSON:API envelope. Wire contract (all paths relative to /api/v2):
//
//	Create: POST   /organizations/:org/ansible/playbooks
//	Read:   GET    /ansible/playbooks/:id
//	Update: PATCH  /ansible/playbooks/:id
//	Delete: DELETE /ansible/playbooks/:id
//	Sync:   POST   /ansible/playbooks/:id/actions/sync
//	List:   GET    /projects/:project_id/ansible/playbooks
type AnsiblePlaybooksService struct {
	client *Client
}

// AnsiblePlaybook is the native representation of an Ansible playbook, flattened
// from the JSON:API resource object (attributes + the project/vcs-connection
// relationship ids) into the shape the Terraform resource consumes.
type AnsiblePlaybook struct {
	ID              string
	ProjectID       string
	Name            string
	Description     string
	VCSConnectionID string
	VCSRepository   string
	VCSBranch       string
	PlaybookPath    string
	SourceMode      string
	LastSyncAt      string
	LastSyncStatus  string
	LastSyncCommit  string
	CachedCommit    string
	CachedAt        string
}

// AnsiblePlaybookListOptions are the filter/pagination options for List.
type AnsiblePlaybookListOptions struct {
	ProjectID string
	PageSize  int
	PageNum   int
}

// AnsiblePlaybookCreateOptions are the fields accepted when creating a playbook.
// Organization scopes the create endpoint; ProjectID/VCSConnectionID are sent as
// JSON:API relationships, the rest as attributes.
type AnsiblePlaybookCreateOptions struct {
	Organization    string
	ProjectID       string
	Name            string
	Description     string
	VCSConnectionID string
	VCSRepository   string
	VCSBranch       string
	PlaybookPath    string
	SourceMode      string
}

// AnsiblePlaybookUpdateOptions are the fields accepted when updating a playbook.
// A nil pointer leaves the corresponding field unchanged.
type AnsiblePlaybookUpdateOptions struct {
	Name            *string
	Description     *string
	VCSConnectionID *string
	VCSRepository   *string
	VCSBranch       *string
	PlaybookPath    *string
	SourceMode      *string
}

// playbookResource mirrors the JSON:API resource object the backend returns, so
// jsonAPICodec.Unmarshal can decode a response straight into it.
type playbookResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		VCSRepository  string `json:"vcs-repository"`
		VCSBranch      string `json:"vcs-branch"`
		PlaybookPath   string `json:"playbook-path"`
		SourceMode     string `json:"source-mode"`
		LastSyncAt     string `json:"last-sync-at"`
		LastSyncStatus string `json:"last-sync-status"`
		LastSyncCommit string `json:"last-sync-commit"`
		CachedCommit   string `json:"cached-commit"`
		CachedAt       string `json:"cached-at"`
	} `json:"attributes"`
	Relationships struct {
		Project       jsonAPIRelationship `json:"project"`
		VCSConnection jsonAPIRelationship `json:"vcs-connection"`
	} `json:"relationships"`
}

// toModel flattens the wire resource into the public AnsiblePlaybook.
func (r *playbookResource) toModel() *AnsiblePlaybook {
	p := &AnsiblePlaybook{
		ID:             r.ID,
		Name:           r.Attributes.Name,
		Description:    r.Attributes.Description,
		VCSRepository:  r.Attributes.VCSRepository,
		VCSBranch:      r.Attributes.VCSBranch,
		PlaybookPath:   r.Attributes.PlaybookPath,
		SourceMode:     r.Attributes.SourceMode,
		LastSyncAt:     r.Attributes.LastSyncAt,
		LastSyncStatus: r.Attributes.LastSyncStatus,
		LastSyncCommit: r.Attributes.LastSyncCommit,
		CachedCommit:   r.Attributes.CachedCommit,
		CachedAt:       r.Attributes.CachedAt,
	}
	if r.Relationships.Project.Data != nil {
		p.ProjectID = r.Relationships.Project.Data.ID
	}
	if r.Relationships.VCSConnection.Data != nil {
		p.VCSConnectionID = r.Relationships.VCSConnection.Data.ID
	}
	return p
}

// jsonAPIRelationship is a to-one relationship member ({"data":{"type","id"}}).
type jsonAPIRelationship struct {
	Data *jsonAPIResourceRef `json:"data"`
}

// jsonAPIResourceRef is a resource linkage (type + id).
type jsonAPIResourceRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// playbookRequest is the JSON:API request envelope for create/update. Attributes
// omit empty values so the server applies its defaults; relationships are added
// only when a linkage is supplied.
type playbookRequest struct {
	Data playbookRequestData `json:"data"`
}

type playbookRequestData struct {
	Type          string                         `json:"type"`
	Attributes    map[string]any                 `json:"attributes"`
	Relationships map[string]jsonAPIRelationship `json:"relationships,omitempty"`
}

// List returns the playbooks in options.ProjectID.
func (s *AnsiblePlaybooksService) List(ctx context.Context, options AnsiblePlaybookListOptions) ([]*AnsiblePlaybook, error) {
	if options.ProjectID == "" {
		return nil, fmt.Errorf("stackweaver: List requires a ProjectID")
	}

	path := fmt.Sprintf("/projects/%s/ansible/playbooks", url.PathEscape(options.ProjectID))
	query := url.Values{}
	if options.PageSize > 0 {
		query.Set("page[size]", fmt.Sprintf("%d", options.PageSize))
	}
	if options.PageNum > 0 {
		query.Set("page[number]", fmt.Sprintf("%d", options.PageNum))
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resources []playbookResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, err
	}

	playbooks := make([]*AnsiblePlaybook, len(resources))
	for i := range resources {
		playbooks[i] = resources[i].toModel()
	}
	return playbooks, nil
}

// Create creates a new playbook under options.Organization.
func (s *AnsiblePlaybooksService) Create(ctx context.Context, options AnsiblePlaybookCreateOptions) (*AnsiblePlaybook, error) {
	if options.Organization == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an Organization")
	}
	if options.ProjectID == "" {
		return nil, fmt.Errorf("stackweaver: Create requires a ProjectID")
	}

	attributes := map[string]any{
		"name": options.Name,
	}
	if options.Description != "" {
		attributes["description"] = options.Description
	}
	if options.VCSRepository != "" {
		attributes["vcs-repository"] = options.VCSRepository
	}
	if options.VCSBranch != "" {
		attributes["vcs-branch"] = options.VCSBranch
	}
	if options.PlaybookPath != "" {
		attributes["playbook-path"] = options.PlaybookPath
	}
	if options.SourceMode != "" {
		attributes["source-mode"] = options.SourceMode
	}

	relationships := map[string]jsonAPIRelationship{
		"project": {Data: &jsonAPIResourceRef{Type: "projects", ID: options.ProjectID}},
	}
	if options.VCSConnectionID != "" {
		relationships["vcs-connection"] = jsonAPIRelationship{
			Data: &jsonAPIResourceRef{Type: "vcs-connections", ID: options.VCSConnectionID},
		}
	}

	reqBody, err := s.client.plain.Marshal(playbookRequest{
		Data: playbookRequestData{
			Type:          playbookResourceType,
			Attributes:    attributes,
			Relationships: relationships,
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/organizations/%s/ansible/playbooks", url.PathEscape(options.Organization))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Read returns the playbook identified by id. A missing playbook surfaces as
// ErrNotFound.
func (s *AnsiblePlaybooksService) Read(ctx context.Context, id string) (*AnsiblePlaybook, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}

	path := fmt.Sprintf("/ansible/playbooks/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// Update updates the playbook identified by id.
func (s *AnsiblePlaybooksService) Update(ctx context.Context, id string, options AnsiblePlaybookUpdateOptions) (*AnsiblePlaybook, error) {
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
	if options.VCSRepository != nil {
		attributes["vcs-repository"] = *options.VCSRepository
	}
	if options.VCSBranch != nil {
		attributes["vcs-branch"] = *options.VCSBranch
	}
	if options.PlaybookPath != nil {
		attributes["playbook-path"] = *options.PlaybookPath
	}
	if options.SourceMode != nil {
		attributes["source-mode"] = *options.SourceMode
	}

	data := playbookRequestData{
		Type:       playbookResourceType,
		Attributes: attributes,
	}
	if options.VCSConnectionID != nil {
		// An empty id clears the linkage; the backend treats a null-id
		// relationship as "detach the VCS connection".
		data.Relationships = map[string]jsonAPIRelationship{
			"vcs-connection": {Data: &jsonAPIResourceRef{Type: "vcs-connections", ID: *options.VCSConnectionID}},
		}
	}

	reqBody, err := s.client.plain.Marshal(playbookRequest{Data: data})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/playbooks/%s", url.PathEscape(id))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Delete removes the playbook identified by id.
func (s *AnsiblePlaybooksService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}

	path := fmt.Sprintf("/ansible/playbooks/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// Sync triggers a VCS sync of the playbook identified by id (POST
// /ansible/playbooks/:id/actions/sync). It returns the playbook with its sync
// status advanced to "syncing".
func (s *AnsiblePlaybooksService) Sync(ctx context.Context, id string) (*AnsiblePlaybook, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Sync requires an id")
	}

	path := fmt.Sprintf("/ansible/playbooks/%s/actions/sync", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// decode unmarshals a single-playbook JSON:API response into the public model.
func (s *AnsiblePlaybooksService) decode(body []byte) (*AnsiblePlaybook, error) {
	var resource playbookResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}
