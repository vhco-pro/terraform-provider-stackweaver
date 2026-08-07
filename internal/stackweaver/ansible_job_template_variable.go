// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// jobTemplateVariableResourceType is the JSON:API `type` member for a job
// template variable. It mirrors TFE variables - the handler
// (handlers/ansible/job_template_variables.go) validates the request type is
// exactly "vars".
const jobTemplateVariableResourceType = "vars"

// SensitiveValueMask is the sentinel a sensitive variable's value is masked with
// on read; the provider must retain the configured value rather than treat this
// mask as drift.
const SensitiveValueMask = "••••••••"

// AnsibleJobTemplateVariablesService is the native service for a single Ansible
// job template variable (the AWX/TFE analogue of a workspace variable). Wire
// contract (all paths relative to /api/v2):
//
//	Create: POST   /ansible/job-templates/:id/vars
//	Read:   GET    /ansible/job-templates/:id/vars  (list; no single-row GET)
//	Update: PATCH  /ansible/job-templates/:id/vars/:variable_id
//	Delete: DELETE /ansible/job-templates/:id/vars/:variable_id
//
// It is self-contained: build it with NewAnsibleJobTemplateVariablesService.
type AnsibleJobTemplateVariablesService struct {
	client *Client
}

// NewAnsibleJobTemplateVariablesService constructs the service against an
// existing native Client.
func NewAnsibleJobTemplateVariablesService(c *Client) *AnsibleJobTemplateVariablesService {
	return &AnsibleJobTemplateVariablesService{client: c}
}

// AnsibleJobTemplateVariable is the native representation of a job template
// variable, flattened from the JSON:API resource object. JobTemplateID comes
// from the URL path, not the response attributes.
type AnsibleJobTemplateVariable struct {
	ID            string
	JobTemplateID string
	Key           string
	Value         string
	Description   string
	Category      string
	HCL           bool
	Sensitive     bool
}

// AnsibleJobTemplateVariableCreateOptions are the fields accepted when creating
// a variable. JobTemplateID scopes the endpoint; Key and Value are required.
type AnsibleJobTemplateVariableCreateOptions struct {
	JobTemplateID string
	Key           string
	Value         string
	Description   string
	Category      string
	HCL           bool
	Sensitive     bool
}

// AnsibleJobTemplateVariableUpdateOptions are the fields accepted when updating
// a variable. Empty Key/Value/Description/Category leave the field unchanged;
// HCL and Sensitive are pointers so a nil leaves them unchanged.
type AnsibleJobTemplateVariableUpdateOptions struct {
	Key         string
	Value       string
	Description string
	Category    string
	HCL         *bool
	Sensitive   *bool
}

// jobTemplateVariableResource mirrors the JSON:API resource object the backend
// returns. Attribute keys here are plain (non-hyphenated), per formatVariableResponse.
type jobTemplateVariableResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		Description string `json:"description"`
		Sensitive   bool   `json:"sensitive"`
		Category    string `json:"category"`
		HCL         bool   `json:"hcl"`
	} `json:"attributes"`
	Relationships struct {
		Configurable jsonAPIRelationship `json:"configurable"`
	} `json:"relationships"`
}

// toModel flattens the wire resource into the public model, stamping the owning
// template id from the caller (the response links it via relationships.configurable).
func (r *jobTemplateVariableResource) toModel(jobTemplateID string) *AnsibleJobTemplateVariable {
	v := &AnsibleJobTemplateVariable{
		ID:            r.ID,
		JobTemplateID: jobTemplateID,
		Key:           r.Attributes.Key,
		Value:         r.Attributes.Value,
		Description:   r.Attributes.Description,
		Category:      r.Attributes.Category,
		HCL:           r.Attributes.HCL,
		Sensitive:     r.Attributes.Sensitive,
	}
	if v.JobTemplateID == "" && r.Relationships.Configurable.Data != nil {
		v.JobTemplateID = r.Relationships.Configurable.Data.ID
	}
	return v
}

// jobTemplateVariableRequest is the JSON:API request envelope for create/update.
type jobTemplateVariableRequest struct {
	Data jobTemplateVariableRequestData `json:"data"`
}

type jobTemplateVariableRequestData struct {
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
}

// Create creates a new variable on options.JobTemplateID.
func (s *AnsibleJobTemplateVariablesService) Create(ctx context.Context, options AnsibleJobTemplateVariableCreateOptions) (*AnsibleJobTemplateVariable, error) {
	if options.JobTemplateID == "" {
		return nil, fmt.Errorf("stackweaver: Create requires a JobTemplateID")
	}
	if options.Key == "" {
		return nil, fmt.Errorf("stackweaver: Create requires a Key")
	}

	attributes := map[string]any{
		"key":       options.Key,
		"value":     options.Value,
		"sensitive": options.Sensitive,
		"hcl":       options.HCL,
	}
	if options.Description != "" {
		attributes["description"] = options.Description
	}
	if options.Category != "" {
		attributes["category"] = options.Category
	}

	reqBody, err := s.client.plain.Marshal(jobTemplateVariableRequest{
		Data: jobTemplateVariableRequestData{
			Type:       jobTemplateVariableResourceType,
			Attributes: attributes,
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/vars", url.PathEscape(options.JobTemplateID))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	var resource jobTemplateVariableResource
	if err := s.client.jsonapi.Unmarshal(respBody, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(options.JobTemplateID), nil
}

// List returns all variables on the given job template.
func (s *AnsibleJobTemplateVariablesService) List(ctx context.Context, jobTemplateID string) ([]*AnsibleJobTemplateVariable, error) {
	if jobTemplateID == "" {
		return nil, fmt.Errorf("stackweaver: List requires a JobTemplateID")
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/vars", url.PathEscape(jobTemplateID))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resources []jobTemplateVariableResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, err
	}

	variables := make([]*AnsibleJobTemplateVariable, len(resources))
	for i := range resources {
		variables[i] = resources[i].toModel(jobTemplateID)
	}
	return variables, nil
}

// Read returns the variable identified by variableID on the given template.
// There is no single-variable GET endpoint, so it fetches the template's
// variable list and selects by id; an absent variable surfaces as ErrNotFound.
func (s *AnsibleJobTemplateVariablesService) Read(ctx context.Context, jobTemplateID, variableID string) (*AnsibleJobTemplateVariable, error) {
	if variableID == "" {
		return nil, fmt.Errorf("stackweaver: Read requires a variableID")
	}

	variables, err := s.List(ctx, jobTemplateID)
	if err != nil {
		return nil, err
	}
	for _, v := range variables {
		if v.ID == variableID {
			return v, nil
		}
	}
	return nil, ErrNotFound
}

// Update updates the variable identified by variableID on the given template.
func (s *AnsibleJobTemplateVariablesService) Update(ctx context.Context, jobTemplateID, variableID string, options AnsibleJobTemplateVariableUpdateOptions) (*AnsibleJobTemplateVariable, error) {
	if jobTemplateID == "" {
		return nil, fmt.Errorf("stackweaver: Update requires a JobTemplateID")
	}
	if variableID == "" {
		return nil, fmt.Errorf("stackweaver: Update requires a variableID")
	}

	attributes := map[string]any{}
	if options.Key != "" {
		attributes["key"] = options.Key
	}
	if options.Value != "" {
		attributes["value"] = options.Value
	}
	if options.Description != "" {
		attributes["description"] = options.Description
	}
	if options.Category != "" {
		attributes["category"] = options.Category
	}
	if options.HCL != nil {
		attributes["hcl"] = *options.HCL
	}
	if options.Sensitive != nil {
		attributes["sensitive"] = *options.Sensitive
	}

	reqBody, err := s.client.plain.Marshal(jobTemplateVariableRequest{
		Data: jobTemplateVariableRequestData{
			Type:       jobTemplateVariableResourceType,
			Attributes: attributes,
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/vars/%s", url.PathEscape(jobTemplateID), url.PathEscape(variableID))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	var resource jobTemplateVariableResource
	if err := s.client.jsonapi.Unmarshal(respBody, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(jobTemplateID), nil
}

// Delete removes the variable identified by variableID on the given template.
func (s *AnsibleJobTemplateVariablesService) Delete(ctx context.Context, jobTemplateID, variableID string) error {
	if jobTemplateID == "" {
		return fmt.Errorf("stackweaver: Delete requires a JobTemplateID")
	}
	if variableID == "" {
		return fmt.Errorf("stackweaver: Delete requires a variableID")
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/vars/%s", url.PathEscape(jobTemplateID), url.PathEscape(variableID))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}
