// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// jobTemplateResourceType is the JSON:API `type` member for a job template,
// matching the backend handler (handlers/ansible/playbooks.go
// formatJobTemplateResponse).
const jobTemplateResourceType = "ansible-job-templates"

// AnsibleJobTemplatesService is the native service for the AWX-style Ansible job
// template. The handler speaks TFE-style JSON:API with hyphenated attribute keys
// and playbook/inventory/credential/agent-pool/project as relationships. Wire
// contract (all paths relative to /api/v2):
//
//	Create: POST   /organizations/:org/ansible/job-templates
//	Read:   GET    /ansible/job-templates/:id
//	Update: PATCH  /ansible/job-templates/:id
//	Delete: DELETE /ansible/job-templates/:id
//	List:   GET    /organizations/:org/ansible/job-templates
//	        GET    /projects/:project_id/ansible/job-templates
//
// It is self-contained: build it with NewAnsibleJobTemplatesService(client).
type AnsibleJobTemplatesService struct {
	client *Client
}

// NewAnsibleJobTemplatesService constructs the service against an existing
// native Client.
func NewAnsibleJobTemplatesService(c *Client) *AnsibleJobTemplatesService {
	return &AnsibleJobTemplatesService{client: c}
}

// AnsibleJobTemplate is the native representation of a job template, flattened
// from the JSON:API resource object (attributes + relationship ids) into the
// shape the Terraform resource consumes. Enabled is the API projection of the
// inverted model field Disabled.
type AnsibleJobTemplate struct {
	ID                string
	ProjectID         string
	PlaybookID        string
	InventoryID       string
	CredentialID      string
	AgentPoolID       string
	Name              string
	Description       string
	ExtraVars         map[string]any
	Limit             string
	Tags              string
	SkipTags          string
	Verbosity         int
	Forks             int
	BecomeEnabled     bool
	DiffMode          bool
	Enabled           bool
	TimeoutSeconds    int
	AllowSimultaneous bool
	RetentionDays     *int
	JobSliceCount     int
	ScheduleEnabled   bool
	ScheduleCron      string
	AllowCallbacks    bool
	LaunchOnWebhook   bool
	HostConfigKey     string
	CreatedAt         string
	UpdatedAt         string
}

// AnsibleJobTemplateCreateOptions are the fields accepted when creating a job
// template. Pointer fields are sent only when non-nil so the server applies its
// defaults (e.g. forks 0 -> 5, enabled -> true, job-slice-count -> 1). Create
// does not accept allow-callbacks/launch-on-webhook/host-config-key.
type AnsibleJobTemplateCreateOptions struct {
	Organization      string
	ProjectID         string
	PlaybookID        string
	InventoryID       string
	CredentialID      string
	AgentPoolID       string
	Name              string
	Description       string
	ExtraVars         map[string]any
	Limit             string
	Tags              string
	SkipTags          string
	Verbosity         int
	Forks             *int
	BecomeEnabled     bool
	DiffMode          bool
	ScheduleEnabled   bool
	ScheduleCron      string
	Enabled           *bool
	TimeoutSeconds    int
	AllowSimultaneous bool
	RetentionDays     *int
	JobSliceCount     *int
}

// AnsibleJobTemplateUpdateOptions are the fields accepted when updating a job
// template. A nil pointer leaves the corresponding attribute unchanged. Empty
// relationship id strings leave the linkage untouched. Update adds
// allow-callbacks and launch-on-webhook over the create set.
type AnsibleJobTemplateUpdateOptions struct {
	PlaybookID        string
	InventoryID       string
	CredentialID      string
	AgentPoolID       string
	Name              *string
	Description       *string
	ExtraVars         *map[string]any
	Limit             *string
	Tags              *string
	SkipTags          *string
	Verbosity         *int
	Forks             *int
	BecomeEnabled     *bool
	DiffMode          *bool
	ScheduleEnabled   *bool
	ScheduleCron      *string
	Enabled           *bool
	TimeoutSeconds    *int
	AllowSimultaneous *bool
	RetentionDays     *int
	JobSliceCount     *int
	AllowCallbacks    *bool
	LaunchOnWebhook   *bool
}

// jobTemplateResource mirrors the JSON:API resource object the backend returns.
type jobTemplateResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name              string         `json:"name"`
		Description       string         `json:"description"`
		ExtraVars         map[string]any `json:"extra-vars"`
		Limit             string         `json:"limit"`
		Tags              string         `json:"tags"`
		SkipTags          string         `json:"skip-tags"`
		Verbosity         int            `json:"verbosity"`
		Forks             int            `json:"forks"`
		BecomeEnabled     bool           `json:"become-enabled"`
		DiffMode          bool           `json:"diff-mode"`
		ScheduleEnabled   bool           `json:"schedule-enabled"`
		ScheduleCron      string         `json:"schedule-cron"`
		Enabled           bool           `json:"enabled"`
		TimeoutSeconds    int            `json:"timeout-seconds"`
		AllowSimultaneous bool           `json:"allow-simultaneous"`
		RetentionDays     *int           `json:"retention-days"`
		JobSliceCount     int            `json:"job-slice-count"`
		AllowCallbacks    bool           `json:"allow-callbacks"`
		LaunchOnWebhook   bool           `json:"launch-on-webhook"`
		HostConfigKey     string         `json:"host-config-key"`
		CreatedAt         string         `json:"created-at"`
		UpdatedAt         string         `json:"updated-at"`
	} `json:"attributes"`
	Relationships struct {
		Project    jsonAPIRelationship `json:"project"`
		Playbook   jsonAPIRelationship `json:"playbook"`
		Inventory  jsonAPIRelationship `json:"inventory"`
		Credential jsonAPIRelationship `json:"credential"`
		AgentPool  jsonAPIRelationship `json:"agent-pool"`
	} `json:"relationships"`
}

// toModel flattens the wire resource into the public AnsibleJobTemplate.
func (r *jobTemplateResource) toModel() *AnsibleJobTemplate {
	t := &AnsibleJobTemplate{
		ID:                r.ID,
		Name:              r.Attributes.Name,
		Description:       r.Attributes.Description,
		ExtraVars:         r.Attributes.ExtraVars,
		Limit:             r.Attributes.Limit,
		Tags:              r.Attributes.Tags,
		SkipTags:          r.Attributes.SkipTags,
		Verbosity:         r.Attributes.Verbosity,
		Forks:             r.Attributes.Forks,
		BecomeEnabled:     r.Attributes.BecomeEnabled,
		DiffMode:          r.Attributes.DiffMode,
		Enabled:           r.Attributes.Enabled,
		TimeoutSeconds:    r.Attributes.TimeoutSeconds,
		AllowSimultaneous: r.Attributes.AllowSimultaneous,
		RetentionDays:     r.Attributes.RetentionDays,
		JobSliceCount:     r.Attributes.JobSliceCount,
		ScheduleEnabled:   r.Attributes.ScheduleEnabled,
		ScheduleCron:      r.Attributes.ScheduleCron,
		AllowCallbacks:    r.Attributes.AllowCallbacks,
		LaunchOnWebhook:   r.Attributes.LaunchOnWebhook,
		HostConfigKey:     r.Attributes.HostConfigKey,
		CreatedAt:         r.Attributes.CreatedAt,
		UpdatedAt:         r.Attributes.UpdatedAt,
	}
	if r.Relationships.Project.Data != nil {
		t.ProjectID = r.Relationships.Project.Data.ID
	}
	if r.Relationships.Playbook.Data != nil {
		t.PlaybookID = r.Relationships.Playbook.Data.ID
	}
	if r.Relationships.Inventory.Data != nil {
		t.InventoryID = r.Relationships.Inventory.Data.ID
	}
	if r.Relationships.Credential.Data != nil {
		t.CredentialID = r.Relationships.Credential.Data.ID
	}
	if r.Relationships.AgentPool.Data != nil {
		t.AgentPoolID = r.Relationships.AgentPool.Data.ID
	}
	return t
}

// jobTemplateRequest is the JSON:API request envelope for create/update.
type jobTemplateRequest struct {
	Data jobTemplateRequestData `json:"data"`
}

type jobTemplateRequestData struct {
	Type          string                         `json:"type"`
	Attributes    map[string]any                 `json:"attributes"`
	Relationships map[string]jsonAPIRelationship `json:"relationships,omitempty"`
}

// Create creates a new job template under options.Organization.
func (s *AnsibleJobTemplatesService) Create(ctx context.Context, options AnsibleJobTemplateCreateOptions) (*AnsibleJobTemplate, error) {
	if options.Organization == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an Organization")
	}
	if options.PlaybookID == "" {
		return nil, fmt.Errorf("stackweaver: Create requires a PlaybookID")
	}
	if options.InventoryID == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an InventoryID")
	}

	attributes := map[string]any{
		"name":               options.Name,
		"description":        options.Description,
		"limit":              options.Limit,
		"tags":               options.Tags,
		"skip-tags":          options.SkipTags,
		"verbosity":          options.Verbosity,
		"become-enabled":     options.BecomeEnabled,
		"diff-mode":          options.DiffMode,
		"schedule-enabled":   options.ScheduleEnabled,
		"schedule-cron":      options.ScheduleCron,
		"timeout-seconds":    options.TimeoutSeconds,
		"allow-simultaneous": options.AllowSimultaneous,
	}
	if options.ExtraVars != nil {
		attributes["extra-vars"] = options.ExtraVars
	}
	if options.Forks != nil {
		attributes["forks"] = *options.Forks
	}
	if options.Enabled != nil {
		attributes["enabled"] = *options.Enabled
	}
	if options.JobSliceCount != nil {
		attributes["job-slice-count"] = *options.JobSliceCount
	}
	if options.RetentionDays != nil {
		attributes["retention-days"] = *options.RetentionDays
	}

	relationships := map[string]jsonAPIRelationship{
		"playbook":  {Data: &jsonAPIResourceRef{Type: playbookResourceType, ID: options.PlaybookID}},
		"inventory": {Data: &jsonAPIResourceRef{Type: "ansible-inventories", ID: options.InventoryID}},
	}
	if options.ProjectID != "" {
		relationships["project"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: "projects", ID: options.ProjectID}}
	}
	if options.CredentialID != "" {
		relationships["credential"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: "ansible-credentials", ID: options.CredentialID}}
	}
	if options.AgentPoolID != "" {
		relationships["agent-pool"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: "agent-pools", ID: options.AgentPoolID}}
	}

	reqBody, err := s.client.plain.Marshal(jobTemplateRequest{
		Data: jobTemplateRequestData{
			Type:          jobTemplateResourceType,
			Attributes:    attributes,
			Relationships: relationships,
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/organizations/%s/ansible/job-templates", url.PathEscape(options.Organization))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Read returns the job template identified by id. A missing template surfaces as
// ErrNotFound.
func (s *AnsibleJobTemplatesService) Read(ctx context.Context, id string) (*AnsibleJobTemplate, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}

	path := fmt.Sprintf("/ansible/job-templates/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// Update updates the job template identified by id.
func (s *AnsibleJobTemplatesService) Update(ctx context.Context, id string, options AnsibleJobTemplateUpdateOptions) (*AnsibleJobTemplate, error) {
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
	if options.ExtraVars != nil {
		attributes["extra-vars"] = *options.ExtraVars
	}
	if options.Limit != nil {
		attributes["limit"] = *options.Limit
	}
	if options.Tags != nil {
		attributes["tags"] = *options.Tags
	}
	if options.SkipTags != nil {
		attributes["skip-tags"] = *options.SkipTags
	}
	if options.Verbosity != nil {
		attributes["verbosity"] = *options.Verbosity
	}
	if options.Forks != nil {
		attributes["forks"] = *options.Forks
	}
	if options.BecomeEnabled != nil {
		attributes["become-enabled"] = *options.BecomeEnabled
	}
	if options.DiffMode != nil {
		attributes["diff-mode"] = *options.DiffMode
	}
	if options.ScheduleEnabled != nil {
		attributes["schedule-enabled"] = *options.ScheduleEnabled
	}
	if options.ScheduleCron != nil {
		attributes["schedule-cron"] = *options.ScheduleCron
	}
	if options.Enabled != nil {
		attributes["enabled"] = *options.Enabled
	}
	if options.TimeoutSeconds != nil {
		attributes["timeout-seconds"] = *options.TimeoutSeconds
	}
	if options.AllowSimultaneous != nil {
		attributes["allow-simultaneous"] = *options.AllowSimultaneous
	}
	if options.RetentionDays != nil {
		attributes["retention-days"] = *options.RetentionDays
	}
	if options.JobSliceCount != nil {
		attributes["job-slice-count"] = *options.JobSliceCount
	}
	if options.AllowCallbacks != nil {
		attributes["allow-callbacks"] = *options.AllowCallbacks
	}
	if options.LaunchOnWebhook != nil {
		attributes["launch-on-webhook"] = *options.LaunchOnWebhook
	}

	data := jobTemplateRequestData{
		Type:       jobTemplateResourceType,
		Attributes: attributes,
	}
	relationships := map[string]jsonAPIRelationship{}
	if options.PlaybookID != "" {
		relationships["playbook"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: playbookResourceType, ID: options.PlaybookID}}
	}
	if options.InventoryID != "" {
		relationships["inventory"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: "ansible-inventories", ID: options.InventoryID}}
	}
	if options.CredentialID != "" {
		relationships["credential"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: "ansible-credentials", ID: options.CredentialID}}
	}
	if options.AgentPoolID != "" {
		relationships["agent-pool"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: "agent-pools", ID: options.AgentPoolID}}
	}
	if len(relationships) > 0 {
		data.Relationships = relationships
	}

	reqBody, err := s.client.plain.Marshal(jobTemplateRequest{Data: data})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/job-templates/%s", url.PathEscape(id))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(respBody)
}

// Delete removes the job template identified by id.
func (s *AnsibleJobTemplatesService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}

	path := fmt.Sprintf("/ansible/job-templates/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// decode unmarshals a single-template JSON:API response into the public model.
func (s *AnsibleJobTemplatesService) decode(body []byte) (*AnsibleJobTemplate, error) {
	var resource jobTemplateResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}
