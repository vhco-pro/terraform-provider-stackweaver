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
// Every verb speaks JSON:API (handlers/ansible/schedules.go):
//   - Create sends a "schedules" document whose attributes carry the target ids
//     and the date window.
//   - Update sends a "schedules" document whose attributes are the in-place
//     updatable ones (name, description, cron-expression, timezone, config).
//     Each is optional and an omitted one is left as it is; the type, target and
//     date window cannot change in place (they are ForceNew).
//   - Create, Read and Update all answer a "schedules" resource object
//     (formatScheduleResponse), decoded into the one AnsibleSchedule model.
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

// scheduleUpdateAttributes are the attributes of the update document
// (UpdateScheduleRequest), field for field AnsibleScheduleUpdateOptions so one
// converts to the other. A nil pointer or nil Config is omitted, which the
// server reads as "leave this attribute as it is".
type scheduleUpdateAttributes struct {
	Name           *string                `json:"name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	CronExpression *string                `json:"cron-expression,omitempty"`
	Timezone       *string                `json:"timezone,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

// scheduleJSONAPIResource mirrors the "schedules" resource object every verb
// answers with (formatScheduleResponse). The server relates no workflow, so a
// workflow target is not echoed back.
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

	return s.decode(respBody)
}

// Read returns the schedule identified by id.
func (s *AnsibleSchedulesService) Read(ctx context.Context, id string) (*AnsibleSchedule, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}
	path := fmt.Sprintf("/ansible/schedules/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
}

// Update applies the in-place-updatable fields to the schedule identified by id.
func (s *AnsibleSchedulesService) Update(ctx context.Context, id string, options AnsibleScheduleUpdateOptions) (*AnsibleSchedule, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Update requires an id")
	}

	reqBody, err := s.client.jsonapi.Marshal(scheduleResourceType, scheduleUpdateAttributes(options))
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/schedules/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodPatch, path, reqBody)
	if err != nil {
		return nil, err
	}

	return s.decode(body)
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

// decode unmarshals a single "schedules" resource document into the model.
func (s *AnsibleSchedulesService) decode(body []byte) (*AnsibleSchedule, error) {
	var resource scheduleJSONAPIResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}
