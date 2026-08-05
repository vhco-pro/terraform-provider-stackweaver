<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_access
tfe_alias: tfe_team_access
kind: resource
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_team_access.go
go_tfe_type: TeamAccess
compat_doc: docs/internal/tfe-compatibility/resources/teams/tfe_team_access.md
---
# stackweaver_team_access

Grants a team a level of access to a single **workspace** — either a fixed level (`read`/`plan`/
`write`/`admin`) or a `custom` set of fine-grained permissions. Maps onto Stackweaver's
`team-workspaces` access record.

## Client approach

`go-tfe-clean`. The upstream resource (legacy SDKv2, `Schema()` at
`internal/provider/resource_tfe_team_access.go:39`) drives the stock `go-tfe` `TeamAccess` service
(`Add/Read/Update/Remove`) against the `team-workspaces` endpoint. Stackweaver accepts and returns the
stock `team-workspaces` JSON:API shape unchanged — no wrapper. A `CustomizeDiff`
(`setCustomOrComputedPermissions`) reconciles the `access` ⇄ `permissions` relationship provider-side;
it emits no extra wire bytes.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `team-workspaces` primary id |
| `team_id` | string | Required | yes | — | no | `team-*` id |
| `workspace_id` | string | Required | yes | — | no | must match `ws-<...>` |
| `access` | string | Optional+Computed | no | server | no | `read`/`plan`/`write`/`admin`; ExactlyOneOf with `permissions`; becomes `custom` when a `permissions` block is set |
| `permissions` | block (list, max 1) | Optional+Computed | no | server | no | fine-grained permissions (below) |

### `permissions` block

| Field | Type | Req/Opt/Computed | Values |
|-------|------|------------------|--------|
| `runs` | string | Required | `read`/`plan`/`apply` |
| `variables` | string | Required | `none`/`read`/`write` |
| `state_versions` | string | Required | `none`/`read-outputs`/`read`/`write` |
| `sentinel_mocks` | string | Required | `none`/`read` |
| `workspace_locking` | bool | Required | — |
| `run_tasks` | bool | Required | — |
| `policy_overrides` | bool | Optional+Computed | — (BETA upstream) |

## Wire contract

- **Create:** `TeamAccess.Add(TeamAccessAddOptions)` → `POST /team-workspaces`. Sends `access` and (when
  `custom`) `runs`, `variables`, `state-versions`, `sentinel-mocks`, `workspace-locking`, `run-tasks`,
  `policy-overrides`; plus `team` and `workspace` relationships.
- **Read:** `TeamAccess.Read(id)` → `GET /team-workspaces/:id`. Returns `access`, all permission attrs,
  and the `team` relationship.
- **Update:** `TeamAccess.Update(id, TeamAccessUpdateOptions)` → `PATCH /team-workspaces/:id` (access +
  permissions in place).
- **Delete:** `TeamAccess.Remove(id)` → `DELETE /team-workspaces/:id`.
- **JSON:API type:** `team-workspaces`. No write-only fields. Import by
  `<ORG>/<WORKSPACE>/<TEAM ACCESS ID>`.

## Acceptance criteria (these ARE the test)

1. `apply` of `{team_id, workspace_id, access="write"}` creates the record; `id`, `access`, and the
   computed `permissions` block round-trip into state.
2. Re-`plan` after apply shows **no drift** (fixed `access` leaves `permissions` computed, stable).
3. A `custom` config (a `permissions` block, no `access`) round-trips: `access` reads back `custom` and
   each named permission matches what was set.
4. Updating `access` from `write` to `read` (and updating a `permissions` field in a custom config)
   applies in place without recreate.
5. `team_id` and `workspace_id` are ForceNew — changing either recreates the record.
6. `destroy` removes it; a subsequent `TeamAccess.Read(id)` returns 404.

## Runtime criterion

The grant is enforced at run time: a member of `team_id` can perform exactly the operations the level/
permissions allow on `workspace_id` (e.g. `access="read"` cannot queue an apply; `runs="apply"` can).
Verified by exercising a run/variable/state operation as a team member. Not config-only.

## Docs + example

- Provider docs page: `docs/resources/team_access.md` — the `access` vs `permissions` mutual exclusion,
  the custom-permission value tables, import format.
- Example: `examples/resources/stackweaver_team_access/resource.tf` — one fixed-level grant and one
  `custom` permissions grant.

## Divergences from upstream / TFE

None. Wire shape and endpoint (`team-workspaces`) match `tfe_team_access`; drop-in.
