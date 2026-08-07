<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_projects
tfe_alias: tfe_projects
kind: data-source
family: projects
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_projects.go
go_tfe_type: Project
compat_doc: docs/internal/tfe-compatibility/data-sources/projects/tfe_projects.md
---
# stackweaver_projects

Lists every project in an organization, exposing a `projects` collection of `{id, name, description,
organization}`. Maps onto Stackweaver's org projects list endpoint.

## Client approach

`go-tfe-clean`. This plugin-framework data source calls `Projects.List(org, {PageSize: 100})` →
`GET /organizations/:org/projects` and paginates to the end (following `NextPage`), appending each item.
Stackweaver's org projects list endpoint already returns the stock go-tfe `projects` JSON:API
collection unchanged (`docs/internal/tfe-compatibility/data-sources/projects/tfe_projects.md`), so no
backend change and no wrapper were needed.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional+Computed | - | provider default | no | org name; falls back to the provider default; echoed back |
| `id` | string | Computed | - | - | no | set to the organization name |
| `projects` | list(object) | Computed | - | - | no | `{id, name, description, organization}` per project |
| `projects[].id` | string | Computed | - | - | no | `projects` id |
| `projects[].name` | string | Computed | - | - | no | |
| `projects[].description` | string | Computed | - | - | no | |
| `projects[].organization` | string | Computed | - | - | no | org name |

## Wire contract

- **Read/lookup:** `Projects.List(org, {ListOptions: {PageSize: 100}})` →
  `GET /organizations/:org/projects`, paginated (follows `NextPage` to the last page).
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `projects` (collection). Each item maps `id`, `name`, `description`, and the
  `organization` relation name. No divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

1. Fixture creates a `stackweaver_project`, then reads `data.stackweaver_projects` for the same org.
2. The data source's computed `id` is set (equals the organization name) and `organization` is echoed.
3. The created project's `id` appears in the `projects` collection (and its `name`/`description` match
   that element).
4. Re-`plan` after apply shows **no drift**.
5. `projects` and `id` are Computed-only, so no input assertion is needed before the read.

## Runtime criterion

Read-only data source. Resolves the org's full (paginated) project list into a `projects` collection; no
runtime side effect beyond the list read.

## Docs + example

- Provider docs page: `docs/data-sources/projects.md` - argument `organization`; computed `projects`
  (each `id`/`name`/`description`/`organization`) and `id`.
- Example: `examples/data-sources/stackweaver_projects/data-source.tf` - list an org's projects.

## Divergences from upstream / TFE

None. Drop-in with `tfe_projects`; lists the org's projects with no new backend code.
