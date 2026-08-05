<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_run_task
tfe_alias: tfe_organization_run_task
kind: resource
family: run-tasks
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_organization_run_task.go
go_tfe_type: RunTask
compat_doc: docs/internal/tfe-compatibility/resources/run-tasks/tfe_organization_run_task.md
---
# stackweaver_organization_run_task

Defines an organization-level run task: an external HTTP service that receives signed webhooks at run
stage boundaries. Maps 1:1 onto Stackweaver's `run_tasks` concept. The task is the reusable definition
that `stackweaver_workspace_run_task` attaches to workspaces and
`stackweaver_organization_run_task_global_settings` applies org-wide.

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework,
`internal/provider/resource_tfe_organization_run_task.go:108`) drives `go-tfe`'s
`RunTasks.Create/Read/Update/Delete` and the stock `RunTask` JSON:API shape (`tasks`, kebab-case).
Stackweaver returns that shape unchanged
(`docs/internal/tfe-compatibility/resources/run-tasks/tfe_organization_run_task.md`); no wrapper. The
one shape subtlety is deliberate and go-tfe-compatible: the `global-configuration` sub-object is
**always emitted** with a boolean `enabled` — go-tfe only decodes the sub-object when that key is a JSON
bool, so emitting it as a minimum `{"enabled": false}` is exactly what the decode quirk requires (and
`stackweaver_organization_run_task_global_settings` depends on it).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `task-` + 16 alphanumerics |
| `name` | string | Required | no | — | no | unique per org |
| `organization` | string | Optional+Computed | yes | provider default | no | org name |
| `url` | string | Required | no | — | no | validated http/https; verified at create/update |
| `category` | string | Optional+Computed | no | `"task"` | no | must be `task` |
| `hmac_key` | string | Optional+Computed | no | `""` | **yes** | write-only on the wire (never echoed); conflicts with `hmac_key_wo` |
| `hmac_key_wo` | string | Optional (write-only) | no | — | **yes** | provider-side write-only variant; requires `hmac_key_wo_version`; conflicts with `hmac_key` |
| `hmac_key_wo_version` | int64 | Optional | no | — | no | bump to trigger an `hmac_key_wo` update |
| `enabled` | bool | Optional+Computed | no | `true` | no | disabled tasks are skipped at stage materialization |
| `description` | string | Optional+Computed | no | `""` | no | null stored as `""` |

## Wire contract

- **Create:** `RunTasks.Create(org, RunTaskCreateOptions)` → `POST /organizations/:org/tasks`. Attrs:
  `name`, `url`, `category`, `enabled`, `description`, `hmac-key?` (from `hmac_key` or `hmac_key_wo`).
  The server runs the URL verification handshake before persisting.
- **Read:** `RunTasks.Read(id)` → `GET /tasks/:id`.
- **Update:** `RunTasks.Update(id, RunTaskUpdateOptions)` → `PATCH /tasks/:id` (name/url/category/
  enabled/description in place; `hmac-key` sent only when it actually changed, via
  `determineHMACKeyForUpdate`). A url/hmac-key change re-runs the handshake.
- **Delete:** `RunTasks.Delete(id)` → `DELETE /tasks/:id` (404 ignored). Workspace attachments cascade.
- **JSON:API type:** `tasks`. `hmac-key` is **write-only** (`omitempty`, never echoed on read).
  `global-configuration` is always emitted with a boolean `enabled` (go-tfe decode quirk).

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, name, url, hmac_key, enabled, description}` creates the task; `id`, `name`,
   `url`, `category` (= `"task"`), `enabled`, `description` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. `hmac_key` is write-only on the wire: the API read response **never echoes** it; state carries only
   the value the config supplied (injected client-side), and `hmac_key_wo` never appears in state at all.
4. Updating `url`, `name`, `enabled`, or `description` applies **in place** (no recreate); updating
   `organization` **recreates** (ForceNew).
5. Bumping `hmac_key_wo_version` (with `hmac_key_wo` set) triggers a PATCH that resends the HMAC key;
   unrelated updates do **not** reset the HMAC key.
6. `category` other than `task` is rejected; a non-http(s) `url` is rejected.
7. `destroy` removes it; a subsequent `RunTasks.Read(id)` returns 404.
8. The task document always carries `global-configuration` with a boolean `enabled` (min
   `{"enabled": false}`), so `stackweaver_organization_run_task_global_settings` can read it.

## Runtime criterion

Run tasks drive external webhook gates at run time. When a run reaches a stage the task is attached to
(via workspace attachment or global settings), the orchestrator fires one signed webhook per result
(hex `HMAC-SHA512` body signature in `X-TFC-Task-Signature`, payload_version 1,
`capabilities: {"outcomes": true}`); the external service PATCHes
`/api/v2/task-results/:id/callback` with passed/failed/running, and the run continues, blocks
(`awaiting_override`), or fails (mandatory). Proven by
`scripts/tfe-compat/runtime/run_tasks_runtime.sh`. Not CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/organization_run_task.md` — arguments (organization/name/url/
  category/enabled/description/hmac_key[/_wo/_wo_version]), computed `id`, the write-only HMAC key
  guidance, import by `<org>/<task_name>`.
- Example: `examples/resources/stackweaver_organization_run_task/resource.tf` — a task with url + a
  variable-sourced `hmac_key`.

## Divergences from upstream / TFE

None at the resource level. Documented, wire-safe scope cuts in the backing runtime (not schema): the
`agent-pool` relation for request forwarding is cut (#555); webhook `run_message` is always `""`;
`vcs_*` URLs are null; outcomes body is capped at 1MB; the `?url_code=` callback variant is not
implemented (Bearer `access_token` is the primary path). The always-emitted `global-configuration`
boolean is a go-tfe decode-quirk match, not a divergence. Compat source:
`docs/internal/tfe-compatibility/resources/run-tasks/tfe_organization_run_task.md:27,38,86-97`.
