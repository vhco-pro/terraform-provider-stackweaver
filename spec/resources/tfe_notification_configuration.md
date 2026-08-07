<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_notification_configuration
tfe_alias: tfe_notification_configuration
kind: resource
family: notifications
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_notification_configuration.go
go_tfe_type: NotificationConfiguration
compat_doc: docs/internal/tfe-compatibility/resources/notifications/tfe_notification_configuration.md
---
# stackweaver_notification_configuration

Defines a workspace-scoped notification that fires on run lifecycle events and delivers to a destination
(generic HMAC-signed webhook, Slack, Microsoft Teams, or email). Maps onto Stackweaver's
`notification_configuration` bound to a workspace.

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework,
`internal/provider/resource_tfe_notification_configuration.go:154`) drives `go-tfe`'s
`NotificationConfigurations.Create/Read/Update/Delete` and the stock `NotificationConfiguration`
JSON:API shape (`notification-configurations`, kebab-case, polymorphic `subscribable` relation).
Stackweaver returns that shape unchanged
(`docs/internal/tfe-compatibility/resources/notifications/tfe_notification_configuration.md`); no
wrapper. `token`/`url` write-only handling (`token_wo`/`url_wo` + `*_wo_version` with hash-in-private-
state auto-detection) is entirely provider-side and does not change the wire.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | `notification-configurations` primary id |
| `name` | string | Required | no | - | no | |
| `destination_type` | string | Required | yes | - | no | `email`/`generic`/`slack`/`microsoft-teams` (validated `OneOf`) |
| `email_addresses` | set(string) | Optional+Computed | no | - | no | only for `email`; conflicts with generic/slack/teams |
| `email_user_ids` | set(string) | Optional+Computed | no | - | no | only for `email`; conflicts with generic/slack/teams |
| `enabled` | bool | Optional+Computed | no | `false` | no | disabled configs send nothing |
| `token` | string | Optional | no | - | **yes** | HMAC secret for `generic`; write-only on the wire; conflicts with `token_wo` |
| `token_wo` | string | Optional (write-only) | no | - | **yes** | write-only alternative; changes tracked via hash in private state |
| `token_wo_version` | int64 | Optional+Computed | no | - | no | auto-incremented on `token_wo` change, or set manually |
| `triggers` | set(string) | Optional | no | - | no | `run:*` (+ assessment / auto_destroy accepted) |
| `url` | string | Optional | no | - | **yes** | webhook URL for generic/slack/teams; conflicts with email + `url_wo` |
| `url_wo` | string | Optional (write-only) | no | - | **yes** | write-only URL; changes tracked via hash in private state |
| `url_wo_version` | int64 | Optional+Computed | no | - | no | auto-incremented on `url_wo` change, or set manually |
| `workspace_id` | string | Required | yes | - | no | owning workspace |

## Wire contract

- **Create:** `NotificationConfigurations.Create(workspace_id, NotificationConfigurationCreateOptions)`
  → `POST /workspaces/:id/notification-configurations`. Attrs: `destination-type`, `enabled`, `name`,
  `url?` (from `url`/`url_wo`), `token?` (from `token`/`token_wo`), `triggers`, `email-addresses`,
  `users` relation; `subscribable` = the workspace.
- **Read:** `NotificationConfigurations.Read(id)` → `GET /notification-configurations/:id`.
- **Update:** `NotificationConfigurations.Update(id, ...UpdateOptions)` →
  `PATCH /notification-configurations/:id` (name/enabled/triggers/url/token/emails in place; `token`/`url`
  resent only when actually changed via `determineTokenForUpdate`/`determineURLForUpdate`).
- **Delete:** `NotificationConfigurations.Delete(id)` → `DELETE /notification-configurations/:id`.
- **JSON:API type:** `notification-configurations`. `token` is **write-only** (`token` returned empty
  on read; the provider carries the plan/state value forward). `url` may be managed write-only via
  `url_wo`. A `POST /notification-configurations/:id/actions/verify` action supports test delivery.

## Acceptance criteria (these ARE the test)

1. `apply` of a `generic` config `{workspace_id, name, destination_type, url, token, triggers, enabled}`
   creates it; `id`, `name`, `destination_type`, `enabled`, `triggers`, `url` round-trip into state.
2. Re-`plan` after apply shows **no drift** (including no drift on the write-only `token`).
3. `token` is write-only: the API read response **never echoes** it; state carries only the config value
   (carried forward client-side), and `token_wo` never appears in state.
4. Updating `name`, `enabled`, or `triggers` applies **in place**; changing `destination_type` or
   `workspace_id` **recreates** (ForceNew). An unrelated update does **not** reset the token/url.
5. `email_addresses`/`email_user_ids` are rejected for generic/slack/teams; `url` is rejected for
   `email`; `token` and `token_wo` (and `url`/`url_wo`) cannot both be set.
6. Bumping `token_wo_version` (with `token_wo` set) resends the token; likewise `url_wo_version` +
   `url_wo` for the URL. Switching an active write-only attribute back to its plaintext form is blocked.
7. `destroy` removes it; a subsequent `NotificationConfigurations.Read(id)` returns 404.

## Runtime criterion

Notification configs deliver webhooks. An orchestrator poll tracks a `notified_status` marker on each
run and, when a run's status advances past it and matches a trigger, delivers: **generic** POSTs the
TFE-shaped payload signed `X-TFE-Notification-Signature: hex(HMAC-SHA512(token, body))`; **slack** POSTs
`{"text": …}`; **teams** POSTs a MessageCard. Delivery is SSRF-guarded (loopback/RFC1918/link-local/
metadata blocked at dial time on the resolved IP). Proven via a local receiver + the `verify` action and
`TriggerForRun`/HMAC unit tests. Not CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/notification_configuration.md` - arguments (workspace_id/name/
  destination_type/url[/_wo/_wo_version]/token[/_wo/_wo_version]/triggers/enabled/email_addresses/
  email_user_ids), computed `id`, the destination-type conflict matrix, write-only token/url guidance,
  import by id. **Document that `email` is accepted but not delivered on Stackweaver.**
- Example: `examples/resources/stackweaver_notification_configuration/resource.tf` - a `generic`
  webhook with `url`, variable-sourced `token`, and `triggers = ["run:completed","run:errored"]`.

## Divergences from upstream / TFE

None at the wire level - all attributes round-trip (bytes match go-tfe). **Behavioral (documented)
deferrals**, not shape divergences: `email` destinations are accepted and round-trip but **not
delivered**; `assessment:*` and `workspace:auto_destroy_*` triggers are accepted but not fired (no
backing feature); `email_user_ids` is exposed for round-trip but not modelled on the backend, and
`delivery_responses` is not persisted. Delivery is additionally SSRF-guarded at send time only (config
CRUD still accepts any URL, as TFE does). Compat source:
`docs/internal/tfe-compatibility/resources/notifications/tfe_notification_configuration.md:43-46,57-68`.
