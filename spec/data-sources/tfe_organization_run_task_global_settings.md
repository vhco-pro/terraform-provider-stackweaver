<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_run_task_global_settings
tfe_alias: tfe_organization_run_task_global_settings
kind: data-source
family: run-tasks
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_organization_run_task_global_settings.go
go_tfe_type: RunTask
compat_doc: docs/internal/tfe-compatibility/data-sources/run-tasks/tfe_organization_run_task_global_settings.md
---
# stackweaver_organization_run_task_global_settings

Reads the global run-task settings of an existing organization run task by `task_id` — whether the task
runs globally, its enforcement level, and which run stages it applies to. Read-only lookup companion to
`stackweaver_organization_run_task_global_settings`.

## Client approach

`go-tfe-clean`. The upstream data source is a plugin-framework data source
(`internal/provider/data_source_organization_run_task_global_settings.go:32`) whose read calls
`RunTasks.Read(task_id)` and extracts the `global-configuration` sub-object. Stackweaver's
`GET /tasks/:id` emits the enabled `global-configuration` (bool `enabled`, `stages`,
`enforcement-level`) in the stock go-tfe `RunTask`/`GlobalRunTask` shape, so no wrapper
(`docs/internal/tfe-compatibility/data-sources/run-tasks/tfe_organization_run_task_global_settings.md`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `task_id` | string | Required | — | — | no | lookup key; the id of the run task to read |
| `id` | string | Computed | — | — | no | `tasks` JSON:API primary id (set to the task id on read) |
| `enabled` | bool | Optional (plan-null quirk) | — | — | no | from `global-configuration.enabled` |
| `enforcement_level` | string | Optional (plan-null quirk) | — | — | no | from `global-configuration.enforcement-level` |
| `stages` | list(string) | Optional (plan-null quirk) | — | — | no | from `global-configuration.stages` |

## Wire contract

- **Read/lookup:** `RunTasks.Read(ctx, task_id)` → `GET /tasks/:id`; the provider reads
  `global-configuration` off the returned `RunTask`. If the task exists but has no global support
  (`Global == nil`), a warning is emitted and no state is set. No create/update/delete.
- **JSON:API type:** `tasks` (with the `global-configuration` attr object). No write-only fields; no
  divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. Fixture creates a `stackweaver_organization_run_task` + a
   `stackweaver_organization_run_task_global_settings` (`enabled = true`,
   `enforcement_level = "advisory"`, `stages = ["post_plan"]`), then a
   `data.stackweaver_organization_run_task_global_settings` reading by `task_id`.
2. `data...id` equals the backing task's `id` (the read sets `id` to the task id).
3. Re-`plan` after apply shows **no drift**.
4. **Optional-not-Computed plan-null quirk:** `enabled`/`enforcement_level`/`stages` are marked Optional
   (not Computed), so they are known-null at plan time and cannot be asserted in plan-safe HCL even
   though they do land in state (`terraform state show` shows `enabled=true` /
   `enforcement_level=advisory` / `stages=[post_plan]`). Assert only on the Computed `id`.

## Runtime criterion

Read-only data source. It resolves the global run-task configuration of a task by id, exposing the
task's global-scope flag, enforcement level, and stages for downstream references. No mutating runtime
effect.

## Docs + example

- Provider docs page: `docs/data-sources/organization_run_task_global_settings.md` — argument
  (`task_id`), computed `id`, and `enabled`/`enforcement_level`/`stages` outputs (with the plan-null
  caveat).
- Example: `examples/data-sources/stackweaver_organization_run_task_global_settings/data-source.tf` —
  read a task's global settings by `task_id`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_organization_run_task_global_settings`.
