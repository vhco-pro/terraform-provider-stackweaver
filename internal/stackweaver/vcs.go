// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// VCSService is the native service backing the VCS discovery data sources
// (stackweaver_vcs_repositories, stackweaver_vcs_repository_branches,
// stackweaver_vcs_yaml_files). Given a VCS connection it enumerates the
// repositories, branches, and candidate files that connection can see. All
// reads; the provider never mutates a VCS connection through this service.
//
// Every listing answers a JSON:API collection document:
//   - repositories are "vcs-repositories" resources keyed by the provider's
//     repository id, branches are "vcs-branches" resources keyed by the branch
//     name, and both are paged: the provider reports no total, so
//     meta.pagination states null totals and a next-page only when the page
//     came back full. The client follows next-page until it is null.
//   - the yaml and inventory file listings are "vcs-files" resources keyed by
//     the file path, with no attributes, answered whole in one response.
//
// Provider-capability failures surface as APIError (501 unsupported, 403 Azure
// DevOps identity not materialized) - never swallowed into an empty list. Wire
// contract (paths relative to /api/v2):
//
//	Repositories:    GET /vcs-connections/:id/repositories?page[number]=&page[size]=
//	Branches:        GET /vcs-connections/:id/repositories/:owner/:repo/branches?page[number]=&page[size]=
//	YamlFiles:       GET /vcs-connections/:id/repositories/:owner/:repo/yaml-files
//	InventoryFiles:  GET /vcs-connections/:id/repositories/:owner/:repo/inventory-files
type VCSService struct {
	client *Client
}

// NewVCSService constructs a self-contained VCSService over c. Data sources
// build it via stackweaver.NewVCSService(cc.Stackweaver).
func NewVCSService(c *Client) *VCSService {
	return &VCSService{client: c}
}

// VCSRepository is a repository reachable through a VCS connection. ID is the
// provider's numeric repository id, which the server sends as the resource id
// string.
type VCSRepository struct {
	ID            int64
	Name          string
	FullName      string
	Description   string
	Private       bool
	DefaultBranch string
	URL           string
	CloneURL      string
	SSHURL        string
}

// VCSBranch is a branch of a repository, with the head commit's SHA.
type VCSBranch struct {
	Name      string
	CommitSHA string
	Protected bool
}

// vcsListPageSize is the page size the paged listings ask for (the server caps
// it at 100).
const vcsListPageSize = 100

// vcsRepositoryResource mirrors one "vcs-repositories" resource object.
type vcsRepositoryResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name          string `json:"name"`
		FullName      string `json:"full-name"`
		Description   string `json:"description"`
		Private       bool   `json:"private"`
		DefaultBranch string `json:"default-branch"`
		URL           string `json:"url"`
		CloneURL      string `json:"clone-url"`
		SSHURL        string `json:"ssh-url"`
	} `json:"attributes"`
}

func (r *vcsRepositoryResource) toModel() (*VCSRepository, error) {
	id, err := strconv.ParseInt(r.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("stackweaver: vcs-repositories id %q is not a provider repository id: %w", r.ID, err)
	}
	return &VCSRepository{
		ID:            id,
		Name:          r.Attributes.Name,
		FullName:      r.Attributes.FullName,
		Description:   r.Attributes.Description,
		Private:       r.Attributes.Private,
		DefaultBranch: r.Attributes.DefaultBranch,
		URL:           r.Attributes.URL,
		CloneURL:      r.Attributes.CloneURL,
		SSHURL:        r.Attributes.SSHURL,
	}, nil
}

// vcsBranchResource mirrors one "vcs-branches" resource object.
type vcsBranchResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Protected bool   `json:"protected"`
		CommitSHA string `json:"commit-sha"`
		CommitURL string `json:"commit-url"`
	} `json:"attributes"`
}

func (r *vcsBranchResource) toModel() *VCSBranch {
	return &VCSBranch{
		Name:      r.ID,
		CommitSHA: r.Attributes.CommitSHA,
		Protected: r.Attributes.Protected,
	}
}

// vcsFileResource mirrors one "vcs-files" resource object of a listing: the
// path is the id and a listing carries no attributes.
type vcsFileResource struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// vcsListDocument is a VCS collection document with the paging member the
// client follows. The JSON:API codec unwraps data only, so the whole document is
// decoded with the plain codec.
type vcsListDocument[R any] struct {
	Data []R `json:"data"`
	Meta struct {
		Pagination struct {
			NextPage *int `json:"next-page"`
		} `json:"pagination"`
	} `json:"meta"`
}

// listAllPages walks a paged VCS listing from page 1, following
// meta.pagination.next-page until it is null. pathFor builds the request path of
// one page. The totals are null in a provider listing, so they are never read.
func listAllPages[R any](ctx context.Context, c *Client, pathFor func(page int) string) ([]R, error) {
	var all []R
	for page := 1; ; {
		body, err := c.do(ctx, http.MethodGet, pathFor(page), nil)
		if err != nil {
			return nil, err
		}
		var doc vcsListDocument[R]
		if err := c.plain.Unmarshal(body, &doc); err != nil {
			return nil, err
		}
		all = append(all, doc.Data...)

		next := doc.Meta.Pagination.NextPage
		if next == nil {
			return all, nil
		}
		if *next <= page {
			return nil, fmt.Errorf("stackweaver: listing on page %d names next-page %d, which does not advance", page, *next)
		}
		page = *next
	}
}

// pageQuery is the paging query of one page of a VCS listing.
func pageQuery(page int) url.Values {
	query := url.Values{}
	query.Set("page[number]", strconv.Itoa(page))
	query.Set("page[size]", strconv.Itoa(vcsListPageSize))
	return query
}

// VCSRepositoryListOptions are the options for ListRepositories. ConnectionID is
// required; Project scopes an Azure DevOps listing (?project=) and is ignored by
// providers without a project layer.
type VCSRepositoryListOptions struct {
	ConnectionID string
	Project      string
}

// ListRepositories enumerates every repository reachable through the connection,
// walking all pages.
func (s *VCSService) ListRepositories(ctx context.Context, options VCSRepositoryListOptions) ([]*VCSRepository, error) {
	if options.ConnectionID == "" {
		return nil, fmt.Errorf("stackweaver: ListRepositories requires a ConnectionID")
	}

	resources, err := listAllPages[vcsRepositoryResource](ctx, s.client, func(page int) string {
		query := pageQuery(page)
		if options.Project != "" {
			query.Set("project", options.Project)
		}
		return fmt.Sprintf("/vcs-connections/%s/repositories?%s", url.PathEscape(options.ConnectionID), query.Encode())
	})
	if err != nil {
		return nil, err
	}

	repos := make([]*VCSRepository, 0, len(resources))
	for i := range resources {
		repo, err := resources[i].toModel()
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

// ListBranches enumerates every branch of owner/repo reachable through the
// connection, walking all pages.
func (s *VCSService) ListBranches(ctx context.Context, connID, owner, repo string) ([]*VCSBranch, error) {
	if connID == "" {
		return nil, fmt.Errorf("stackweaver: ListBranches requires a ConnectionID")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("stackweaver: ListBranches requires an owner and repo")
	}

	resources, err := listAllPages[vcsBranchResource](ctx, s.client, func(page int) string {
		return fmt.Sprintf("/vcs-connections/%s/repositories/%s/%s/branches?%s",
			url.PathEscape(connID), url.PathEscape(owner), url.PathEscape(repo), pageQuery(page).Encode())
	})
	if err != nil {
		return nil, err
	}

	branches := make([]*VCSBranch, len(resources))
	for i := range resources {
		branches[i] = resources[i].toModel()
	}
	return branches, nil
}

// ListYamlFiles returns the .yaml/.yml file paths in owner/repo at the optional
// ref (empty = default branch). The listing is not paged: one response holds
// every path.
func (s *VCSService) ListYamlFiles(ctx context.Context, connID, owner, repo, ref string) ([]string, error) {
	return s.listFiles(ctx, connID, owner, repo, ref, "yaml-files")
}

// ListInventoryFiles returns the inventory (.ini/.yaml/.yml/.json) file paths in
// owner/repo at the optional ref (empty = default branch). Not paged.
func (s *VCSService) ListInventoryFiles(ctx context.Context, connID, owner, repo, ref string) ([]string, error) {
	return s.listFiles(ctx, connID, owner, repo, ref, "inventory-files")
}

// listFiles is the shared implementation behind ListYamlFiles/ListInventoryFiles;
// segment selects the yaml-files vs inventory-files endpoint (they differ only in
// the server-side extension filter). Each "vcs-files" resource's id is its path.
func (s *VCSService) listFiles(ctx context.Context, connID, owner, repo, ref, segment string) ([]string, error) {
	if connID == "" {
		return nil, fmt.Errorf("stackweaver: listFiles requires a ConnectionID")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("stackweaver: listFiles requires an owner and repo")
	}

	path := fmt.Sprintf("/vcs-connections/%s/repositories/%s/%s/%s",
		url.PathEscape(connID), url.PathEscape(owner), url.PathEscape(repo), segment)
	if ref != "" {
		query := url.Values{}
		query.Set("ref", ref)
		path += "?" + query.Encode()
	}

	body, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resources []vcsFileResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, err
	}

	paths := make([]string, len(resources))
	for i := range resources {
		paths[i] = resources[i].ID
	}
	return paths, nil
}
