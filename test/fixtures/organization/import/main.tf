# Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0
#
# Resource criterion 5: importing the organization the main fixture created, under the
# configuration it was last applied with, yields a plan with no changes. Data source criterion 3:
# with `name` omitted, the data source reads the provider's `organization`.

terraform {
  required_providers {
    stackweaver = {
      source = "vhco-pro/stackweaver"
    }
  }
}

variable "name" {
  type = string
}

variable "email" {
  type = string
}

variable "assessments_enforced" {
  type = bool
}

provider "stackweaver" {
  organization = var.name
}

resource "stackweaver_organization" "imported" {
  name  = var.name
  email = var.email

  collaborator_auth_policy            = "password"
  cost_estimation_enabled             = true
  allow_force_delete_workspaces       = true
  assessments_enforced                = var.assessments_enforced
  speculative_plan_management_enabled = false
  aggregated_commit_status_enabled    = true
}

data "stackweaver_organization" "provider_default" {}

output "provider_default_name" {
  value = data.stackweaver_organization.provider_default.name
}
