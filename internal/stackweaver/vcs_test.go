// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// The golden corpus records the VCS listings as 404s (no connection exists in
// the recording fixture), so these documents follow the shapes the server's
// TestPlatformRequestContract asserts (handlers/request_contract_test.go):
// repositories are "vcs-repositories" keyed by the provider's repository id,
// branches are "vcs-branches" keyed by name, both with the provider page block
// (null totals, next-page only after a full page), and the file listings are
// "vcs-files" keyed by path with no attributes and the unpaged block.

const testVCSConnectionID = "6e7f8091-a2b3-4c4d-9e5f-000000000001"

const vcsReposPage1 = `{
  "data": [
    {
      "id": "1000",
      "type": "vcs-repositories",
      "attributes": {
        "name": "repo-0",
        "full-name": "acme/repo-0",
        "description": "First repository",
        "private": true,
        "default-branch": "main",
        "url": "https://github.com/acme/repo-0",
        "clone-url": "https://github.com/acme/repo-0.git",
        "ssh-url": "git@github.com:acme/repo-0.git"
      }
    },
    {
      "id": "1001",
      "type": "vcs-repositories",
      "attributes": {
        "name": "repo-1",
        "full-name": "acme/repo-1",
        "description": "",
        "private": false,
        "default-branch": "trunk",
        "url": "https://github.com/acme/repo-1",
        "clone-url": "https://github.com/acme/repo-1.git",
        "ssh-url": "git@github.com:acme/repo-1.git"
      }
    }
  ],
  "meta": {"pagination": {"current-page": 1, "page-size": 100, "prev-page": null, "next-page": 2, "total-pages": null, "total-count": null}}
}`

const vcsReposPage2 = `{
  "data": [
    {
      "id": "1002",
      "type": "vcs-repositories",
      "attributes": {
        "name": "infra",
        "full-name": "acme/infra",
        "description": "Fleet playbooks",
        "private": true,
        "default-branch": "main",
        "url": "https://github.com/acme/infra",
        "clone-url": "https://github.com/acme/infra.git",
        "ssh-url": "git@github.com:acme/infra.git"
      }
    }
  ],
  "meta": {"pagination": {"current-page": 2, "page-size": 100, "prev-page": 1, "next-page": null, "total-pages": null, "total-count": null}}
}`

const vcsBranchesPage = `{
  "data": [
    {"id": "main", "type": "vcs-branches", "attributes": {"protected": true, "commit-sha": "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "commit-url": "https://api.github.com/repos/acme/infra/commits/4b825dc6"}},
    {"id": "feature/x", "type": "vcs-branches", "attributes": {"protected": false, "commit-sha": "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", "commit-url": "https://api.github.com/repos/acme/infra/commits/e69de29b"}}
  ],
  "meta": {"pagination": {"current-page": 1, "page-size": 100, "prev-page": null, "next-page": null, "total-pages": null, "total-count": null}}
}`

const vcsFilesListing = `{
  "data": [
    {"id": "site.yml", "type": "vcs-files"},
    {"id": "inventories/prod/hosts.yml", "type": "vcs-files"}
  ],
  "meta": {"pagination": {"current-page": 1, "page-size": 2, "prev-page": null, "next-page": null, "total-pages": null, "total-count": null}}
}`

// vcsRequest is one request a case expects, the query compared as parsed values.
type vcsRequest struct {
	path  string
	query url.Values
}

func pageValues(page string) url.Values {
	return url.Values{"page[number]": {page}, "page[size]": {"100"}}
}

func TestVCSDocuments(t *testing.T) {
	const repoBase = "/api/v2/vcs-connections/" + testVCSConnectionID + "/repositories"
	repo1002 := &VCSRepository{
		ID: 1002, Name: "infra", FullName: "acme/infra", Description: "Fleet playbooks", Private: true,
		DefaultBranch: "main", URL: "https://github.com/acme/infra",
		CloneURL: "https://github.com/acme/infra.git", SSHURL: "git@github.com:acme/infra.git",
	}

	tests := []struct {
		name         string
		call         func(context.Context, *VCSService) (any, error)
		responses    []string
		wantRequests []vcsRequest
		want         any
		wantErr      string
	}{
		{
			name: "repositories follow next-page until it is null",
			call: func(ctx context.Context, s *VCSService) (any, error) {
				return s.ListRepositories(ctx, VCSRepositoryListOptions{ConnectionID: testVCSConnectionID})
			},
			responses: []string{vcsReposPage1, vcsReposPage2},
			wantRequests: []vcsRequest{
				{path: repoBase, query: pageValues("1")},
				{path: repoBase, query: pageValues("2")},
			},
			want: []*VCSRepository{
				{
					ID: 1000, Name: "repo-0", FullName: "acme/repo-0", Description: "First repository", Private: true,
					DefaultBranch: "main", URL: "https://github.com/acme/repo-0",
					CloneURL: "https://github.com/acme/repo-0.git", SSHURL: "git@github.com:acme/repo-0.git",
				},
				{
					ID: 1001, Name: "repo-1", FullName: "acme/repo-1",
					DefaultBranch: "trunk", URL: "https://github.com/acme/repo-1",
					CloneURL: "https://github.com/acme/repo-1.git", SSHURL: "git@github.com:acme/repo-1.git",
				},
				repo1002,
			},
		},
		{
			name: "repositories scoped to an Azure DevOps project send the project",
			call: func(ctx context.Context, s *VCSService) (any, error) {
				return s.ListRepositories(ctx, VCSRepositoryListOptions{ConnectionID: testVCSConnectionID, Project: "Platform"})
			},
			responses: []string{strings.Replace(vcsReposPage2, `"current-page": 2`, `"current-page": 1`, 1)},
			wantRequests: []vcsRequest{
				{path: repoBase, query: url.Values{"page[number]": {"1"}, "page[size]": {"100"}, "project": {"Platform"}}},
			},
			want: []*VCSRepository{repo1002},
		},
		{
			name: "a repository id that is not numeric is an error",
			call: func(ctx context.Context, s *VCSService) (any, error) {
				return s.ListRepositories(ctx, VCSRepositoryListOptions{ConnectionID: testVCSConnectionID})
			},
			responses:    []string{strings.Replace(vcsReposPage2, `"id": "1002"`, `"id": "acme/infra"`, 1)},
			wantRequests: []vcsRequest{{path: repoBase, query: pageValues("1")}},
			wantErr:      `vcs-repositories id "acme/infra"`,
		},
		{
			name: "a next-page that does not advance is an error",
			call: func(ctx context.Context, s *VCSService) (any, error) {
				return s.ListRepositories(ctx, VCSRepositoryListOptions{ConnectionID: testVCSConnectionID})
			},
			responses:    []string{strings.Replace(vcsReposPage1, `"next-page": 2`, `"next-page": 1`, 1)},
			wantRequests: []vcsRequest{{path: repoBase, query: pageValues("1")}},
			wantErr:      "does not advance",
		},
		{
			name: "branches are vcs-branches keyed by name",
			call: func(ctx context.Context, s *VCSService) (any, error) {
				return s.ListBranches(ctx, testVCSConnectionID, "acme", "infra")
			},
			responses:    []string{vcsBranchesPage},
			wantRequests: []vcsRequest{{path: repoBase + "/acme/infra/branches", query: pageValues("1")}},
			want: []*VCSBranch{
				{Name: "main", CommitSHA: "4b825dc642cb6eb9a060e54bf8d69288fbee4904", Protected: true},
				{Name: "feature/x", CommitSHA: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", Protected: false},
			},
		},
		{
			name: "yaml files are vcs-files keyed by path, read at a ref",
			call: func(ctx context.Context, s *VCSService) (any, error) {
				return s.ListYamlFiles(ctx, testVCSConnectionID, "acme", "infra", "main")
			},
			responses:    []string{vcsFilesListing},
			wantRequests: []vcsRequest{{path: repoBase + "/acme/infra/yaml-files", query: url.Values{"ref": {"main"}}}},
			want:         []string{"site.yml", "inventories/prod/hosts.yml"},
		},
		{
			name: "inventory files on the default branch send no query",
			call: func(ctx context.Context, s *VCSService) (any, error) {
				return s.ListInventoryFiles(ctx, testVCSConnectionID, "acme", "infra", "")
			},
			responses:    []string{vcsFilesListing},
			wantRequests: []vcsRequest{{path: repoBase + "/acme/infra/inventory-files", query: url.Values{}}},
			want:         []string{"site.yml", "inventories/prod/hosts.yml"},
		},
		{
			name: "an empty file listing decodes to no paths",
			call: func(ctx context.Context, s *VCSService) (any, error) {
				return s.ListYamlFiles(ctx, testVCSConnectionID, "acme", "infra", "")
			},
			responses:    []string{`{"data": [], "meta": {"pagination": {"current-page": 1, "page-size": 1, "prev-page": null, "next-page": null, "total-pages": null, "total-count": null}}}`},
			wantRequests: []vcsRequest{{path: repoBase + "/acme/infra/yaml-files", query: url.Values{}}},
			want:         []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := newReplayClient(t, http.StatusOK, tt.responses...)
			result, err := tt.call(t.Context(), NewVCSService(c))

			if len(*got) != len(tt.wantRequests) {
				t.Fatalf("made %d requests, want %d: %+v", len(*got), len(tt.wantRequests), *got)
			}
			for i, want := range tt.wantRequests {
				req := (*got)[i]
				query, perr := url.ParseQuery(req.Query)
				if perr != nil {
					t.Fatalf("request %d: parse query %q: %v", i+1, req.Query, perr)
				}
				if req.Method != http.MethodGet || req.Path != want.path || !reflect.DeepEqual(query, want.query) {
					t.Errorf("request %d = %s %s ? %v, want GET %s ? %v", i+1, req.Method, req.Path, query, want.path, want.query)
				}
			}

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("decoded %+v, want %+v", result, tt.want)
			}
		})
	}
}
