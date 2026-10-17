// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"net/http"
	"reflect"
	"testing"
)

const (
	testCredentialJobTemplateID = "3f4e5d6c-7b8a-4c9d-8e0f-000000000001"
	testCredentialID            = "3f4e5d6c-7b8a-4c9d-8e0f-000000000002"
)

func TestAnsibleJobTemplateCredentialsDocuments(t *testing.T) {
	tests := []struct {
		name         string
		credentialID string
		response     string
		wantBody     string
		want         *AnsibleTemplateCredential
	}{
		{
			// request.body and response of golden
			// writes/post_api_v2_ansible_job-templates_id_credentials.json.
			name:         "attach sends the ansible-credentials identifier the server accepted",
			credentialID: testCredentialID,
			response: `{
  "data": {
    "attributes": {"credential-type": "ssh", "name": "fleet-ssh", "username": "ansible", "vault-id": ""},
    "id": "` + testCredentialID + `",
    "type": "ansible-credentials"
  }
}`,
			wantBody: `{"data": {"id": "` + testCredentialID + `", "type": "ansible-credentials"}}`,
			want: &AnsibleTemplateCredential{
				ID:             testCredentialID,
				Name:           "fleet-ssh",
				CredentialType: "ssh",
				Username:       "ansible",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := newRecordingClient(t, http.StatusCreated, tt.response)
			cred, err := NewAnsibleJobTemplateCredentialsService(c).Attach(t.Context(), testCredentialJobTemplateID, tt.credentialID)
			if err != nil {
				t.Fatalf("Attach: %v", err)
			}
			assertRequest(t, got, http.MethodPost, "/api/v2/ansible/job-templates/"+testCredentialJobTemplateID+"/credentials", tt.wantBody)
			if !reflect.DeepEqual(cred, tt.want) {
				t.Errorf("decoded\n%+v\nwant\n%+v", cred, tt.want)
			}
		})
	}
}
