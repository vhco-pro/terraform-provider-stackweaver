---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_organization"
description: |-
  Gets information about an organization.
---

# Data Source: stackweaver_organization

Gets information about an organization. Also available as `tfe_organization`; set
`provider = stackweaver` on a block that uses the `tfe_` name.

## Example Usage

```terraform
data "stackweaver_organization" "example" {
  name = "my-org-name"
}
```

## Schema

### Optional

- `name` (String) Name of the organization. If omitted, the provider's `organization` is used.

### Read-Only

- `aggregated_commit_status_enabled` (Boolean) Whether one aggregated commit status is posted for all of the organization's workspaces.
- `assessments_enforced` (Boolean) Whether drift detection runs on every eligible workspace.
- `collaborator_auth_policy` (String) Authentication policy (`password` or `two_factor_mandatory`).
- `cost_estimation_enabled` (Boolean) Stored for compatibility; Stackweaver has no cost estimation feature.
- `default_project_id` (String) ID of the organization's `default` project. Workspaces created without a project are created in it.
- `email` (String) Admin email address.
- `enforce_hyok` (Boolean) Not supported by Stackweaver; always `false`.
- `external_id` (String) The organization's UUID.
- `id` (String) The organization's UUID. Use `external_id` instead.
- `max_ttl_enabled` (Boolean) Not supported by Stackweaver; always `false`.
- `owners_team_saml_role_id` (String) Not supported by Stackweaver; always empty.
- `send_passing_statuses_for_untriggered_speculative_plans` (Boolean) Whether a passing commit status is posted for workspaces a pull request does not trigger.
- `speculative_plan_management_enabled` (Boolean) Whether a newer commit on a branch cancels the pending speculative plans of older commits on that branch.
- `two_factor_conformant` (Boolean) Always `false`: Stackweaver does not yet read two-factor status from the identity provider.
