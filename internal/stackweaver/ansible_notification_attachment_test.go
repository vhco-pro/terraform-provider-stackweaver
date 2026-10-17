// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"net/http"
	"reflect"
	"testing"
)

const (
	testAttachmentID           = "9d8e7f6a-5b4c-4d3e-8f2a-000000000001"
	testAttachmentTemplateID   = "9d8e7f6a-5b4c-4d3e-8f2a-000000000002"
	testAttachmentJobTemplate  = "9d8e7f6a-5b4c-4d3e-8f2a-000000000003"
	testAttachmentWorkflowID   = "9d8e7f6a-5b4c-4d3e-8f2a-000000000004"
	testAttachmentOrganization = "acme"
)

// attachmentJobTemplateRequest and attachmentJobTemplateResponse are
// request.body and response of golden
// writes/post_api_v2_organizations_name_ansible_notification-attachments.json.
const attachmentJobTemplateRequest = `{
  "data": {
    "attributes": {"on-failure": true, "on-started": true, "on-success": false},
    "relationships": {
      "job-template": {"data": {"id": "` + testAttachmentJobTemplate + `", "type": "ansible-job-templates"}},
      "notification-template": {"data": {"id": "` + testAttachmentTemplateID + `", "type": "ansible-notification-templates"}}
    },
    "type": "ansible-notification-attachments"
  }
}`

const attachmentJobTemplateResponse = `{
  "data": {
    "attributes": {
      "notification-template-name": "golden-write-webhook",
      "on-failure": true,
      "on-started": true,
      "on-success": false
    },
    "id": "` + testAttachmentID + `",
    "relationships": {
      "job-template": {"data": {"id": "` + testAttachmentJobTemplate + `", "type": "ansible-job-templates"}},
      "notification-template": {"data": {"id": "` + testAttachmentTemplateID + `", "type": "ansible-notification-templates"}}
    },
    "type": "ansible-notification-attachments"
  }
}`

// attachmentWorkflowRequest and attachmentWorkflowResponse are the workflow
// target form of the same documents. The golden harness records only the
// job-template target; this pair follows NotificationAttachmentRequest and
// formatAttachment, which relate a workflow as "ansible-workflows".
const attachmentWorkflowRequest = `{
  "data": {
    "attributes": {"on-failure": true, "on-started": false, "on-success": true},
    "relationships": {
      "workflow": {"data": {"id": "` + testAttachmentWorkflowID + `", "type": "ansible-workflows"}},
      "notification-template": {"data": {"id": "` + testAttachmentTemplateID + `", "type": "ansible-notification-templates"}}
    },
    "type": "ansible-notification-attachments"
  }
}`

const attachmentWorkflowResponse = `{
  "data": {
    "attributes": {
      "notification-template-name": "golden-write-webhook",
      "on-failure": true,
      "on-started": false,
      "on-success": true
    },
    "id": "` + testAttachmentID + `",
    "relationships": {
      "workflow": {"data": {"id": "` + testAttachmentWorkflowID + `", "type": "ansible-workflows"}},
      "notification-template": {"data": {"id": "` + testAttachmentTemplateID + `", "type": "ansible-notification-templates"}}
    },
    "type": "ansible-notification-attachments"
  }
}`

func TestAnsibleNotificationAttachmentsDocuments(t *testing.T) {
	tests := []struct {
		name     string
		options  AnsibleNotificationAttachmentCreateOptions
		response string
		wantBody string
		want     *AnsibleNotificationAttachment
	}{
		{
			name: "attach to a job template sends the document the server accepted",
			options: AnsibleNotificationAttachmentCreateOptions{
				Organization:           testAttachmentOrganization,
				NotificationTemplateID: testAttachmentTemplateID,
				JobTemplateID:          testAttachmentJobTemplate,
				OnStarted:              true,
				OnSuccess:              false,
				OnFailure:              true,
			},
			response: attachmentJobTemplateResponse,
			wantBody: attachmentJobTemplateRequest,
			want: &AnsibleNotificationAttachment{
				ID:                     testAttachmentID,
				NotificationTemplateID: testAttachmentTemplateID,
				JobTemplateID:          testAttachmentJobTemplate,
				OnStarted:              true,
				OnSuccess:              false,
				OnFailure:              true,
			},
		},
		{
			name: "attach to a workflow relates the workflow",
			options: AnsibleNotificationAttachmentCreateOptions{
				Organization:           testAttachmentOrganization,
				NotificationTemplateID: testAttachmentTemplateID,
				WorkflowID:             testAttachmentWorkflowID,
				OnStarted:              false,
				OnSuccess:              true,
				OnFailure:              true,
			},
			response: attachmentWorkflowResponse,
			wantBody: attachmentWorkflowRequest,
			want: &AnsibleNotificationAttachment{
				ID:                     testAttachmentID,
				NotificationTemplateID: testAttachmentTemplateID,
				WorkflowID:             testAttachmentWorkflowID,
				OnStarted:              false,
				OnSuccess:              true,
				OnFailure:              true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := newRecordingClient(t, http.StatusCreated, tt.response)
			attachment, err := NewAnsibleNotificationAttachmentsService(c).Create(t.Context(), tt.options)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			assertRequest(t, got, http.MethodPost, "/api/v2/organizations/acme/ansible/notification-attachments", tt.wantBody)
			if !reflect.DeepEqual(attachment, tt.want) {
				t.Errorf("decoded\n%+v\nwant\n%+v", attachment, tt.want)
			}
		})
	}
}
