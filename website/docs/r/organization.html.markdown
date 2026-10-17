---
layout: "stackweaver"
page_title: "Stackweaver: stackweaver_organization"
description: |-
  Manages organizations.
---

# stackweaver_organization

Manages a Stackweaver organization. Creating one also creates its `default` project and an owners
team with the creator in it, so the token that created the organization can manage it straight away.

This resource is also available as `tfe_organization`. Set `provider = stackweaver` on a block that
uses a `tfe_` name, or Terraform looks for it in the `hashicorp/tfe` provider.

~> **A user token is required to create an organization.** An organization-scoped token is refused
with 403 ("organization-scoped tokens cannot create organizations; use a user token"), as in Terraform
Enterprise. Create a user token in the Stackweaver UI under your user settings, or with
`terraform login`.

~> **Organization names cannot be reused.** A deleted organization's name stays reserved, because it is
part of the subject of the workload identity tokens its runs were issued. Creating an organization
with a reserved name fails with 422.

## Example Usage

```terraform
resource "stackweaver_organization" "example" {
  name  = "my-org-name"
  email = "admin@example.com"
}
```

## Schema

### Required

- `email` (String) Admin email address.
- `name` (String) Name of the organization. Compared without case; changing it renames the organization in place.

### Optional

- `aggregated_commit_status_enabled` (Boolean) Whether to post one aggregated commit status for all of the organization's workspaces on a commit instead of one per workspace. If enabled, `send_passing_statuses_for_untriggered_speculative_plans` must be `false`.
- `allow_force_delete_workspaces` (Boolean) Whether workspace administrators may delete workspaces that still have resources under management. If false, only organization owners may. Defaults to `false`.
- `assessments_enforced` (Boolean) Whether drift detection runs on every eligible workspace, overriding each workspace's own setting.
- `collaborator_auth_policy` (String) Authentication policy (`password` or `two_factor_mandatory`). Defaults to `password`. Stored; `two_factor_mandatory` is not yet enforced.
- `cost_estimation_enabled` (Boolean) Stored for compatibility. Stackweaver has no cost estimation feature.
- `enforce_hyok` (Boolean) Not supported by Stackweaver; always read back as `false`. Defaults to `false`.
- `max_ttl_enabled` (Boolean) Not supported by Stackweaver; always read back as `false`. Defaults to `false`.
- `owners_team_saml_role_id` (String) Not supported by Stackweaver, which signs users in through OIDC; always read back as empty. Setting it causes a change on every plan.
- `send_passing_statuses_for_untriggered_speculative_plans` (Boolean) Whether to post a passing commit status for workspaces a pull request does not trigger. Defaults to `false`.
- `session_remember_minutes` (Number) Not supported by Stackweaver; sessions are governed by the identity provider. Always read back as unset, so setting it causes a change on every plan.
- `session_timeout_minutes` (Number) Not supported by Stackweaver; sessions are governed by the identity provider. Always read back as unset, so setting it causes a change on every plan.
- `speculative_plan_management_enabled` (Boolean) Whether a newer commit on a branch cancels the pending speculative plans of older commits on that branch. Defaults to `true`.
- `stacks_enabled` (Boolean) Not supported by Stackweaver; always read back as `false`.
- `user_tokens_enabled` (Boolean) Whether user tokens may be used on the organization's resources. Owners are exempt, so an owner cannot lock themselves out. Defaults to `true`.

### Read-Only

- `default_project_id` (String) The ID of the organization's `default` project.
- `id` (String) The name of the organization.

## Import

An organization can be imported by name:

```shell
terraform import stackweaver_organization.example my-org-name
```
