<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_membership
tfe_alias: tfe_organization_membership
kind: data-source
family: organizations
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_organization_membership.go
go_tfe_type: OrganizationMembership
compat_doc: docs/internal/tfe-compatibility/data-sources/organizations/tfe_organization_membership.md
---
# stackweaver_organization_membership

Resolves a single organization membership by `email` (or `username`, or an explicit
`organization_membership_id`) within an organization, returning its id, email, `user_id`, and
`username`. Maps onto Stackweaver's organization-membership record.

## Client approach

`go-tfe-clean`. The legacy-SDK data source resolves the membership via
`fetchOrganizationMemberByNameOrEmail` (`OrganizationMemberships.List` with `filter[email]` / `q`,
`include=user`; `internal/provider/organization_members_helpers.go`), then confirms with
`OrganizationMemberships.ReadWithOptions(id, include=user)`. If `organization_membership_id` is supplied
directly, the list step is skipped and the id is read straight. Stackweaver returns the stock go-tfe
`organization-memberships` JSON:API shape unchanged
(`docs/internal/tfe-compatibility/data-sources/organizations/tfe_organization_membership.md`); no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `email` | string | Optional+Computed | - | - | no | lookup key; set from the read if resolved by another key |
| `username` | string | Optional+Computed | - | - | no | lookup key; set from the read's `user` relation |
| `organization` | string | Optional+Computed | - | provider default | no | org name; falls back to the provider default; set from the read |
| `organization_membership_id` | string | Optional+Computed | - | - | no | explicit lookup id; `AtLeastOneOf(email, username)`; **stays null on an email/username lookup** (see quirk) |
| `user_id` | string | Computed | - | - | no | `users` id from the `user` relation |
| `id` | string | Computed | - | - | no | the resolved `organization-memberships` id (SetId) |

## Wire contract

- **Read/lookup:** when `organization_membership_id` is empty →
  `OrganizationMemberships.List(org, {Emails|Query, Include: user})` →
  `GET /organizations/:org/organization-memberships?filter[email]=…&q=…&include=user` to resolve the id,
  then `OrganizationMemberships.ReadWithOptions(id, {Include: user})` →
  `GET /organization-memberships/:id?include=user`. When the id is supplied, only the Read runs.
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `organization-memberships`; `email` on the resource, `user_id`/`username` from the
  embedded `user` relation, `organization` from the `organization` relation. `ErrResourceNotFound` on
  the read clears state (`SetId("")`). No divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

1. Fixture creates a `stackweaver_organization_membership` (an invited email), then reads
   `data.stackweaver_organization_membership` for the same org **by `email`**.
2. The data source's computed `id` equals the created membership's id, and `email` round-trips.
3. `user_id` and `username` are non-empty (populated from the `user` relation).
4. Re-`plan` after apply shows **no drift**.
5. **Provider quirk (do not assert before the read):** on an email/username lookup the provider sets the
   data source's `id` (SetId), **not** the `organization_membership_id` attribute - that attribute is
   Optional-not-set and stays known-null. Assert `.id` (and `email`), never `organization_membership_id`.

## Runtime criterion

Read-only data source. Resolves one membership to its id + user identity within an org; no runtime side
effect beyond the lookup/read.

## Docs + example

- Provider docs page: `docs/data-sources/organization_membership.md` - arguments `email`/`username`/
  `organization`/`organization_membership_id`; computed `user_id`, `id`; note the id-vs-attribute quirk.
- Example: `examples/data-sources/stackweaver_organization_membership/data-source.tf` - look up a
  membership by email.

## Divergences from upstream / TFE

None at the wire level. Behavioral note (upstream provider quirk, not a Stackweaver divergence): resolving
by `email`/`username` populates the data source `id` via `SetId`, leaving the
`organization_membership_id` attribute unset - compare `.id`.
