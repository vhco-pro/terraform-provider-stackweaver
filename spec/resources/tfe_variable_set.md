<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_variable_set
tfe_alias: tfe_variable_set
kind: resource
family: variables
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_variable_set.go
go_tfe_type: VariableSet
compat_doc: docs/internal/tfe-compatibility/resources/variables/tfe_variable_set.md
---
# stackweaver_variable_set

Manages a variable set — a named, reusable bundle of variables — with a scope: `global` (all workspaces
in the org), organization-owned but attached to specific workspaces/projects, or project-owned (via
`parent_project_id`). Maps onto Stackweaver's variable set (`core/models/variable_set.go`).

## Client approach

`go-tfe-clean`. The upstream resource (SDKv2 legacy,
`internal/provider/resource_tfe_variable_set.go:24`) drives the `go-tfe` `VariableSets` service —
`Create`/`Read`/`Update`/`Delete`, plus `UpdateWorkspaces`/`UpdateStacks` for the deprecated inline id
lists. Stackweaver serves the stock `varsets` JSON:API shape (`name`, `description`, `global`,
`priority`, `parent` polyrelation) unchanged; the `global` wire attr round-trips. The Stackweaver
`scope`/`organization_id`/`project_id` model fields are internal mappings behind that same wire shape,
not new bytes. No wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `varsets` primary id (`varset-…`) |
| `name` | string | Required | no | — | no | |
| `description` | string | Optional | no | — | no | |
| `global` | bool | Optional | no | `false` | no | `ConflictsWith` `workspace_ids`; applies to all org workspaces |
| `priority` | bool | Optional | no | `false` | no | set overrides more-specific scopes / CLI |
| `organization` | string | Optional+Computed | yes | provider default | no | org name |
| `workspace_ids` | set(string) | Optional+Computed | no | — | no | **deprecated** — prefer `stackweaver_workspace_variable_set` |
| `stack_ids` | set(string) | Optional+Computed | no | — | no | stacks the set is applied to (TFE-version gated on read) |
| `parent_project_id` | string | Optional+Computed | yes | — | no | project-owned set; `global` must be `false` (validated in CustomizeDiff) |

## Wire contract

- **Create:** `VariableSets.Create(org, VariableSetCreateOptions)` →
  `POST /organizations/:org/varsets`. Sends `name`, `global?`, `priority?`, `description?`, and the
  `parent` polyrelation (`{project: {id}}`) when `parent_project_id` is set. Then, if not global and
  ids are given, `UpdateWorkspaces` / `UpdateStacks` apply the inline id lists.
- **Read:** `VariableSets.Read(id, {Include: [workspaces (,stacks)]})` → `GET /varsets/:id?include=…`.
  Reads back `name`, `description`, `global`, `priority`, `organization`, the workspace/stack id sets,
  and `parent.project` id.
- **Update:** `VariableSets.Update(id, VariableSetUpdateOptions)` → `PATCH /varsets/:id` for
  name/description/global/priority; `UpdateWorkspaces`/`UpdateStacks` when those id sets change.
- **Delete:** `VariableSets.Delete(id)` → `DELETE /varsets/:id` (404 tolerated).
- **JSON:API type:** `varsets`. No write-only fields. `global` round-trips as a wire attr. No divergence
  from stock go-tfe.

## Acceptance criteria (these ARE the test)

1. `apply` of `{name, description, global=true, organization}` creates the set; `id` (`varset-…`),
   `name`, `description`, `global`, `priority`, `organization` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. `global` round-trips faithfully: applying `global=true` then re-reading returns `global=true`;
   toggling to `global=false` (with no inline ids) updates in place.
4. Setting `parent_project_id` creates a project-owned set and round-trips the id; setting both
   `parent_project_id` and `global=true` fails (`validateParentProjectID`); setting both `global=true`
   and `workspace_ids` fails (`ConflictsWith`).
5. Updating `name`/`description`/`priority` applies **in place**; changing `organization` or
   `parent_project_id` recreates (ForceNew).
6. `destroy` removes it; a subsequent `VariableSets.Read(id)` returns 404.

## Runtime criterion

Not `CRUD-only`. The set's variables must reach runs of the workspaces in its scope: a `global` set
applies to every workspace in the org; a scoped set applies only where attached. Verified via the
resolver `core/repository/variable_set.go` `ListByWorkspace`/`ListByProject`, which feed the run-config
assembly on both runner paths (proven by `TestListByWorkspace_AUD150_OwnershipAndGlobal`).

## Docs + example

- Provider docs page: `docs/resources/variable_set.md` — arguments (name/description/global/priority/
  organization/parent_project_id), the deprecation of `workspace_ids` in favor of
  `stackweaver_workspace_variable_set`, and the `global` ⨯ `parent_project_id` mutual exclusivity.
- Example: `examples/resources/stackweaver_variable_set/resource.tf` — a global org set and a
  project-owned set attached to a workspace via `stackweaver_workspace_variable_set`.

## Divergences from upstream / TFE

None on the wire — the `global` attr and the `varsets` shape round-trip unchanged. Stackweaver's
`scope`/`organization_id`/`project_id` are **internal-field mappings** for the same wire shape, and
`workspace_ids` is the **deprecated** inline-id form (upstream deprecated it in 0.33.0; prefer
`stackweaver_workspace_variable_set`). `stack_ids` read is gated on a minimum TFE version include and is
not the migration focus. Source:
`docs/internal/tfe-compatibility/resources/variables/tfe_variable_set.md:21-24,83-88`.
