<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_agent_pool
tfe_alias: tfe_agent_pool
kind: data-source
family: agent-pools
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_agent_pool.go
go_tfe_type: AgentPool
compat_doc: docs/internal/tfe-compatibility/data-sources/agent-pools/tfe_agent_pool.md
---
# stackweaver_agent_pool

Resolves an existing agent pool by `name` within an organization and exposes its organization-scoped
flag and its allowed/excluded workspace and project bindings. Maps onto Stackweaver's agent-pool
concept; read-only lookup companion to `stackweaver_agent_pool`.

## Client approach

`go-tfe-clean`. The upstream data source is a legacy SDKv2 resource
(`internal/provider/data_source_agent_pool.go:15`) whose read calls the `fetchAgentPool` helper
(`internal/provider/agent_pool_helpers.go:12`), which lists via `AgentPools.List(org)` and matches on
`name`. Stackweaver's `GET /organizations/:org/agent-pools` returns the stock go-tfe `AgentPool`
JSON:API shape - including `organization-scoped` and the allowed/excluded relations (preloaded) - so no
wrapper is needed (`docs/internal/tfe-compatibility/data-sources/agent-pools/tfe_agent_pool.md`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `name` | string | Required | - | - | no | lookup key; matched against the org agent-pool list |
| `organization` | string | Optional | - | provider default | no | org name; falls back to provider default |
| `id` | string | Computed | - | - | no | `agent-pools` JSON:API primary id of the matched pool |
| `organization_scoped` | bool | Computed | - | - | no | whether the pool is available org-wide |
| `allowed_workspace_ids` | set(string) | Computed | - | - | no | ids from the `allowed-workspaces` relation |
| `allowed_project_ids` | set(string) | Computed | - | - | no | ids from the `allowed-projects` relation |
| `excluded_workspace_ids` | set(string) | Computed | - | - | no | ids from the `excluded-workspaces` relation |

## Wire contract

- **Read/lookup:** `AgentPools.List(org, AgentPoolListOptions)` → `GET /organizations/:org/agent-pools`,
  paginated; the provider matches the item whose `name` equals the input and reads its attributes +
  relations. No create/update/delete.
- **JSON:API type:** `agent-pools`. `organization_scoped` from `organization-scoped`;
  `allowed_workspace_ids`/`allowed_project_ids`/`excluded_workspace_ids` collapsed from the
  `allowed-workspaces` / `allowed-projects` / `excluded-workspaces` relations to id sets. No write-only
  or divergent fields.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. Fixture applies a `stackweaver_agent_pool` (`organization_scoped = false`) + a `stackweaver_workspace`
   + an allowed-workspaces binding, then a `data.stackweaver_agent_pool` reading the pool by `name`.
2. `data...id` equals the backing resource's `id`.
3. `data...organization_scoped` equals `false` (matches the resource).
4. `data...allowed_workspace_ids` contains the bound workspace id.
5. Re-`plan` after apply shows **no drift**.

## Runtime criterion

Read-only data source. It resolves an agent pool by name to its id, scope flag, and allowed/excluded
bindings so other config can reference the pool without hardcoding its id. No mutating runtime effect.

## Docs + example

- Provider docs page: `docs/data-sources/agent_pool.md` - arguments (`name`, `organization`), computed
  attributes (`id`, `organization_scoped`, `allowed_workspace_ids`, `allowed_project_ids`,
  `excluded_workspace_ids`).
- Example: `examples/data-sources/stackweaver_agent_pool/data-source.tf` - look up a pool by name and
  reference `data.stackweaver_agent_pool.this.id`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_agent_pool`.
