# Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0
#
# Resource criterion 4: the same configuration under the tfe_organization alias applies and
# re-plans clean.

terraform {
  required_providers {
    stackweaver = {
      source = "vhco-pro/stackweaver"
    }
  }
}

variable "run_id" {
  type = string
}

provider "stackweaver" {}

resource "tfe_organization" "test" {
  provider = stackweaver

  name  = "pv-alias-${var.run_id}"
  email = "provider-acceptance@example.test"

  collaborator_auth_policy            = "password"
  cost_estimation_enabled             = true
  allow_force_delete_workspaces       = true
  assessments_enforced                = true
  speculative_plan_management_enabled = false
  aggregated_commit_status_enabled    = true
}

output "name" {
  value = tfe_organization.test.name
}
