<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team
tfe_alias: tfe_team
kind: resource
family: teams
origin: forked
backing_api: partial
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_team.go
go_tfe_type: Team
compat_doc: docs/internal/tfe-compatibility/resources/teams/tfe_team.md
---
# stackweaver_team

A team is an RBAC principal inside an organization: a named group that carries an organization-access
grant and, through the sibling access resources, workspace/project permissions that its members
inherit. Maps 1:1 onto Stackweaver's team concept.

## Client approach

`go-tfe-clean`. The upstream resource (legacy SDKv2, `Schema()` at
`internal/provider/resource_tfe_team.go:55`) drives the stock `go-tfe` `Teams` service
(`Create/Read/Update/Delete`). Stackweaver accepts and returns the stock `teams` JSON:API shape
unchanged — no wrapper. Some attributes are accepted on the wire but not yet enforced by the
Stackweaver backend (see divergences); that is a backing-completeness gap, not a wire-shape difference,
so it needs no client change.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `teams` JSON:API primary id (`team-*`) |
| `name` | string | Required | no | — | no | team name, unique within the org |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; changing it recreates |
| `visibility` | string | Optional+Computed | no | server | no | `secret` or `organization` |
| `sso_team_id` | string | Optional | no | — | no | SAML/SSO team id; sent as `sso-team-id` |
| `allow_member_token_management` | bool | Optional | no | `true` | no | `allow-member-token-management` |
| `organization_access` | block (list, max 1) | Optional+Computed | no | — | no | org-level permission booleans (below) |

### `organization_access` block (all bool, Optional, Default `false`)

`manage_policies`, `manage_policy_overrides`, `delegate_policy_overrides`, `manage_workspaces`,
`manage_vcs_settings`, `manage_providers`, `manage_modules`, `manage_run_tasks`, `manage_projects`,
`read_workspaces`, `read_projects`, `manage_membership`, `manage_teams`, `manage_organization_access`,
`access_secret_teams`, `manage_agent_pools`. Sent nested as the `organization-access` object.

## Wire contract

- **Create:** `Teams.Create(org, TeamCreateOptions)` → `POST /organizations/:org/teams`. Attrs sent:
  `name`, `sso-team-id?`, `visibility?`, `allow-member-token-management`, `organization-access` object.
- **Read:** `Teams.Read(id)` → `GET /teams/:id`. Returns `name`, `visibility`, `sso-team-id`,
  `allow-member-token-management`, `organization-access`, `users-count`.
- **Update:** `Teams.Update(id, TeamUpdateOptions)` → `PATCH /teams/:id`. All attrs in place; the
  resource always sends `sso-team-id` (empty string when unset) and `allow-member-token-management`.
- **Delete:** `Teams.Delete(id)` → `DELETE /teams/:id`.
- **JSON:API type:** `teams`. No write-only fields. Import by `<ORG>/<TEAM ID>` or `<ORG>/<TEAM NAME>`.

## Acceptance criteria (these ARE the test)

1. `apply` of `{name, organization, visibility="organization"}` creates the team; `id`, `name`,
   `visibility` round-trip into state.
2. Re-`plan` after apply shows **no drift** (including the computed `organization_access` block).
3. Setting `organization_access { read_workspaces = true, read_projects = true }` round-trips: both
   booleans read back `true`, unset booleans read back `false`.
4. Updating `name` and `visibility` applies in place without recreate.
5. `organization` is ForceNew — changing it recreates the team.
6. `destroy` removes it; a subsequent `Teams.Read(id)` returns 404.
7. `allow_member_token_management` defaults to `true` when omitted and round-trips.

## Runtime criterion

The team is an RBAC principal: after apply it can be referenced by `stackweaver_team_access`,
`stackweaver_team_project_access`, and the membership resources, and its members inherit the granted
access at run time. Verified indirectly (a member of a team with a workspace grant can perform the
granted operation); the team resource itself is otherwise CRUD.

## Docs + example

- Provider docs page: `docs/resources/team.md` — arguments (name/organization/visibility/sso_team_id/
  allow_member_token_management/organization_access), attribute `id`, import formats.
- Example: `examples/resources/stackweaver_team/resource.tf` — a named team with a minimal
  `organization_access` block.

## Divergences from upstream / TFE

Wire shape is identical to `tfe_team`; drop-in on the schema. Backing is **partial** per the compat doc
(`docs/internal/tfe-compatibility/resources/teams/tfe_team.md`): `sso_team_id` (SSO team management) and
the `manage_teams` / `manage_organization_access` / `access_secret_teams` organization-access booleans
are accepted on the wire but **not yet enforced** by the Stackweaver backend. These are ignored, not
rejected, so no drift and no client change — a backing-completeness gap only.
