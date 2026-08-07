<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_current_user
tfe_alias: tfe_current_user
kind: data-source
family: organizations
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_current_user.go
go_tfe_type: User
compat_doc: docs/internal/tfe-compatibility/data-sources/organizations/tfe_current_user.md
---
# stackweaver_current_user

Returns the user behind the current API credentials (API-key or JWT identity): id, username, email, and
`is_service_account`. Takes no input. Maps onto Stackweaver's authenticated caller - there is no backing
resource.

## Client approach

`go-tfe-clean`. This is a plugin-framework data source that calls `Users.ReadCurrent`, which is
`GET account/details`. Stackweaver added that account-level endpoint
(`backend/internal/api/v2/handlers/account.go`): it resolves the authenticated user from request context
and returns a stock go-tfe `users` JSON:API resource. The endpoint is org-wall **agnostic** (like
`/tokens`), so no target organization is required. No wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | `users` primary id (uuid) |
| `username` | string | Computed | - | - | no | current user's username (Stackweaver falls back to email) |
| `email` | string | Computed | - | - | no | current user's email |
| `is_service_account` | string(bool) | Computed | - | - | no | **always `false`** on Stackweaver - no service-account user kind yet |

## Wire contract

- **Read/lookup:** `Users.ReadCurrent(ctx)` → `GET /api/v2/account/details`. No request body / no input
  attributes. Returns the authenticated caller; 401 when unauthenticated.
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `users`. Maps `id`, `username`, `email`, `is-service-account`. **Value-level
  divergence:** `is-service-account` is always `false` (Stackweaver has no service-account user kind
  yet); all other fields match stock go-tfe.

## Acceptance criteria (these ARE the test)

1. Fixture declares only `data.stackweaver_current_user` (no backing resource - the caller is the
   backing concept).
2. Computed `id`, `username`, and `email` are all non-empty after the read.
3. Re-`plan` after apply shows **no drift**.
4. **Documented value-level divergence:** `is_service_account` is `false` (Stackweaver has no
   service-account kind). A fixture may assert `is_service_account == false`; do not expect `true`.
5. All attributes are Computed, so nothing is asserted before the read.

## Runtime criterion

Read-only data source. Resolves the identity behind the current credentials via `GET /account/details`;
no runtime side effect beyond the account read.

## Docs + example

- Provider docs page: `docs/data-sources/current_user.md` - no arguments; computed `id`, `username`,
  `email`, `is_service_account`; note `is_service_account` is always `false` on Stackweaver.
- Example: `examples/data-sources/stackweaver_current_user/data-source.tf` - read the current caller.

## Divergences from upstream / TFE

Value-level: `is_service_account` is always `false` - Stackweaver has no service-account user kind yet.
Schema and wire shape are otherwise a drop-in for `tfe_current_user`. Stackweaver adds the backing
`GET /api/v2/account/details` endpoint (org-wall agnostic).
