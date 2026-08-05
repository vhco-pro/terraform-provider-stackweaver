<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_project_variable_set
tfe_alias: tfe_project_variable_set
kind: resource
family: variables
origin: forked
backing_api: implemented
client_approach: go-tfe-clean   # value-level divergence: bare-UUID project ids round-trip verbatim; project-owned sets rejected (wire shape unchanged)
status: spec'd
upstream_file: internal/provider/resource_tfe_project_variable_set.go
go_tfe_type: VariableSetApplyToProjectsOptions / Project
compat_doc: docs/internal/tfe-compatibility/resources/variables/tfe_project_variable_set.md
---
# stackweaver_project_variable_set

A **relationship resource** with no object of its own: it attaches an existing organization-owned
variable set to an existing project. Create applies the set to the project; delete removes it. Maps
onto Stackweaver's varset↔project join (`VariableSetProject`).

## Client approach

`go-tfe-clean` **with a documented value-level divergence**. The upstream resource (SDKv2 legacy,
`internal/provider/resource_tfe_project_variable_set.go:22`) drives `go-tfe`
`VariableSets.ApplyToProjects` / `RemoveFromProjects` and reads membership via `VariableSets.Read` with
`Include: [projects]`. The wire *shape* is stock (`POST`/`DELETE /varsets/:id/relationships/projects`,
`type: projects` in `data[]`), so no wrapper is needed. The only differences are **values**:
Stackweaver returns **bare UUID** project ids (not `prj-…`), which stock go-tfe round-trips verbatim,
and it **rejects attaching a project-owned set**. Captured as migration notes below, not client code
(`docs/internal/tfe-compatibility/resources/variables/tfe_project_variable_set.md`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | composite `{project_id}_{variable_set_id}` (provider-side; no server row id) |
| `variable_set_id` | string | Required | yes | — | no | `varset-…`; must be **organization-owned** (see divergence) |
| `project_id` | string | Required | yes | — | no | the target project; **bare UUID on Stackweaver** (not `prj-…`) |

## Wire contract

- **Create:** `VariableSets.ApplyToProjects(variable_set_id, {Projects})` →
  `POST /varsets/:id/relationships/projects`, body `{"data":[{"type":"projects","id":"<project_id>"}]}`.
  Additive. State id set to `{project_id}_{variable_set_id}`.
- **Read:** `VariableSets.Read(variable_set_id, {Include:[projects]})` →
  `GET /varsets/:id?include=projects`. The provider scans `relationships.projects` for `project_id`;
  if absent (or the set is gone) it drops the resource from state.
- **Update:** none — both attributes are ForceNew, so any change recreates.
- **Delete:** `VariableSets.RemoveFromProjects(variable_set_id, {Projects})` →
  `DELETE /varsets/:id/relationships/projects`, same body shape.
- **JSON:API type:** `projects` (relationship data). No write-only fields. **Divergence:** the
  relationship `id` is a **bare UUID** on Stackweaver (TFE emits `prj-…`); stock go-tfe treats it as an
  opaque string and round-trips it unchanged.

## Acceptance criteria (these ARE the test)

1. `apply` of `{variable_set_id, project_id}` (org-owned set) attaches it; state `id` =
   `{project_id}_{variable_set_id}` and both ids round-trip.
2. Re-`plan` after apply shows **no drift**; on read `project_id` is present in the set's
   `relationships.projects`.
3. The `project_id` round-trips **verbatim as a bare UUID** (not rewritten to `prj-…`) — no drift is
   introduced by the id format.
4. Attaching a **project-owned** variable set (one created with a `parent_project_id`) is rejected with
   an error like "Only organization-owned variable sets can be assigned to projects".
5. Changing either `variable_set_id` or `project_id` recreates (both ForceNew); an out-of-band removal
   (project no longer in the set's projects) drops the resource from state on the next read.
6. `destroy` removes the attachment; a subsequent read of the set no longer lists the project.

## Runtime criterion

Not `CRUD-only`. The attachment must make the org-owned set's variables reach that project's workspaces'
runs and **no others**. Verified by `core/repository` `TestListByWorkspace_AUD150_OwnershipAndGlobal`:
an org-owned, non-global set attached to project P is returned by `ListByWorkspace` for a workspace in P
and not for one outside P — the resolver used by run-config assembly on both runner paths.

## Docs + example

- Provider docs page: `docs/resources/project_variable_set.md` — arguments (variable_set_id,
  project_id), the composite `id`, and a prominent note that (a) `project_id` is a **bare UUID** on
  Stackweaver and (b) only **organization-owned** sets can be attached.
- Example: `examples/resources/stackweaver_project_variable_set/resource.tf` — an org-owned
  `stackweaver_variable_set` + a `stackweaver_project`, joined by this resource.

## Divergences from upstream / TFE

**Value-level (documented), wire shape unchanged:**

1. **Bare-UUID project ids.** `project_id` (and the read-back relationship id) is a bare UUID on
   Stackweaver, not TFE's `prj-…`. Stock go-tfe treats it as an opaque string and round-trips it
   verbatim — no client change.
2. **Project-owned sets rejected.** Stackweaver rejects attaching a variable set that is itself owned
   by a project (created with a `parent_project_id`) to another project; only organization-owned sets
   may be attached. TFE enforces the same via ownership.

Both are usage/migration notes, not client code. Source:
`docs/internal/tfe-compatibility/resources/variables/tfe_project_variable_set.md:27,56-62`.
