<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_agent_pool_allowed_projects
tfe_alias: tfe_agent_pool_allowed_projects
kind: resource
family: agent-pools
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_agent_pool_allowed_projects.go
go_tfe_type: AgentPoolAllowedProjectsUpdateOptions
compat_doc: docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_pool_allowed_projects.md
---
# stackweaver_agent_pool_allowed_projects

Manages the authoritative allowlist of projects that may use an agent pool. Every workspace within an
allowed project inherits access to the pool - a coarser-grained alternative to listing individual
workspaces. Maps onto the pool's `allowed-projects` relationship.

## Client approach

`go-tfe-clean`. The upstream legacy SDKv2 resource (`resourceTFEAgentPoolAllowedProjects` at
`internal/provider/resource_tfe_agent_pool_allowed_projects.go:20`) drives
`AgentPools.UpdateAllowedProjects` with `AgentPoolAllowedProjectsUpdateOptions` and reads the set back
via `AgentPools.Read`. Stackweaver accepts the stock `agent-pools` PATCH with an `allowed-projects`
relationship and echoes it back on read
(`docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_pool_allowed_projects.md`); no
wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | set to `agent_pool_id` |
| `agent_pool_id` | string | Required | yes | - | no | `agent-pools` id; changing it recreates |
| `allowed_project_ids` | set(string) | Required | no | - | no | authoritative set of `prj-*` ids |

## Wire contract

- **Create:** `AgentPools.UpdateAllowedProjects(pool_id, {AllowedProjects})` →
  `PATCH /agent-pools/:id` with `relationships.allowed-projects.data = [{type:"projects",id}]`. Sets
  `id = agent_pool_id`.
- **Read:** `AgentPools.Read(pool_id)` → `GET /agent-pools/:id`; stores `allowed-projects`
  relationship ids back into `allowed_project_ids`. 404 → resource removed from state.
- **Update:** diff the set → `UpdateAllowedProjects` with the full new list (send-whole-set).
- **Delete:** `UpdateAllowedProjects(pool_id, {AllowedProjects: []})` - clears the relationship; the
  pool itself is untouched.
- **JSON:API type:** `agent-pools` (relationship members are `projects`). No write-only fields.
  `allowed-projects` is sent **without** `omitempty`, so an empty list clears the set.

## Acceptance criteria (these ARE the test)

1. `apply` with `allowed_project_ids = [prj_a, prj_b]` adds both; on read the set round-trips to the
   same members.
2. Re-`plan` after apply shows **no drift**.
3. Removing `prj_b` and re-`apply` leaves exactly `[prj_a]` (set remove).
4. Adding `prj_c` and re-`apply` yields exactly `[prj_a, prj_c]` (set add), order-insensitive.
5. `agent_pool_id` is ForceNew - changing it recreates.
6. `destroy` clears the pool's `allowed-projects` to empty; a subsequent read shows none of the
   managed projects, and the parent pool still exists.

## Runtime criterion

Not `CRUD-only`. The project allowlist gates run routing at project granularity: any workspace inside
an allowed project may dispatch agent-mode runs to the pool. Verified indirectly - a workspace in an
allowed project is picked up by the pool, one in a non-allowed project is denied. The resource itself
is CRUD over the relationship set.

## Docs + example

- Provider docs page: `docs/resources/agent_pool_allowed_projects.md` - arguments
  (agent_pool_id/allowed_project_ids), note that project access is additive with
  `stackweaver_agent_pool_allowed_workspaces`, and that an empty list clears the allowlist.
- Example: `examples/resources/stackweaver_agent_pool_allowed_projects/resource.tf` - a pool plus an
  allowlist of one project.

## Divergences from upstream / TFE

None. Drop-in with `tfe_agent_pool_allowed_projects`.
