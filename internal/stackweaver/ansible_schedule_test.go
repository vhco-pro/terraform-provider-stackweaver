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
	testScheduleID    = "5c1e0a3e-7d42-4b8f-9a10-000000000001"
	testScheduleOrgID = "5c1e0a3e-7d42-4b8f-9a10-000000000002"
	testScheduleJTID  = "5c1e0a3e-7d42-4b8f-9a10-000000000003"
)

// scheduleReadResponse is golden reads/get_api_v2_ansible_schedules_schedule_id.json.
const scheduleReadResponse = `{
  "data": {
    "attributes": {
      "config": {},
      "created-at": "2026-10-15T08:00:00Z",
      "cron-expression": "0 2 * * *",
      "description": "Applies the patch baseline across the fleet overnight.",
      "name": "nightly-patch-baseline",
      "schedule-type": "job_template",
      "status": "disabled",
      "timezone": "UTC",
      "updated-at": "2026-10-15T08:00:00Z"
    },
    "id": "` + testScheduleID + `",
    "relationships": {
      "job-template": {"data": {"id": "` + testScheduleJTID + `", "type": "ansible-job-templates"}},
      "organization": {"data": {"id": "` + testScheduleOrgID + `", "type": "organizations"}}
    },
    "type": "schedules"
  }
}`

// scheduleUpdateRequest and scheduleUpdateResponse are request.body and response
// of golden writes/patch_api_v2_ansible_schedules_schedule_id.json.
const scheduleUpdateRequest = `{
  "data": {
    "attributes": {
      "config": {"extra_vars": {"reboot": false}},
      "cron-expression": "15 1 * * *",
      "description": "Updated by the golden write harness",
      "name": "nightly-patch-baseline",
      "timezone": "Europe/Brussels"
    },
    "type": "schedules"
  }
}`

const scheduleUpdateResponse = `{
  "data": {
    "attributes": {
      "config": {"extra_vars": {"reboot": false}},
      "created-at": "2026-10-15T08:00:00Z",
      "cron-expression": "15 1 * * *",
      "description": "Updated by the golden write harness",
      "name": "nightly-patch-baseline",
      "next-run-at": "2026-10-16T23:15:00Z",
      "schedule-type": "job_template",
      "status": "disabled",
      "timezone": "Europe/Brussels",
      "updated-at": "2026-10-15T09:00:00Z"
    },
    "id": "` + testScheduleID + `",
    "relationships": {
      "job-template": {"data": {"id": "` + testScheduleJTID + `", "type": "ansible-job-templates"}},
      "organization": {"data": {"id": "` + testScheduleOrgID + `", "type": "organizations"}}
    },
    "type": "schedules"
  }
}`

func TestAnsibleSchedulesDocuments(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	updated := &AnsibleSchedule{
		ID:             testScheduleID,
		OrganizationID: testScheduleOrgID,
		Name:           "nightly-patch-baseline",
		Description:    "Updated by the golden write harness",
		Type:           "job_template",
		Status:         "disabled",
		JobTemplateID:  testScheduleJTID,
		CronExpression: "15 1 * * *",
		Timezone:       "Europe/Brussels",
		Config:         map[string]interface{}{"extra_vars": map[string]interface{}{"reboot": false}},
		NextRunAt:      "2026-10-16T23:15:00Z",
	}

	tests := []struct {
		name       string
		call       func(context.Context, *AnsibleSchedulesService) (*AnsibleSchedule, error)
		response   string
		wantMethod string
		wantBody   string
		want       *AnsibleSchedule
	}{
		{
			name: "read decodes a schedules resource",
			call: func(ctx context.Context, s *AnsibleSchedulesService) (*AnsibleSchedule, error) {
				return s.Read(ctx, testScheduleID)
			},
			response:   scheduleReadResponse,
			wantMethod: http.MethodGet,
			want: &AnsibleSchedule{
				ID:             testScheduleID,
				OrganizationID: testScheduleOrgID,
				Name:           "nightly-patch-baseline",
				Description:    "Applies the patch baseline across the fleet overnight.",
				Type:           "job_template",
				Status:         "disabled",
				JobTemplateID:  testScheduleJTID,
				CronExpression: "0 2 * * *",
				Timezone:       "UTC",
				Config:         map[string]interface{}{},
			},
		},
		{
			name: "update sends the schedules document the server accepted",
			call: func(ctx context.Context, s *AnsibleSchedulesService) (*AnsibleSchedule, error) {
				return s.Update(ctx, testScheduleID, AnsibleScheduleUpdateOptions{
					Name:           strPtr("nightly-patch-baseline"),
					Description:    strPtr("Updated by the golden write harness"),
					CronExpression: strPtr("15 1 * * *"),
					Timezone:       strPtr("Europe/Brussels"),
					Config:         map[string]interface{}{"extra_vars": map[string]interface{}{"reboot": false}},
				})
			},
			response:   scheduleUpdateResponse,
			wantMethod: http.MethodPatch,
			wantBody:   scheduleUpdateRequest,
			want:       updated,
		},
		{
			name: "update omits the attributes it leaves untouched",
			call: func(ctx context.Context, s *AnsibleSchedulesService) (*AnsibleSchedule, error) {
				return s.Update(ctx, testScheduleID, AnsibleScheduleUpdateOptions{
					CronExpression: strPtr("15 1 * * *"),
				})
			},
			response:   scheduleUpdateResponse,
			wantMethod: http.MethodPatch,
			wantBody:   `{"data":{"type":"schedules","attributes":{"cron-expression":"15 1 * * *"}}}`,
			want:       updated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, got := newRecordingClient(t, http.StatusOK, tt.response)
			schedule, err := tt.call(t.Context(), NewAnsibleSchedulesService(c))
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			assertRequest(t, got, tt.wantMethod, "/api/v2/ansible/schedules/"+testScheduleID, tt.wantBody)
			if !reflect.DeepEqual(schedule, tt.want) {
				t.Errorf("decoded\n%+v\nwant\n%+v", schedule, tt.want)
			}
		})
	}
}
