// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// scheduleResourceType is the JSON:API `type` member the backend requires on a
// schedule create body and stamps on the create response (handler
// CreateScheduleRequest / formatScheduleResponse). Note it is "schedules", NOT
// "ansible-schedules".
const scheduleResourceType = "schedules"

// AnsibleSchedulesService is a native service for cron schedules that trigger an
// Ansible target (job template / inventory-source sync / playbook sync /
// workflow). It is built stand-alone via NewAnsibleSchedulesService.
//
// Envelope is INCONSISTENT across verbs (confirmed in handlers/ansible/schedules.go):
//   - Create speaks JSON:API: body under data.attributes with hyphenated keys,
//     data.type = "schedules"; the response is a JSON:API resource object.
//   - Read (GET) and Update (PATCH) return the RAW GORM model (c.JSON(200,
//     schedule)) with underscore json keys — NOT a JSON:API envelope.
//   - Update's request body is plain JSON ({name, description, cron_expression,
//     timezone, config}) with underscore keys — NOT JSON:API, and it does NOT
//     accept target/type/date-window changes (those are ForceNew).
//
// The service therefore decodes create responses with the JSON:API codec and
// read/update responses with the plain-JSON codec, both flattening to the same
// AnsibleSchedule model.
//
// Wire contract (all paths relative to /api/v2):
//
//	Create: POST   /organizations/:org/ansible/schedules
//	Read:   GET    /ansible/schedules/:schedule_id
//	Update: PATCH  /ansible/schedules/:schedule_id
//	Delete: DELETE /ansible/schedules/:schedule_id
//
// The enable/disable/run-now actions (POST .../actions/{enable,disable,run-now})
// are ephemeral, not CRUD, and are not modeled here.
type AnsibleSchedulesService struct {
	client *Client
}

// NewAnsibleSchedulesService constructs the service over c.
func NewAnsibleSchedulesService(c *Client) *AnsibleSchedulesService {
	return &AnsibleSchedulesService{client: c}
}

// AnsibleSchedule is the native representation of a schedule.
type AnsibleSchedule struct {
	ID                string
	OrganizationID    string
	Name              string
	Description       string
	Type              string
	Status            string
	JobTemplateID     string
	InventorySourceID string
	PlaybookID        string
	WorkflowID        string
	CronExpression    string
	Timezone          string
	StartDateTime     string
	EndDateTime       string
	Config            map[string]interface{}
	NextRunAt         string
	LastRunAt         string
	LastRunStatus     string
	LastJobID         string
	RunCount          int
}

// AnsibleScheduleCreateOptions are the fields accepted on create. Organization is
// the org NAME scoping the create URL. Exactly one target id must match Type.
type AnsibleScheduleCreateOptions struct {
	Organization      string
	Name              string
	Description       string
	Type              string
	JobTemplateID     string
	InventorySourceID string
	PlaybookID        string
	WorkflowID        string
	CronExpression    string
	Timezone          string
	StartDateTime     string
	EndDateTime       string
	Config            map[string]interface{}
}

// AnsibleScheduleUpdateOptions are the in-place-updatable fields (a nil pointer /
// nil Config leaves the field unchanged). Type/target/date-window are ForceNew.
type AnsibleScheduleUpdateOptions struct {
	Name           *string
	Description    *string
	CronExpression *string
	Timezone       *string
	Config         map[string]interface{}
}

// scheduleCreateRequest is the JSON:API create envelope.
type scheduleCreateRequest struct {
	Data scheduleCreateData `json:"data"`
}

type scheduleCreateData struct {
	Type       string                   `json:"type"`
	Attributes scheduleCreateAttributes `json:"attributes"`
}

type scheduleCreateAttributes struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	ScheduleType      string                 `json:"schedule-type"`
	JobTemplateID     string                 `json:"job-template-id,omitempty"`
	InventorySourceID string                 `json:"inventory-source-id,omitempty"`
	PlaybookID        string                 `json:"playbook-id,omitempty"`
	WorkflowID        string                 `json:"workflow-id,omitempty"`
	CronExpression    string                 `json:"cron-expression"`
	Timezone          string                 `json:"timezone"`
	StartDateTime     string                 `json:"start-date-time,omitempty"`
	EndDateTime       string                 `json:"end-date-time,omitempty"`
	Config            map[string]interface{} `json:"config,omitempty"`
}

// scheduleUpdateRequest is the plain-JSON update body (underscore keys).
type scheduleUpdateRequest struct {
	Name           *string                `json:"name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	CronExpression *string                `json:"cron_expression,omitempty"`
	Timezone       *string                `json:"timezone,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

// scheduleJSONAPIResource mirrors the JSON:API resource object returned by Create.
type scheduleJSONAPIResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name           string                 `json:"name"`
		Description    string                 `json:"description"`
		ScheduleType   string                 `json:"schedule-type"`
		Status         string                 `json:"status"`
		CronExpression string                 `json:"cron-expression"`
		Timezone       string                 `json:"timezone"`
		Config         map[string]interface{} `json:"config"`
		StartDateTime  string                 `json:"start-date-time"`
		EndDateTime    string                 `json:"end-date-time"`
		NextRunAt      string                 `json:"next-run-at"`
		LastRunAt      string                 `json:"last-run-at"`
		LastRunStatus  string                 `json:"last-run-status"`
		RunCount       int                    `json:"run-count"`
	} `json:"attributes"`
	Relationships struct {
		Organization    jsonAPIRelationship `json:"organization"`
		JobTemplate     jsonAPIRelationship `json:"job-template"`
		InventorySource jsonAPIRelationship `json:"inventory-source"`
		Playbook        jsonAPIRelationship `json:"playbook"`
		LastJob         jsonAPIRelationship `json:"last-job"`
	} `json:"relationships"`
}

func (r *scheduleJSONAPIResource) toModel() *AnsibleSchedule {
	s := &AnsibleSchedule{
		ID:             r.ID,
		Name:           r.Attributes.Name,
		Description:    r.Attributes.Description,
		Type:           r.Attributes.ScheduleType,
		Status:         r.Attributes.Status,
		CronExpression: r.Attributes.CronExpression,
		Timezone:       r.Attributes.Timezone,
		Config:         r.Attributes.Config,
		StartDateTime:  r.Attributes.StartDateTime,
		EndDateTime:    r.Attributes.EndDateTime,
		NextRunAt:      r.Attributes.NextRunAt,
		LastRunAt:      r.Attributes.LastRunAt,
		LastRunStatus:  r.Attributes.LastRunStatus,
		RunCount:       r.Attributes.RunCount,
	}
	if r.Relationships.Organization.Data != nil {
		s.OrganizationID = r.Relationships.Organization.Data.ID
	}
	if r.Relationships.JobTemplate.Data != nil {
		s.JobTemplateID = r.Relationships.JobTemplate.Data.ID
	}
	if r.Relationships.InventorySource.Data != nil {
		s.InventorySourceID = r.Relationships.InventorySource.Data.ID
	}
	if r.Relationships.Playbook.Data != nil {
		s.PlaybookID = r.Relationships.Playbook.Data.ID
	}
	if r.Relationships.LastJob.Data != nil {
		s.LastJobID = r.Relationships.LastJob.Data.ID
	}
	return s
}

// scheduleRawModel mirrors the RAW GORM model returned by Read and Update.
type scheduleRawModel struct {
	ID                string                 `json:"id"`
	OrganizationID    string                 `json:"organization_id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Type              string                 `json:"type"`
	Status            string                 `json:"status"`
	JobTemplateID     string                 `json:"job_template_id"`
	InventorySourceID string                 `json:"inventory_source_id"`
	PlaybookID        string                 `json:"playbook_id"`
	WorkflowID        string                 `json:"workflow_id"`
	CronExpression    string                 `json:"cron_expression"`
	Timezone          string                 `json:"timezone"`
	StartDateTime     string                 `json:"start_date_time"`
	EndDateTime       string                 `json:"end_date_time"`
	Config            map[string]interface{} `json:"config"`
	NextRunAt         string                 `json:"next_run_at"`
	LastRunAt         string                 `json:"last_run_at"`
	LastJobID         string                 `json:"last_job_id"`
	LastRunStatus     string                 `json:"last_run_status"`
	RunCount          int                    `json:"run_count"`
}

func (r *scheduleRawModel) toModel() *AnsibleSchedule {
	return &AnsibleSchedule{
		ID:                r.ID,
		OrganizationID:    r.OrganizationID,
		Name:              r.Name,
		Description:       r.Description,
		Type:              r.Type,
		Status:            r.Status,
		JobTemplateID:     r.JobTemplateID,
		InventorySourceID: r.InventorySourceID,
		PlaybookID:        r.PlaybookID,
		WorkflowID:        r.WorkflowID,
		CronExpression:    r.CronExpression,
		Timezone:          r.Timezone,
		StartDateTime:     r.StartDateTime,
		EndDateTime:       r.EndDateTime,
		Config:            r.Config,
		NextRunAt:         r.NextRunAt,
		LastRunAt:         r.LastRunAt,
		LastJobID:         r.LastJobID,
		LastRunStatus:     r.LastRunStatus,
		RunCount:          r.RunCount,
	}
}

// Create creates a schedule under options.Organization.
func (s *AnsibleSchedulesService) Create(ctx context.Context, options AnsibleScheduleCreateOptions) (*AnsibleSchedule, error) {
	if options.Organization == "" {
		return nil, fmt.Errorf("stackweaver: Create requires an Organization")
	}

	reqBody, err := s.client.plain.Marshal(scheduleCreateRequest{
		Data: scheduleCreateData{
			Type: scheduleResourceType,
			Attributes: scheduleCreateAttributes{
				Name:              options.Name,
				Description:       options.Description,
				ScheduleType:      options.Type,
				JobTemplateID:     options.JobTemplateID,
				InventorySourceID: options.InventorySourceID,
				PlaybookID:        options.PlaybookID,
				WorkflowID:        options.WorkflowID,
				CronExpression:    options.CronExpression,
				Timezone:          options.Timezone,
				StartDateTime:     options.StartDateTime,
				EndDateTime:       options.EndDateTime,
				Config:            options.Config,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/organizations/%s/ansible/schedules", url.PathEscape(options.Organization))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	var resource scheduleJSONAPIResource
	if err := s.client.jsonapi.Unmarshal(respBody, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}

// Read returns the schedule identified by id (raw-model envelope).
func (s *AnsibleSchedulesService) Read(ctx context.Context, id string) (*AnsibleSchedule, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}
	path := fmt.Sprintf("/ansible/schedules/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var raw scheduleRawModel
	if err := s.client.plain.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return raw.toModel(), nil
}

// Update applies the in-place-updatable fields to the schedule identified by id
// (raw-model envelope in the response).
func (s *AnsibleSchedulesService) Update(ctx context.Context, id string, options AnsibleScheduleUpdateOptions) (*AnsibleSchedule, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Update requires an id")
	}

	reqBody, err := s.client.plain.Marshal(scheduleUpdateRequest{
		Name:           options.Name,
		Description:    options.Description,
		CronExpression: options.CronExpression,
		Timezone:       options.Timezone,
		Config:         options.Config,
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/schedules/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	var raw scheduleRawModel
	if err := s.client.plain.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return raw.toModel(), nil
}

// Delete removes the schedule identified by id.
func (s *AnsibleSchedulesService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}
	path := fmt.Sprintf("/ansible/schedules/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}
