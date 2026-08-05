<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_project
tfe_alias: tfe_project
kind: data-source
family: projects
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_project.go
go_tfe_type: Project
compat_doc: docs/internal/tfe-compatibility/data-sources/projects/tfe_project.md
---
# stackweaver_project

Reads a single project by `name` within an organization and exposes its id, description,
`auto_destroy_activity_duration`, the ids/names of its workspaces, and its `effective_tags`. Maps onto
Stackweaver's project record.

## Client approach

`go-tfe-clean`. This plugin-framework data source lists projects with
`Projects.List(org, {Name: name})` → `GET /organizations/:org/projects?filter[names]=<name>` and picks
the case-insensitive name match. It then lists the project's workspaces
(`Workspaces.List(org, {ProjectID})`, paged) and reads effective tags via
`Projects.ListEffectiveTagBindings(id)` → `GET /projects/:id/effective-tag-bindings` (tolerating
`ErrResourceNotFound` on instances without the endpoint). Stackweaver returns the stock go-tfe `projects`
JSON:API shape (`docs/internal/tfe-compatibility/data-sources/projects/tfe_project.md`); no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `name` | string | Required | — | — | no | project name (lookup key; case-insensitive match) |
| `organization` | string | Optional+Computed | — | provider default | no | org name; falls back to the provider default |
| `id` | string | Computed | — | — | no | `projects` id |
| `description` | string | Computed | — | — | no | |
| `auto_destroy_activity_duration` | string | Computed | — | — | no | nullable on the wire; set only when specified |
| `workspace_ids` | set(string) | Computed | — | — | no | ids of the project's workspaces |
| `workspace_names` | set(string) | Computed | — | — | no | names of the project's workspaces |
| `effective_tags` | map(string) | Computed | — | — | no | project's effective tag bindings (key→value) |

## Wire contract

- **Read/lookup:** `Projects.List(org, {Name})` → `GET /organizations/:org/projects?filter[names]=<name>`
  (case-insensitive match on `Name`); then `Workspaces.List(org, {ProjectID})` (paged) for
  `workspace_ids`/`workspace_names`, and `Projects.ListEffectiveTagBindings(id)` →
  `GET /projects/:id/effective-tag-bindings` for `effective_tags`. Not-found → an explicit
  "Could not find project" diagnostic.
- **Create/Update/Delete:** n/a — read-only data source.
- **JSON:API type:** `projects`; effective tags via the `effective-tag-bindings` collection.
  `auto-destroy-activity-duration` is nullable on the wire. No divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

1. Fixture creates a `stackweaver_project` with `tags = {env=…, team=…}`, then reads
   `data.stackweaver_project` by `name`.
2. The data source's computed `id` equals the created project's id.
3. `effective_tags["env"]` and `effective_tags["team"]` round-trip the tags set on the project.
4. Re-`plan` after apply shows **no drift**.
5. `id`, `description`, `effective_tags`, `workspace_ids`/`workspace_names` are all Computed — assert on
   those; `auto_destroy_activity_duration` is set only when specified, so do not assert it unless the
   fixture sets it.

## Runtime criterion

Read-only data source. Resolves one project to its id, description, member workspaces, and effective
tags; no runtime side effect beyond the reads.

## Docs + example

- Provider docs page: `docs/data-sources/project.md` — arguments `name`/`organization`; computed `id`,
  `description`, `auto_destroy_activity_duration`, `workspace_ids`, `workspace_names`, `effective_tags`.
- Example: `examples/data-sources/stackweaver_project/data-source.tf` — read a project by name.

## Divergences from upstream / TFE

None. Drop-in with `tfe_project`; `effective_tags` depends on the project/workspace tags feature.
