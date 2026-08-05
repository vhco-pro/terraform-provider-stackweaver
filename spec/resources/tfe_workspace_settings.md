<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_workspace_settings
tfe_alias: tfe_workspace_settings
kind: resource
family: workspaces
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_workspace_settings.go
go_tfe_type: Workspace (WorkspaceUpdateOptions / WorkspaceSettingOverwritesOptions)
compat_doc: docs/internal/tfe-compatibility/resources/workspaces/tfe_workspace_settings.md
---
# stackweaver_workspace_settings

Manages the mutable execution and state-sharing settings of an existing workspace — chiefly
`execution_mode` and `agent_pool_id` — as the authoritative replacement for the deprecated equivalents
on `tfe_workspace`. It is the supported way to attach a workspace to a self-hosted runner (agent pool).
Maps onto the same Stackweaver workspace object, patched by id.

## Client approach

`go-tfe-clean`. The resource is Plugin-Framework
(`internal/provider/resource_tfe_workspace_settings.go`) and has no object of its own: it reads and
patches the workspace via `Workspaces.ReadByIDWithOptions`/`ReadByID` and `Workspaces.UpdateByID`
(`:563,:652`). Stackweaver serves the stock `workspaces` JSON:API shape unchanged
(`docs/internal/tfe-compatibility/resources/workspaces/tfe_workspace_settings.md`), critically exposing
`agent-pool` as a **relationship** (not only the `agent-pool-id` attribute) and the `setting-overwrites`
object; no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | set equal to `workspace_id` |
| `workspace_id` | string | Required | yes | — | no | `RequiresReplace`; the workspace to configure |
| `execution_mode` | string | Optional+Computed | no | — | no | `remote`/`local`/`agent`; unset defers to project/org defaults |
| `agent_pool_id` | string | Optional+Computed | no | — | no | required when `execution_mode = agent`, forbidden otherwise |
| `overwrites` | list(obj{execution_mode,agent_pool}) | Computed | — | — | no | the `setting-overwrites` booleans; drives defer-vs-own logic |
| `global_remote_state` | bool | Optional+Computed | no | — | no | mutually exclusive with `project_remote_state` |
| `project_remote_state` | bool | Optional+Computed | no | — | no | mutually exclusive with `global_remote_state` |
| `remote_state_consumer_ids` | set(string) | Optional+Computed | no | — | no | explicit consumers when neither global nor project sharing |
| `description` | string | Optional+Computed | no | — | no | |
| `auto_apply` | bool | Optional+Computed | no | — | no | |
| `assessments_enabled` | bool | Optional+Computed | no | — | no | |
| `tags` | map(string) | Optional+Computed | no | — | no | key-value tag bindings (`tag-bindings`) |
| `effective_tags` | map(string) | Optional+Computed | no | — | no | tags including inherited |

## Wire contract

- **Create:** first `Workspaces.ReadByIDWithOptions(workspace_id)` to preserve unset settings, then
  `Workspaces.UpdateByID(workspace_id, WorkspaceUpdateOptions)` → `PATCH /workspaces/:id`. Sends
  `execution-mode`, `agent-pool-id`, `global-remote-state`, `project-remote-state`, and a
  `setting-overwrites` object. When `execution_mode` is provided the request marks
  `setting-overwrites.execution-mode` **and** `agent-pool` `true` (the workspace owns its settings);
  when `execution_mode` is unset the overwrites are sent `false/false` so the workspace defers to
  project/org defaults. `agent_pool_id` may be `""` to clear.
- **Read:** `Workspaces.ReadByIDWithOptions(id, {Include:[effective-tag-bindings]})` (fallback
  `ReadByID`) → `GET /workspaces/:id`. Response must include `agent-pool` as a **relationship**
  (`data:{id,type:"agent-pools"}` or `null`), `setting-overwrites`, `execution-mode`, `agent-pool-id`.
- **Update:** same `Workspaces.UpdateByID` → `PATCH /workspaces/:id`. `tags` diffs may also call
  `DeleteAllTagBindings`; `remote_state_consumer_ids` diffs call
  `AddRemoteStateConsumers`/`RemoveRemoteStateConsumers`.
- **Delete:** no destroy of the workspace — `Delete` patches the workspace back to defaults
  (`setting-overwrites` false/false, `execution-mode = "remote"`) via `UpdateByID`, then removes the
  resource from state. State-toggle contract, not lifecycle.
- **JSON:API type:** `workspaces`. `WorkspaceSettingOverwritesOptions` fields are `json:`
  (`execution-mode`, `agent-pool`) (`go-tfe/v1.go:24799`); the read-side `setting-overwrites` are
  `*bool` (`:24545`). No sensitive/write-only fields.

## Acceptance criteria (these ARE the test)

1. `apply` of `{workspace_id, execution_mode = "agent", agent_pool_id = <pool>}` patches the workspace;
   on read `execution_mode` and `agent_pool_id` round-trip, and the `agent-pool` **relationship** is
   present in the workspace response.
2. Re-`plan` after apply shows **no drift**, including the computed `overwrites` list reading back
   `{execution_mode = true, agent_pool = true}`.
3. `execution_mode = "agent"` without `agent_pool_id` fails at plan
   ("If execution mode is \"agent\", \"agent_pool_id\" is required"); `agent_pool_id` with a non-agent
   `execution_mode` also fails ("must not be set").
4. Clearing `execution_mode` from config reverts the workspace to deferring: the request sends
   `setting-overwrites` false/false, `execution_mode` reads back the inherited/`remote` default, and
   `agent_pool_id` reads back null — no drift on the following plan.
5. Changing `execution_mode` in place (e.g. `agent` → `remote`) applies via `PATCH /workspaces/:id`
   without recreate and clears `agent_pool_id`; changing `workspace_id` is ForceNew and recreates.
6. `destroy` does **not** delete the workspace: it resets the workspace to `execution_mode = "remote"`
   with overwrites false/false and removes only this resource from state (assert the workspace still
   exists afterwards, with default execution — a state-toggle, not a 404).

## Runtime criterion

Not CRUD-only — this is the resource that actually binds a workspace to a self-hosted runner. Verified:
after applying `execution_mode = "agent"` with an `agent_pool_id`, a run for that workspace is
dispatched to the named agent pool (a self-hosted runner in that pool executes it); after reverting to
`remote`, subsequent runs execute in remote mode / defer to project/org defaults per the inheritance
chain.

## Docs + example

- Provider docs page: `docs/resources/workspace_settings.md` — arguments (`workspace_id`,
  `execution_mode`, `agent_pool_id`, plus `global_remote_state`/`project_remote_state`/
  `remote_state_consumer_ids`, `description`, `auto_apply`, `assessments_enabled`, `tags`), the
  agent-mode/pool constraint, the "authoritative replacement for the deprecated `tfe_workspace`
  attributes" note, and import by workspace id or `<org>/<name>`.
- Example: `examples/resources/stackweaver_workspace_settings/resource.tf` — a workspace plus a
  `tfe_workspace_settings` pinning it to `agent` mode + an agent pool (mirrors the compat doc example).

## Divergences from upstream / TFE

None on the wire shape or the `execution_mode`/`agent_pool_id` contract — drop-in with
`tfe_workspace_settings` (client is go-tfe-clean). Stackweaver returns both the `agent-pool`
relationship and the `agent-pool-id` attribute and the `setting-overwrites` object, which the go-tfe
client requires for correct state. `run_timeout` and other Stackweaver-only workspace knobs are handled
via the workspace update path, not this resource, so they are out of scope here.
