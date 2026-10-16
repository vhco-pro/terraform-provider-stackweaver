// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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

// The service table tests replay documents recorded by the Stackweaver server's
// golden write harness (backend/internal/api/v2/routes/testdata/golden/ in the
// monorepo). The recordings carry "<uuid>", "<timestamp>" and "<secret>"
// placeholders; the literals in the tests replace each with a concrete,
// distinct value so that an id decoded into the wrong field is caught.

// recordedRequest is what the test server saw of the one request a case makes.
type recordedRequest struct {
	Method      string
	Path        string
	Query       string
	ContentType string
	Body        []byte
}

// newRecordingClient returns a Client pointed at a test server that records the
// request into the returned recordedRequest and answers status with response.
func newRecordingClient(t *testing.T, status int, response string) (*Client, *recordedRequest) {
	t.Helper()
	got := &recordedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		got.Method = r.Method
		got.Path = r.URL.Path
		got.Query = r.URL.RawQuery
		got.ContentType = r.Header.Get("Content-Type")
		got.Body = body
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{BaseURL: srv.URL, Token: "test-token", HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, got
}

// assertRequest checks the method, path and body of the recorded request. An
// empty wantBody asserts that no body was sent; otherwise the body must be the
// same JSON document as wantBody (member order and whitespace aside).
func assertRequest(t *testing.T, got *recordedRequest, wantMethod, wantPath, wantBody string) {
	t.Helper()
	if got.Method != wantMethod {
		t.Errorf("method = %s, want %s", got.Method, wantMethod)
	}
	if got.Path != wantPath {
		t.Errorf("path = %s, want %s", got.Path, wantPath)
	}
	if wantBody == "" {
		if len(got.Body) != 0 {
			t.Errorf("sent a body %s, want none", got.Body)
		}
		return
	}
	if got.ContentType != "application/vnd.api+json" {
		t.Errorf("Content-Type = %q, want application/vnd.api+json", got.ContentType)
	}
	var gotDoc, wantDoc any
	if err := json.Unmarshal(got.Body, &gotDoc); err != nil {
		t.Fatalf("sent body is not JSON: %v (%s)", err, got.Body)
	}
	if err := json.Unmarshal([]byte(wantBody), &wantDoc); err != nil {
		t.Fatalf("want body is not JSON: %v", err)
	}
	if !reflect.DeepEqual(gotDoc, wantDoc) {
		t.Errorf("sent body\n%s\nwant\n%s", got.Body, wantBody)
	}
}
