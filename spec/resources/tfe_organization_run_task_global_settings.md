<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_run_task_global_settings
tfe_alias: tfe_organization_run_task_global_settings
kind: resource
family: run-tasks
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_organization_run_task_global_settings.go
go_tfe_type: GlobalRunTask / GlobalRunTaskOptions
compat_doc: docs/internal/tfe-compatibility/resources/run-tasks/tfe_organization_run_task_global_settings.md
---
# stackweaver_organization_run_task_global_settings

Applies an existing organization run task to **every** workspace in the organization at the given
stages/enforcement. It has no object of its own - it reads and writes the `global-configuration`
sub-object of the task document (`stackweaver_organization_run_task`). Maps onto the three `Global*`
columns on Stackweaver's `run_tasks`.

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework,
`internal/provider/resource_tfe_organization_run_task_global_settings.go:73`) drives `go-tfe`'s
`RunTasks.Read` (to read the sub-object) and `RunTasks.Update` (to write `GlobalRunTaskOptions`) - there
is no dedicated global-settings client method. Stackweaver serializes/parses the stock `RunTask` shape
with its `global-configuration` sub-object (`GlobalRunTask` / `GlobalRunTaskOptions`, kebab-case)
unchanged; no wrapper. Because it rides the tasks document, it depends on that document **always**
carrying `global-configuration` with a boolean `enabled` (the go-tfe decode quirk; see
`stackweaver_organization_run_task`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | set to the run task's id (there is no separate object) |
| `task_id` | string | Required | yes | - | no | the `task-*` id whose global config this manages |
| `enabled` | bool | Optional+Computed | no | `true` | no | whether the task applies globally |
| `enforcement_level` | string | Required | no | - | no | `advisory` or `mandatory` (validated `OneOf`) |
| `stages` | list(string) | Required | no | - | no | ≥1 unique of `pre_plan`/`post_plan`/`pre_apply`/`post_apply` |

## Wire contract

- **Create:** `RunTasks.Update(task_id, {Global: GlobalRunTaskOptions{Enabled, Stages, EnforcementLevel}})`
  → `PATCH /tasks/:id` with only the `global-configuration` attr sub-object. (`Read` first confirms the
  task exists and supports global config.)
- **Read:** `RunTasks.Read(task_id)` → `GET /tasks/:id`; the resource reads
  `global-configuration.{enabled,stages,enforcement-level}` and errors if the sub-object is absent.
- **Update:** same PATCH as Create. `task_id` changing is ForceNew (recreate).
- **Delete:** `RunTasks.Update(task_id, {Global: {Enabled: false}})` → `PATCH /tasks/:id` writing
  `{"enabled": false}`; the task itself is **never** removed.
- **JSON:API type:** `tasks` (this resource has **no object of its own on the wire** - it rides
  `/tasks/:id`). `global-configuration` is always emitted with a boolean `enabled`.

## Acceptance criteria (these ARE the test)

1. `apply` of `{task_id, enabled, enforcement_level, stages}` PATCHes the task; on the task document
   `global-configuration.enabled` = true and `stages`/`enforcement-level` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. Updating `stages` or `enforcement_level` applies **in place** on the same task document (no recreate);
   changing `task_id` **recreates** (ForceNew).
4. `stages` rejects duplicates, an empty list, and values outside the four valid stages;
   `enforcement_level` rejects anything but `advisory`/`mandatory`.
5. `destroy` writes `global-configuration = {"enabled": false}` on the task and leaves the task itself
   intact - there is **no by-id object to 404**; the contract is that the sub-object reads back disabled
   (`"enabled":false`) and the task's own CRUD still works.
6. Reading global settings on a task that lacks a `global-configuration` sub-object errors (the quirk
   guard), matching upstream.

## Runtime criterion

At run creation, every enabled org task with `global_enabled` applies to the workspace at
`global_stages`/`global_enforcement_level` - **unless** the workspace has its own attachment of the same
task, in which case the attachment's stages/enforcement win (TFE precedence, pinned by the snapshot
dedupe test). Global-settings changes never affect in-flight runs (snapshot semantics). Verified via
`scripts/tfe-compat/runtime/run_tasks_runtime.sh` + the materialization precedence test. Not CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/organization_run_task_global_settings.md` - arguments
  (task_id/enabled/enforcement_level/stages), computed `id`, note that destroy disables (does not delete)
  the task, import by `<org>/<task_name>`.
- Example: `examples/resources/stackweaver_organization_run_task_global_settings/resource.tf` -
  references a `stackweaver_organization_run_task.id`, `stages = ["post_plan"]`,
  `enforcement_level = "advisory"`.

## Divergences from upstream / TFE

None. It has no object of its own - it rides the tasks document (`/tasks/:id`) via the
`global-configuration` sub-object, exactly as upstream. The always-emitted boolean `enabled` is a
go-tfe decode-quirk match, not a divergence. The data source warns (rather than errors) when a task has
no global configuration, matching the provider. Compat source:
`docs/internal/tfe-compatibility/resources/run-tasks/tfe_organization_run_task_global_settings.md:19-25`.
