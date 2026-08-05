<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_token
tfe_alias: tfe_organization_token
kind: resource
family: tokens
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_organization_token.go
go_tfe_type: OrganizationToken
compat_doc: docs/internal/tfe-compatibility/resources/tokens/tfe_organization_token.md
---
# stackweaver_organization_token

Manages the single **authentication token per organization** — the credential CI/automation uses to act
on the org. There is exactly one; creating again **regenerates** it (revoking the previous value).
Stackweaver backs it with the existing API-key infrastructure: an org-bound key (`Kind=org`,
`OrganizationID`, scope `org:<id>:admin`) flagged `IsOrgToken=true`, making it a distinct singleton from
ordinary org API keys.

## Client approach

`go-tfe-clean`. Stackweaver's org authentication-token endpoint accepts and returns the stock `go-tfe`
`OrganizationToken` JSON:API shape unchanged
(`docs/internal/tfe-compatibility/resources/tokens/tfe_organization_token.md`); no wrapper. The upstream
resource (legacy SDKv2, `internal/provider/resource_tfe_organization_token.go:22`) drives the `go-tfe`
`OrganizationTokens` service (`Read`, `CreateWithOptions`, `Delete`) verbatim. Because the minted token
is a real api_key, it authenticates through the normal API-key path with no org-wall special-casing.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | set to the organization name |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; changing it recreates |
| `token` | string | Computed | — | — | **yes** | plaintext value returned **once** on create; never read back |
| `expired_at` | string | Optional+Computed | yes | 24 months (server) | no | iso8601/RFC3339; null at create → server defaults to 24 months; changing it recreates |
| `force_regenerate` | bool | Optional | yes | — | no | **provider-side only**; when true, allows recreate over an existing token |

## Wire contract

- **Create:** first `OrganizationTokens.Read(org)` → `GET /organizations/:org/authentication-token` to
  check for an existing token; if one exists and `force_regenerate` is not set, error. Otherwise
  `OrganizationTokens.CreateWithOptions(org, OrganizationTokenCreateOptions{ExpiredAt?})` →
  `POST /organizations/:org/authentication-token`, which replaces any existing token and returns the
  **`token`** value (only here). `id` is set to the org name; `token` and `expired_at` are stored from
  the response.
- **Read:** `OrganizationTokens.Read(org)` → `GET /organizations/:org/authentication-token` — metadata
  only (`expired-at`, `created-at`, `last-used-at`); the `token` value is **never** returned again. 404
  → resource removed from state.
- **Update:** none — all writable attrs (`organization`, `expired_at`, `force_regenerate`) are ForceNew,
  so any change recreates.
- **Delete:** `OrganizationTokens.Delete(org)` → `DELETE /organizations/:org/authentication-token`
  (idempotent; 404 treated as gone).
- **JSON:API type:** `authentication-tokens`. `token` is **write-only** (echoed once on create, never on
  read). `token_type` in go-tfe (`OrganizationTokenCreateOptions.TokenType`, sent as the `?token=` query
  param) is HCP-only and **ignored by TFE and Stackweaver** — the upstream resource does not expose it.

## Acceptance criteria (these ARE the test)

Concrete, testable — the `implement` pipeline generates fixture assertions from these.

1. `apply` of `stackweaver_organization_token { organization = <org> }` creates the token; `id` (= org
   name) and `expired_at` round-trip into state, and `token` is populated (non-empty) from the create
   response.
2. Re-`plan` after apply shows **no drift** (in particular `token` and `expired_at` are stable — read
   returns metadata only and does not clear them).
3. `token` is **sensitive and write-only**: it is present in state after create but the Read/refresh path
   never re-fetches or changes it, and it is masked in plan/apply output.
4. Setting `expired_at` to an explicit iso8601 value round-trips; changing `organization`, `expired_at`,
   or `force_regenerate` recreates (all ForceNew).
5. `force_regenerate = true` lets a second create succeed over an existing org token (mints a new value,
   revoking the old); without it, creating when a token already exists errors.
6. `destroy` revokes it; a subsequent `OrganizationTokens.Read(org)` returns **404**.
7. Import by organization name populates `organization` and reads token metadata (the `token` value
   stays null on import — never retrievable).

## Runtime criterion

Not CRUD-only — the minted token genuinely authenticates. The `token` value acts as an org-admin
automation credential: a request bearing it succeeds against the org's API (authorized by the
`org:<id>:admin` scope through the normal api-key auth path), and after `destroy` (revoke) the same
token is rejected. Verified with a live auth check in the harness.

## Docs + example

- Provider docs page: `docs/resources/organization_token.md` — arguments `organization`, `expired_at`,
  `force_regenerate`; computed sensitive `token`; the singleton/regenerate semantics; import by org
  name; note that `token_type` is HCP-only and not exposed.
- Example: `examples/resources/stackweaver_organization_token/resource.tf` — a token for an org with an
  optional `expired_at`, showing the `sensitive` `token` output usage.

## Divergences from upstream / TFE

None at the wire/attribute level. Notes: `force_regenerate` is a **provider-side-only** flag (no wire
field — each regenerating apply simply re-POSTs). `token_type` is HCP-only and ignored (TFE ignores it
too); the upstream resource does not surface it. Stackweaver grants the org token the `org:<id>:admin`
scope (closest existing scope to TFE's fixed org-token permission set); finer-grained org-token
permissions are out of scope until the RBAC model exposes them.
