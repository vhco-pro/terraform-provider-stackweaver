<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_run_task
tfe_alias: tfe_organization_run_task
kind: data-source
family: run-tasks
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_organization_run_task.go
go_tfe_type: RunTask
compat_doc: docs/internal/tfe-compatibility/resources/run-tasks/tfe_organization_run_task.md
---
# stackweaver_organization_run_task

Resolves an existing organization run task by `name` within an organization and exposes its category,
description, enabled flag, and callback URL. Maps onto Stackweaver's run-task concept; read-only lookup
companion to `stackweaver_organization_run_task`.

## Client approach

`go-tfe-clean`. The upstream data source is a plugin-framework data source
(`internal/provider/data_source_organization_run_task.go:65`) whose read calls the
`fetchOrganizationRunTask` helper (`internal/provider/run_task_helpers.go:13`), which lists via
`RunTasks.List(org)` and matches on `name`. Stackweaver's `GET /tasks` (org run-task list) returns the
stock go-tfe `RunTask` JSON:API shape unchanged, so no wrapper. The HMAC key is write-only and never
read (upstream assumes empty).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `name` | string | Required | - | - | no | lookup key; matched against the org run-task list |
| `organization` | string | Optional | - | provider default | no | org name; falls back to provider default |
| `id` | string | Computed | - | - | no | `tasks` JSON:API primary id of the matched task |
| `url` | string | Optional (plan-null quirk) | - | - | no | callback URL; set from `RunTask.URL` on read |
| `category` | string | Optional (plan-null quirk) | - | - | no | task category (e.g. `task`); set from `RunTask.Category` |
| `enabled` | bool | Optional (plan-null quirk) | - | - | no | whether the task is enabled; set from `RunTask.Enabled` |
| `description` | string | Optional (plan-null quirk) | - | - | no | set from `RunTask.Description` |

## Wire contract

- **Read/lookup:** `RunTasks.List(org, RunTaskListOptions)` → `GET /organizations/:org/tasks`,
  paginated; the provider matches the item whose `name` equals the input, then maps its attributes. No
  create/update/delete.
- **JSON:API type:** `tasks`. `hmac-key` is write-only (never echoed) - the data source does not expose
  it. No divergent fields.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. Fixture applies a `stackweaver_organization_run_task` (`name`, `url`, `category = "task"`,
   `enabled`), then a `data.stackweaver_organization_run_task` reading it by `name`.
2. `data...id` equals the backing resource's `id`.
3. Re-`plan` after apply shows **no drift**.
4. Note the Optional-not-Computed plan-null quirk: `url`/`category`/`enabled`/`description` are marked
   Optional (not Computed), so they are known-null at plan time and cannot participate in a plan-safe
   HCL assertion even though they do land in state. Assert on the Computed `id` (the plan-safe value).

## Runtime criterion

Read-only data source. It resolves an org run task by name to its id and metadata so a
`stackweaver_workspace_run_task` (or other config) can attach the task without hardcoding its id. No
mutating runtime effect.

## Docs + example

- Provider docs page: `docs/data-sources/organization_run_task.md` - arguments (`name`,
  `organization`), computed `id`, and the `url`/`category`/`enabled`/`description` outputs (with the
  plan-null caveat).
- Example: `examples/data-sources/stackweaver_organization_run_task/data-source.tf` - look up a task by
  name and reference `data.stackweaver_organization_run_task.this.id`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_organization_run_task`. (The `hmac-key` write-only field being absent from the
read is upstream behavior, not a divergence.)
