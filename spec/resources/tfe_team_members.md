<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team_members
tfe_alias: tfe_team_members
kind: resource
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_team_members.go
go_tfe_type: TeamMemberAddOptions / User
compat_doc: docs/internal/tfe-compatibility/resources/teams/tfe_team_members.md
---
# stackweaver_team_members

Manages the full membership set of a team as an authoritative list. Maps onto Stackweaver team
membership (org members added to / removed from a team).

## Client approach

`go-tfe-clean` **with a documented value-level divergence**. The upstream resource (SDKv2,
`internal/provider/resource_tfe_team_members.go:20`) drives `go-tfe`'s `TeamMembers.Add/List/Remove`,
which send/read a `users` relationship as `{"data":[{"type":"users","id":"<x>"}]}`. Stackweaver
accepts this exact shape unchanged - **but interprets the relationship `id` as the user's email**, not
a TFE username. So `go-tfe` needs no wrapper; the only difference is the *value* callers put in
`usernames`. Captured as a migration note below, not client code.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | set to `team_id` |
| `team_id` | string | Required | yes | - | no | `team-*` id |
| `usernames` | set(string) | Required | no | - | no | **Stackweaver: put user emails here** (see divergence) |

## Wire contract

- **Create:** `TeamMembers.Add(team_id, {Usernames})` → `POST /teams/:id/relationships/users`, body
  `{"data":[{"type":"users","id":"<value>"}]}` per member.
- **Read:** `TeamMembers.List(team_id)` → `GET /teams/:id/relationships/users` (or team `?include`),
  returns `users`; the resource stores each user's identifier back into `usernames`.
- **Update:** diff the set → `Add` new, `Remove` (`DELETE /teams/:id/relationships/users`) dropped.
- **Delete:** `Remove` all current members.
- **JSON:API type:** `users`. Divergence: the `id` is an **email** on Stackweaver, a **username** on TFE.

## Acceptance criteria (these ARE the test)

1. `apply` with `usernames = ["a@example.com","b@example.com"]` adds both; on read `usernames`
   round-trips to the same set (as emails).
2. Re-`plan` after apply shows **no drift**.
3. Removing one entry and re-`apply` removes exactly that membership; the other remains.
4. Adding an entry and re-`apply` adds exactly that membership.
5. `team_id` is ForceNew - changing it recreates.
6. `destroy` removes all managed memberships; a subsequent list shows none of them.

## Runtime criterion

The listed users gain the team's access at run time (team RBAC). Verified: after apply, the users
appear in the team's effective membership and inherit the team's workspace/project access. Not
config-only.

## Docs + example

- Provider docs page: `docs/resources/team_members.md` - **must prominently document** that
  `usernames` takes **emails** on Stackweaver, with a migration note for users coming from TFE
  (where these were usernames).
- Example: `examples/resources/stackweaver_team_members/resource.tf` - a team + two members by email.

## Divergences from upstream / TFE

**Value-level (documented):** `usernames` entries are matched by **email**, not TFE username - the
`users` relationship `id` on the wire carries the email. Wire *shape* is identical to go-tfe, so no
client change; this is a usage/migration note only. Compat source:
`docs/internal/tfe-compatibility/resources/teams/tfe_team_members.md:21,29,47`.
