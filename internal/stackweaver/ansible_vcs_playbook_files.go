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
// NewAnsibleVCSPlaybookFilesService(cc.Stackweaver). Its endpoint answers a
// collection of "ansible-playbook-files" resources: the id is the file path,
// the attributes are name and registered, and a registered file relates to its
// playbook. The listing is the provider's whole file tree in one response, so
// it is not paged. Wire contract (path relative to /api/v2):
//
//	List: GET /organizations/:org/ansible/vcs-playbook-files?vcs_connection_id=&repository=&branch=&path=
type AnsibleVCSPlaybookFilesService struct {
	client *Client
}

// NewAnsibleVCSPlaybookFilesService constructs the service against c.
func NewAnsibleVCSPlaybookFilesService(c *Client) *AnsibleVCSPlaybookFilesService {
	return &AnsibleVCSPlaybookFilesService{client: c}
}

// VCSPlaybookFile is one discovered playbook candidate. PlaybookID is populated
// only when Registered is true. The server relates a registered file to its
// playbook by id alone, so the playbook's name is not part of the listing.
type VCSPlaybookFile struct {
	Path       string
	Name       string
	Registered bool
	PlaybookID string
}

// VCSPlaybookFilesListOptions are the required (and optional) inputs for List.
type VCSPlaybookFilesListOptions struct {
	VCSConnectionID string // required
	Repository      string // required, "owner/repo"
	Branch          string // required
	Path            string // optional scope prefix
}

// playbookFileResource mirrors one "ansible-playbook-files" resource object.
type playbookFileResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name       string `json:"name"`
		Registered bool   `json:"registered"`
	} `json:"attributes"`
	Relationships struct {
		Playbook jsonAPIRelationship `json:"playbook"`
	} `json:"relationships"`
}

func (r *playbookFileResource) toModel() *VCSPlaybookFile {
	f := &VCSPlaybookFile{
		Path:       r.ID,
		Name:       r.Attributes.Name,
		Registered: r.Attributes.Registered,
	}
	if r.Relationships.Playbook.Data != nil {
		f.PlaybookID = r.Relationships.Playbook.Data.ID
	}
	return f
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

	var resources []playbookFileResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, err
	}

	files := make([]*VCSPlaybookFile, len(resources))
	for i := range resources {
		files[i] = resources[i].toModel()
	}
	return files, nil
}
