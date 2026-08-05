<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_project_access
tfe_alias: tfe_team_project_access
kind: resource
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_team_project_access.go
go_tfe_type: TeamProjectAccess
compat_doc: docs/internal/tfe-compatibility/resources/teams/tfe_team_project_access.md
---
# stackweaver_team_project_access

Grants a team a level of access to a **project** — either a fixed level (`read`/`write`/`maintain`/
`admin`) or `custom`, in which case explicit project-level and workspace-level permission blocks apply
to the project and all workspaces within it. Maps onto Stackweaver's `team-projects` access record.

## Client approach

`go-tfe-clean`. The upstream resource (legacy SDKv2 with context CRUD, `Schema()` at
`internal/provider/resource_tfe_team_project_access.go:23`) drives the stock `go-tfe`
`TeamProjectAccess` service (`Add/Read/Update/Remove`) against the `team-projects` endpoint. Stackweaver
accepts and returns the stock `team-projects` JSON:API shape unchanged — no wrapper. A `CustomizeDiff`
(`checkForCustomPermissions`) rejects permission blocks when `access != "custom"` provider-side; no
extra wire bytes.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `team-projects` primary id |
| `team_id` | string | Required | yes | — | no | `team-*` id |
| `project_id` | string | Required | yes | — | no | `prj-*` id |
| `access` | string | Required | no | — | no | `admin`/`write`/`maintain`/`read`/`custom` |
| `project_access` | block (list) | Optional+Computed | no | server | no | only with `access="custom"` (below) |
| `workspace_access` | block (list) | Optional+Computed | no | server | no | only with `access="custom"` (below) |

### `project_access` block (strings, Optional+Computed)

`settings` (`read`/`update`/`delete`), `teams` (`none`/`read`/`manage`),
`variable_sets` (`none`/`read`/`write`). Wire attrs: `settings`, `teams`, `variable-sets`.

### `workspace_access` block (Optional+Computed)

Strings: `runs` (`read`/`plan`/`apply`), `sentinel_mocks` (`none`/`read`),
`state_versions` (`none`/`read-outputs`/`read`/`write`), `variables` (`none`/`read`/`write`).
Bools: `create`, `locking`, `move`, `delete`, `run_tasks`, `policy_overrides` (BETA upstream).
Wire attrs use the same names with `state-versions`/`sentinel-mocks`/`run-tasks`/`policy-overrides`.

## Wire contract

- **Create:** `TeamProjectAccess.Add(TeamProjectAccessAddOptions)` → `POST /team-projects`. Sends
  `access`, and (when `custom`) the `project-access` and `workspace-access` nested objects; plus `team`
  and `project` relationships.
- **Read:** `TeamProjectAccess.Read(id)` → `GET /team-projects/:id`. Returns `access`, both nested
  permission objects, and `team`/`project` relationships.
- **Update:** `TeamProjectAccess.Update(id, TeamProjectAccessUpdateOptions)` → `PATCH /team-projects/:id`
  (access + permissions in place).
- **Delete:** `TeamProjectAccess.Remove(id)` → `DELETE /team-projects/:id`.
- **JSON:API type:** `team-projects`. No write-only fields. Import by id (passthrough).

## Acceptance criteria (these ARE the test)

1. `apply` of `{team_id, project_id, access="maintain"}` creates the record; `id`, `access`, and the
   computed `project_access` / `workspace_access` blocks round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. A `custom` config with explicit `project_access` and `workspace_access` round-trips: every named
   permission (e.g. `project_access.settings="update"`, `workspace_access.runs="apply"`,
   `workspace_access.create=true`) reads back exactly as set.
4. Setting a `project_access`/`workspace_access` block with `access != "custom"` is rejected at plan by
   the `checkForCustomPermissions` diff.
5. Updating `access` (fixed→fixed, or a custom permission value) applies in place without recreate.
6. `team_id` and `project_id` are ForceNew — changing either recreates the record.
7. `destroy` removes it; a subsequent `TeamProjectAccess.Read(id)` returns 404.

## Runtime criterion

The grant is enforced at run time across the whole project: a member of `team_id` gets the specified
access to the project and to every workspace in it (e.g. `workspace_access.create=true` lets them create
workspaces in the project; `access="read"` cannot). Verified by exercising a project/workspace operation
as a team member. Not config-only.

## Docs + example

- Provider docs page: `docs/resources/team_project_access.md` — the fixed-vs-custom rule, both
  permission-value tables, import by id.
- Example: `examples/resources/stackweaver_team_project_access/resource.tf` — one fixed-level grant and
  one `custom` grant with both permission blocks.

## Divergences from upstream / TFE

None. Wire shape and endpoint (`team-projects`) match `tfe_team_project_access`; drop-in.
