// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// RunnersService is the native service backing the stackweaver_runners data
// source — a read-only observability view over an organization's self-hosted
// runner fleet. Runners self-register and heartbeat out of band; the provider
// never mutates the fleet, so this service exposes only reads.
//
// Envelope note: unlike the plain-JSON VCS endpoints in this package, the
// runners endpoints emit a JSON:API-shaped body whose attribute keys are
// dash-cased (agent-pool-id, runner-type, last-heartbeat-at, ...), with a
// meta.pagination block (current-page, page-size, total-count, total-pages).
// Wire contract (paths relative to /api/v2):
//
//	List:  GET /organizations/:name/runners        (+ filter[...] + page[...])
//	Stats: GET /organizations/:name/runners/stats
type RunnersService struct {
	client *Client
}

// NewRunnersService constructs a self-contained RunnersService over c. Data
// sources build it via stackweaver.NewRunnersService(cc.Stackweaver).
func NewRunnersService(c *Client) *RunnersService {
	return &RunnersService{client: c}
}

// Runner is the native representation of a single runner, flattened from the
// JSON:API resource object into the shape the Terraform data source consumes.
type Runner struct {
	ID               string
	Name             string
	Description      string
	AgentPoolID      string
	RunnerType       string
	Status           string
	Hostname         string
	OSType           string
	AgentVersion     string
	Labels           []string
	TerraformVersion string
	AnsibleVersion   string
	LastHeartbeatAt  string
}

// RunnerStats is the fleet summary returned by the stats endpoint.
type RunnerStats struct {
	Total   int
	Online  int
	Offline int
}

// RunnerListOptions are the filter options for List. Organization is required;
// the rest map to the server-side filter[...] query params and are omitted when
// empty.
type RunnerListOptions struct {
	Organization string
	AgentPoolID  string
	RunnerType   string
	Status       string
}

// runnerResource mirrors the dash-cased JSON:API resource object the runners
// handler returns (buildRunnerResponse in handlers/runners.go).
type runnerResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name             string   `json:"name"`
		Description      string   `json:"description"`
		AgentPoolID      string   `json:"agent-pool-id"`
		RunnerType       string   `json:"runner-type"`
		Status           string   `json:"status"`
		Hostname         string   `json:"hostname"`
		OSType           string   `json:"os-type"`
		AgentVersion     string   `json:"agent-version"`
		Labels           []string `json:"labels"`
		TerraformVersion string   `json:"terraform-version"`
		AnsibleVersion   string   `json:"ansible-version"`
		LastHeartbeatAt  *string  `json:"last-heartbeat-at"`
	} `json:"attributes"`
}

// toModel flattens a wire resource into the public Runner.
func (r *runnerResource) toModel() *Runner {
	m := &Runner{
		ID:               r.ID,
		Name:             r.Attributes.Name,
		Description:      r.Attributes.Description,
		AgentPoolID:      r.Attributes.AgentPoolID,
		RunnerType:       r.Attributes.RunnerType,
		Status:           r.Attributes.Status,
		Hostname:         r.Attributes.Hostname,
		OSType:           r.Attributes.OSType,
		AgentVersion:     r.Attributes.AgentVersion,
		Labels:           r.Attributes.Labels,
		TerraformVersion: r.Attributes.TerraformVersion,
		AnsibleVersion:   r.Attributes.AnsibleVersion,
	}
	if r.Attributes.LastHeartbeatAt != nil {
		m.LastHeartbeatAt = *r.Attributes.LastHeartbeatAt
	}
	return m
}

// runnerListEnvelope is the full JSON:API collection envelope, including the
// meta.pagination block List uses to walk every page.
type runnerListEnvelope struct {
	Data []runnerResource `json:"data"`
	Meta struct {
		Pagination struct {
			CurrentPage int `json:"current-page"`
			TotalPages  int `json:"total-pages"`
		} `json:"pagination"`
	} `json:"meta"`
}

// List returns every runner in options.Organization, walking all pages via
// meta.pagination and applying the optional filters. A capability/permission
// failure (403) or any non-404 error surfaces as an error, never an empty list.
func (s *RunnersService) List(ctx context.Context, options RunnerListOptions) ([]*Runner, error) {
	if options.Organization == "" {
		return nil, fmt.Errorf("stackweaver: List requires an Organization")
	}

	const pageSize = 100
	var runners []*Runner

	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("page[size]", fmt.Sprintf("%d", pageSize))
		query.Set("page[number]", fmt.Sprintf("%d", page))
		if options.AgentPoolID != "" {
			query.Set("filter[agent_pool_id]", options.AgentPoolID)
		}
		if options.RunnerType != "" {
			query.Set("filter[runner_type]", options.RunnerType)
		}
		if options.Status != "" {
			query.Set("filter[status]", options.Status)
		}

		path := fmt.Sprintf("/organizations/%s/runners?%s", url.PathEscape(options.Organization), query.Encode())
		body, err := s.client.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var env runnerListEnvelope
		if err := s.client.plain.Unmarshal(body, &env); err != nil {
			return nil, err
		}

		for i := range env.Data {
			runners = append(runners, env.Data[i].toModel())
		}

		if env.Meta.Pagination.TotalPages <= page || len(env.Data) == 0 {
			break
		}
	}

	return runners, nil
}

// Stats returns the fleet summary (total/online/offline) for org.
func (s *RunnersService) Stats(ctx context.Context, org string) (*RunnerStats, error) {
	if org == "" {
		return nil, fmt.Errorf("stackweaver: Stats requires an Organization")
	}

	path := fmt.Sprintf("/organizations/%s/runners/stats", url.PathEscape(org))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res struct {
		Attributes struct {
			Total   int `json:"total"`
			Online  int `json:"online"`
			Offline int `json:"offline"`
		} `json:"attributes"`
	}
	if err := s.client.jsonapi.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	return &RunnerStats{
		Total:   res.Attributes.Total,
		Online:  res.Attributes.Online,
		Offline: res.Attributes.Offline,
	}, nil
}
