<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_organization_member
tfe_alias: tfe_team_organization_member
kind: resource
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_team_organization_member.go
go_tfe_type: TeamMemberAddOptions / OrganizationMembership
compat_doc: docs/internal/tfe-compatibility/resources/teams/tfe_team_organization_member.md
---
# stackweaver_team_organization_member

Adds a **single** organization membership to a team, keyed by `organization_membership_id` (not user
id). Non-authoritative: each resource manages exactly one (team, membership) link and leaves other
memberships alone. Maps onto Stackweaver's team-membership relationship.

## Client approach

`go-tfe-clean`. The upstream resource (legacy SDKv2, `Schema()` at
`internal/provider/resource_tfe_team_organization_member.go:21`) drives the stock `go-tfe` `TeamMembers`
service (`Add` / `ListOrganizationMemberships` / `Remove`) against the
`teams/:id/relationships/organization-memberships` endpoint. Stackweaver accepts the stock relationship
payload unchanged - no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | synthesized `<TEAM ID>/<ORG MEMBERSHIP ID>` |
| `team_id` | string | Required | yes | - | no | `team-*` id |
| `organization_membership_id` | string | Required | yes | - | no | `ou-*` org-membership id |

There is **no update** operation - both real attributes are ForceNew, so any change recreates.

## Wire contract

- **Create:** `TeamMembers.Add(team_id, {OrganizationMembershipIDs: [id]})` →
  `POST /teams/:id/relationships/organization-memberships`, body
  `{"data":[{"type":"organization-memberships","id":"<ou-...>"}]}` (single entry). Resource id set to
  `<team_id>/<org_membership_id>`.
- **Read:** `TeamMembers.ListOrganizationMemberships(team_id)` →
  `GET /teams/:id/relationships/organization-memberships`; the resource scans the list for its
  `organization_membership_id` and drops from state if absent.
- **Update:** none - recreate (both attrs ForceNew).
- **Delete:** `TeamMembers.Remove(team_id, {OrganizationMembershipIDs: [id]})` →
  `DELETE /teams/:id/relationships/organization-memberships` with the single entry.
- **JSON:API type:** `organization-memberships` (relationship endpoint; no standalone primary resource).
  No write-only fields.

## Acceptance criteria (these ARE the test)

1. `apply` of `{team_id, organization_membership_id}` adds the one membership; `id` round-trips as
   `<team_id>/<org_membership_id>`, and `team_id` / `organization_membership_id` round-trip unchanged.
2. Re-`plan` after apply shows **no drift**.
3. Read finds the membership in the team's org-membership list; if it is removed out-of-band, the next
   read drops the resource from state (id cleared).
4. Changing `organization_membership_id` (or `team_id`) recreates - ForceNew, no in-place update.
5. `destroy` removes exactly that one membership from the team; other memberships on the team remain.
6. Import accepts both `<TEAM ID>/<ORG MEMBERSHIP ID>` and `<ORG NAME>/<USER EMAIL>/<TEAM NAME>`.

## Runtime criterion

The referenced user gains the team's effective access at run time (team RBAC): after apply the user
appears in the team's membership and inherits the team's workspace/project grants. Verified by the user
resolving the team's access. Not config-only.

## Docs + example

- Provider docs page: `docs/resources/team_organization_member.md` - that it keys on
  `organization_membership_id` (from `stackweaver_organization_membership`), the two import formats, and
  that it is non-authoritative (single link).
- Example: `examples/resources/stackweaver_team_organization_member/resource.tf` - a team + one
  membership referencing an `stackweaver_organization_membership`.

## Divergences from upstream / TFE

None on the wire. Stackweaver supports the `organization-memberships` relationship only (not the
alternative `relationships/users` user-id path), which is exactly what this resource uses - see
`docs/internal/tfe-compatibility/resources/teams/tfe_team_organization_member.md`.
