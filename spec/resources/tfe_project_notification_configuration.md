<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_project_notification_configuration
tfe_alias: tfe_project_notification_configuration
kind: resource
family: notifications
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_project_notification_configuration.go
go_tfe_type: NotificationConfiguration
compat_doc: docs/internal/tfe-compatibility/resources/notifications/tfe_project_notification_configuration.md
---
# stackweaver_project_notification_configuration

Defines a project-scoped notification that fires on run lifecycle events for **any** workspace in the
project and delivers to a destination (generic HMAC-signed webhook, Slack, Microsoft Teams, or email).
Maps onto Stackweaver's shared `notification_configuration` bound to a project (sibling of
`stackweaver_notification_configuration`).

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework,
`internal/provider/resource_tfe_project_notification_configuration.go:138`) drives `go-tfe`'s
`NotificationConfigurations.Create/Read/Update/Delete` and the stock `NotificationConfiguration`
JSON:API shape (`notification-configurations`, kebab-case). The only difference from the workspace
resource is the polymorphic `subscribable` relation, which is `{type: projects}` here. Stackweaver
returns that shape unchanged; no wrapper. `token` write-only handling (`token_wo` + `token_wo_version`)
is provider-side and does not change the wire.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `notification-configurations` primary id |
| `name` | string | Required | no | — | no | |
| `destination_type` | string | Required | yes | — | no | `email`/`generic`/`slack`/`microsoft-teams` (validated `OneOf`) |
| `email_addresses` | set(string) | Optional+Computed | no | — | no | only for `email`; conflicts with generic/slack/teams |
| `email_user_ids` | set(string) | Optional+Computed | no | — | no | only for `email`; conflicts with generic/slack/teams |
| `enabled` | bool | Optional+Computed | no | `false` | no | disabled configs send nothing |
| `token` | string | Optional | no | — | **yes** | HMAC secret for `generic`; write-only on the wire; conflicts with `token_wo` |
| `token_wo` | string | Optional (write-only) | no | — | **yes** | write-only alternative; requires `token_wo_version` |
| `token_wo_version` | int64 | Optional | no | — | no | bump to trigger a `token_wo` update |
| `triggers` | set(string) | Optional | no | — | no | `run:*` (+ assessment accepted-not-fired) |
| `url` | string | Optional | no | — | **yes** | webhook URL for generic/slack/teams; conflicts with email |
| `project_id` | string | Required | yes | — | no | owning project |

Note vs the workspace resource: `project_id` replaces `workspace_id`, and there is **no** `url_wo`/
`url_wo_version` pair here.

## Wire contract

- **Create:** `NotificationConfigurations.Create(project_id, NotificationConfigurationCreateOptions)`
  → `POST /projects/:id/notification-configurations`. Attrs: `destination-type`, `enabled`, `name`,
  `url?`, `token?` (from `token`/`token_wo`), `triggers`, `email-addresses`, `users` relation;
  `subscribable` = the project (`{type: projects}`).
- **Read:** `NotificationConfigurations.Read(id)` → `GET /notification-configurations/:id` (shared).
- **Update:** `NotificationConfigurations.Update(id, ...UpdateOptions)` →
  `PATCH /notification-configurations/:id` (name/enabled/triggers/url/token/emails in place; `token`
  resent only when actually changed via `determineTokenForUpdate`).
- **Delete:** `NotificationConfigurations.Delete(id)` → `DELETE /notification-configurations/:id`.
- **JSON:API type:** `notification-configurations`; `subscribable` = `projects`. `token` is
  **write-only** (never echoed; carried forward from state). A shared
  `POST /notification-configurations/:id/actions/verify` action supports test delivery.

## Acceptance criteria (these ARE the test)

1. `apply` of a `generic` config `{project_id, name, destination_type, url, token, triggers, enabled}`
   creates it; `id`, `name`, `destination_type`, `enabled`, `triggers`, `url`, `project_id` round-trip
   into state, and the `subscribable` relation is `{type: projects}`.
2. Re-`plan` after apply shows **no drift** (including no drift on the write-only `token`).
3. `token` is write-only: the API read response **never echoes** it; state carries only the config value
   (carried forward client-side), and `token_wo` never appears in state.
4. Updating `name`, `enabled`, or `triggers` applies **in place**; changing `destination_type` or
   `project_id` **recreates** (ForceNew). An unrelated update does **not** reset the token.
5. `email_addresses`/`email_user_ids` are rejected for generic/slack/teams; `url` is rejected for
   `email`; `token` and `token_wo` cannot both be set (and `token_wo` requires `token_wo_version`).
6. Bumping `token_wo_version` (with `token_wo` set) resends the token.
7. `destroy` removes it; a subsequent `NotificationConfigurations.Read(id)` returns 404.

## Runtime criterion

Notification configs deliver webhooks. The orchestrator poll passes the run's workspace's `ProjectID`;
a run event in any workspace of the project delivers to that project's enabled configs (in addition to
the workspace's own), using the shared delivery service: **generic** signed
`X-TFE-Notification-Signature: hex(HMAC-SHA512(token, body))`, **slack** `{"text": …}`, **teams**
MessageCard, all SSRF-guarded. Proven live: a real run reached `run:errored` and delivered to the
project's config with a valid HMAC-SHA512 signature. Not CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/project_notification_configuration.md` — arguments (project_id/
  name/destination_type/url/token[/_wo/_wo_version]/triggers/enabled/email_addresses/email_user_ids),
  computed `id`, the destination-type conflict matrix, write-only token guidance, note that project
  notifications fire only on **run** events (change requests notify teams), and that `email` is accepted
  but not delivered. Import by id.
- Example: `examples/resources/stackweaver_project_notification_configuration/resource.tf` — a `generic`
  webhook bound to a `stackweaver_project.id` with a variable-sourced `token` and `run:*` triggers.

## Divergences from upstream / TFE

None at the wire level — all attributes round-trip (bytes match go-tfe). **Behavioral (documented)
deferrals**, shared with the workspace sibling: `email` destinations round-trip but are not delivered;
`assessment:*` triggers are accepted but not fired; `email_user_ids` is exposed for round-trip only;
`delivery_responses` is not persisted; delivery is SSRF-guarded at send time only. Project notifications
fire only on **run** events, which matches TFE (`change_request:created` is valid only on the team-scoped
resource). Compat source:
`docs/internal/tfe-compatibility/resources/notifications/tfe_project_notification_configuration.md:34-36,47-52`.
