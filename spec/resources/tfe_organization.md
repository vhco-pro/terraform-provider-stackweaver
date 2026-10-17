<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization
tfe_alias: tfe_organization
kind: resource
family: organizations
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: implemented
upstream_file: internal/provider/resource_tfe_organization.go
go_tfe_type: Organization
compat_doc: docs/internal/tfe-compatibility/resources/organizations/tfe_organization.md
---
# stackweaver_organization

Manages a Stackweaver **organization**: the top-level tenant that owns projects, workspaces, teams,
Ansible resources and the registry. Creating one also creates its `default` project, its owners team
and its owners membership for the caller, exactly as creating it in the UI does. The organization's
`id` is its name, as in TFE.

This resource was held out of v0.1 because the backing API was not green at the time (`plan/tasks.md`,
"Explicitly out of v0.1"). The Stackweaver API completed the full `tfe_organization` surface on
2026-08-26 (monorepo #709): JSON:API list, every attribute below stored or echoed drift-free, the
default-project relationship, and the user-token-only create rule. It was end-to-end proven there with
the stock `hashicorp/tfe` provider (create, no drift, destroy, 404).

## Client approach

`go-tfe-clean`. Stackweaver's organization endpoints accept and return the stock `go-tfe`
`Organization` JSON:API shape (compat doc above). The upstream SDKv2 resource
(`internal/provider/resource_tfe_organization.go`) drives `Organizations.Create`, `Read`, `Update`
and `Delete` unchanged. Registration only: `"tfe_organization": resourceTFEOrganization()` in the
SDKv2 `ResourcesMap` of `provider.go`, which `addStackweaverAliases` registers under both names.

## Schema

As upstream; nothing added or removed.

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `name` | string | Required | no | - | no | Case-insensitive diff; a rename is an in-place update and `id` follows the new name |
| `email` | string | Required | no | - | no | |
| `collaborator_auth_policy` | string | Optional | no | `password` | no | `password` or `two_factor_mandatory`; stored, MFA enforcement not yet wired (divergence) |
| `cost_estimation_enabled` | bool | Optional+Computed | no | server | no | Stored; no cost-estimation subsystem (divergence) |
| `send_passing_statuses_for_untriggered_speculative_plans` | bool | Optional+Computed | no | `false` | no | Real; mutually exclusive with `aggregated_commit_status_enabled` (server answers 422) |
| `aggregated_commit_status_enabled` | bool | Optional+Computed | no | `false` | no | Real |
| `assessments_enforced` | bool | Optional | no | - | no | Real (drift detection) |
| `allow_force_delete_workspaces` | bool | Optional | no | `false` | no | Real (force-delete gate) |
| `speculative_plan_management_enabled` | bool | Optional | no | `true` | no | Real (superseded speculative runs cancelled) |
| `user_tokens_enabled` | bool | Optional | no | `true` | no | Real; owners are exempt (divergence) |
| `default_project_id` | string | Computed | - | - | no | The org's `default` project (UUID) |
| `session_timeout_minutes` | number | Optional | no | - | no | Declined: echoed `null`; setting it drifts (divergence) |
| `session_remember_minutes` | number | Optional | no | - | no | Declined: echoed `null`; setting it drifts (divergence) |
| `owners_team_saml_role_id` | string | Optional | no | - | no | Declined: echoed `""` (divergence) |
| `enforce_hyok` | bool | Optional | no | `false` | no | Declined: echoed `false` |
| `stacks_enabled` | bool | Optional | no | - | no | Declined: echoed `false` |
| `max_ttl_enabled` | bool | Optional | no | `false` | no | Declined: echoed `false` |

## Wire contract

- **Create:** `Organizations.Create` → `POST /api/v2/organizations`, JSON:API type `organizations`,
  attributes as above. Requires a **user-bound** token (a user API token or `terraform login`); an
  organization-scoped token is refused with 403 "organization-scoped tokens cannot create
  organizations; use a user token", matching TFE, where organization tokens cannot create
  organizations. Deleted organization names are permanently reserved, so re-creating a deleted name
  answers 422.
- **Read:** `Organizations.Read` → `GET /api/v2/organizations/:name`; 404 removes it from state.
  `relationships.default-project` carries `default_project_id`.
- **Update:** `Organizations.Update` → `PATCH /api/v2/organizations/:name`, including a rename.
- **Delete:** `Organizations.Delete` → `DELETE /api/v2/organizations/:name`.
- **Import:** by organization name (upstream importer, unchanged).

## Acceptance criteria (these ARE the test)

The fixture is `test/fixtures/organization/`, run with a **user-bound** `TFE_TOKEN` against the dev
stack through `dev_overrides`. Every name carries a per-run suffix, because deleted names stay
reserved.

1. `apply` of `stackweaver_organization` with `name`, `email`, `collaborator_auth_policy =
   "password"`, `cost_estimation_enabled = true`, `allow_force_delete_workspaces = true`,
   `assessments_enforced = true`, `speculative_plan_management_enabled = false` and
   `aggregated_commit_status_enabled = true` creates the organization. Every one of those values, `id`
   (= name) and a non-empty `default_project_id` are in state.
2. Re-`plan` after apply shows **no changes**.
3. Updating `email` and `assessments_enforced` applies in place (no replacement) and re-`plan` shows
   no changes.
4. The same configuration written as `tfe_organization` (the alias) applies and re-plans clean.
5. `terraform import stackweaver_organization.x <name>` of the created organization yields a plan
   with no changes for the matching configuration.
6. `destroy` removes it; `GET /api/v2/organizations/<name>` then answers **404**.
7. With an organization-scoped token, `apply` fails with the 403 above and creates nothing.

## Runtime criterion

The created organization is usable: its `default_project_id` names the project that
`GET /api/v2/organizations/:name/projects` lists as `default`, and the token that created it can
create a workspace in it (the creator is in the owners team). Checked in the fixture by a
`stackweaver_workspace` in the new organization that applies and destroys.

## Docs + example

- Provider docs page: `docs/resources/organization.md` (and the `tfe_` alias page as for other
  resources): every attribute above, the user-token requirement, the reserved-name rule, and the
  declined attributes.
- Example: `examples/resources/stackweaver_organization/resource.tf`: `name` + `email`.

## Divergences from upstream / TFE

- **Declined attributes** (`session_timeout_minutes`, `session_remember_minutes`,
  `owners_team_saml_role_id`, `enforce_hyok`, `stacks_enabled`, `max_ttl_enabled`) are accepted and
  echoed as null, empty or false. Leaving them unset never drifts; setting the two session attributes
  or the SAML role drifts on every plan, so a migrated configuration should remove them. Sessions are
  owned by the identity provider.
- **`collaborator_auth_policy = "two_factor_mandatory"`** is stored but not yet enforced.
- **`cost_estimation_enabled`** is stored; there is no cost-estimation subsystem.
- **`user_tokens_enabled = false`** denies user tokens on the organization's routes, except to owners,
  so a lone owner cannot lock themselves out.
- **Deleted names are reserved forever** (they are part of workload-identity subjects).
