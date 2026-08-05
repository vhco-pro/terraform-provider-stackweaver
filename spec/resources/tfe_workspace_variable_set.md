<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_workspace_variable_set
tfe_alias: tfe_workspace_variable_set
kind: resource
family: variables
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_workspace_variable_set.go
go_tfe_type: VariableSetApplyToWorkspacesOptions / Workspace
compat_doc: docs/internal/tfe-compatibility/resources/variables/tfe_variable_set.md
---
# stackweaver_workspace_variable_set

A **relationship resource** with no object of its own: it attaches an existing variable set to an
existing workspace. Create applies the set to the workspace; delete removes it. This is the preferred,
non-deprecated way to associate a set with a workspace (vs. `variable_set.workspace_ids`). Maps onto
Stackweaver's varset↔workspace join.

## Client approach

`go-tfe-clean`. The upstream resource (SDKv2 legacy,
`internal/provider/resource_tfe_workspace_variable_set.go:22`) drives `go-tfe`
`VariableSets.ApplyToWorkspaces` / `RemoveFromWorkspaces` and reads membership via `VariableSets.Read`
with `Include: [workspaces]`. Stackweaver serves the stock `POST`/`DELETE /varsets/:id/relationships/
workspaces` endpoints and the `?include=workspaces` read unchanged
(`docs/internal/tfe-compatibility/resources/variables/tfe_variable_set.md:47`). No wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | composite `{workspace_id}_{variable_set_id}` (provider-side; no server row id) |
| `variable_set_id` | string | Required | yes | — | no | `varset-…`; the set being attached |
| `workspace_id` | string | Required | yes | — | no | `ws-…`; the target workspace |

## Wire contract

- **Create:** `VariableSets.ApplyToWorkspaces(variable_set_id, {Workspaces})` →
  `POST /varsets/:id/relationships/workspaces`, body `{"data":[{"type":"workspaces","id":"<ws>"}]}`.
  Additive. State id set to `{workspace_id}_{variable_set_id}`.
- **Read:** `VariableSets.Read(variable_set_id, {Include:[workspaces]})` →
  `GET /varsets/:id?include=workspaces`. The provider scans `relationships.workspaces` for
  `workspace_id`; if absent (or the set is gone) it drops the resource from state.
- **Update:** none — both attributes are ForceNew, so any change recreates.
- **Delete:** `VariableSets.RemoveFromWorkspaces(variable_set_id, {Workspaces})` →
  `DELETE /varsets/:id/relationships/workspaces`, same body shape.
- **JSON:API type:** `workspaces` (relationship data). No write-only fields; no divergence from stock
  go-tfe.

## Acceptance criteria (these ARE the test)

1. `apply` of `{variable_set_id, workspace_id}` attaches the set; state `id` =
   `{workspace_id}_{variable_set_id}` and both ids round-trip.
2. Re-`plan` after apply shows **no drift**; on read the workspace appears in the set's
   `relationships.workspaces`.
3. Changing either `variable_set_id` or `workspace_id` recreates (both ForceNew) — the old attachment
   is removed and the new one applied.
4. If the attachment is removed out-of-band (workspace no longer in the set's workspaces), the next
   read drops the resource from state (planning a re-create) rather than erroring.
5. `destroy` removes the attachment; a subsequent read of the set no longer lists the workspace.

## Runtime criterion

Not `CRUD-only`. The attachment must make the set's variables reach the workspace's runs: after apply,
a run in `workspace_id` sees the variables from `variable_set_id` (resolved by
`core/repository/variable_set.go` `ListByWorkspace`). CRUD alone only records the join row.

## Docs + example

- Provider docs page: `docs/resources/workspace_variable_set.md` — arguments (variable_set_id,
  workspace_id), the composite `id`, and a note preferring this resource over the deprecated
  `variable_set.workspace_ids`.
- Example: `examples/resources/stackweaver_workspace_variable_set/resource.tf` — a variable set + a
  workspace joined by this resource.

## Divergences from upstream / TFE

None. Drop-in with `tfe_workspace_variable_set`. Note: there is **no dedicated compat doc** for this
resource; the varset↔workspace relationship endpoints it uses are documented under
`docs/internal/tfe-compatibility/resources/variables/tfe_variable_set.md:47` (the `compat_doc`
frontmatter points there).
