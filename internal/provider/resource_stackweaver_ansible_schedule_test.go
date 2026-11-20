// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-tfe/internal/stackweaver"
)

func TestAnsibleScheduleImportTarget(t *testing.T) {
	r := &resourceStackweaverAnsibleSchedule{}
	for _, tc := range []struct {
		name string
		data stackweaver.AnsibleSchedule
	}{
		{"job template", stackweaver.AnsibleSchedule{Type: "job_template", JobTemplateID: "template-id"}},
		{"inventory source", stackweaver.AnsibleSchedule{Type: "inventory_source", InventorySourceID: "source-id"}},
		{"playbook", stackweaver.AnsibleSchedule{Type: "playbook_sync", PlaybookID: "playbook-id", StartDateTime: "2026-11-19T08:00:00Z"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := r.stateFromSchedule(&tc.data, modelStackweaverAnsibleSchedule{}, false)
			if state.JobTemplateID.ValueString() != tc.data.JobTemplateID ||
				state.InventorySourceID.ValueString() != tc.data.InventorySourceID ||
				state.PlaybookID.ValueString() != tc.data.PlaybookID ||
				state.StartDateTime.ValueString() != tc.data.StartDateTime {
				t.Fatalf("schedule import discarded the readable target")
			}
		})
	}
	t.Run("retain configured target when response omits it", func(t *testing.T) {
		prior := modelStackweaverAnsibleSchedule{JobTemplateID: types.StringValue("configured")}
		state := r.stateFromSchedule(&stackweaver.AnsibleSchedule{Type: "job_template"}, prior, false)
		if state.JobTemplateID.ValueString() != "configured" {
			t.Fatal("refresh lost the configured target")
		}
	})
}
