<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_organization_membership
tfe_alias: tfe_organization_membership
kind: resource
family: organizations
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_organization_membership.go
go_tfe_type: OrganizationMembership
compat_doc: docs/internal/tfe-compatibility/resources/organizations/tfe_organization_membership.md
---
# stackweaver_organization_membership

Adds a user to an organization, identified by **email**. The invited user need not exist yet -
Stackweaver (like TFE) provisions a placeholder user for the email and attaches an active membership.
Maps onto Stackweaver's organization-member concept.

## Client approach

`go-tfe-clean`. The upstream resource is a legacy SDKv2 resource
(`internal/provider/resource_tfe_organization_membership.go:22`) that drives `go-tfe`'s
`OrganizationMemberships.Create` / `ReadWithOptions` / `Delete` verbatim. Stackweaver serves the stock
`organization-memberships` JSON:API shape unchanged
(`docs/internal/tfe-compatibility/resources/organizations/tfe_organization_membership.md`); no wrapper.
The create body carries only `email` on the wire, so there is no username/email value divergence at the
resource level (contrast `tfe_team_members`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | `organization-memberships` primary id (UUID) |
| `email` | string | Required | yes | - | no | user's email; lookup-or-create by email |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; changing it recreates |
| `user_id` | string | Computed | - | - | no | resolved via `user` relationship include |
| `username` | string | Computed | - | - | no | resolved via `user` relationship include |

## Wire contract

- **Create:** `OrganizationMemberships.Create(org, OrganizationMembershipCreateOptions{Email})` →
  `POST /organizations/:org/organization-memberships`. Body attr: `email` only (JSON:API type
  `organization-memberships`). Duplicate email in the org → 409.
- **Read:** `OrganizationMemberships.ReadWithOptions(id, {Include: [user]})` →
  `GET /organization-memberships/:id?include=user`. Response attrs: `email`, `status`; relationships
  `organization`, `user` - the resource stores `email`, `organization.name`, `user.id`, `user.username`.
- **Update:** none - the resource has no `Update`; both mutable-looking attributes (`email`,
  `organization`) are ForceNew, so any change recreates.
- **Delete:** `OrganizationMemberships.Delete(id)` → `DELETE /organization-memberships/:id`.
- **JSON:API type:** `organization-memberships`. No write-only fields. `status` reads back as `active`.
  Note: TFE's deprecated org-level `role` attribute is **not** in the provider schema and not sent.

## Acceptance criteria (these ARE the test)

1. `apply` of `{email, organization}` creates the membership; `id`, `email`, `organization`, `user_id`,
   `username` round-trip into state.
2. Re-`plan` after apply shows **no drift** (notably `user_id`/`username` are stable computed values).
3. Applying an email that has no existing user still succeeds: a placeholder user is created and
   `user_id` is populated on read.
4. `email` and `organization` are ForceNew - changing either recreates the resource (destroy + create),
   not an in-place update.
5. Applying a second membership with the same `email` in the same org fails with a 409 conflict.
6. `destroy` removes it; a subsequent `OrganizationMemberships.ReadWithOptions(id)` returns 404
   (`ErrResourceNotFound`).

## Runtime criterion

The membership grants the user org-scoped presence: after apply the user appears in the org's member
list with `status = active` and can authenticate against the org. It does **not** by itself grant any
team/RBAC access (that is `stackweaver_team_organization_member(s)`). Not `CRUD-only` in the dead
sense - the placeholder-user provisioning and active-membership are observable behaviors.

## Docs + example

- Provider docs page: `docs/resources/organization_membership.md` - arguments (`email`,
  `organization`), computed `id`/`user_id`/`username`, note that membership alone adds no team access,
  import by id or by `<org>/<email>`.
- Example: `examples/resources/stackweaver_organization_membership/resource.tf` - a single member added
  to an org by email.

## Divergences from upstream / TFE

None on the wire shape or attribute set exposed by the provider - drop-in with `tfe_organization_membership`.
Behavioral notes (not divergences): create resolves the user by email (exact, then case-insensitive) and
provisions a placeholder user when none exists; TFE's org-level `role` is deprecated and absent from the
provider schema, so Stackweaver's team-based permissions model is unaffected.
