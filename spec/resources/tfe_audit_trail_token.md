<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_audit_trail_token
tfe_alias: tfe_audit_trail_token
kind: resource
family: tokens
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_audit_trail_token.go
go_tfe_type: OrganizationToken
compat_doc: docs/internal/tfe-compatibility/resources/tokens/tfe_audit_trail_token.md
---
# stackweaver_audit_trail_token

Manages the single **read-only audit-trail token per organization** — the audit sibling of
`stackweaver_organization_token`. It is driven through the **same** org authentication-token endpoint,
selected by a `?token=audit-trails` query param, and is a **distinct per-org singleton** (an org can
hold both the regular org token and the audit token at once). Stackweaver backs it with an org-bound
api_key (`Kind=org`, `OrganizationID`) flagged `IsAuditToken`, carrying a **read-only** scope
(`org:<id>:read`) rather than the org token's `org:<id>:admin`.

## Client approach

`go-tfe-clean`. Stackweaver's audit-trail-token variant accepts and returns the stock `go-tfe`
`OrganizationToken` JSON:API shape unchanged, dispatched via `?token=audit-trails`
(`docs/internal/tfe-compatibility/resources/tokens/tfe_audit_trail_token.md`); no wrapper. The upstream
resource (Plugin Framework, `internal/provider/resource_tfe_audit_trail_token.go:27`) drives the
`go-tfe` `OrganizationTokens` service `…WithOptions` methods with
`OrganizationTokenReadOptions/CreateOptions/DeleteOptions{TokenType: &tfe.AuditTrailToken}` — i.e. it
sets go-tfe's own `TokenType` field (`audit-trails`), which serializes to the `?token=` query param.
This is TFE's own mechanism, not a Stackweaver invention.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | set to the organization name (UseStateForUnknown) |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; changing it recreates (RequiresReplace) |
| `token` | string | Computed | — | — | **yes** | plaintext value returned **once** on create; never read back |
| `expired_at` | string | Optional+Computed | yes | 24 months (server) | no | RFC3339; null at create → server defaults to 24 months; RequiresReplace, warns if null on create |
| `force_regenerate` | bool | Optional | — | yes | — | no | **provider-side only**; RequiresReplace — when true, allows recreate over an existing audit token |

## Wire contract

- **Create:** first `OrganizationTokens.ReadWithOptions(org, {TokenType: audit-trails})` →
  `GET /organizations/:org/authentication-token?token=audit-trails` to check for an existing audit token;
  if one exists and `force_regenerate` is not set, error. Otherwise
  `OrganizationTokens.CreateWithOptions(org, {TokenType: audit-trails, ExpiredAt?})` →
  `POST /organizations/:org/authentication-token?token=audit-trails`, which replaces any existing audit
  token and returns the **`token`** value (only here). `id` is set to the org name.
- **Read:** `OrganizationTokens.ReadWithOptions(org, {TokenType: audit-trails})` →
  `GET …?token=audit-trails` — metadata only (`expired-at`, `created-at`); the `token` value is **never**
  returned again (state's stored token is carried forward). 404 → resource removed from state.
- **Update:** unsupported — the resource returns an explicit "audit trail tokens cannot be updated,
  regenerate instead" error; every writable attr is ForceNew, so the plan recreates rather than updates.
- **Delete:** `OrganizationTokens.DeleteWithOptions(org, {TokenType: audit-trails})` →
  `DELETE …?token=audit-trails` (idempotent; 404 treated as gone).
- **JSON:API type:** `authentication-tokens` (shared with the org token; the `?token=audit-trails`
  param selects this variant). `token` is **write-only** (echoed once on create, never on read).

## Acceptance criteria (these ARE the test)

Concrete, testable — the `implement` pipeline generates fixture assertions from these.

1. `apply` of `stackweaver_audit_trail_token { organization = <org> }` creates the token via the
   `?token=audit-trails` variant; `id` (= org name) and `expired_at` round-trip into state, and `token`
   is populated (non-empty) from the create response.
2. Re-`plan` after apply shows **no drift** (`token` and `expired_at` stable; read returns metadata only
   and carries the stored token forward).
3. `token` is **sensitive and write-only**: present after create, never re-fetched or altered by
   refresh, and masked in plan/apply output.
4. Setting an explicit `expired_at` round-trips; changing `organization`, `expired_at`, or
   `force_regenerate` recreates (all RequiresReplace). An in-place `update` attempt errors ("regenerate
   token").
5. `force_regenerate = true` lets a second create succeed over an existing audit token (mints a new
   value, revoking the old); without it, creating when one already exists errors.
6. `destroy` revokes it; a subsequent read via `…?token=audit-trails` returns **404** (the 404 check
   targets the audit variant specifically).
7. **Distinct singleton:** creating the audit token leaves any existing `stackweaver_organization_token`
   untouched — each variant's GET returns its own id; an org can hold both simultaneously.

## Runtime criterion

Not CRUD-only — the minted audit token genuinely authenticates **read-only**. A request bearing it
**reads** an org endpoint successfully (200, authorized by the `org:<id>:read` scope) but is **denied
mutations** (403, enforced by the org-wall); after `destroy` (revoke) it is rejected entirely. Verified
with a live auth check that also confirms it coexists with the regular org token as a distinct
singleton.

## Docs + example

- Provider docs page: `docs/resources/audit_trail_token.md` — arguments `organization`, `expired_at`,
  `force_regenerate`; computed sensitive `token`; the shared-endpoint / `?token=audit-trails` dispatch;
  distinct-singleton-vs-org-token note; read-only semantics; import by org name; no-update behavior.
- Example: `examples/resources/stackweaver_audit_trail_token/resource.tf` — an audit token for an org,
  showing the `sensitive` `token` output usage.

## Divergences from upstream / TFE

None at the wire/attribute level. Notes: `force_regenerate` is a **provider-side-only** flag (no wire
field — each regenerating apply re-POSTs). The `?token=audit-trails` dispatch is **TFE's own mechanism**
(go-tfe's `OrganizationTokenOptions.TokenType`), shared endpoint with the org token but a distinct
singleton (`IsAuditToken` marker, separate from `IsOrgToken`). Stackweaver grants the audit token
`org:<id>:read` (read-only, closest faithful mapping of TFE's audit-read-only semantics); a dedicated
audit-log-only scope is out of scope until the RBAC model exposes one.
