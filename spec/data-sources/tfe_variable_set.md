<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_variable_set
tfe_alias: tfe_variable_set
kind: data-source
family: variables
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_variable_set.go
go_tfe_type: VariableSet
compat_doc: n/a
---
# stackweaver_variable_set

Looks up a variable set in an organization by name and exposes its metadata plus the ids of the
variables, workspaces, and projects attached to it. Maps onto Stackweaver's variable-set concept.
Read-only: it resolves the `varsets` `id`, `description`, `global`/`priority` flags, and the attachment
id sets.

## Client approach

`go-tfe-clean`. Lists variable sets via `VariableSets.List` to find the name match, then re-reads the
matched set with relation includes via `VariableSets.Read` (`workspaces`, `vars`, and — where the
remote version supports it — `stacks`) to populate the id sets. Consumes the stock `VariableSet`
JSON:API shape unchanged; no wrapper. No compatibility detail doc exists yet
(`docs/internal/tfe-compatibility/data-sources/variables/tfe_variable_set.md` is absent) — this spec is
the source of record.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `name` | string | Required | — | — | no | variable set name to look up |
| `organization` | string | Optional | — | provider default | no | org name; falls back to the provider's default org |
| `id` | string | Computed | — | — | no | `varsets` JSON:API primary id |
| `description` | string | Computed | — | — | no | variable set description |
| `global` | bool | Computed | — | — | no | applied to all workspaces in the org |
| `priority` | bool | Computed | — | — | no | overrides workspace-level variables |
| `workspace_ids` | set(string) | Optional+Computed | — | — | no | ids of attached workspaces |
| `variable_ids` | set(string) | Optional+Computed | — | — | no | ids of variables in the set |
| `project_ids` | set(string) | Optional+Computed | — | — | no | ids of attached projects |
| `stack_ids` | set(string) | Optional+Computed | — | — | no | ids of attached stacks — **not populated** (gap, see below) |
| `parent_project_id` | string | Optional+Computed | — | — | no | parent project id when parented to a project — **not populated** (gap, see below) |

## Wire contract

- **Read (lookup):** `VariableSets.List(org, VariableSetListOptions{})` → `GET
  /organizations/:org/varsets`. Paginates until a set whose `Name` equals `name` is found, then
  `VariableSets.Read(id, VariableSetReadOptions{Include: [workspaces, vars, (stacks)]})` → `GET
  /varsets/:id?include=...` to load the relations. Sets `id`, copies `description`/`global`/`priority`,
  and flattens the `Workspaces`/`Variables`/`Projects`/`Stacks` relation ids into the corresponding
  sets; sets `parent_project_id` from `Parent.Project.ID` when present.
- No create/update/delete — data source.
- **JSON:API type:** `varsets`. The `stacks` include is gated behind a minimum remote TFE version
  check (`minTFEVersionVariableSetStacks`). On Stackweaver `stacks` and the `parent` project relation
  are not surfaced (Stacks is unimplemented), so `stack_ids` and `parent_project_id` come back empty.

## Acceptance criteria (these ARE the test)

1. Fixture creates a backing `stackweaver_variable_set` (with a `description` and, e.g., a variable and
   a workspace attachment), then this data source reads it by `name`; `apply` succeeds.
2. Computed `id` is set and equals the created variable set's `id` (the `varsets` primary id).
3. `description`, `global`, and `priority` round-trip: they equal the values on the backing set.
4. `variable_ids` contains the attached variable's id; `workspace_ids` contains the attached
   workspace's id (when the backing set attaches them).
5. Looking up a non-existent name fails the read with a "could not find variable set" error.
6. **Plan-null quirk:** `workspace_ids`/`variable_ids`/`project_ids`/`stack_ids`/`parent_project_id` are
   Optional+Computed and `organization` is Optional-not-Computed, so assert the clearly-Computed fields
   (`id`, `description`, `global`, `priority`) and the populated id sets, not the plan-null inputs.
7. **Gap assertion:** `stack_ids` and `parent_project_id` are expected empty on Stackweaver; the fixture
   must not assert Stacks/parent-project values.

## Runtime criterion

Read-only data source. It resolves an existing variable set's identity, flags, and attachment id sets
so a config can reference the set or its members by name. No runtime side effect of its own.

## Docs + example

- Provider docs page: `docs/data-sources/variable_set.md` — arguments (`name`, `organization`),
  computed attributes (`id`, `description`, `global`, `priority`, `workspace_ids`, `variable_ids`,
  `project_ids`), with a note that `stack_ids`/`parent_project_id` are unpopulated on Stackweaver.
- Example: `examples/data-sources/stackweaver_variable_set/data-source.tf` — look up a variable set by
  name and reference `data.stackweaver_variable_set.x.id`.

## Divergences from upstream / TFE

None (schema/wire drop-in with `tfe_variable_set`). The unpopulated `stack_ids` and `parent_project_id`
are an unimplemented-feature gap (Stacks / project-parented variable sets are not yet supported), not a
wire divergence — the fields exist and are exposed, they simply resolve empty.
