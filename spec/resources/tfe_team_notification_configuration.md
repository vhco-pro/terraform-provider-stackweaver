<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_notification_configuration
tfe_alias: tfe_team_notification_configuration
kind: resource
family: notifications
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_team_notification_configuration.go
go_tfe_type: NotificationConfiguration
compat_doc: docs/internal/tfe-compatibility/resources/notifications/tfe_team_notification_configuration.md
---
# stackweaver_team_notification_configuration

A team-scoped notification configuration. It shares the notification model, delivery service, and GUI
with the workspace and project notification resources; the difference is the scope (`team`) and the
event — it fires **only** on `change_request:created`, delivering to every workspace the team can reach.
Maps onto Stackweaver's `NotificationConfiguration` bound to a team.

## Client approach

`go-tfe-clean`. The upstream resource (Plugin Framework, `Schema()` at
`internal/provider/resource_tfe_team_notification_configuration.go:125`) drives the stock `go-tfe`
`NotificationConfigurations` service (`Create` scoped to a team, then shared `Read/Update/Delete`).
Stackweaver accepts and returns the stock `notification-configurations` JSON:API shape unchanged — no
wrapper. The `subscribable` polyrelation is `{type: teams}`.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `notification-configurations` primary id |
| `name` | string | Required | no | — | no | |
| `destination_type` | string | Required | yes (RequiresReplace) | — | no | `email`/`generic`/`slack`/`microsoft-teams` |
| `team_id` | string | Required | yes (RequiresReplace) | — | no | `team-*` id; the subscribable |
| `enabled` | bool | Optional+Computed | no | `false` | no | |
| `triggers` | set(string) | Optional | no | — | no | only `change_request:created` accepted |
| `url` | string | Optional | no | — | **yes** | required for `generic`/`slack`/`microsoft-teams`; forbidden for `email`/with email fields |
| `email_addresses` | set(string) | Optional+Computed | no | — | no | `email` only |
| `email_user_ids` | set(string) | Optional+Computed | no | — | no | `email` only |
| `token` | string | Optional | no | — | **yes** | HMAC secret for `generic`; never echoed back on read |
| `token_wo` | string | Optional | no | — | **yes** (WriteOnly) | write-only variant; requires `token_wo_version`; conflicts with `token` |
| `token_wo_version` | int64 | Optional | no | — | no | bumping it triggers a `token_wo` update; conflicts with `token` |

## Wire contract

- **Create:** `NotificationConfigurations.Create(team_id, opts)` →
  `POST /teams/:id/notification-configurations`. Sends `destination-type`, `enabled`, `name`, `url?`,
  `token?`, `triggers?`, `email-addresses?`, `users` relation (`email_user_ids`), and the `subscribable`
  polyrelation `{type: teams, id: team_id}`.
- **Read:** `NotificationConfigurations.Read(id)` → `GET /notification-configurations/:id` (shared).
  Returns everything **except** the token value (the API never returns it).
- **Update:** `NotificationConfigurations.Update(id, opts)` → `PATCH /notification-configurations/:id`
  (shared). `name`, `enabled`, `url`, `token`, `triggers`, email fields in place.
- **Delete:** `NotificationConfigurations.Delete(id)` → `DELETE /notification-configurations/:id`.
- **JSON:API type:** `notification-configurations`. `token`/`token_wo` are write-only: the token is
  echoed only implicitly and is carried forward in state (the wire never returns it); `token_wo` and
  `token_wo_version` never land in plan/state as a value.

## Acceptance criteria (these ARE the test)

1. `apply` of a `generic` config `{name, destination_type="generic", team_id, url, triggers=["change_request:created"], token}` creates it; `id`, `name`, `destination_type`, `enabled`, `triggers`,
   `url`, `team_id` round-trip into state.
2. Re-`plan` after apply shows **no drift** (including that the never-returned `token` is carried
   forward, not re-fetched).
3. `token` is never present in any read/refresh response; `token_wo` (WriteOnly) never appears in plan
   or state at all; both are sensitive/redacted.
4. Updating `name`, `enabled`, `url`, or `triggers` applies in place; `destination_type` and `team_id`
   are ForceNew and recreate.
5. `triggers` accepts only `change_request:created`; any other trigger value is rejected at validation.
6. Destination-type conflict rules hold: `email_addresses`/`email_user_ids` only with `email`; `url`
   only with `generic`/`slack`/`microsoft-teams` and forbidden alongside email fields; `token`/`token_wo`
   forbidden for `email`/`slack`/`microsoft-teams`.
7. `destroy` removes it; a subsequent `NotificationConfigurations.Read(id)` returns 404.

## Runtime criterion

Real delivery: when a change request is filed against a workspace the team can reach, a webhook is
delivered to `url` with an HMAC-SHA512 signature derived from `token`, carrying the change-request
payload (`change_request_*` fields, not run fields). A team with no access to the workspace receives
nothing. Verified by driving a live change request and confirming signed delivery to the entitled teams
only. **Not** CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/team_notification_configuration.md` — the single-trigger
  constraint (`change_request:created`), destination-type conflict rules, `token` vs write-only
  `token_wo`/`token_wo_version`, and the change-request payload shape.
- Example: `examples/resources/stackweaver_team_notification_configuration/resource.tf` — a
  `stackweaver_team` + a `generic` notification with a token and the `change_request:created` trigger.

## Divergences from upstream / TFE

Per `docs/internal/tfe-compatibility/resources/notifications/tfe_team_notification_configuration.md`:

- Team notifications fire **only** on `change_request:created` (exactly what the provider allows).
- A team with org-wide read access is notified for **every** change request in the org — TFE's
  documented "all workspaces the team has access to" semantics, not an accident.
- `email` delivery and `delivery_responses` are deferred (as for the workspace/project scopes); the
  attributes are accepted but delivery is not wired for `email`.
- The change-request generic payload is a distinct shape from the run payload (no `notifications` array;
  `change_request_*` fields), matching HashiCorp's documented team-notification fields.
