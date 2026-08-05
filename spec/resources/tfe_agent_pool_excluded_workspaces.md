<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_agent_pool_excluded_workspaces
tfe_alias: tfe_agent_pool_excluded_workspaces
kind: resource
family: agent-pools
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_agent_pool_excluded_workspaces.go
go_tfe_type: AgentPoolExcludedWorkspacesUpdateOptions
compat_doc: docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_pool_excluded_workspaces.md
---
# stackweaver_agent_pool_excluded_workspaces

Manages the authoritative exclusion list for an **organization-scoped** agent pool
(`organization_scoped = true`). All org workspaces may use such a pool by default; excluded
workspaces are explicitly denied even so. Useful for blocking, e.g., dev workspaces from a production
pool. Maps onto the pool's `excluded-workspaces` relationship.

## Client approach

`go-tfe-clean`. The upstream legacy SDKv2 resource (`resourceTFEAgentPoolExcludedWorkspaces` at
`internal/provider/resource_tfe_agent_pool_excluded_workspaces.go:20`) drives
`AgentPools.UpdateExcludedWorkspaces` with `AgentPoolExcludedWorkspacesUpdateOptions` and reads the
set back via `AgentPools.Read`. Stackweaver accepts the stock `agent-pools` PATCH with an
`excluded-workspaces` relationship and echoes it back on read
(`docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_pool_excluded_workspaces.md`); no
wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | set to `agent_pool_id` |
| `agent_pool_id` | string | Required | yes | — | no | `agent-pools` id; changing it recreates |
| `excluded_workspace_ids` | set(string) | Required | no | — | no | authoritative set of `ws-*` ids |

## Wire contract

- **Create:** `AgentPools.UpdateExcludedWorkspaces(pool_id, {ExcludedWorkspaces})` →
  `PATCH /agent-pools/:id` with `relationships.excluded-workspaces.data = [{type:"workspaces",id}]`.
  Sets `id = agent_pool_id`.
- **Read:** `AgentPools.Read(pool_id)` → `GET /agent-pools/:id`; stores `excluded-workspaces`
  relationship ids back into `excluded_workspace_ids`. 404 → resource removed from state.
- **Update:** diff the set → `UpdateExcludedWorkspaces` with the full new list (send-whole-set).
- **Delete:** `UpdateExcludedWorkspaces(pool_id, {ExcludedWorkspaces: []})` — clears the relationship;
  the pool itself is untouched.
- **JSON:API type:** `agent-pools` (relationship members are `workspaces`). No write-only fields.
  `excluded-workspaces` is sent **without** `omitempty`, so an empty list clears the set.

## Acceptance criteria (these ARE the test)

1. `apply` with `excluded_workspace_ids = [ws_a, ws_b]` on an org-scoped pool excludes both; on read
   the set round-trips to the same members.
2. Re-`plan` after apply shows **no drift**.
3. Removing `ws_b` and re-`apply` leaves exactly `[ws_a]` excluded (set remove).
4. Adding `ws_c` and re-`apply` yields exactly `[ws_a, ws_c]` excluded (set add), order-insensitive.
5. `agent_pool_id` is ForceNew — changing it recreates.
6. `destroy` clears the pool's `excluded-workspaces` to empty; a subsequent read shows no exclusions,
   and the parent pool still exists.

## Runtime criterion

Not `CRUD-only`. The exclusion list negatively gates run routing: on an org-scoped pool a workspace in
`excluded_workspace_ids` is denied run dispatch even though it would otherwise inherit access.
Verified indirectly — an excluded workspace's agent-mode run is refused while a non-excluded workspace
in the same org is picked up. The resource itself is CRUD over the relationship set.

## Docs + example

- Provider docs page: `docs/resources/agent_pool_excluded_workspaces.md` — arguments
  (agent_pool_id/excluded_workspace_ids), note that it only has effect when the pool is
  `organization_scoped = true`, that exclusions take precedence over project-level allowlists, and
  that an empty list clears the exclusions.
- Example: `examples/resources/stackweaver_agent_pool_excluded_workspaces/resource.tf` — an
  org-scoped pool plus an exclusion of two workspaces.

## Divergences from upstream / TFE

None. Drop-in with `tfe_agent_pool_excluded_workspaces`.
