# Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0
#
# Acceptance fixture for stackweaver_organization and the organization data sources
# (spec/resources/tfe_organization.md, spec/data-sources/tfe_organization.md,
# spec/data-sources/tfe_organizations.md). Driven by ./run.sh with a USER token: an
# organization-scoped token cannot create organizations.
#
# Deleted organization names stay reserved, so every name carries var.run_id.

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

variable "email" {
  type    = string
  default = "provider-acceptance@example.test"
}

variable "assessments_enforced" {
  type    = bool
  default = true
}

provider "stackweaver" {}

# Resource criterion 1: every attribute below round-trips.
resource "stackweaver_organization" "test" {
  name  = "pv-org-${var.run_id}"
  email = var.email

  collaborator_auth_policy            = "password"
  cost_estimation_enabled             = true
  allow_force_delete_workspaces       = true
  assessments_enforced                = var.assessments_enforced
  speculative_plan_management_enabled = false
  aggregated_commit_status_enabled    = true
}

# Runtime criterion: the creator can use the new organization straight away.
resource "stackweaver_workspace" "probe" {
  name         = "probe"
  organization = stackweaver_organization.test.name
}

# An organization-scoped token for the new organization, used by ./refusal. Throwaway: it dies with
# the organization.
resource "stackweaver_organization_token" "test" {
  organization = stackweaver_organization.test.name
}

# Data source criteria: both names read the same organization.
data "stackweaver_organization" "test" {
  name = stackweaver_organization.test.name
}

data "tfe_organization" "test" {
  provider = stackweaver
  name     = stackweaver_organization.test.name
}

data "stackweaver_organizations" "all" {
  depends_on = [stackweaver_organization.test]
}

data "tfe_organizations" "all" {
  provider   = stackweaver
  depends_on = [stackweaver_organization.test]
}

output "name" {
  value = stackweaver_organization.test.name
}

output "organization" {
  value = stackweaver_organization.test
}

output "workspace_id" {
  value = stackweaver_workspace.probe.id
}

output "org_token" {
  value     = stackweaver_organization_token.test.token
  sensitive = true
}

output "ds" {
  value = data.stackweaver_organization.test
}

output "ds_alias" {
  value = data.tfe_organization.test
}

output "list_names" {
  value = data.stackweaver_organizations.all.names
}

output "list_ids" {
  value = data.stackweaver_organizations.all.ids
}

output "list_alias_ids" {
  value = data.tfe_organizations.all.ids
}
