// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"encoding/json"
	"testing"
)

func TestJSONAPICodecMarshal(t *testing.T) {
	codec := &jsonAPICodec{}
	body, err := codec.Marshal("ansible-playbooks", map[string]any{
		"name":          "site",
		"playbook-path": "site.yml",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got struct {
		Data struct {
			Type       string            `json:"type"`
			Attributes map[string]string `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode marshalled body: %v", err)
	}
	if got.Data.Type != "ansible-playbooks" {
		t.Errorf("type = %q, want ansible-playbooks", got.Data.Type)
	}
	if got.Data.Attributes["name"] != "site" {
		t.Errorf("attributes.name = %q, want site", got.Data.Attributes["name"])
	}
	if got.Data.Attributes["playbook-path"] != "site.yml" {
		t.Errorf("attributes.playbook-path = %q, want site.yml", got.Data.Attributes["playbook-path"])
	}
}

func TestJSONAPICodecUnmarshalSingle(t *testing.T) {
	codec := &jsonAPICodec{}
	body := []byte(`{
		"data": {
			"id": "11111111-1111-1111-1111-111111111111",
			"type": "ansible-playbooks",
			"attributes": {
				"name": "site",
				"playbook-path": "site.yml",
				"source-mode": "cached"
			},
			"relationships": {
				"project": {"data": {"id": "22222222-2222-2222-2222-222222222222", "type": "projects"}},
				"vcs-connection": {"data": {"id": "33333333-3333-3333-3333-333333333333", "type": "vcs-connections"}}
			}
		}
	}`)

	var res playbookResource
	if err := codec.Unmarshal(body, &res); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	got := res.toModel()
	if got.ID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("ID = %q", got.ID)
	}
	if got.Name != "site" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.PlaybookPath != "site.yml" {
		t.Errorf("PlaybookPath = %q", got.PlaybookPath)
	}
	if got.SourceMode != "cached" {
		t.Errorf("SourceMode = %q", got.SourceMode)
	}
	if got.ProjectID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("ProjectID = %q", got.ProjectID)
	}
	if got.VCSConnectionID != "33333333-3333-3333-3333-333333333333" {
		t.Errorf("VCSConnectionID = %q", got.VCSConnectionID)
	}
}

func TestJSONAPICodecUnmarshalNoData(t *testing.T) {
	codec := &jsonAPICodec{}
	var res playbookResource
	if err := codec.Unmarshal([]byte(`{}`), &res); err == nil {
		t.Fatal("expected error for missing data member, got nil")
	}
}

func TestNewAPIErrorParsesEnvelope(t *testing.T) {
	body := []byte(`{"errors":[{"status":"404","title":"Not Found","detail":"Playbook not found"}]}`)
	err := newAPIError(422, body)
	if err.Detail != "Playbook not found" {
		t.Errorf("Detail = %q, want Playbook not found", err.Detail)
	}
	if err.Title != "Not Found" {
		t.Errorf("Title = %q, want Not Found", err.Title)
	}
	if err.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 422", err.StatusCode)
	}
}
