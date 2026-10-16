// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

const (
	testPlaybookFilesConnectionID = "2b3c4d5e-6f70-4a81-9b2c-000000000001"
	testPlaybookFilesPlaybookID   = "2b3c4d5e-6f70-4a81-9b2c-000000000002"
)

// playbookFilesResponse is the listing the server answers. The golden harness
// records no 2xx for this route (its read fixture is the 400 for a malformed
// repository), so this literal follows the shape the server's request contract
// test asserts ("playbook files are ansible-playbook-files resources" in
// handlers/ansible/request_contract_test.go): the path is the id, name and
// registered are the attributes, a registered file relates to its playbook, an
// unregistered one carries no relationships, and the pagination block states
// no totals and no next page.
const playbookFilesResponse = `{
  "data": [
    {
      "id": "site.yml",
      "type": "ansible-playbook-files",
      "attributes": {"name": "site.yml", "registered": true},
      "relationships": {
        "playbook": {"data": {"id": "` + testPlaybookFilesPlaybookID + `", "type": "ansible-playbooks"}}
      }
    },
    {
      "id": "playbooks/deploy.yml",
      "type": "ansible-playbook-files",
      "attributes": {"name": "deploy.yml", "registered": false}
    }
  ],
  "meta": {
    "pagination": {
      "current-page": 1,
      "page-size": 2,
      "prev-page": null,
      "next-page": null,
      "total-pages": null,
      "total-count": null
    }
  }
}`

func TestAnsibleVCSPlaybookFilesDocuments(t *testing.T) {
	tests := []struct {
		name      string
		options   VCSPlaybookFilesListOptions
		response  string
		wantQuery url.Values
		want      []*VCSPlaybookFile
	}{
		{
			name: "list decodes ansible-playbook-files resources",
			options: VCSPlaybookFilesListOptions{
				VCSConnectionID: testPlaybookFilesConnectionID,
				Repository:      "acme/infra",
				Branch:          "main",
			},
			response: playbookFilesResponse,
			wantQuery: url.Values{
				"vcs_connection_id": {testPlaybookFilesConnectionID},
				"repository":        {"acme/infra"},
				"branch":            {"main"},
			},
			want: []*VCSPlaybookFile{
				{Path: "site.yml", Name: "site.yml", Registered: true, PlaybookID: testPlaybookFilesPlaybookID},
				{Path: "playbooks/deploy.yml", Name: "deploy.yml", Registered: false},
			},
		},
		{
			name: "an empty listing decodes to no files",
			options: VCSPlaybookFilesListOptions{
				VCSConnectionID: testPlaybookFilesConnectionID,
				Repository:      "acme/infra",
				Branch:          "main",
				Path:            "playbooks",
			},
			response: `{"data": [], "meta": {"pagination": {"current-page": 1, "page-size": 1, "prev-page": null, "next-page": null, "total-pages": null, "total-count": null}}}`,
			wantQuery: url.Values{
				"vcs_connection_id": {testPlaybookFilesConnectionID},
				"repository":        {"acme/infra"},
				"branch":            {"main"},
				"path":              {"playbooks"},
			},
			want: []*VCSPlaybookFile{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := newRecordingClient(t, http.StatusOK, tt.response)
			files, err := NewAnsibleVCSPlaybookFilesService(c).List(t.Context(), "acme", tt.options)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			assertRequest(t, got, http.MethodGet, "/api/v2/organizations/acme/ansible/vcs-playbook-files", "")
			query, err := url.ParseQuery(got.Query)
			if err != nil {
				t.Fatalf("parse query %q: %v", got.Query, err)
			}
			if !reflect.DeepEqual(query, tt.wantQuery) {
				t.Errorf("query = %v, want %v", query, tt.wantQuery)
			}
			if !reflect.DeepEqual(files, tt.want) {
				t.Errorf("decoded %+v, want %+v", files, tt.want)
			}
		})
	}
}
