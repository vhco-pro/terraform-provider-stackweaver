<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_teams
tfe_alias: tfe_teams
kind: data-source
family: teams
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_teams.go
go_tfe_type: Team
compat_doc: n/a
---
# stackweaver_teams

Lists every team in an organization, exposing their names and a name→id map. Maps onto Stackweaver's
team concept. Read-only: it resolves the full set of teams in the org so a config can iterate names or
resolve any team's id by name.

## Client approach

`go-tfe-clean`. Calls the stock `go-tfe` `Teams.List` service, paging through all results, and consumes
the stock `Team` JSON:API shape unchanged; no wrapper. No compatibility detail doc exists yet
(`docs/internal/tfe-compatibility/data-sources/teams/tfe_teams.md` is absent) - this spec is the source
of record.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional | - | provider default | no | org name; falls back to the provider's default org |
| `id` | string | Computed | - | - | no | set to the organization name (synthetic id) |
| `names` | list(string) | Computed | - | - | no | names of all teams in the org |
| `ids` | map(string) | Computed | - | - | no | team name → `teams` primary id |

## Wire contract

- **Read (lookup):** `Teams.List(org, TeamListOptions{})` → `GET /organizations/:org/teams`. Paginates
  through every page, accumulating `names` (each team's `Name`) and `ids` (`Name` → `ID`). Sets the
  synthetic `id` to the organization name.
- No create/update/delete - data source.
- **JSON:API type:** `teams`. Only `name` and the primary `id` of each team are consumed; no divergent
  fields.

## Acceptance criteria (these ARE the test)

1. Fixture creates one or more backing teams in an org, then this data source reads them; `apply`
   succeeds.
2. Computed `id` is set and equals the organization name.
3. `names` contains each created team's name; `ids[<name>]` equals that team's `id` (the `teams`
   primary id).
4. Reading an org with zero teams fails with a "could not find teams in <org>" error.
5. **Plan-null quirk:** `organization` is Optional-not-Computed, so assert the clearly-Computed outputs
   (`id`, `names`, `ids`) rather than the input arg.

## Runtime criterion

Read-only data source. It resolves the org's full team roster (names + name→id map) so configs can
reference teams by name. No runtime side effect of its own.

## Docs + example

- Provider docs page: `docs/data-sources/teams.md` - argument (`organization`), computed attributes
  (`id`, `names`, `ids`).
- Example: `examples/data-sources/stackweaver_teams/data-source.tf` - list org teams and reference
  `data.stackweaver_teams.x.ids["<name>"]`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_teams`.
