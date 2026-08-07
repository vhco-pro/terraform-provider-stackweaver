<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_webhook_events
tfe_alias: n/a
kind: data-source
family: vcs
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/webhook_event.go)
---
# stackweaver_webhook_events

**Native data source - no TFE equivalent.** A read-only audit/debug helper that lists recent VCS
webhook deliveries received for an organization: the delivery log Stackweaver records for each inbound
push / pull_request / ping / installation event, with the status and HTTP response code it returned.
Use it to observe whether a repository's webhooks are reaching the platform. Model:
`core/models/webhook_event.go` (`WebhookEvent`); handler:
`backend/internal/api/v2/handlers/webhook_events.go` (`WebhookEventHandlerV2.List`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `WebhookEvents` service with
`ListByOrganization(ctx, org, {pageSize, pageNumber})` →
`GET /organizations/:org/webhook-events`. The endpoint returns a JSON:API-shaped envelope
(`{"data": [{"type":"webhook-events", ...}], "meta": {"pagination": {...}}}`) with dash-cased
attribute keys (a `?format=simple` snake_case variant also exists for the frontend; the native client
should use the default JSON:API form). Read-only audit log - no create/update/delete.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional+Computed | - | provider default | no | org name; falls back to the provider default |
| `id` | string | Computed | - | - | no | set to the organization name |
| `events` | list(object) | Computed | - | - | no | recent webhook deliveries, newest first |
| `events[].id` | string (uuid) | Computed | - | - | no | delivery id |
| `events[].event_type` | string | Computed | - | - | no | `push` \| `pull_request` \| `ping` \| `installation` \| … |
| `events[].provider` | string | Computed | - | - | no | `github` \| `gitlab` \| … |
| `events[].repository` | string | Computed | - | - | no | `owner/repo`, when applicable |
| `events[].branch` | string | Computed | - | - | no | branch, when applicable |
| `events[].commit` | string | Computed | - | - | no | commit SHA, when applicable |
| `events[].status` | string | Computed | - | - | no | `success` \| `failed` \| `ignored` |
| `events[].response_code` | number | Computed | - | - | no | HTTP code Stackweaver returned |
| `events[].message` | string | Computed | - | - | no | additional info / error message |
| `events[].delivered_at` | string (rfc3339) | Computed | - | - | no | receipt time |
| `events[].processed_at` | string (rfc3339) | Computed | - | - | no | nullable until processed |

Note: the raw webhook `payload` is stored server-side but never exposed by the API (`json:"-"`), so it
is not part of this schema.

## Wire contract

- **Read/lookup:** `WebhookEvents.ListByOrganization(ctx, org, {pageSize, pageNumber})` →
  `GET /organizations/:org/webhook-events?page[size]=&page[number]=` (default page size 50, max 100).
  Paginate via `meta.pagination.total-count` to cover the log.
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `webhook-events`. Attributes are dash-cased (`event-type`, `response-code`,
  `delivered-at`, `processed-at`, …); the native client maps them to the snake_case schema above. A
  `?format=simple` snake_case envelope exists for the frontend - the provider uses the default
  JSON:API form.
- **Auth/scope:** filtered to the organization (`:name` path param); unknown org → 404. The raw
  `payload` field is never returned.

## Acceptance criteria (these ARE the test)

Assert against known dev-stack state: an org whose repository has delivered at least one webhook (e.g.
a `ping` on connection, or a `push`), so the log is non-empty.

1. Reading `data.stackweaver_webhook_events` for the fixture org returns an `events` list; each entry
   carries an `id`, an `event_type`, a `provider`, a `status`, and a `delivered_at`.
2. The computed `id` equals the organization name.
3. At least one entry's `status` is a known value (`success`/`failed`/`ignored`) and `response_code`
   is a populated integer.
4. No entry exposes a raw `payload` field.
5. Re-`plan` after apply shows **no drift** (`events`, `id` are Computed-only).

## Runtime criterion

Read-only audit/observability helper over the webhook delivery log. No runtime side effect beyond the
list read; its value is confirming inbound webhooks are being received and how the platform responded
(e.g. surfacing `failed` deliveries).

## Docs + example

- Provider docs page: `docs/data-sources/webhook_events.md` - argument `organization`; computed
  `events` (each `id`/`event_type`/`provider`/`repository`/`branch`/`commit`/`status`/`response_code`/
  `message`/`delivered_at`/`processed_at`) and `id`.
- Example: `examples/data-sources/stackweaver_webhook_events/data-source.tf` - list an org's recent
  webhook deliveries and output the latest event's type and status.

## Divergences from upstream / TFE

Native data source - no TFE equivalent. JSON:API envelope with dash-cased keys (mapped to snake_case
schema); a `?format=simple` variant exists for the frontend but is not used by the provider. The raw
webhook `payload` is intentionally never exposed. Read-only audit surface with no lifecycle.
