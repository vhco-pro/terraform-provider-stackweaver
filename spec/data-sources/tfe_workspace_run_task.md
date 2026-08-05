<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_workspace_run_task
tfe_alias: tfe_workspace_run_task
kind: data-source
family: run-tasks
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_workspace_run_task.go
go_tfe_type: WorkspaceRunTask
compat_doc: docs/internal/tfe-compatibility/resources/run-tasks/tfe_workspace_run_task.md
---
# stackweaver_workspace_run_task

Resolves the association between a workspace and a run task by `workspace_id` + `task_id`, exposing the
association's enforcement level and the stage(s) it runs in. Read-only lookup companion to
`stackweaver_workspace_run_task`.

## Client approach

`go-tfe-clean`. The upstream data source is a plugin-framework data source
(`internal/provider/data_source_workspace_run_task.go:33`) whose read pages
`WorkspaceRunTasks.List(workspace_id)` and matches the item whose `RunTask.ID` equals the input
`task_id`. Stackweaver's `GET /workspaces/:id/tasks` returns the stock go-tfe `WorkspaceRunTask`
JSON:API shape unchanged, so no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `workspace_id` | string | Required | — | — | no | lookup key; the workspace to list tasks for |
| `task_id` | string | Required | — | — | no | lookup key; matched against each item's `RunTask.ID` |
| `id` | string | Computed | — | — | no | `workspace-tasks` JSON:API primary id of the matched association |
| `enforcement_level` | string | Computed | — | — | no | from `WorkspaceRunTask.EnforcementLevel` |
| `stage` | string | Computed | — | — | no | deprecated; single stage from `WorkspaceRunTask.Stage` |
| `stages` | list(string) | Computed | — | — | no | from `WorkspaceRunTask.Stages` |

## Wire contract

- **Read/lookup:** `WorkspaceRunTasks.List(ctx, workspace_id, WorkspaceRunTaskListOptions)` →
  `GET /workspaces/:id/tasks`, paginated; the provider matches the item whose `task` relation id equals
  `task_id`, erroring if none is found. No create/update/delete.
- **JSON:API type:** `workspace-tasks`. `stage` is the deprecated single-stage field (kept alongside
  `stages`). No write-only fields; no divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. Fixture creates a `stackweaver_organization_run_task` + a `stackweaver_workspace` + a
   `stackweaver_workspace_run_task` (`enforcement_level = "advisory"`), then a
   `data.stackweaver_workspace_run_task` reading by `workspace_id` + `task_id`.
2. `data...id` equals the backing `stackweaver_workspace_run_task` resource's `id`.
3. `data...enforcement_level` equals `"advisory"` (these outputs are Computed, so plan-safe to assert).
4. Re-`plan` after apply shows **no drift**.

## Runtime criterion

Read-only data source. It resolves a workspace/run-task association by workspace + task id to its
`workspace-tasks` id, enforcement level, and stages so other config can reference the association
without hardcoding its id. No mutating runtime effect.

## Docs + example

- Provider docs page: `docs/data-sources/workspace_run_task.md` — arguments (`workspace_id`,
  `task_id`), computed `id`, `enforcement_level`, `stage` (deprecated), `stages`.
- Example: `examples/data-sources/stackweaver_workspace_run_task/data-source.tf` — resolve an
  association by workspace + task id.

## Divergences from upstream / TFE

None. Drop-in with `tfe_workspace_run_task`.
