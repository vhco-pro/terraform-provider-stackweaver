// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// VCSService is the native service backing the VCS discovery data sources
// (stackweaver_vcs_repositories, stackweaver_vcs_repository_branches,
// stackweaver_vcs_yaml_files). Given a VCS connection it enumerates the
// repositories, branches, and candidate files that connection can see. All
// reads; the provider never mutates a VCS connection through this service.
//
// Envelope note: unlike the JSON:API runners endpoints, every VCS endpoint here
// speaks plain JSON. The list endpoints return { "data": [ <item>, ... ], "meta":
// { "pagination": { "page", "per_page" } } } with snake_case keys; the file
// endpoints return { "data": [ "<path>", ... ] } (a flat []string, no meta, not
// paginated). Provider-capability failures surface as APIError (501 unsupported,
// 403 Azure DevOps identity not materialized) - never swallowed into an empty
// list. Wire contract (paths relative to /api/v2):
//
//	Repositories:    GET /vcs-connections/:id/repositories
//	Branches:        GET /vcs-connections/:id/repositories/:owner/:repo/branches
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

// VCSRepository is a repository reachable through a VCS connection, matching the
// core/services/vcs.Repository wire struct (snake_case JSON tags).
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

// VCSBranch is a branch of a repository, flattening the nested commit.sha into a
// top-level CommitSHA for the Terraform schema.
type VCSBranch struct {
	Name      string
	CommitSHA string
	Protected bool
}

// vcsRepository mirrors the plain-JSON Repository object (vcs.Repository).
type vcsRepository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	URL           string `json:"url"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
}

// vcsBranch mirrors the plain-JSON Branch object (vcs.Branch).
type vcsBranch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
	Protected bool `json:"protected"`
}

// VCSRepositoryListOptions are the options for ListRepositories. ConnectionID is
// required; Project scopes an Azure DevOps listing (?project=) and is ignored by
// providers without a project layer.
type VCSRepositoryListOptions struct {
	ConnectionID string
	Project      string
}

// ListRepositories enumerates every repository reachable through the connection,
// walking all pages. The server's meta echoes only page/per_page (no total), so
// pagination terminates when a page returns fewer than per_page items.
func (s *VCSService) ListRepositories(ctx context.Context, options VCSRepositoryListOptions) ([]*VCSRepository, error) {
	if options.ConnectionID == "" {
		return nil, fmt.Errorf("stackweaver: ListRepositories requires a ConnectionID")
	}

	const perPage = 100
	var repos []*VCSRepository

	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("page[size]", fmt.Sprintf("%d", perPage))
		query.Set("page[number]", fmt.Sprintf("%d", page))
		if options.Project != "" {
			query.Set("project", options.Project)
		}

		path := fmt.Sprintf("/vcs-connections/%s/repositories?%s", url.PathEscape(options.ConnectionID), query.Encode())
		body, err := s.client.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var env struct {
			Data []vcsRepository `json:"data"`
		}
		if err := s.client.plain.Unmarshal(body, &env); err != nil {
			return nil, err
		}

		for _, r := range env.Data {
			repos = append(repos, &VCSRepository{
				ID:            r.ID,
				Name:          r.Name,
				FullName:      r.FullName,
				Description:   r.Description,
				Private:       r.Private,
				DefaultBranch: r.DefaultBranch,
				URL:           r.URL,
				CloneURL:      r.CloneURL,
				SSHURL:        r.SSHURL,
			})
		}

		if len(env.Data) < perPage {
			break
		}
	}

	return repos, nil
}

// ListBranches enumerates every branch of owner/repo reachable through the
// connection, walking all pages and flattening commit.sha into CommitSHA.
func (s *VCSService) ListBranches(ctx context.Context, connID, owner, repo string) ([]*VCSBranch, error) {
	if connID == "" {
		return nil, fmt.Errorf("stackweaver: ListBranches requires a ConnectionID")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("stackweaver: ListBranches requires an owner and repo")
	}

	const perPage = 100
	var branches []*VCSBranch

	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("page[size]", fmt.Sprintf("%d", perPage))
		query.Set("page[number]", fmt.Sprintf("%d", page))

		path := fmt.Sprintf("/vcs-connections/%s/repositories/%s/%s/branches?%s",
			url.PathEscape(connID), url.PathEscape(owner), url.PathEscape(repo), query.Encode())
		body, err := s.client.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		var env struct {
			Data []vcsBranch `json:"data"`
		}
		if err := s.client.plain.Unmarshal(body, &env); err != nil {
			return nil, err
		}

		for _, b := range env.Data {
			branches = append(branches, &VCSBranch{
				Name:      b.Name,
				CommitSHA: b.Commit.SHA,
				Protected: b.Protected,
			})
		}

		if len(env.Data) < perPage {
			break
		}
	}

	return branches, nil
}

// ListYamlFiles returns the .yaml/.yml file paths in owner/repo at the optional
// ref (empty = default branch). The endpoint is not paginated: a single response
// holds the full flat []string of repo-relative paths.
func (s *VCSService) ListYamlFiles(ctx context.Context, connID, owner, repo, ref string) ([]string, error) {
	return s.listFiles(ctx, connID, owner, repo, ref, "yaml-files")
}

// ListInventoryFiles returns the inventory (.ini/.yaml/.yml/.json) file paths in
// owner/repo at the optional ref (empty = default branch). Not paginated.
func (s *VCSService) ListInventoryFiles(ctx context.Context, connID, owner, repo, ref string) ([]string, error) {
	return s.listFiles(ctx, connID, owner, repo, ref, "inventory-files")
}

// listFiles is the shared implementation behind ListYamlFiles/ListInventoryFiles;
// segment selects the yaml-files vs inventory-files endpoint (they differ only in
// the server-side extension filter).
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

	var env struct {
		Data []string `json:"data"`
	}
	if err := s.client.plain.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	return env.Data, nil
}
