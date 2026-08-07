<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_notification_attachment
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_notification.go)
---
# stackweaver_ansible_notification_attachment

**Native resource - no TFE equivalent.** A **relationship** resource that binds a
`stackweaver_ansible_notification_template` (channel) to exactly one target - a job template **or** a
workflow - with the per-trigger flags (`on_started`/`on_success`/`on_failure`) that decide when the
channel fires. It has **create + delete only, no update**: every attribute is ForceNew. Model:
`core/models/ansible_notification.go` (`AnsibleNotificationAttachment`).

## Client approach

`native-client`. Not in `go-tfe`; served by the `internal/stackweaver`
`AnsibleNotificationTemplates` (or a dedicated `Attachments`) service. Endpoints use **plain JSON**
bodies (`{notification_template_id, job_template_id|workflow_id, on_started, on_success, on_failure}`),
not JSON:API - confirm from `backend/internal/api/v2/handlers/ansible/notifications.go` (`attachRequest`)
at implement time.

**Read surface (IMPORTANT):** there is no GET-by-attachment-id endpoint. Attachments are read through
the **target's** notifications list:
`GET /ansible/job-templates/:id/notifications` for a job-template target (handler
`ListForJobTemplate`). The provider `Read` must list the target's attachments and filter by attachment
`id`; "not present" means deleted. For a **workflow** target the equivalent read is the workflow's
notifications listing (verify the exact route at implement time - the job-template read path is the one
confirmed in `ansible_routes.go`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | - | - | no | server-assigned |
| `organization_id` | string (uuid) | Required | yes | - | no | org that owns the channel + target (URL is `/organizations/:name/...`) |
| `notification_template_id` | string (uuid) | Required | yes | - | no | the channel being attached |
| `job_template_id` | string (uuid) | Optional | yes | - | no | **exactly one** of `job_template_id`/`workflow_id` |
| `workflow_id` | string (uuid) | Optional | yes | - | no | **exactly one** of `job_template_id`/`workflow_id` |
| `on_started` | bool | Optional | yes | `false` | no | fire when the target starts |
| `on_success` | bool | Optional | yes | `true` | no | fire on success |
| `on_failure` | bool | Optional | yes | `true` | no | fire on failure |

There is **no PATCH** - changing any trigger flag or target recreates the attachment (delete + create).

## Wire contract

- **Create:** `POST /organizations/:org/ansible/notification-attachments` - plain-JSON body:
  `notification_template_id`, exactly one of `job_template_id`/`workflow_id`, `on_started?`,
  `on_success?`, `on_failure?`. Server rejects zero-or-both targets (`exactly one of ... is required`).
- **Read:** no by-id GET. `GET /ansible/job-templates/:id/notifications` (job-template target) →
  filter by attachment `id`; workflow target read via the workflow notifications listing. A missing id
  means deleted.
- **Update:** **none** - all attributes ForceNew (recreate).
- **Delete:** `DELETE /organizations/:org/ansible/notification-attachments/:attachment_id`.
- **JSON:API type:** n/a - plain JSON. Native client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization_id, notification_template_id, job_template_id, on_failure = true}` creates
   the attachment; `id`, `notification_template_id`, `job_template_id`, and the three trigger flags
   round-trip into state.
2. Re-`plan` after apply shows **no drift** - trigger-flag defaults (`on_success = true`,
   `on_failure = true`, `on_started = false`) settle without a perpetual diff.
3. **Read via target list:** the provider resolves the attachment by listing the target's notifications
   (`GET /ansible/job-templates/:id/notifications` for a job-template target) and filtering by `id`;
   an attachment deleted out-of-band disappears from that list and is dropped from state.
4. Setting **both** or **neither** of `job_template_id`/`workflow_id` is rejected (plan-time validation
   or apply-time 400 `exactly one of ... is required`).
5. Changing any attribute (a trigger flag, the target, or the template) **recreates** the attachment -
   there is no in-place update.
6. `destroy` removes it; it no longer appears in the target's notifications list.

## Runtime criterion

The attachment is the wiring that makes a channel fire: with it in place, a run of the target dispatches
through the notification worker on the enabled triggers. Verified: attach a `webhook` template to a job
template with `on_failure = true`, run a failing job, and observe the webhook fire once.
Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_notification_attachment.md` - the exactly-one-target rule,
  the three trigger flags and their defaults, the create/delete-only (no update) lifecycle, and the
  target-list Read behavior (no import-by-attachment-id GET).
- Example: `examples/resources/stackweaver_ansible_notification_attachment/resource.tf` - attaches a
  `stackweaver_ansible_notification_template` to a `stackweaver_ansible_job_template` with
  `on_failure = true`.

## Divergences from upstream / TFE

Native resource - no TFE equivalent. Relationship resource with **no update path** (all-ForceNew) and
**no by-id GET** (Read is target-list-and-filter). The workflow-target read route is inferred; the
job-template read route (`GET /ansible/job-templates/:id/notifications`) is the confirmed one.
