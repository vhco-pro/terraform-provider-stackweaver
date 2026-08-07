<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_agent_pool_allowed_workspaces
tfe_alias: tfe_agent_pool_allowed_workspaces
kind: resource
family: agent-pools
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_agent_pool_allowed_workspaces.go
go_tfe_type: AgentPoolAllowedWorkspacesUpdateOptions
compat_doc: docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_pool_allowed_workspaces.md
---
# stackweaver_agent_pool_allowed_workspaces

Manages the authoritative allowlist of workspaces that may use a **non**-organization-scoped agent
pool (`organization_scoped = false`). Only workspaces present in the allowlist can route runs to the
pool. Maps onto the pool's `allowed-workspaces` relationship.

## Client approach

`go-tfe-clean`. The upstream legacy SDKv2 resource (`resourceTFEAgentPoolAllowedWorkspaces` at
`internal/provider/resource_tfe_agent_pool_allowed_workspaces.go:20`) drives
`AgentPools.UpdateAllowedWorkspaces` with `AgentPoolAllowedWorkspacesUpdateOptions` and reads the set
back via `AgentPools.Read`. Stackweaver accepts the stock `agent-pools` PATCH with an
`allowed-workspaces` relationship and echoes it back on read
(`docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_pool_allowed_workspaces.md`); no
wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | set to `agent_pool_id` |
| `agent_pool_id` | string | Required | yes | - | no | `agent-pools` id; changing it recreates |
| `allowed_workspace_ids` | set(string) | Required | no | - | no | authoritative set of `ws-*` ids |

## Wire contract

- **Create:** `AgentPools.UpdateAllowedWorkspaces(pool_id, {AllowedWorkspaces})` →
  `PATCH /agent-pools/:id` with `relationships.allowed-workspaces.data = [{type:"workspaces",id}]`.
  Sets `id = agent_pool_id`.
- **Read:** `AgentPools.Read(pool_id)` → `GET /agent-pools/:id`; stores `allowed-workspaces`
  relationship ids back into `allowed_workspace_ids`. 404 → resource removed from state.
- **Update:** diff the set → `UpdateAllowedWorkspaces` with the full new list (send-whole-set, not
  add/remove deltas).
- **Delete:** `UpdateAllowedWorkspaces(pool_id, {AllowedWorkspaces: []})` - clears the relationship;
  the pool itself is untouched.
- **JSON:API type:** `agent-pools` (relationship members are `workspaces`). No write-only fields.
  This resource sends `allowed-workspaces` **without** `omitempty`, so an empty list clears the set.

## Acceptance criteria (these ARE the test)

1. `apply` with `allowed_workspace_ids = [ws_a, ws_b]` on a non-org-scoped pool adds both; on read the
   set round-trips to the same members.
2. Re-`plan` after apply shows **no drift**.
3. Removing `ws_b` and re-`apply` leaves exactly `[ws_a]` on the pool (set remove).
4. Adding `ws_c` and re-`apply` yields exactly `[ws_a, ws_c]` (set add), order-insensitive.
5. `agent_pool_id` is ForceNew - changing it recreates.
6. `destroy` clears the pool's `allowed-workspaces` to empty; a subsequent read shows none of the
   managed workspaces, and the parent pool still exists.

## Runtime criterion

Not `CRUD-only`. The allowlist gates run routing: a workspace present in `allowed_workspace_ids` may
dispatch agent-mode runs to the pool; one absent from it is denied. Verified indirectly - an allowed
workspace's agent-mode run is picked up, a non-allowed workspace's is refused. The resource itself is
CRUD over the relationship set.

## Docs + example

- Provider docs page: `docs/resources/agent_pool_allowed_workspaces.md` - arguments
  (agent_pool_id/allowed_workspace_ids), note that it only has effect when the pool is
  `organization_scoped = false`, and that an empty list clears the allowlist.
- Example: `examples/resources/stackweaver_agent_pool_allowed_workspaces/resource.tf` - a restricted
  pool plus an allowlist of two workspaces.

## Divergences from upstream / TFE

None. Drop-in with `tfe_agent_pool_allowed_workspaces`.
