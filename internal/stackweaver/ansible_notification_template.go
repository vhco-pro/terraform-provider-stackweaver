// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// notificationTemplateResourceType is the JSON:API `type` member of a
// notification-template resource object and of the create and update documents
// the backend accepts (handlers/ansible/notifications.go).
const notificationTemplateResourceType = "ansible-notification-templates"

// AnsibleNotificationTemplatesService is a native service for org-scoped Ansible
// notification templates (webhook / email / Teams channels). It is built
// stand-alone via NewAnsibleNotificationTemplatesService rather than hung off the
// Client struct.
//
// Every verb speaks JSON:API: create and update send an
// "ansible-notification-templates" document whose attributes are name,
// description, notification-type, config and secret (NotificationTemplateRequest),
// and the answers are resource objects of the same type with hyphenated keys
// (notification-type, has-secret). The sensitive secret is write-only - the
// backend returns only has-secret, never the value.
//
// Read gap: there is NO GET-by-id route. The only read surface is the org-scoped
// List; Read lists and filters client-side by id.
//
// Wire contract (all paths relative to /api/v2):
//
//	Create: POST   /organizations/:org/ansible/notification-templates
//	List:   GET    /organizations/:org/ansible/notification-templates
//	Update: PATCH  /ansible/notification-templates/:id
//	Delete: DELETE /ansible/notification-templates/:id
type AnsibleNotificationTemplatesService struct {
	client *Client
}

// NewAnsibleNotificationTemplatesService constructs the service over c.
func NewAnsibleNotificationTemplatesService(c *Client) *AnsibleNotificationTemplatesService {
	return &AnsibleNotificationTemplatesService{client: c}
}

// AnsibleNotificationTemplate is the native representation of a notification
// template. Secret is never populated from the API (write-only).
type AnsibleNotificationTemplate struct {
	ID          string
	Name        string
	Description string
	Type        string
	Config      map[string]interface{}
	HasSecret   bool
}

// AnsibleNotificationTemplateCreateOptions are the fields accepted on create.
// Organization is the org NAME scoping the create URL.
type AnsibleNotificationTemplateCreateOptions struct {
	Organization string
	Name         string
	Description  string
	Type         string
	Config       map[string]interface{}
	// Secret, when non-nil, sets the channel's sensitive value. Nil leaves it unset.
	Secret *string
}

// AnsibleNotificationTemplateUpdateOptions are the fields accepted on update. A
// nil pointer leaves the field unchanged (Config nil = unchanged).
type AnsibleNotificationTemplateUpdateOptions struct {
	Name        *string
	Description *string
	Config      map[string]interface{}
	// Secret: nil leaves unchanged; a pointer to "" clears it; a value sets it.
	Secret *string
}

// notificationTemplateAttributes are the attributes of the create and update
// document. On update the server keeps the name when it is empty, always sets
// the description, keeps the config when it is absent, and keeps the secret when
// it is absent (an empty secret clears it); notification-type is not updatable
// and is omitted there.
type notificationTemplateAttributes struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	NotificationType string                 `json:"notification-type,omitempty"`
	Config           map[string]interface{} `json:"config,omitempty"`
	Secret           *string                `json:"secret,omitempty"`
}

// notificationTemplateResource mirrors the JSON:API resource object returned by
// the backend so jsonAPICodec.Unmarshal can decode straight into it.
type notificationTemplateResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name             string                 `json:"name"`
		Description      string                 `json:"description"`
		NotificationType string                 `json:"notification-type"`
		Config           map[string]interface{} `json:"config"`
		HasSecret        bool                   `json:"has-secret"`
	} `json:"attributes"`
}

func (r *notificationTemplateResource) toModel() *AnsibleNotificationTemplate {
	return &AnsibleNotificationTemplate{
		ID:          r.ID,
		Name:        r.Attributes.Name,
		Description: r.Attributes.Description,
		Type:        r.Attributes.NotificationType,
		Config:      r.Attributes.Config,
		HasSecret:   r.Attributes.HasSecret,
	}
}

// Create creates a notification template under options.Organization.
func (s *AnsibleNotificationTemplatesService) Create(ctx context.Context, options AnsibleNotificationTemplateCreateOptions) (*AnsibleNotificationTemplate, error) {
	if options.Organization == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an Organization")
	}

	reqBody, err := s.client.jsonapi.Marshal(notificationTemplateResourceType, notificationTemplateAttributes{
		Name:             options.Name,
		Description:      options.Description,
		NotificationType: options.Type,
		Config:           options.Config,
		Secret:           options.Secret,
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/organizations/%s/ansible/notification-templates", url.PathEscape(options.Organization))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}
	return s.decode(respBody)
}

// List returns all notification templates in the org named organization.
func (s *AnsibleNotificationTemplatesService) List(ctx context.Context, organization string) ([]*AnsibleNotificationTemplate, error) {
	if organization == "" {
		return nil, fmt.Errorf("stackweaver: List requires an Organization")
	}

	path := fmt.Sprintf("/organizations/%s/ansible/notification-templates", url.PathEscape(organization))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resources []notificationTemplateResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, err
	}
	templates := make([]*AnsibleNotificationTemplate, len(resources))
	for i := range resources {
		templates[i] = resources[i].toModel()
	}
	return templates, nil
}

// Read resolves a template by id via List (there is no GET-by-id). A template
// absent from the org listing surfaces as ErrNotFound so Read can drop it.
func (s *AnsibleNotificationTemplatesService) Read(ctx context.Context, organization, id string) (*AnsibleNotificationTemplate, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}
	templates, err := s.List(ctx, organization)
	if err != nil {
		return nil, err
	}
	for _, t := range templates {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, ErrNotFound
}

// Update updates the template identified by id.
func (s *AnsibleNotificationTemplatesService) Update(ctx context.Context, id string, options AnsibleNotificationTemplateUpdateOptions) (*AnsibleNotificationTemplate, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Update requires an id")
	}

	attributes := notificationTemplateAttributes{
		Config: options.Config,
		Secret: options.Secret,
	}
	if options.Name != nil {
		attributes.Name = *options.Name
	}
	if options.Description != nil {
		attributes.Description = *options.Description
	}

	reqBody, err := s.client.jsonapi.Marshal(notificationTemplateResourceType, attributes)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/notification-templates/%s", url.PathEscape(id))
	respBody, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}
	return s.decode(respBody)
}

// Delete removes the template identified by id.
func (s *AnsibleNotificationTemplatesService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}
	path := fmt.Sprintf("/ansible/notification-templates/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// decode unmarshals a single-template JSON:API response into the public model.
func (s *AnsibleNotificationTemplatesService) decode(body []byte) (*AnsibleNotificationTemplate, error) {
	var resource notificationTemplateResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}
