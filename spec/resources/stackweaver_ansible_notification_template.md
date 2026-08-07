<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_notification_template
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
# stackweaver_ansible_notification_template

**Native resource - no TFE equivalent.** An org-scoped notification channel (AWX-style notification
template) that can be attached to job templates and workflows. Delivery type is `webhook`, `email`, or
`teams`; channel-specific non-secret settings live in `config`; the single sensitive value (webhook
basic-auth password / SMTP password) is a **write-only** encrypted `secret`. Model:
`core/models/ansible_notification.go` (`AnsibleNotificationTemplate`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleNotificationTemplates`
service. The endpoints use **plain JSON** bodies (`{name, description, type, config, secret}`), not
JSON:API - confirm the exact envelope from
`backend/internal/api/v2/handlers/ansible/notifications.go` (`notificationTemplateRequest`) at
implement time.

**Read gap (IMPORTANT):** there is **no GET-by-id endpoint** for notification templates. The only read
surface is the org-scoped **List** (`GET /organizations/:org/ansible/notification-templates`). The
provider's `Read` must therefore **List and filter client-side by `id`**, treating "not present in the
list" as deleted (drop from state / return not-found). The service method must expose this List and the
resource `Read` must not assume a by-id fetch.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | - | - | no | server-assigned |
| `organization_id` | string (uuid) | Required | yes | - | no | owning org; `(organization_id, name)` unique |
| `name` | string | Required | no | - | no | unique within the org |
| `description` | string | Optional | no | `""` | no | |
| `type` | string | Required | yes | - | no | one of `webhook` \| `email` \| `teams`; determines expected `config` keys |
| `config` | map/json (jsonb) | Required | no | `{}` | no | channel-specific non-secret settings (see below) |
| `secret` | string | Optional | no | - | **yes (write-only)** | encrypted at rest, `json:"-"` on the model - **never echoed** in any read/list response |

`config` shape by `type` (from the model doc comment):
- `webhook`: `{"url", "method", "headers": {..}, "username", "skip_tls_verify"}` (password → `secret`)
- `email`: `{"host", "port", "username", "from", "to": [..], "use_tls"}` (SMTP password → `secret`)
- `teams`: `{"url"}`

## Wire contract

- **Create:** `POST /organizations/:org/ansible/notification-templates` - plain-JSON body:
  `name`, `description?`, `type`, `config`, `secret?`.
- **Read:** **no GET-by-id.** `GET /organizations/:org/ansible/notification-templates` (List) → filter
  by `id` client-side. A missing id means deleted.
- **Update:** `PATCH /ansible/notification-templates/:id` - name/description/config/secret in place.
- **Delete:** `DELETE /ansible/notification-templates/:id`.
- **Action (not part of CRUD):** `POST /ansible/notification-templates/:id/test` - sends a test message
  through the channel. Exposed as an optional action, never a lifecycle step.
- **JSON:API type:** n/a - plain JSON (`json:` tags). `secret` is `json:"-"` (write-only, never
  serialized back). Native client owns marshalling.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization_id, name, type = "webhook", config = {url=...}}` creates the template;
   `id`, `name`, `type`, `config` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. **Read via List:** the provider `Read` resolves the resource by Listing the org's templates and
   filtering by `id` (there is no by-id GET); a template deleted out-of-band disappears from the list
   and the resource is dropped from state on next refresh.
4. `secret` is **write-only** - it is accepted on create/update but **never appears** in state read back
   from the API (the List/Read response omits it); Terraform must not report drift on it.
5. Updating `config`/`description`/`secret` applies in place; changing `type` (or `organization_id`)
   forces recreate.
6. `destroy` removes it; the template no longer appears in the org List.

## Runtime criterion

The template is config with a **real runtime effect**: attached to a job template/workflow (via
`stackweaver_ansible_notification_attachment`) it delivers messages on the configured triggers.
Verified: create a `webhook` template pointed at a capture endpoint and invoke the `test` action (or
attach it and run a job) - the webhook fires with the configured `url`/auth. Config-with-real-effect,
not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_notification_template.md` - the three `type` values and
  their `config` keys, the write-only `secret`, the **List-based Read** behavior (no import-by-id GET),
  and the `test` action.
- Example: `examples/resources/stackweaver_ansible_notification_template/resource.tf` - a `webhook`
  template with `config = { url = ..., method = "POST" }` and a `secret`.

## Divergences from upstream / TFE

Native resource - no TFE equivalent. Notable Stackweaver-specific behaviors: (1) **no GET-by-id** -
Read is List-and-filter; (2) `secret` is write-only (`json:"-"`), never returned; (3) `test` is a
Stackweaver action with no Terraform analogue (optional action, not lifecycle).
