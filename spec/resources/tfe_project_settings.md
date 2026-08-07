<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_project_settings
tfe_alias: tfe_project_settings
kind: resource
family: projects
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_project_settings.go
go_tfe_type: Project
compat_doc: docs/internal/tfe-compatibility/resources/projects/tfe_project_settings.md
---
# stackweaver_project_settings

Manages the default execution settings **on an existing project** - the default execution mode
(`remote`/`agent`/`local`) and, for agent execution, the default agent pool. It has no object of its
own: it PATCHes the project. Workspaces in the project inherit these defaults at run time unless they
overwrite their own execution mode. Maps onto Stackweaver's per-project execution defaults.

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework,
`internal/provider/resource_tfe_project_settings.go:217`) drives the `go-tfe` `Projects` service -
`Projects.Read` and `Projects.Update` with `ProjectUpdateOptions` - the exact same endpoint and wire
shape `tfe_project` uses. Stackweaver's `PATCH /projects/:id` accepts and returns the stock `Project`
JSON:API shape (`default-execution-mode`, `default-agent-pool-id` on write / `default-agent-pool`
relation on read, `setting-overwrites`) unchanged
(`docs/internal/tfe-compatibility/resources/projects/tfe_project_settings.md`). No wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | equals `project_id` (the `projects` primary id) |
| `project_id` | string | Required | yes | - | no | `RequiresReplace`; the project to configure |
| `default_execution_mode` | string | Optional+Computed | no | server (`remote`) | no | one of `agent`/`local`/`remote` |
| `default_agent_pool_id` | string | Optional+Computed | no | - | no | required iff mode is `agent`; must be unset otherwise (validated) |
| `overwrites` | object | Computed | - | - | no | `{default_execution_mode: bool, default_agent_pool_id: bool}` - which settings overwrite org defaults |

## Wire contract

- **Create:** no dedicated create - `Projects.Update(project_id, ProjectUpdateOptions)` →
  `PATCH /projects/:id`. Sends `default-execution-mode`, `default-agent-pool-id`, and
  `setting-overwrites` = `{default-execution-mode:true, default-agent-pool:true}` when a mode is set
  (both `false` when unset, deferring to defaults).
- **Read:** `Projects.Read(project_id)` → `GET /projects/:id`. Reads `default-execution-mode`, the
  `default-agent-pool` relation (only when mode is `agent`), and the computed `setting-overwrites`.
- **Update:** `Projects.Update` → `PATCH /projects/:id` - same as create, in place.
- **Delete:** not a real delete - `Projects.Update` with `setting-overwrites`
  `{default-execution-mode:false, default-agent-pool:false}`, reverting the project to the built-in
  `remote` default; then the resource is removed from state.
- **JSON:API type:** `projects`. No write-only fields. `default-agent-pool-id` is a write attr;
  `default-agent-pool` is the read relation. `overwrites` is server-computed. No divergence from stock
  go-tfe.

## Acceptance criteria (these ARE the test)

1. `apply` of `{project_id, default_execution_mode = "local"}` PATCHes the project; `id`,
   `project_id`, `default_execution_mode` round-trip into state and the project reports
   `default-execution-mode: local`.
2. Re-`plan` after apply shows **no drift**.
3. `apply` of `{default_execution_mode = "agent", default_agent_pool_id = <pool>}` round-trips both;
   on read `overwrites.default_execution_mode` and `overwrites.default_agent_pool_id` are `true`.
4. Setting `default_execution_mode = "agent"` **without** `default_agent_pool_id` is rejected at plan
   (validator); setting `default_agent_pool_id` with a non-`agent` mode is rejected.
5. Updating `default_execution_mode` from `local` to `remote` applies **in place** (no recreate);
   changing `project_id` recreates (ForceNew).
6. `destroy` reverts the project's `default-execution-mode` to `remote` (setting-overwrites cleared to
   `false`) and removes the resource from state; a subsequent read shows the setting absent/default.

## Runtime criterion

Not `CRUD-only`. The behavior is **inheritance**: a workspace in the project that does not overwrite
its own execution mode inherits the project default at run time. The meaningful case is `agent` - when
the project defaults to `agent` with a default pool, an inheriting workspace's runs dispatch to that
pool (`backend/internal/api/v2/handlers/terraform/runs.go`); `remote`/`local` change no dispatch
target. Verified live per the compat doc's runtime check (an inheriting workspace's run picks up the
project's default pool; a workspace with its own explicit mode is unaffected).

## Docs + example

- Provider docs page: `docs/resources/project_settings.md` - arguments (project_id,
  default_execution_mode, default_agent_pool_id), the computed `overwrites` object, the "agent requires
  a pool" rule, and the note that a cleared setting reverts to `remote` (no org-level defaults yet).
- Example: `examples/resources/stackweaver_project_settings/resource.tf` - a project +
  project_settings with `default_execution_mode = "agent"` and a default agent pool.

## Divergences from upstream / TFE

None on the wire. One documented behavioral note (already in the compat doc): a cleared setting reverts
to the built-in `remote` default because Stackweaver has no organization-level default settings
(`tfe_organization_default_settings`) yet; TFE would defer to the org default. Drop-in with
`tfe_project_settings`.
