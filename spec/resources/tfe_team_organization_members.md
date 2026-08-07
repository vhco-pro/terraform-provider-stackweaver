<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_organization_members
tfe_alias: tfe_team_organization_members
kind: resource
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_team_organization_members.go
go_tfe_type: TeamMemberAddOptions / OrganizationMembership
compat_doc: docs/internal/tfe-compatibility/resources/teams/tfe_team_organization_members.md
---
# stackweaver_team_organization_members

Manages the **full set** of organization memberships on a team as an authoritative list of
`organization_membership_id`s. On update it diffs the set (adds new, removes dropped) so the team ends
up with exactly the listed members. Maps onto Stackweaver's team-membership relationship.

## Client approach

`go-tfe-clean`. The upstream resource (legacy SDKv2, `Schema()` at
`internal/provider/resource_tfe_team_organization_members.go:20`) drives the stock `go-tfe`
`TeamMembers` service (`Add` / `ListOrganizationMemberships` / `Remove`) against the
`teams/:id/relationships/organization-memberships` endpoint. Stackweaver accepts the stock relationship
payload unchanged - no wrapper. Read filters out service-account memberships
(`filterNonServiceAccountOrganizationMembers`) provider-side so they never enter the managed set.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | set to `team_id` |
| `team_id` | string | Required | yes | - | no | `team-*` id |
| `organization_membership_ids` | set(string) | Required | no | - | no | `ou-*` org-membership ids; diffed on update |

## Wire contract

- **Create:** `TeamMembers.Add(team_id, {OrganizationMembershipIDs: [...]})` →
  `POST /teams/:id/relationships/organization-memberships`, body
  `{"data":[{"type":"organization-memberships","id":"<ou-...>"}, ...]}` (all ids). Resource id set to
  `team_id`.
- **Read:** `TeamMembers.ListOrganizationMemberships(team_id)` →
  `GET /teams/:id/relationships/organization-memberships`; the non-service-account ids populate
  `organization_membership_ids`. If none remain, the resource drops from state.
- **Update:** diff the set → `Add` the added ids, then `Remove` (`DELETE .../organization-memberships`)
  the dropped ids that still exist on the team.
- **Delete:** `Remove` every currently-managed (non-service-account) membership.
- **JSON:API type:** `organization-memberships` (relationship endpoint; no standalone primary resource).
  No write-only fields.

## Acceptance criteria (these ARE the test)

1. `apply` with two membership ids adds both; on read `organization_membership_ids` round-trips to the
   same set and `id` equals `team_id`.
2. Re-`plan` after apply shows **no drift**.
3. Adding a third id and re-`apply` adds exactly that membership; the existing two remain.
4. Removing one id and re-`apply` removes exactly that membership; the others remain.
5. Service-account memberships on the team are never pulled into the managed set (no drift from them).
6. `team_id` is ForceNew - changing it recreates.
7. `destroy` removes all managed memberships from the team; a subsequent list shows none of them.

## Runtime criterion

Each listed user gains the team's effective access at run time (team RBAC): after apply all listed users
appear in the team's membership and inherit its workspace/project grants; removed users lose it.
Verified via the team's effective membership. Not config-only.

## Docs + example

- Provider docs page: `docs/resources/team_organization_members.md` - authoritative-set semantics, that
  it keys on `organization_membership_id`, and that service accounts are excluded.
- Example: `examples/resources/stackweaver_team_organization_members/resource.tf` - a team + a set of two
  `stackweaver_organization_membership` ids.

## Divergences from upstream / TFE

None on the wire. Same `organization-memberships`-only relationship as
`stackweaver_team_organization_member` (no user-id path) - see
`docs/internal/tfe-compatibility/resources/teams/tfe_team_organization_members.md`.
