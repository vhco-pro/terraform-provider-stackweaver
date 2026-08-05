// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// AnsibleVCSPlaybookFilesService lists the playbook candidate files in a
// connected VCS repository at a branch, each annotated with whether it is
// already registered as a stackweaver_ansible_playbook. It backs the
// stackweaver_ansible_vcs_playbook_files data source.
//
// Unlike the reference AnsiblePlaybooks service (a Client field), this service
// is self-contained: the data source constructs it on demand via
// NewAnsibleVCSPlaybookFilesService(cc.Stackweaver). Its endpoint is
// plain-JSON, not JSON:API. Wire contract (path relative to /api/v2):
//
//	List: GET /organizations/:org/ansible/vcs-playbook-files?vcs_connection_id=&repository=&branch=&path=
type AnsibleVCSPlaybookFilesService struct {
	client *Client
}

// NewAnsibleVCSPlaybookFilesService constructs the service against c.
func NewAnsibleVCSPlaybookFilesService(c *Client) *AnsibleVCSPlaybookFilesService {
	return &AnsibleVCSPlaybookFilesService{client: c}
}

// VCSPlaybookFile is one discovered playbook candidate. PlaybookID/PlaybookName
// are populated only when Registered is true.
type VCSPlaybookFile struct {
	Path         string
	Name         string
	Registered   bool
	PlaybookID   string
	PlaybookName string
}

// VCSPlaybookFilesListOptions are the required (and optional) inputs for List.
type VCSPlaybookFilesListOptions struct {
	VCSConnectionID string // required
	Repository      string // required, "owner/repo"
	Branch          string // required
	Path            string // optional scope prefix
}

// vcsPlaybookFileWire mirrors one entry of the plain-JSON {"data":[...]} envelope.
type vcsPlaybookFileWire struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	Registered   bool   `json:"registered"`
	PlaybookID   string `json:"playbook_id"`
	PlaybookName string `json:"playbook_name"`
}

// List returns the playbook candidate files in the repository at the branch.
func (s *AnsibleVCSPlaybookFilesService) List(ctx context.Context, org string, options VCSPlaybookFilesListOptions) ([]*VCSPlaybookFile, error) {
	if org == "" {
		return nil, fmt.Errorf("stackweaver: List requires an organization")
	}
	if options.VCSConnectionID == "" {
		return nil, fmt.Errorf("stackweaver: List requires a VCSConnectionID")
	}
	if options.Repository == "" {
		return nil, fmt.Errorf("stackweaver: List requires a Repository")
	}
	if options.Branch == "" {
		return nil, fmt.Errorf("stackweaver: List requires a Branch")
	}

	query := url.Values{}
	query.Set("vcs_connection_id", options.VCSConnectionID)
	query.Set("repository", options.Repository)
	query.Set("branch", options.Branch)
	if options.Path != "" {
		query.Set("path", options.Path)
	}

	path := fmt.Sprintf("/organizations/%s/ansible/vcs-playbook-files?%s", url.PathEscape(org), query.Encode())
	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	// Plain-JSON envelope: {"data":[ {path,name,registered,...} ]}.
	var env struct {
		Data []vcsPlaybookFileWire `json:"data"`
	}
	if err := s.client.plain.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	files := make([]*VCSPlaybookFile, len(env.Data))
	for i, f := range env.Data {
		files[i] = &VCSPlaybookFile{
			Path:         f.Path,
			Name:         f.Name,
			Registered:   f.Registered,
			PlaybookID:   f.PlaybookID,
			PlaybookName: f.PlaybookName,
		}
	}
	return files, nil
}
