<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_project_access
tfe_alias: tfe_team_project_access
kind: data-source
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_team_project_access.go
go_tfe_type: TeamProjectAccess
compat_doc: n/a
---
# stackweaver_team_project_access

Looks up a team's access grant on a specific project and exposes the access level plus the resolved
project-level and workspace-level permission sets. Maps onto Stackweaver's team-project access concept.
Read-only: given a `team_id` and `project_id` it resolves the `team-projects` grant `id`, its `access`
level, and the computed `project_access` / `workspace_access` blocks.

## Client approach

`go-tfe-clean`. Reads the project via `Projects.Read`, lists team-project grants via
`TeamProjectAccess.List` filtered by project, then re-reads the matched grant through the shared
resource read path (`TeamProjectAccess.Read`). Consumes the stock `TeamProjectAccess` JSON:API shape
unchanged; no wrapper. No compatibility detail doc exists yet
(`docs/internal/tfe-compatibility/data-sources/teams/tfe_team_project_access.md` is absent) — this spec
is the source of record.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `team_id` | string | Required | — | — | no | team whose grant to look up |
| `project_id` | string | Required | — | — | no | project to look up the grant on |
| `id` | string | Computed | — | — | no | `team-projects` primary id of the grant |
| `access` | string | Computed | — | — | no | access level (`admin`/`write`/`maintain`/`read`/`custom`) |
| `project_access` | list(object) | Computed | — | — | no | single-element list of project-level permissions |
| `project_access.settings` | string | Computed | — | — | no | project settings permission |
| `project_access.teams` | string | Computed | — | — | no | project teams permission |
| `project_access.variable_sets` | string | Computed | — | — | no | project variable-sets permission |
| `workspace_access` | list(object) | Computed | — | — | no | single-element list of workspace-level permissions |
| `workspace_access.create` | bool | Computed | — | — | no | may create workspaces |
| `workspace_access.locking` | bool | Computed | — | — | no | may lock/unlock workspaces |
| `workspace_access.move` | bool | Computed | — | — | no | may move workspaces |
| `workspace_access.delete` | bool | Computed | — | — | no | may delete workspaces |
| `workspace_access.run_tasks` | bool | Computed | — | — | no | may manage run tasks |
| `workspace_access.policy_overrides` | bool | Computed | — | — | no | may override policy checks (BETA) |
| `workspace_access.runs` | string | Computed | — | — | no | runs permission |
| `workspace_access.sentinel_mocks` | string | Computed | — | — | no | sentinel-mocks permission |
| `workspace_access.state_versions` | string | Computed | — | — | no | state-versions permission |
| `workspace_access.variables` | string | Computed | — | — | no | variables permission |

## Wire contract

- **Read (lookup):** `Projects.Read(project_id)` → `GET /projects/:id`, then
  `TeamProjectAccess.List(TeamProjectAccessListOptions{ProjectID})` → `GET
  /team-projects?filter[project][id]=:id`. Paginates until a grant whose `Team.ID` equals `team_id` is
  found, sets `id` to that grant's id, and delegates to the resource read
  (`TeamProjectAccess.Read(id)` → `GET /team-projects/:id`) to populate `access`, `project_access`, and
  `workspace_access`.
- No create/update/delete — data source.
- **JSON:API type:** `team-projects`. No divergent fields; the permission enums map straight from the
  go-tfe `TeamProjectAccess` struct.

## Acceptance criteria (these ARE the test)

1. Fixture creates a backing `stackweaver_team_project_access` grant (a team + project + access level),
   then this data source reads it by `team_id` + `project_id`; `apply` succeeds.
2. Computed `id` is set and equals the created grant's `id` (the `team-projects` primary id).
3. `access` round-trips: it equals the access level configured on the backing grant.
4. `project_access.0.*` and `workspace_access.0.*` reflect the level's implicit (or custom) permissions
   consistent with the backing grant.
5. A `team_id`/`project_id` pair with no grant fails the read with a "could not find team project
   access" error.
6. **Plan-null quirk:** both input args are Required (not Optional-null), so assert the clearly-Computed
   fields (`id`, `access`, the two nested permission blocks).

## Runtime criterion

Read-only data source. It resolves an existing team's effective access level and permission sets on a
project so a config can branch on or export those permissions. No runtime side effect of its own.

## Docs + example

- Provider docs page: `docs/data-sources/team_project_access.md` — arguments (`team_id`, `project_id`),
  computed attributes (`id`, `access`, nested `project_access` and `workspace_access`).
- Example: `examples/data-sources/stackweaver_team_project_access/data-source.tf` — look up a team's
  access on a project and reference `data.stackweaver_team_project_access.x.access`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_team_project_access`.
