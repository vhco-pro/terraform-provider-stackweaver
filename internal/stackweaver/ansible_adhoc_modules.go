// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// AnsibleAdHocModulesService reads the effective ad hoc module allowlist for an
// organization (the org's configured list, or the built-in AWX default). It
// backs the stackweaver_ansible_adhoc_modules data source.
//
// Self-contained service: the data source constructs it via
// NewAnsibleAdHocModulesService(cc.Stackweaver). The endpoint returns a JSON:API
// single object ({"data":{"type":"adhoc-modules","id":<org uuid>,"attributes":
// {"modules":[...]}}}). Wire contract (path relative to /api/v2):
//
//	List: GET /organizations/:org/ansible/adhoc-modules
type AnsibleAdHocModulesService struct {
	client *Client
}

// NewAnsibleAdHocModulesService constructs the service against c.
func NewAnsibleAdHocModulesService(c *Client) *AnsibleAdHocModulesService {
	return &AnsibleAdHocModulesService{client: c}
}

// AdHocModules is the effective allowlist for an organization. OrganizationID is
// the server-returned org uuid (the resource id).
type AdHocModules struct {
	OrganizationID string
	Modules        []string
}

// adhocModulesResource mirrors the JSON:API single resource object.
type adhocModulesResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Modules []string `json:"modules"`
	} `json:"attributes"`
}

// List returns the effective ad hoc module allowlist for org.
func (s *AnsibleAdHocModulesService) List(ctx context.Context, org string) (*AdHocModules, error) {
	if org == "" {
		return nil, fmt.Errorf("stackweaver: List requires an organization")
	}

	path := fmt.Sprintf("/organizations/%s/ansible/adhoc-modules", url.PathEscape(org))
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resource adhocModulesResource
	if err := s.client.jsonapi.Unmarshal(body, &resource); err != nil {
		return nil, err
	}

	return &AdHocModules{
		OrganizationID: resource.ID,
		Modules:        resource.Attributes.Modules,
	}, nil
}
