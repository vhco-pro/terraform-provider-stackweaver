// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// AnsibleNotificationAttachmentsService is a native service that binds a
// notification template (channel) to a single target - a job template OR a
// workflow - with per-trigger flags. It is a create/delete-only relationship
// resource: there is no update path (every field is ForceNew).
//
// Every verb speaks JSON:API: the create request is an
// "ansible-notification-attachments" document whose attributes are the three
// trigger flags (on-started, on-success, on-failure) and whose relationships name
// the notification-template and exactly one of job-template or workflow
// (NotificationAttachmentRequest); the answers are resource objects of the same
// type, related the same way.
//
// Read gap: there is NO GET-by-attachment-id route. Attachments are read through
// the TARGET's notifications listing. Only the job-template read path exists
// (GET /ansible/job-templates/:id/notifications); there is NO workflow
// notifications route, so a workflow-target attachment cannot be refreshed
// server-side (ReadByJobTemplate is the only confirmed read).
//
// Wire contract (all paths relative to /api/v2):
//
//	Create: POST   /organizations/:org/ansible/notification-attachments
//	Read:   GET    /ansible/job-templates/:job_template_id/notifications  (filter by id)
//	Delete: DELETE /organizations/:org/ansible/notification-attachments/:attachment_id
type AnsibleNotificationAttachmentsService struct {
	client *Client
}

// NewAnsibleNotificationAttachmentsService constructs the service over c.
func NewAnsibleNotificationAttachmentsService(c *Client) *AnsibleNotificationAttachmentsService {
	return &AnsibleNotificationAttachmentsService{client: c}
}

// AnsibleNotificationAttachment is the native representation of an attachment.
// Exactly one of JobTemplateID / WorkflowID is set.
type AnsibleNotificationAttachment struct {
	ID                     string
	NotificationTemplateID string
	JobTemplateID          string
	WorkflowID             string
	OnStarted              bool
	OnSuccess              bool
	OnFailure              bool
}

// AnsibleNotificationAttachmentCreateOptions are the fields accepted on create.
// Organization is the org NAME scoping the create URL; exactly one of
// JobTemplateID / WorkflowID must be set.
type AnsibleNotificationAttachmentCreateOptions struct {
	Organization           string
	NotificationTemplateID string
	JobTemplateID          string
	WorkflowID             string
	OnStarted              bool
	OnSuccess              bool
	OnFailure              bool
}

// The JSON:API resource types the attach document names that no other service
// declares (the job template and notification template types live with their
// services).
const (
	notificationAttachmentResourceType = "ansible-notification-attachments"
	workflowResourceType               = "ansible-workflows"
)

// attachRequest builds the attach document. The three flags are always sent,
// so the server's defaults (on-success and on-failure on when omitted) never
// override what the caller chose.
func attachRequest(options AnsibleNotificationAttachmentCreateOptions) jsonAPIRequest {
	relationships := map[string]jsonAPIRelationship{
		"notification-template": {Data: &jsonAPIResourceRef{Type: notificationTemplateResourceType, ID: options.NotificationTemplateID}},
	}
	if options.JobTemplateID != "" {
		relationships["job-template"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: jobTemplateResourceType, ID: options.JobTemplateID}}
	}
	if options.WorkflowID != "" {
		relationships["workflow"] = jsonAPIRelationship{Data: &jsonAPIResourceRef{Type: workflowResourceType, ID: options.WorkflowID}}
	}
	return jsonAPIRequest{Data: jsonAPIRequestData{
		Type: notificationAttachmentResourceType,
		Attributes: map[string]any{
			"on-started": options.OnStarted,
			"on-success": options.OnSuccess,
			"on-failure": options.OnFailure,
		},
		Relationships: relationships,
	}}
}

// attachmentResource mirrors the JSON:API resource object the backend returns.
type attachmentResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		OnStarted bool `json:"on-started"`
		OnSuccess bool `json:"on-success"`
		OnFailure bool `json:"on-failure"`
	} `json:"attributes"`
	Relationships struct {
		NotificationTemplate jsonAPIRelationship `json:"notification-template"`
		JobTemplate          jsonAPIRelationship `json:"job-template"`
		Workflow             jsonAPIRelationship `json:"workflow"`
	} `json:"relationships"`
}

func (r *attachmentResource) toModel() *AnsibleNotificationAttachment {
	a := &AnsibleNotificationAttachment{
		ID:        r.ID,
		OnStarted: r.Attributes.OnStarted,
		OnSuccess: r.Attributes.OnSuccess,
		OnFailure: r.Attributes.OnFailure,
	}
	if r.Relationships.NotificationTemplate.Data != nil {
		a.NotificationTemplateID = r.Relationships.NotificationTemplate.Data.ID
	}
	if r.Relationships.JobTemplate.Data != nil {
		a.JobTemplateID = r.Relationships.JobTemplate.Data.ID
	}
	if r.Relationships.Workflow.Data != nil {
		a.WorkflowID = r.Relationships.Workflow.Data.ID
	}
	return a
}

// Create attaches a notification template to a target.
func (s *AnsibleNotificationAttachmentsService) Create(ctx context.Context, options AnsibleNotificationAttachmentCreateOptions) (*AnsibleNotificationAttachment, error) {
	if options.Organization == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an Organization")
	}
	if options.NotificationTemplateID == "" {
		return nil, fmt.Errorf("stackweaver: Create requires a NotificationTemplateID")
	}
	if (options.JobTemplateID == "") == (options.WorkflowID == "") {
		return nil, fmt.Errorf("stackweaver: exactly one of JobTemplateID or WorkflowID is required")
	}

	reqBody, err := s.client.plain.Marshal(attachRequest(options))
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/organizations/%s/ansible/notification-attachments", url.PathEscape(options.Organization))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	var resource attachmentResource
	if err := s.client.jsonapi.Unmarshal(respBody, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}

// ReadByJobTemplate resolves an attachment by id from its job-template target's
// notifications listing. A missing id surfaces as ErrNotFound. This is the only
// confirmed read path - workflow-target attachments have no listing route.
func (s *AnsibleNotificationAttachmentsService) ReadByJobTemplate(ctx context.Context, jobTemplateID, id string) (*AnsibleNotificationAttachment, error) {
	if jobTemplateID == "" {
		return nil, fmt.Errorf("stackweaver: ReadByJobTemplate requires a JobTemplateID")
	}
	if id == "" {
		return nil, fmt.Errorf("stackweaver: ReadByJobTemplate requires an id")
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/notifications", url.PathEscape(jobTemplateID))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resources []attachmentResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, err
	}
	for i := range resources {
		if resources[i].ID == id {
			return resources[i].toModel(), nil
		}
	}
	return nil, ErrNotFound
}

// Delete removes the attachment identified by id from options.Organization.
func (s *AnsibleNotificationAttachmentsService) Delete(ctx context.Context, organization, id string) error {
	if organization == "" {
		return fmt.Errorf("stackweaver: Delete requires an Organization")
	}
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}
	path := fmt.Sprintf("/organizations/%s/ansible/notification-attachments/%s", url.PathEscape(organization), url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}
