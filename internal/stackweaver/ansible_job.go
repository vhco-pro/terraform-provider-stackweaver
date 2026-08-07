// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// jobResourceType is the JSON:API `type` member for an Ansible job
// (handlers/ansible/jobs.go formatJobResponse).
const jobResourceType = "ansible-jobs"

// AnsibleJobsService is a native service that LAUNCHES Ansible jobs from a job
// template and reads back the resulting execution. It backs a lifecycle-trigger
// resource modeled on tfe_workspace_run - a job is immutable execution history,
// not reconciled config. Built stand-alone via NewAnsibleJobsService.
//
// Backing-API gap (IMPORTANT): the template-launch endpoint
// (POST /ansible/job-templates/:id/launch, handler LaunchFromTemplate) currently
// accepts ONLY an extra-vars override - limit/tags/inventory overrides are NOT
// wired on this path today (the org-scoped POST /organizations/:name/ansible/jobs
// accepts them but is playbook+inventory driven, not template driven). So Launch
// here sends only extra-vars.
//
// Envelope is JSON:API throughout (data.attributes with hyphenated keys).
//
// Wire contract (all paths relative to /api/v2):
//
//	Launch: POST   /ansible/job-templates/:job_template_id/launch
//	Read:   GET    /ansible/jobs/:id
//	Cancel: POST   /ansible/jobs/:id/actions/cancel
//	Delete: DELETE /ansible/jobs/:id
type AnsibleJobsService struct {
	client *Client
}

// NewAnsibleJobsService constructs the service over c.
func NewAnsibleJobsService(c *Client) *AnsibleJobsService {
	return &AnsibleJobsService{client: c}
}

// AnsibleJob is the native representation of a launched job.
type AnsibleJob struct {
	ID               string
	JobTemplateID    string
	Status           string
	ExtraVars        map[string]interface{}
	StartedAt        string
	FinishedAt       string
	ExitCode         *int
	HostsOk          int
	HostsChanged     int
	HostsFailed      int
	HostsUnreachable int
}

// terminalJobStatuses are the statuses at which a job has stopped executing.
var terminalJobStatuses = map[string]bool{
	"successful": true,
	"failed":     true,
	"canceled":   true,
	"error":      true,
}

// IsTerminalJobStatus reports whether status is a terminal (stopped) job status.
func IsTerminalJobStatus(status string) bool {
	return terminalJobStatuses[status]
}

// launchFromTemplateRequest is the JSON:API launch body. Only extra-vars is
// honored by the backend today (see the service doc gap note).
type launchFromTemplateRequest struct {
	Data launchFromTemplateData `json:"data"`
}

type launchFromTemplateData struct {
	Type       string                  `json:"type"`
	Attributes launchFromTemplateAttrs `json:"attributes"`
}

type launchFromTemplateAttrs struct {
	ExtraVars map[string]interface{} `json:"extra-vars,omitempty"`
}

// jobResource mirrors the JSON:API job resource object returned by the backend.
type jobResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Status           string                 `json:"status"`
		ExtraVars        map[string]interface{} `json:"extra-vars"`
		StartedAt        *string                `json:"started-at"`
		FinishedAt       *string                `json:"finished-at"`
		ExitCode         *int                   `json:"exit-code"`
		HostsOk          int                    `json:"hosts-ok"`
		HostsChanged     int                    `json:"hosts-changed"`
		HostsFailed      int                    `json:"hosts-failed"`
		HostsUnreachable int                    `json:"hosts-unreachable"`
	} `json:"attributes"`
	Relationships struct {
		Template jsonAPIRelationship `json:"template"`
	} `json:"relationships"`
}

func (r *jobResource) toModel() *AnsibleJob {
	j := &AnsibleJob{
		ID:               r.ID,
		Status:           r.Attributes.Status,
		ExtraVars:        r.Attributes.ExtraVars,
		ExitCode:         r.Attributes.ExitCode,
		HostsOk:          r.Attributes.HostsOk,
		HostsChanged:     r.Attributes.HostsChanged,
		HostsFailed:      r.Attributes.HostsFailed,
		HostsUnreachable: r.Attributes.HostsUnreachable,
	}
	if r.Attributes.StartedAt != nil {
		j.StartedAt = *r.Attributes.StartedAt
	}
	if r.Attributes.FinishedAt != nil {
		j.FinishedAt = *r.Attributes.FinishedAt
	}
	if r.Relationships.Template.Data != nil {
		j.JobTemplateID = r.Relationships.Template.Data.ID
	}
	return j
}

// Launch launches a job from jobTemplateID with an optional extra-vars override.
func (s *AnsibleJobsService) Launch(ctx context.Context, jobTemplateID string, extraVars map[string]interface{}) (*AnsibleJob, error) {
	if jobTemplateID == "" {
		return nil, fmt.Errorf("stackweaver: Launch requires a JobTemplateID")
	}

	reqBody, err := s.client.plain.Marshal(launchFromTemplateRequest{
		Data: launchFromTemplateData{
			Type:       jobResourceType,
			Attributes: launchFromTemplateAttrs{ExtraVars: extraVars},
		},
	})
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/ansible/job-templates/%s/launch", url.PathEscape(jobTemplateID))
	respBody, err := s.client.do(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}
	return s.decode(respBody)
}

// Read returns the job identified by id. A missing job surfaces as ErrNotFound.
func (s *AnsibleJobsService) Read(ctx context.Context, id string) (*AnsibleJob, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Read requires an id")
	}
	path := fmt.Sprintf("/ansible/jobs/%s", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return s.decode(body)
}

// Cancel requests cancellation of the still-running job identified by id.
func (s *AnsibleJobsService) Cancel(ctx context.Context, id string) (*AnsibleJob, error) {
	if id == "" {
		return nil, fmt.Errorf("stackweaver: Cancel requires an id")
	}
	path := fmt.Sprintf("/ansible/jobs/%s/actions/cancel", url.PathEscape(id))
	body, err := s.client.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	return s.decode(body)
}

// Delete drops the job history record identified by id.
func (s *AnsibleJobsService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("stackweaver: Delete requires an id")
	}
	path := fmt.Sprintf("/ansible/jobs/%s", url.PathEscape(id))
	_, err := s.client.do(ctx, http.MethodDelete, path, nil)
	return err
}

// decode unmarshals a single-job JSON:API response into the public model.
func (s *AnsibleJobsService) decode(body []byte) (*AnsibleJob, error) {
	var resource jobResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}
	return resource.toModel(), nil
}
