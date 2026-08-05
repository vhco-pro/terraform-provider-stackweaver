<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_token
tfe_alias: tfe_team_token
kind: resource
family: tokens
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_team_token.go
go_tfe_type: TeamToken
compat_doc: docs/internal/tfe-compatibility/resources/tokens/tfe_team_token.md
---
# stackweaver_team_token

Generates the single (legacy, descriptionless) **authentication token** for a team — the credential CI/
automation uses to act as that team. It is the team-scoped sibling of `stackweaver_organization_token`
and is backed by Stackweaver's api-key infrastructure (a team-scoped key flagged `IsTeamToken`, scope
`team:<id>:admin`).

## Client approach

`go-tfe-clean`. The upstream resource (Plugin Framework, `Schema()` at
`internal/provider/resource_tfe_team_token.go:70`) drives the stock `go-tfe` `TeamTokens` service
(`CreateWithOptions` / `Read` / `Delete`) against `teams/:id/authentication-token`. Stackweaver accepts
and returns the stock `authentication-tokens` JSON:API shape unchanged — no wrapper. The resource has no
Update (any change is `RequiresReplace`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | on the legacy path this is the **team id**, not the token id (not load-bearing) |
| `team_id` | string | Required | yes (RequiresReplace) | — | no | `team-*` id |
| `token` | string | Computed | — | — | **yes** | the generated secret; returned **once** on create |
| `expired_at` | string | Optional+Computed | yes (RequiresReplace) | — | no | iso8601 expiry; `null` ⇒ non-expiring (see divergence) |
| `force_regenerate` | bool | Optional | yes (RequiresReplace) | — | no | **provider-side only**, never on the wire; conflicts with `description` |
| `description` | string | Optional | yes (RequiresReplace) | — | no | always `null` on the legacy path; conflicts with `force_regenerate` (see divergence) |

## Wire contract

- **Create/regenerate:** `TeamTokens.CreateWithOptions(team_id, {ExpiredAt?})` →
  `POST /teams/:id/authentication-token`. Sends only `expired-at?`. Returns the `token` value (only
  here). Deletes any existing team token first. Resource id set to `team_id`.
- **Read:** `TeamTokens.Read(team_id)` → `GET /teams/:id/authentication-token`. Metadata only
  (`expired-at`, `created-at`, `last-used-at`); the token value is **never** returned again and is
  carried forward from state.
- **Update:** none — `RequiresReplace` on every attribute.
- **Delete:** `TeamTokens.Delete(team_id)` → `DELETE /teams/:id/authentication-token`.
- **JSON:API type:** `authentication-tokens`. `token` is write-only in the read sense (echoed only on
  create). `description` is always `null` on the wire here; `force_regenerate` never reaches the wire.

## Acceptance criteria (these ARE the test)

1. `apply` of `{team_id}` creates the token; `id` (= `team_id`) and `token` are set in state.
2. Re-`plan` after apply shows **no drift** — confirms the state-id = team-id mapping and that the
   never-returned `token` is not re-derived.
3. `token` never appears in any read/refresh response; it is only ever populated from the create result
   and is marked sensitive (redacted in output).
4. Setting `expired_at` round-trips the iso8601 value; omitting it leaves it `null` (non-expiring) with
   no drift.
5. `team_id`, `expired_at`, `description`, and `force_regenerate` are all ForceNew — changing any of
   them (or toggling `force_regenerate = true`) recreates the token, minting a **new** `token` value.
6. `destroy` removes the token; a subsequent `TeamTokens.Read(team_id)` returns 404 and the old token
   value no longer authenticates.

## Runtime criterion

The minted token genuinely authenticates as a team automation credential: a request bearing the returned
`token` value is accepted and acts with `team:<id>:admin` scope; after `destroy` the same value is
rejected (401). This runtime auth is the point of the resource — **not** CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/team_token.md` — that `token` is shown once and sensitive, the
  `expired_at` null-means-non-expiring behavior, `force_regenerate` semantics, and the out-of-scope
  descriptioned/multi-token note.
- Example: `examples/resources/stackweaver_team_token/resource.tf` — a `stackweaver_team` +
  `stackweaver_team_token`, with the token consumed via a sensitive output.

## Divergences from upstream / TFE

Per `docs/internal/tfe-compatibility/resources/tokens/tfe_team_token.md`:

- **Legacy path only.** The descriptionless single-token-per-team path is implemented. The provider's
  BETA descriptioned / **multiple-tokens-per-team** path (`POST /teams/:id/authentication-tokens` plural,
  by-id `at-*` resource ids) is **out of scope**; setting `description` has no effect and the wire
  `description` is always `null`.
- **`force_regenerate` is provider-side only** — a client flag that forces recreate; it never appears on
  the wire.
- **Expiry default:** TFE defaults a null `expired_at` to 24 months from creation; Stackweaver treats
  `null` as **non-expiring** (drift-free, since Stackweaver returns `null` back on read).
- **`id`:** on the legacy path the resource id is the **team id**, not the token id, so the returned
  token id is not load-bearing.
- **Scope:** the token is granted `team:<id>:admin`; finer-grained team-token permissions are deferred.
