<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_agent_pool
tfe_alias: tfe_agent_pool
kind: resource
family: agent-pools
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_agent_pool.go
go_tfe_type: AgentPool
compat_doc: docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_pool.md
---
# stackweaver_agent_pool

An agent pool is a named group that self-hosted agents (runners) register into; workspaces with
`execution_mode = "agent"` route their runs to a pool. `organization_scoped` controls the default
access mode: when true every workspace in the org may use the pool (narrowed by
`stackweaver_agent_pool_excluded_workspaces`), when false only explicitly allowed
workspaces/projects may. Maps 1:1 onto Stackweaver's agent-pool concept.

## Client approach

`go-tfe-clean`. Stackweaver's agent-pools endpoints accept and return the stock `go-tfe` `AgentPool`
JSON:API shape (`agent-pools` type) unchanged
(`docs/internal/tfe-compatibility/resources/agent-pools/tfe_agent_pool.md`); no wrapper. The upstream
resource is a legacy SDKv2 resource (`resourceTFEAgentPool` at
`internal/provider/resource_tfe_agent_pool.go:22`) driving the `go-tfe` `AgentPools` service verbatim
(`Create`/`Read`/`Update`/`Delete`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | `agent-pools` JSON:API primary id (UUID) |
| `name` | string | Required | no | - | no | pool name, unique within the org |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; changing it recreates |
| `organization_scoped` | bool | Optional | no | `true` | no | true = all workspaces may use by default |

Note: `agent_count` exists on the go-tfe `AgentPool` struct (computed on read) but the upstream SDKv2
resource does not expose it in its schema, so it is not a resource attribute here.

## Wire contract

- **Create:** `AgentPools.Create(org, AgentPoolCreateOptions)` → `POST /organizations/:org/agent-pools`.
  Attrs sent: `name`, `organization-scoped?` (omitempty).
- **Read:** `AgentPools.Read(id)` → `GET /agent-pools/:id`. Response attrs: `name`,
  `organization-scoped`, `agent-count`, `created-at`; `organization` relation (Read sets
  `organization` from `agentPool.Organization.Name`).
- **Update:** `AgentPools.Update(id, AgentPoolUpdateOptions)` → `PATCH /agent-pools/:id` - `name` and
  `organization_scoped` in place.
- **Delete:** `AgentPools.Delete(id)` → `DELETE /agent-pools/:id`.
- **JSON:API type:** `agent-pools`. No write-only fields. The `allowed-workspaces` /
  `allowed-projects` / `excluded-workspaces` relations are managed by the separate
  `stackweaver_agent_pool_*` resources, not this one.

## Acceptance criteria (these ARE the test)

1. `apply` of `{name, organization, organization_scoped = false}` creates the pool; `id`, `name`,
   `organization`, `organization_scoped` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. Updating `name` in place applies without recreate; toggling `organization_scoped` applies in place.
4. `organization` is ForceNew - changing it recreates the pool (new `id`).
5. `destroy` removes it; a subsequent `AgentPools.Read(id)` returns 404.
6. Default assertion: applying with `organization_scoped` unset reads back `true`.

## Runtime criterion

Not `CRUD-only`. The pool is the routing target for agent-mode runs: after a runner registers into the
pool (via `stackweaver_agent_token`) a workspace with `execution_mode = "agent"` bound to the pool
dispatches its plan/apply to a registered agent. Verified indirectly - a registered runner appears in
the pool's `agent-count` and an agent-mode run is picked up. The resource itself is CRUD; the routing
is exercised by the runner + workspace.

## Docs + example

- Provider docs page: `docs/resources/agent_pool.md` - arguments (name/organization/
  organization_scoped), computed `id`, import by id or `<ORGANIZATION>/<NAME>`.
- Example: `examples/resources/stackweaver_agent_pool/resource.tf` - a minimal named pool in an org
  (organization-scoped and a restricted variant).

## Divergences from upstream / TFE

None. Drop-in with `tfe_agent_pool`.
