<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_job
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — modeled on internal/provider/resource_tfe_workspace_run.go)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_job.go)
---
# stackweaver_ansible_job

**Native resource — no TFE equivalent; modeled on `tfe_workspace_run`.** An Ansible **job** is an
ephemeral execution, not a reconciled config object. This resource is a **lifecycle trigger** (an
owner-approved pattern, exactly like `stackweaver_workspace_run` / `tfe_workspace_run`): on `create` it
**launches** a job from a job template (with optional overrides) and waits for the job to reach a
**terminal** status, recording the outcome in state; `update` is a no-op for the trigger itself (any
change to the launch inputs is ForceNew — a new launch is a new job); `delete` is a no-op (optionally a
cancel of a still-running job). It orchestrates Stackweaver's existing Ansible jobs API — the job object
it records is immutable execution history, **not** drift-reconciled config. Model:
`core/models/ansible_job.go` (`AnsibleJob`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleJobs` service
(Launch/Read/Cancel/Relaunch + poll). The resource is pure client-side orchestration over that
service — mirror the `tfe_workspace_run` shape: launch → poll `GET /ansible/jobs/:id` until a terminal
`status` → record. The endpoints speak **JSON:API** with hyphenated attribute keys (`job-type`,
`extra-vars`, `skip-tags`, …) and relationship blocks (`playbook`, `inventory`, `project`,
`credential`) — confirm from `backend/internal/api/v2/handlers/ansible/jobs.go`
(`LaunchJobRequest`, `LaunchFromTemplateRequest`) at implement time.

**Backing-API reconciliation (IMPORTANT — flag):** two launch paths exist and they differ:
- `POST /ansible/job-templates/:id/launch` (`LaunchFromTemplate`) is the **template-driven** launch this
  resource wants (create from `job_template_id`), but it currently accepts **only an `extra_vars`
  override** — `limit`/`tags`/`inventory` overrides are **not** wired on this path today.
- `POST /organizations/:name/ansible/jobs` (`LaunchByOrganization`) accepts `limit`/`tags`/`extra_vars`
  (and more) but is driven by **`playbook` + `inventory` relationships**, not `job_template_id`.

So the requested schema (`job_template_id` + `inventory_id?`/`limit?`/`tags?`/`extra_vars?` overrides)
is only **partially** backed: `job_template_id` + `extra_vars` works via the template-launch path today;
`inventory_id`/`limit`/`tags` overrides on a template launch require a small backend addition (extend
`LaunchFromTemplateRequest`/`jobService.LaunchFromTemplate` to accept them, mirroring the org-scoped
launch). Spec them as ForceNew overrides and treat the extra overrides as `backing_api: partial` pending
that server change.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | — | — | no | the launched job's id |
| `organization_id` | string (uuid) | Required | yes | — | no | org to launch in (resolves the template scope) |
| `job_template_id` | string (uuid) | Required | yes | — | no | template to launch from; a new value = a new launch |
| `inventory_id` | string (uuid) | Optional | yes | — | no | override the template's inventory (**partial** — needs template-launch backend support) |
| `limit` | string | Optional | yes | — | no | `--limit` override (**partial** — needs template-launch backend support) |
| `tags` | string | Optional | yes | — | no | `--tags` override (**partial** — needs template-launch backend support) |
| `extra_vars` | map/json (jsonb) | Optional | yes | `{}` | no | extra-vars override (**supported today** on the template-launch path) |
| `wait_for_completion` | bool | Optional | no | `true` | no | provider-side: poll to a terminal status; `false` = fire-and-forget (record `pending`/`running`) |
| `status` | string | Computed | — | — | no | terminal status after wait: `successful` \| `failed` \| `canceled` \| `error` (or `pending`/`running` if not waiting) |
| `started_at` | string (RFC3339) | Computed | — | — | no | |
| `finished_at` | string (RFC3339) | Computed | — | — | no | |
| `exit_code` | int | Computed | — | — | no | ansible-playbook exit code |
| `hosts_ok` / `hosts_changed` / `hosts_failed` / `hosts_unreachable` | int | Computed | — | — | no | recap stats (optional to surface) |

Everything that defines the launch is **ForceNew**: a job is immutable, so changing any input means a
new launch (a new resource), never an in-place mutation.

## Wire contract

- **Create (launch + wait):** `POST /ansible/job-templates/:job_template_id/launch`
  (`LaunchFromTemplate`) with `data.attributes.extra-vars` → returns the created job (JSON:API `data`
  with the job id + `status`). Then poll `GET /ansible/jobs/:id` until `status` is terminal
  (`successful`/`failed`/`canceled`/`error`) when `wait_for_completion = true`. Record `id`, `status`,
  `started_at`, `finished_at`, `exit_code`.
  - The `inventory_id`/`limit`/`tags` overrides map onto this body **once the backend accepts them**
    (see reconciliation note); until then only `extra_vars` is honored on the template-launch path.
    (The org-scoped `POST /organizations/:name/ansible/jobs` accepts `limit`/`tags`/`extra-vars` but is
    playbook+inventory driven, not template driven — not this resource's primary path.)
- **Read:** `GET /ansible/jobs/:id` — reads back `status` + timing + recap. A job deleted/expired
  server-side reads 404; the resource treats a terminal, already-recorded job as stable (does not
  re-launch).
- **Update:** **no-op** for the trigger. All launch inputs are ForceNew, so a change forces a new
  launch rather than an in-place update; the provider `Update` for non-ForceNew fields (only
  `wait_for_completion`) returns without launching.
- **Delete:** **no-op** by default (a completed job is immutable history); optionally
  `POST /ansible/jobs/:id/actions/cancel` if the job is still running, or `DELETE /ansible/jobs/:id` to
  drop the history record. Document as trigger semantics: destroy removes the resource from state
  without "undoing" the run.
- **Actions (not part of CRUD):** `POST /ansible/jobs/:id/actions/cancel`,
  `POST /ansible/jobs/:id/actions/relaunch`; `GET /ansible/jobs/:id/events`,
  `GET /ansible/jobs/:id/output` for live logs. Optional actions, not lifecycle steps.
- **JSON:API type:** `jobs`. Hyphenated attribute keys; `playbook`/`inventory`/`project`/`credential`
  relationship blocks on the org-scoped launch. Native client owns marshalling.

## Acceptance criteria (these ARE the test)

Modeled on `tfe_workspace_run` — this is a trigger, not reconciled config.

1. `apply` of `{organization_id, job_template_id}` (optionally `extra_vars`) **launches a job** and,
   with `wait_for_completion = true` (default), the job reaches a **terminal** status
   (`successful`/`failed`/`canceled`/`error`); `id`, `status`, `started_at`, `finished_at` round-trip
   into state.
2. Re-`plan` after apply with unchanged inputs shows **no drift** and does **not** re-launch — the
   launch is not re-run on a no-op re-plan (same manage-state semantics as `tfe_workspace_run`: the
   recorded job is treated as stable, not reconciled).
3. `job_template_id` (and every override) is **ForceNew** — changing any of them recreates the resource,
   i.e. performs a **new launch**; there is no in-place update of a job.
4. `wait_for_completion = false` fire-and-forget: `apply` launches and returns immediately with
   `status` = `pending`/`running`; it does not block on terminal state.
5. `destroy` removes the resource from state without re-running or reverting the job (no-op delete; a
   still-running job may be `cancel`ed as an option).
6. This resource is documented as a **trigger, not reconciled config**: it records a point-in-time
   execution, and Terraform will not detect or "fix" post-run changes to the job.
7. **Override backing (partial):** `extra_vars` is honored on the template-launch path today; `limit`,
   `tags`, and `inventory_id` are specced as ForceNew but require the noted backend extension to the
   template-launch endpoint — until then their fixtures are gated on that change.

## Runtime criterion

The job **is** the behavior — not CRUD. Verified live: `apply` launches a real Ansible job from a
template that executes against an inventory (runner/agent-pool), streams events, and reaches a terminal
status the resource records; a no-op re-plan does not launch a second job. The launch + terminal-status
wait exercises the real `POST /ansible/job-templates/:id/launch` → `GET /ansible/jobs/:id` polling loop.

## Docs + example

- Provider docs page: `docs/resources/ansible_job.md` — `job_template_id` + the override fields, the
  `wait_for_completion` behavior, the ForceNew-launch / no-update / no-op-destroy trigger semantics
  (contrast with a reconciled resource), and the cancel/relaunch/events/output actions. Mirror the
  `workspace_run` docs framing.
- Example: `examples/resources/stackweaver_ansible_job/resource.tf` — a `stackweaver_ansible_job_template`
  + a `stackweaver_ansible_job` that launches it with an `extra_vars` override and
  `wait_for_completion = true`.

## Divergences from upstream / TFE

Native resource — no TFE equivalent; **modeled on `tfe_workspace_run`** (lifecycle trigger, no stored
reconciled object, ForceNew launch inputs, no-op update, trigger-semantics destroy). Divergence from the
requested schema: `limit`/`tags`/`inventory_id` overrides are **not yet backed** on the template-launch
endpoint (only `extra_vars` is) — specced as ForceNew and flagged `partial` pending a small backend
addition; the org-scoped launch endpoint that does accept those overrides is playbook+inventory driven,
not template driven.
