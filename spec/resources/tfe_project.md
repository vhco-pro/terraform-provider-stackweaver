<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_project
tfe_alias: tfe_project
kind: resource
family: projects
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_project.go
go_tfe_type: Project
compat_doc: docs/internal/tfe-compatibility/resources/projects/tfe_project.md
---
# stackweaver_project

A project groups workspaces and carries default settings (execution mode, agent pool, tag bindings)
that its workspaces inherit at run time. Maps 1:1 onto Stackweaver's project concept.

## Client approach

`go-tfe-clean`. Stackweaver's projects endpoint accepts and returns the stock `go-tfe` `Project`
JSON:API shape unchanged (`docs/internal/tfe-compatibility/resources/projects/tfe_project.md`); no
wrapper. The upstream resource uses the plugin framework (`Schema()` at
`internal/provider/resource_tfe_project.go:113`) and the `go-tfe` `Projects` service verbatim.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `projects` JSON:API primary id |
| `name` | string | Required | no | — | no | unique within the org |
| `description` | string | Optional | no | `""` | no | |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; changing it recreates |
| `auto_destroy_activity_duration` | string | Optional+Computed | no | — | no | Go-ish duration ("24h"/"1d"); nullable on the wire |
| `tags` | map(string) | Optional | no | — | no | key/value tag bindings; sent as `tag-bindings` relation |

## Wire contract

- **Create:** `Projects.Create(org, ProjectCreateOptions)` → `POST /organizations/:org/projects`.
  Attrs sent: `name`, `description?`, `auto-destroy-activity-duration?`, `tag-bindings?` relation.
- **Read:** `Projects.Read(id)` → `GET /projects/:id`.
- **Update:** `Projects.Update(id, ProjectUpdateOptions)` → `PATCH /projects/:id` (name, description,
  auto-destroy, tags — all in place).
- **Delete:** `Projects.Delete(id)` → `DELETE /projects/:id`.
- **JSON:API type:** `projects`. No write-only fields. `default-execution-mode` /
  `default-agent-pool` live on `stackweaver_project_settings`, not here.

## Acceptance criteria (these ARE the test)

1. `apply` of `{name, description}` creates the project; `id`, `name`, `description` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. Setting `tags = {env = "dev"}` round-trips: the binding is present on read.
4. Updating `description` in place applies without recreate; updating `organization` forces recreate.
5. `destroy` removes it; a subsequent `Projects.Read(id)` returns 404.

## Runtime criterion

Container resource — its runtime effect is inheritance: a workspace placed in the project inherits its
default execution mode / agent pool / tags at run time. Verified indirectly (a workspace referencing
the project resolves those defaults); the resource itself is otherwise CRUD. Not `CRUD-only` in the
dead sense — the inheritance is exercised by `stackweaver_workspace` + `stackweaver_project_settings`.

## Docs + example

- Provider docs page: `docs/resources/project.md` — arguments (name/description/organization/
  auto_destroy_activity_duration/tags), attribute `id`, import by id.
- Example: `examples/resources/stackweaver_project/resource.tf` — a minimal named project in an org.

## Divergences from upstream / TFE

None. Drop-in with `tfe_project`.
