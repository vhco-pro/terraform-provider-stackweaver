<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_schedule
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_schedule.go)
---
# stackweaver_ansible_schedule

**Native resource — no TFE equivalent.** A **declarative** cron schedule that periodically triggers an
Ansible target: a job template, an inventory-source sync, a playbook VCS sync, or a workflow. The
schedule is a persisted, reconciled object (cron expression + timezone + optional date-range window +
enabled/disabled state) — the server-side scheduler fires it; the resource only manages its definition.
Model: `core/models/ansible_schedule.go` (`AnsibleSchedule`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleSchedules` service
(List/Create/Read/Update/Delete). Unlike the plain-`json` playbook endpoints, the schedule endpoints
speak **JSON:API** with hyphenated attribute keys (`schedule-type`, `job-template-id`,
`cron-expression`, `start-date-time`, …) — confirm the exact envelope from
`backend/internal/api/v2/handlers/ansible/schedules.go` (`CreateScheduleRequest`) at implement time.
The enable/disable/run-now actions are exposed as separate action calls, not lifecycle CRUD.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | — | — | no | server-assigned |
| `organization_id` | string (uuid) | Required | yes | — | no | owning org (URL is `/organizations/:name/...`; provider resolves name→scope) |
| `name` | string | Required | no | — | no | display name |
| `description` | string | Optional | no | `""` | no | |
| `type` | string | Required | yes | — | no | one of `job_template` \| `inventory_source` \| `playbook_sync` \| `workflow` (wire key `schedule-type`); determines which target id is required |
| `job_template_id` | string (uuid) | Optional | yes | — | no | set iff `type = job_template`; exactly one target id must match `type` |
| `inventory_source_id` | string (uuid) | Optional | yes | — | no | set iff `type = inventory_source` |
| `playbook_id` | string (uuid) | Optional | yes | — | no | set iff `type = playbook_sync` |
| `workflow_id` | string (uuid) | Optional | yes | — | no | set iff `type = workflow` |
| `cron_expression` | string | Required | no | — | no | standard 5-field cron (`minute hour day month weekday`) |
| `timezone` | string | Required | no | `UTC` | no | IANA tz (e.g. `America/New_York`); server default `UTC` |
| `start_date_time` | string (RFC3339) | Optional | no | — | no | schedule active after this instant |
| `end_date_time` | string (RFC3339) | Optional | no | — | no | schedule stops after this instant |
| `config` | map/json (jsonb) | Optional | no | `{}` | no | extra config (e.g. `extra_vars` override for job-template schedules) |
| `status` | string | Optional+Computed | no | `enabled` | no | `enabled` \| `disabled`; manage via `type`-appropriate value or the enable/disable actions |
| `next_run_at` / `last_run_at` / `last_run_status` / `last_job_id` / `run_count` | (various) | Computed | — | — | no | execution-tracking readback |

## Wire contract

- **Create:** `POST /organizations/:org/ansible/schedules` — JSON:API body under `data.attributes`:
  `name`, `description?`, `schedule-type`, one of `job-template-id`/`inventory-source-id`/`playbook-id`/`workflow-id`,
  `cron-expression`, `timezone`, `start-date-time?`, `end-date-time?`, `config?`.
- **Read:** `GET /ansible/schedules/:schedule_id`.
- **Update:** `PATCH /ansible/schedules/:schedule_id` — name/description/cron/timezone/date-window/config
  in place. Changing `type` or the target id forces recreate (ForceNew).
- **Delete:** `DELETE /ansible/schedules/:schedule_id`.
- **Actions (ephemeral — NOT part of CRUD state):**
  `POST /ansible/schedules/:schedule_id/actions/enable`,
  `POST /ansible/schedules/:schedule_id/actions/disable`,
  `POST /ansible/schedules/:schedule_id/actions/run-now` (fires the target immediately, out of band).
  `enable`/`disable` flip `status` and may be reflected by setting the `status` attribute; `run-now`
  is a fire-once trigger with no Terraform analogue — document as an optional action, never a lifecycle
  step.
- **Envelope:** JSON:API with hyphenated attribute keys. Native client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization_id, name, type = "job_template", job_template_id, cron_expression, timezone}`
   creates the schedule; `id`, `name`, `type`, `cron_expression`, `timezone`, `status` round-trip into
   state.
2. Re-`plan` after apply shows **no drift** — computed `next_run_at`/`last_*`/`run_count` fields must
   not produce a perpetual diff.
3. `timezone` defaults to `UTC` and `status` to `enabled` when omitted.
4. Updating `cron_expression`/`description`/`config` applies in place; changing `type` or the matching
   target id (e.g. `job_template_id`) forces recreate.
5. Exactly one target id may be set and it must correspond to `type`; a mismatch (e.g. `type =
   inventory_source` with `job_template_id`) is rejected by the API and surfaced as a plan/apply error.
6. `destroy` removes it; a subsequent `GET /ansible/schedules/:schedule_id` returns 404.

## Runtime criterion

The schedule is declarative config with a **real runtime effect**: the server-side scheduler evaluates
`cron_expression`/`timezone` within the optional `start`/`end` window and fires the target, advancing
`next_run_at`/`last_run_at`/`run_count`. Verified: create a `job_template` schedule with a
near-future/short-interval cron and observe `run_count`/`last_run_at` advance (or invoke the `run-now`
action and observe `last_job_id` populate). Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_schedule.md` — the four `type` values and their matching
  target id, cron/timezone semantics, the optional date window, and the enable/disable/run-now actions
  (ephemeral, not lifecycle).
- Example: `examples/resources/stackweaver_ansible_schedule/resource.tf` — a job-template schedule
  (`type = "job_template"`, `cron_expression = "0 2 * * *"`, `timezone = "UTC"`) referencing a
  `stackweaver_ansible_job_template` id.

## Divergences from upstream / TFE

Native resource — no TFE equivalent. `run-now` is a Stackweaver action with no Terraform analogue
(surfaced as an optional action, not a lifecycle requirement). `status` is manageable both as an
attribute and via the enable/disable actions; the spec models it as an attribute and documents the
actions as the imperative alternative.
