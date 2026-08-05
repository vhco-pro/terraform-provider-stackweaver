<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_access
tfe_alias: tfe_team_access
kind: data-source
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_team_access.go
go_tfe_type: TeamAccess
compat_doc: n/a
---
# stackweaver_team_access

Looks up a team's access grant on a specific workspace and exposes the access level plus the resolved
custom permission set. Maps onto Stackweaver's team-workspace access concept. Read-only: given a
`team_id` and `workspace_id` it resolves the `team-workspaces` grant `id`, its `access` level, and the
computed `permissions` block.

## Client approach

`go-tfe-clean`. Reads the workspace via `Workspaces.ReadByID`, lists team-workspace grants via
`TeamAccess.List` filtered by workspace, then re-reads the matched grant through the shared resource
read path (`TeamAccess.Read`). Consumes the stock `TeamAccess` JSON:API shape unchanged; no wrapper.
No compatibility detail doc exists yet
(`docs/internal/tfe-compatibility/data-sources/teams/tfe_team_access.md` is absent) — this spec is the
source of record.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `team_id` | string | Required | — | — | no | team whose grant to look up |
| `workspace_id` | string | Required | — | — | no | workspace to look up the grant on |
| `id` | string | Computed | — | — | no | `team-workspaces` primary id of the grant |
| `access` | string | Computed | — | — | no | access level (`read`/`plan`/`write`/`admin`/`custom`) |
| `permissions` | list(object) | Computed | — | — | no | single-element list of the resolved custom permissions |
| `permissions.runs` | string | Computed | — | — | no | runs permission |
| `permissions.variables` | string | Computed | — | — | no | variables permission |
| `permissions.state_versions` | string | Computed | — | — | no | state-versions permission |
| `permissions.sentinel_mocks` | string | Computed | — | — | no | sentinel-mocks permission |
| `permissions.workspace_locking` | bool | Computed | — | — | no | may lock/unlock the workspace |
| `permissions.run_tasks` | bool | Computed | — | — | no | may manage run tasks |
| `permissions.policy_overrides` | bool | Computed | — | — | no | may override policy checks (BETA) |

## Wire contract

- **Read (lookup):** `Workspaces.ReadByID(workspace_id)` → `GET /workspaces/:id`, then
  `TeamAccess.List(TeamAccessListOptions{WorkspaceID})` → `GET
  /team-workspaces?filter[workspace][id]=:id`. Paginates until a grant whose `Team.ID` equals `team_id`
  is found, sets `id` to that grant's id, and delegates to the resource read
  (`TeamAccess.Read(id)` → `GET /team-workspaces/:id`) to populate `access` and `permissions`.
- No create/update/delete — data source.
- **JSON:API type:** `team-workspaces`. No divergent fields; the permission enums map straight from the
  go-tfe `TeamAccess` struct.

## Acceptance criteria (these ARE the test)

1. Fixture creates a backing `stackweaver_team_access` grant (a team + workspace + access level), then
   this data source reads it by `team_id` + `workspace_id`; `apply` succeeds.
2. Computed `id` is set and equals the created grant's `id` (the `team-workspaces` primary id).
3. `access` round-trips: it equals the access level configured on the backing grant.
4. `permissions.0.*` reflect the level's implicit (or custom) permissions consistent with the backing
   grant.
5. A `team_id`/`workspace_id` pair with no grant fails the read with a "could not find team access"
   error.
6. **Plan-null quirk:** both input args are Required (not Optional-null), so assert the clearly-Computed
   fields (`id`, `access`, `permissions`).

## Runtime criterion

Read-only data source. It resolves an existing team's effective access level and permission set on a
workspace so a config can branch on or export those permissions. No runtime side effect of its own.

## Docs + example

- Provider docs page: `docs/data-sources/team_access.md` — arguments (`team_id`, `workspace_id`),
  computed attributes (`id`, `access`, nested `permissions`).
- Example: `examples/data-sources/stackweaver_team_access/data-source.tf` — look up a team's access on
  a workspace and reference `data.stackweaver_team_access.x.access`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_team_access`.
