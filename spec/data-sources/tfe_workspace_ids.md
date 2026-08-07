<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_workspace_ids
tfe_alias: tfe_workspace_ids
kind: data-source
family: workspaces
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_workspace_ids.go
go_tfe_type: Workspace
compat_doc: docs/internal/tfe-compatibility/data-sources/workspaces/tfe_workspace_ids.md
---
# stackweaver_workspace_ids

Lists workspaces in an organization, filtered by `names` (a `*` wildcard or explicit list) and/or by
tags, returning `ids` (a **map** name→id) and `full_names` (name→"org/name"). Maps onto Stackweaver's
workspace list endpoint.

## Client approach

`go-tfe-clean`. The legacy-SDK data source calls `Workspaces.List(org, options)` (paged) →
`GET /organizations/:org/workspaces`, passing `search[tags]` (from `tag_names`),
`search[exclude-tags]` (from `exclude_tags`), `filter[tagged][N][key|value]` (from
`tag_filters.include`), and `?include=effective_tag_bindings` when a tag filter is active. Name matching
(`includedByName`, incl. `*` wildcards) and `tag_filters.exclude` are applied client-side over the
returned effective tag bindings. Stackweaver's list endpoint honours these params against its binding
model (`docs/internal/tfe-compatibility/data-sources/workspaces/tfe_workspace_ids.md`); no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `names` | list(string) | Optional | - | - | no | explicit names or a single `*` (all); `AtLeastOneOf(tag_filters, names, tag_names)` |
| `tag_names` | list(string) | Optional | - | - | no | legacy tag names → `search[tags]`; **matches on binding KEY (any value)** - see divergence |
| `exclude_tags` | set(string) | Optional | - | - | no | legacy tag names → `search[exclude-tags]`; **excludes on binding KEY (any value)** - see divergence |
| `tag_filters` | list(object) | Optional | - | - | no | single block `{include = map, exclude = map}`; key=value bindings (AND) |
| `organization` | string | Optional | - | provider default | no | org name; falls back to the provider default |
| `ids` | map(string) | Computed | - | - | no | workspace name → workspace id |
| `full_names` | map(string) | Computed | - | - | no | workspace name → "org/name" |
| `id` | string | Computed | - | - | no | `"<org>/<hash(names+tag_names)>"` (SetId) |

## Wire contract

- **Read/lookup:** `Workspaces.List(org, options)` → `GET /organizations/:org/workspaces`, paginated
  (follows `NextPage`). Options carry `Tags` (`search[tags]`), `ExcludeTags` (`search[exclude-tags]`),
  `TagBindings` (`filter[tagged][N][key|value]`), and `Include: [effective_tag_bindings]` when
  `tag_filters` is set. Name wildcard/exact matching and `tag_filters.exclude` are done client-side.
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `workspaces` (collection). Each item's `effective-tag-bindings` relation drives
  client-side tag exclusion. **Value-level divergence** (see below) in how legacy tag names map to
  key/value bindings.

## Acceptance criteria (these ARE the test)

1. **Name path:** fixture creates a `stackweaver_workspace` and reads it via `names = [<name>]`;
   `ids[<name>] == workspace.id` and `full_names[<name>] == "<org>/<name>"`.
2. **Tag path:** fixture creates two workspaces differing only by an `env` tag, both inheriting
   `team=platform` from their project; assert `tag_filters.include {env="prod"}` selects only the prod
   workspace, `exclude_tags = ["env"]`-style exclusion drops matches on that key, and an inherited
   `team=platform` include selects both.
3. The data source's computed `id` is set (`"<org>/<hash>"`).
4. Re-`plan` after apply shows **no drift**.
5. `ids`/`full_names` are Computed-only; input filter blocks are Optional and may be known-null at plan -
   assert on `ids`/`full_names`/`id`, not on the filter inputs.

## Runtime criterion

Read-only data source. Resolves a filtered set of workspaces to `{name→id}` / `{name→full_name}` maps;
no runtime side effect beyond the list read.

## Docs + example

- Provider docs page: `docs/data-sources/workspace_ids.md` - arguments `names`/`tag_names`/
  `exclude_tags`/`tag_filters`/`organization`; computed `ids`, `full_names`, `id`; document the
  KEY-match semantics of `tag_names`/`exclude_tags`.
- Example: `examples/data-sources/stackweaver_workspace_ids/data-source.tf` - list by names and by tags.

## Divergences from upstream / TFE

Value-level: Stackweaver models tags as key/value **bindings**, not TFE's flat tag names. So
`tag_names` and `exclude_tags` (legacy flat names) match/exclude on the binding **key** (any value) -
a legacy name maps to a binding key. `tag_filters {include/exclude = {k=v}}` matches key=value pairs
exactly (server-side AND for include). "Effective" tags include project-inherited bindings. Schema is
otherwise a drop-in for `tfe_workspace_ids`.
