<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_members
tfe_alias: tfe_organization_members
kind: data-source
family: organizations
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_organization_members.go
go_tfe_type: OrganizationMembership
compat_doc: docs/internal/tfe-compatibility/data-sources/organizations/tfe_organization_members.md
---
# stackweaver_organization_members

Lists every membership in an organization, split into `members` (active) and `members_waiting`
(invited-but-not-yet-accepted). Each entry carries the user's `user_id` and its
`organization_membership_id`. Maps onto Stackweaver's organization-membership listing.

## Client approach

`go-tfe-clean`. The legacy-SDK data source calls `OrganizationMemberships.List` (paged) via the helper
`fetchOrganizationMembers` (`internal/provider/organization_members_helpers.go`) and buckets each item
by `Status` (`active` → `members`, `invited` → `members_waiting`). Stackweaver's org-memberships list
endpoint returns the stock go-tfe `organization-memberships` JSON:API collection unchanged
(`docs/internal/tfe-compatibility/data-sources/organizations/tfe_organization_members.md`); no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional | - | provider default | no | org whose members are listed; falls back to the provider's default org |
| `members` | list(object) | Computed | - | - | no | active memberships; each `{user_id, organization_membership_id}` |
| `members_waiting` | list(object) | Computed | - | - | no | invited/pending memberships; each `{user_id, organization_membership_id}` |
| `members[].user_id` | string | Computed | - | - | no | `users` id of the member |
| `members[].organization_membership_id` | string | Computed | - | - | no | `organization-memberships` id |
| `id` | string | Computed | - | - | no | set to the organization name |

## Wire contract

- **Read/lookup:** `OrganizationMemberships.List(org, options)` → `GET /organizations/:org/organization-memberships`,
  paginated (follows `NextPage` to end). No input filters are sent; the client buckets by `Status`.
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `organization-memberships` (each item embeds a `user` relation for `user_id`).
  `Status` is `active` → `members`, `invited` → `members_waiting`; other statuses are logged and skipped.
  No divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

1. Fixture creates a `stackweaver_organization_membership` (an invited email), then reads
   `data.stackweaver_organization_members` for the same org.
2. The data source's computed `id` is set (equals the organization name).
3. The created membership's `organization_membership_id` appears in `members` ∪ `members_waiting`
   (an invited-but-unaccepted membership lands in `members_waiting`); its `user_id` is non-empty.
4. Re-`plan` after apply shows **no drift**.
5. `members` / `members_waiting` are Computed-only, so no input assertion is needed before the read.

## Runtime criterion

Read-only data source. Resolves the org's active and pending memberships into two lists; no runtime
side effect beyond the list read.

## Docs + example

- Provider docs page: `docs/data-sources/organization_members.md` - argument `organization`; computed
  `members`/`members_waiting` (each `user_id` + `organization_membership_id`) and `id`.
- Example: `examples/data-sources/stackweaver_organization_members/data-source.tf` - list an org's members.

## Divergences from upstream / TFE

None. Drop-in with `tfe_organization_members`.
