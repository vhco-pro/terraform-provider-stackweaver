<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_workspace_run_task
tfe_alias: tfe_workspace_run_task
kind: resource
family: run-tasks
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_workspace_run_task.go
go_tfe_type: WorkspaceRunTask
compat_doc: docs/internal/tfe-compatibility/resources/run-tasks/tfe_workspace_run_task.md
---
# stackweaver_workspace_run_task

Attaches an organization run task to a single workspace at one or more run stages with advisory or
mandatory enforcement. Maps onto Stackweaver's `workspace_task` (join of a workspace and a run task).

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework, schema V1 in
`internal/provider/resource_tfe_workspace_run_task_schemas.go:91`, logic in
`internal/provider/resource_tfe_workspace_run_task.go`) drives `go-tfe`'s
`WorkspaceRunTasks.Create/Read/Update/Delete` (plus `RunTasks.Read` and `Workspaces.ReadByID` for
validation at create) and the stock `WorkspaceRunTask` JSON:API shape (`workspace-tasks`, kebab-case).
Stackweaver returns that shape unchanged; no wrapper. The one deliberate subtlety is wire-compat: the
response carries **both** the deprecated singular `stage` and the plural `stages`, and a write with only
`stage` is normalized into a one-element `stages` — exactly what go-tfe expects, so no client change.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `wstask-` + 16 alphanumerics |
| `workspace_id` | string | Required | yes | — | no | the workspace to attach to |
| `task_id` | string | Required | yes | — | no | the `task-*` id; must belong to the workspace's org (cross-org → 404) |
| `enforcement_level` | string | Required | no | — | no | `advisory` or `mandatory` (validated `OneOf`) |
| `stage` | string | Optional+Computed | no | server default | no | **deprecated** upstream; prefer `stages`; `OneOf` the four stages |
| `stages` | list(string) | Optional+Computed | no | server default (`["post_plan"]`) | no | ≥1 unique of `pre_plan`/`post_plan`/`pre_apply`/`post_apply` |

## Wire contract

- **Create:** `WorkspaceRunTasks.Create(workspace_id, WorkspaceRunTaskCreateOptions)` →
  `POST /workspaces/:id/tasks`. Attrs: `enforcement-level`, `task` relation, and `stage?`/`stages?`
  (the provider's `extractStageAndStages` picks `stages` when the server supports it — Stackweaver
  advertises `X-TFE-Version 202501-1`, past the `v202404-1` gate). `task_id` and `workspace_id` are
  validated by a `RunTasks.Read` + `Workspaces.ReadByID` first.
- **Read:** `WorkspaceRunTasks.Read(workspace_id, id)` → `GET /workspaces/:id/tasks/:tid`.
- **Update:** `WorkspaceRunTasks.Update(workspace_id, id, WorkspaceRunTaskUpdateOptions)` →
  `PATCH /workspaces/:id/tasks/:tid` (enforcement-level + stages in place).
- **Delete:** `WorkspaceRunTasks.Delete(workspace_id, id)` → `DELETE /workspaces/:id/tasks/:tid`
  (404 ignored). In-flight runs keep their snapshotted results.
- **JSON:API type:** `workspace-tasks`. Responses carry **both** `stage` and `stages`; `stages` is the
  source of truth, `stage` is served as `stages[0]` for round-trip.

## Acceptance criteria (these ARE the test)

1. `apply` of `{workspace_id, task_id, enforcement_level, stages = ["post_plan","pre_apply"]}` creates
   the attachment; `id`, `enforcement_level`, `stages` round-trip into state, and `stage` round-trips as
   `stages[0]`.
2. Re-`plan` after apply shows **no drift** (both `stage` and `stages` present in the response do not
   cause a diff).
3. A write with only the deprecated `stage` normalizes into a one-element `stages` and reads back
   consistently.
4. Updating `enforcement_level` or `stages` applies **in place**; changing `workspace_id` or `task_id`
   **recreates** (ForceNew).
5. `task_id` from another organization returns **404** (not 403 — no tenant disclosure).
6. `stages` rejects duplicates, empty, and out-of-set values.
7. `destroy` removes the attachment; a subsequent `WorkspaceRunTasks.Read(...)` returns 404.

## Runtime criterion

The attachment gates the workspace's runs: at run creation the attachment is snapshotted into
`task_stages`/`task_results`, and when the run reaches each configured stage boundary the task's webhook
fires. `advisory` never blocks; a `mandatory` failure blocks the run pending a human override
(`POST /task-stages/:id/actions/override`). All four stages and enforcement semantics are exercised live
by `scripts/tfe-compat/runtime/run_tasks_runtime.sh`. Not CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/workspace_run_task.md` — arguments (workspace_id/task_id/
  enforcement_level/stages), the `stage` deprecation notice (prefer `stages`), computed `id`, import by
  `<org>/<workspace_name>/<task_name>`.
- Example: `examples/resources/stackweaver_workspace_run_task/resource.tf` — attaches a
  `stackweaver_organization_run_task` to a `stackweaver_workspace`, `stages = ["post_plan","pre_apply"]`,
  `enforcement_level = "mandatory"`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_workspace_run_task`; prefer `stages` over the upstream-deprecated `stage` (both
round-trip). The dual `stage`+`stages` emission and single-`stage` normalization are wire-compat matches
of go-tfe, not divergences. Compat source:
`docs/internal/tfe-compatibility/resources/run-tasks/tfe_workspace_run_task.md:26,61`.
