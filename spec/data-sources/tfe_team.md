<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_team
tfe_alias: tfe_team
kind: data-source
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_team.go
go_tfe_type: Team
compat_doc: n/a
---
# stackweaver_team

Looks up a single team in an organization by name and exposes its identity and SSO/SCIM attributes.
Maps onto Stackweaver's team concept. Read-only: it resolves the team's `id` (the `teams` primary id)
plus `sso_team_id` and the SCIM attributes for use elsewhere in a config.

## Client approach

`go-tfe-clean`. The data source calls the stock `go-tfe` `Teams` service (`Teams.List` with a `Names`
filter) and consumes the stock `Team` JSON:API shape unchanged; no wrapper. No compatibility detail doc
exists yet (`docs/internal/tfe-compatibility/data-sources/teams/tfe_team.md` is absent) — this spec is
the source of record. The divergence below is value-level (fields left unpopulated), not a wire-shape
change, so it does not force a wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `name` | string | Required | — | — | no | team name to look up; exact match within the org |
| `organization` | string | Optional | — | provider default | no | org name; falls back to the provider's default org |
| `id` | string | Computed | — | — | no | `teams` JSON:API primary id of the matched team |
| `sso_team_id` | string | Computed | — | — | no | SSO team id (`sso-team-id`) |
| `scim_linked` | bool | Computed | — | — | no | **DIVERGENCE — not populated** (see below) |
| `scim_group_name` | string | Computed | — | — | no | **DIVERGENCE — not populated** (see below) |
| `scim_sync_paused` | bool | Computed | — | — | no | **DIVERGENCE — not populated** (see below) |
| `scim_updated_at` | string | Computed | — | — | no | **DIVERGENCE — not populated** (see below), RFC3339 |

## Wire contract

- **Read (lookup):** `Teams.List(org, TeamListOptions{Names: [name]})` → `GET
  /organizations/:org/teams?filter[names]=:name`. Paginates until a `Team` with an exact `Name` match
  is found; sets `id` to that team's id and copies `sso_team_id` and the SCIM fields into state.
- No create/update/delete — data source.
- **JSON:API type:** `teams`. SCIM attributes (`scim-linked`, `scim-group-name`, `scim-sync-paused`,
  `scim-updated-at`) are read via nil-guards on the go-tfe pointers; on Stackweaver they are not emitted
  by the backend and therefore stay at their zero values in state.

## Acceptance criteria (these ARE the test)

1. Fixture creates a backing team (e.g. `stackweaver_team` with a known `name`) in an org, then this
   data source reads it by that `name`; `apply` succeeds.
2. Computed `id` is set and equals the created team's `id` (the `teams` primary id).
3. `sso_team_id` round-trips: it matches the value configured on the backing team (or `""` when unset).
4. Looking up a non-existent name fails the read with a "could not find team org/name" error.
5. **Plan-null quirk:** `organization` is Optional-not-Computed, so assert the clearly-Computed fields
   (`id`, `sso_team_id`) rather than the input args.
6. **DIVERGENCE assertion:** `scim_linked`, `scim_group_name`, `scim_sync_paused`, `scim_updated_at`
   are expected to be empty/zero on Stackweaver; the fixture must not assert real SCIM values.

## Runtime criterion

Read-only data source. It resolves an existing team's identity (`id`, `sso_team_id`) so downstream
resources (team access, team tokens) can reference the team by name instead of a hard-coded id. No
runtime side effect of its own.

## Docs + example

- Provider docs page: `docs/data-sources/team.md` — arguments (`name`, `organization`), computed
  attributes (`id`, `sso_team_id`, and the SCIM attributes with a note that they are unpopulated on
  Stackweaver).
- Example: `examples/data-sources/stackweaver_team/data-source.tf` — look up a team by name in an org
  and reference `data.stackweaver_team.x.id`.

## Divergences from upstream / TFE

**Value-level divergence (documented):** the SCIM attributes `scim_linked`, `scim_group_name`,
`scim_sync_paused`, and `scim_updated_at` are not populated by Stackweaver — SCIM group linkage is a
Zitadel concern and is not surfaced through this API, so these Computed fields stay at their zero
values. The schema and wire shape are otherwise a drop-in for `tfe_team`.
