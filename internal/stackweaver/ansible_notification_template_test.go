// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

const (
	testNotificationTemplateID     = "7a2b9c4d-1e3f-4a5b-8c6d-000000000001"
	testNotificationTemplateSecret = "hook-basic-auth-password"
)

// notificationTemplateCreateRequest and notificationTemplateCreateResponse are
// request.body and response of golden
// writes/post_api_v2_organizations_name_ansible_notification-templates.json.
const notificationTemplateCreateRequest = `{
  "data": {
    "attributes": {
      "config": {
        "headers": {"X-Source": "stackweaver"},
        "method": "POST",
        "url": "https://hooks.example.com/stackweaver/deploys",
        "username": "stackweaver"
      },
      "description": "Recorded by the golden write harness",
      "name": "golden-write-deploy-hook",
      "notification-type": "webhook",
      "secret": "` + testNotificationTemplateSecret + `"
    },
    "type": "ansible-notification-templates"
  }
}`

const notificationTemplateCreateResponse = `{
  "data": {
    "attributes": {
      "config": {
        "headers": {"X-Source": "stackweaver"},
        "method": "POST",
        "url": "https://hooks.example.com/stackweaver/deploys",
        "username": "stackweaver"
      },
      "created-at": "2026-10-15T08:00:00Z",
      "description": "Recorded by the golden write harness",
      "has-secret": true,
      "name": "golden-write-deploy-hook",
      "notification-type": "webhook"
    },
    "id": "` + testNotificationTemplateID + `",
    "type": "ansible-notification-templates"
  }
}`

// notificationTemplateUpdateRequest and notificationTemplateUpdateResponse are
// request.body and response of golden
// writes/patch_api_v2_ansible_notification-templates_id.json.
const notificationTemplateUpdateRequest = `{
  "data": {
    "attributes": {
      "config": {
        "method": "PUT",
        "skip_tls_verify": false,
        "url": "https://hooks.example.com/stackweaver/releases",
        "username": "stackweaver"
      },
      "description": "Updated by the golden write harness",
      "name": "golden-write-webhook",
      "secret": "` + testNotificationTemplateSecret + `"
    },
    "type": "ansible-notification-templates"
  }
}`

const notificationTemplateUpdateResponse = `{
  "data": {
    "attributes": {
      "config": {
        "method": "PUT",
        "skip_tls_verify": false,
        "url": "https://hooks.example.com/stackweaver/releases",
        "username": "stackweaver"
      },
      "created-at": "2026-10-15T08:00:00Z",
      "description": "Updated by the golden write harness",
      "has-secret": true,
      "name": "golden-write-webhook",
      "notification-type": "webhook"
    },
    "id": "` + testNotificationTemplateID + `",
    "type": "ansible-notification-templates"
  }
}`

func TestAnsibleNotificationTemplatesDocuments(t *testing.T) {
	secret := testNotificationTemplateSecret
	name := "golden-write-webhook"
	description := "Updated by the golden write harness"

	tests := []struct {
		name       string
		call       func(context.Context, *AnsibleNotificationTemplatesService) (*AnsibleNotificationTemplate, error)
		status     int
		response   string
		wantMethod string
		wantPath   string
		wantBody   string
		want       *AnsibleNotificationTemplate
	}{
		{
			name: "create sends the ansible-notification-templates document the server accepted",
			call: func(ctx context.Context, s *AnsibleNotificationTemplatesService) (*AnsibleNotificationTemplate, error) {
				return s.Create(ctx, AnsibleNotificationTemplateCreateOptions{
					Organization: "acme",
					Name:         "golden-write-deploy-hook",
					Description:  "Recorded by the golden write harness",
					Type:         "webhook",
					Config: map[string]interface{}{
						"headers":  map[string]interface{}{"X-Source": "stackweaver"},
						"method":   "POST",
						"url":      "https://hooks.example.com/stackweaver/deploys",
						"username": "stackweaver",
					},
					Secret: &secret,
				})
			},
			status:     http.StatusCreated,
			response:   notificationTemplateCreateResponse,
			wantMethod: http.MethodPost,
			wantPath:   "/api/v2/organizations/acme/ansible/notification-templates",
			wantBody:   notificationTemplateCreateRequest,
			want: &AnsibleNotificationTemplate{
				ID:          testNotificationTemplateID,
				Name:        "golden-write-deploy-hook",
				Description: "Recorded by the golden write harness",
				Type:        "webhook",
				Config: map[string]interface{}{
					"headers":  map[string]interface{}{"X-Source": "stackweaver"},
					"method":   "POST",
					"url":      "https://hooks.example.com/stackweaver/deploys",
					"username": "stackweaver",
				},
				HasSecret: true,
			},
		},
		{
			name: "update sends the ansible-notification-templates document the server accepted",
			call: func(ctx context.Context, s *AnsibleNotificationTemplatesService) (*AnsibleNotificationTemplate, error) {
				return s.Update(ctx, testNotificationTemplateID, AnsibleNotificationTemplateUpdateOptions{
					Name:        &name,
					Description: &description,
					Config: map[string]interface{}{
						"method":          "PUT",
						"skip_tls_verify": false,
						"url":             "https://hooks.example.com/stackweaver/releases",
						"username":        "stackweaver",
					},
					Secret: &secret,
				})
			},
			status:     http.StatusOK,
			response:   notificationTemplateUpdateResponse,
			wantMethod: http.MethodPatch,
			wantPath:   "/api/v2/ansible/notification-templates/" + testNotificationTemplateID,
			wantBody:   notificationTemplateUpdateRequest,
			want: &AnsibleNotificationTemplate{
				ID:          testNotificationTemplateID,
				Name:        "golden-write-webhook",
				Description: "Updated by the golden write harness",
				Type:        "webhook",
				Config: map[string]interface{}{
					"method":          "PUT",
					"skip_tls_verify": false,
					"url":             "https://hooks.example.com/stackweaver/releases",
					"username":        "stackweaver",
				},
				HasSecret: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := newRecordingClient(t, tt.status, tt.response)
			template, err := tt.call(t.Context(), NewAnsibleNotificationTemplatesService(c))
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			assertRequest(t, got, tt.wantMethod, tt.wantPath, tt.wantBody)
			if !reflect.DeepEqual(template, tt.want) {
				t.Errorf("decoded\n%+v\nwant\n%+v", template, tt.want)
			}
		})
	}
}
