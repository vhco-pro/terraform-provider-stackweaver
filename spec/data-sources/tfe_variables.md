<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_variables
tfe_alias: tfe_variables
kind: data-source
family: variables
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_variables.go
go_tfe_type: Variable
compat_doc: n/a
---
# stackweaver_variables

Retrieves all variables belonging to either a workspace or a variable set, split into `terraform` and
`env` category lists plus a combined `variables` list. Maps onto Stackweaver's variable and
variable-set-variable concepts. Read-only: given exactly one of `workspace_id` or `variable_set_id` it
resolves the full variable listing.

## Client approach

`go-tfe-clean`. Plugin-framework data source. When `workspace_id` is set it pages `Variables.List`;
when `variable_set_id` is set it pages `VariableSetVariables.List`. Both consume the stock go-tfe
`Variable` / `VariableSetVariable` JSON:API shapes unchanged; no wrapper. No compatibility detail doc
exists yet (`docs/internal/tfe-compatibility/data-sources/variables/tfe_variables.md` is absent) - this
spec is the source of record.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `workspace_id` | string | Optional | - | - | no | source workspace; conflicts with `variable_set_id` |
| `variable_set_id` | string | Optional | - | - | no | source variable set; conflicts with `workspace_id` |
| `id` | string | Computed | - | - | no | synthetic id `variables/<workspace_id or variable_set_id>` |
| `env` | list(object) | Computed | - | - | no | variables with `category = env` |
| `terraform` | list(object) | Computed | - | - | no | variables with `category = terraform` |
| `variables` | list(object) | Computed | - | - | no | all variables (env + terraform) |
| `<list>.id` | string | Computed | - | - | no | variable `vars` primary id |
| `<list>.name` | string | Computed | - | - | no | variable key |
| `<list>.value` | string | Computed | - | - | no | variable value (empty for sensitive variables) |
| `<list>.category` | string | Computed | - | - | no | `terraform` or `env` |
| `<list>.hcl` | bool | Computed | - | - | no | value evaluated as HCL |
| `<list>.sensitive` | bool | Computed | - | - | no | whether the variable is sensitive |

## Wire contract

- **Read (lookup), workspace mode:** `Variables.List(workspace_id, VariableListOptions{})` → `GET
  /workspaces/:id/vars`, paged; each `Variable` is appended to `variables` and bucketed into `env` /
  `terraform` by `Category`.
- **Read (lookup), variable-set mode:** `VariableSetVariables.List(variable_set_id,
  VariableSetVariableListOptions{})` → `GET /varsets/:id/relationships/vars`, paged; same bucketing.
- Exactly one of `workspace_id` / `variable_set_id` is provided (enforced by a `ConflictsWith`
  validator); the synthetic `id` is `variables/<that id>`.
- No create/update/delete - data source.
- **JSON:API type:** `vars`. Each object exposes `id`, `name` (from `key`), `value`, `category`, `hcl`,
  `sensitive`. Sensitive variable values are not echoed by the API and surface as empty strings.

## Acceptance criteria (these ARE the test)

1. Fixture creates a backing workspace with one `terraform` and one `env` variable, then this data
   source reads them via `workspace_id`; `apply` succeeds.
2. Computed `id` is set and equals `variables/<workspace_id>` (or `variables/<variable_set_id>` in
   variable-set mode).
3. `variables` contains both created variables; `terraform` contains only the terraform one and `env`
   only the env one; each object's `name`/`category`/`hcl`/`sensitive` match what was created.
4. A sensitive variable appears in the lists with its `sensitive = true` and an empty `value`.
5. Variable-set mode: pointing `variable_set_id` at a backing variable set returns that set's variables
   with the same bucketing.
6. **Plan-null quirk:** both input args are Optional (plan-null until one is set), so assert the
   clearly-Computed outputs (`id`, `variables`, `terraform`, `env`) rather than the inputs.

## Runtime criterion

Read-only data source. It resolves the full variable listing of a workspace or variable set (split by
category) so a config can reference or export those variables. No runtime side effect of its own.

## Docs + example

- Provider docs page: `docs/data-sources/variables.md` - arguments (`workspace_id`, `variable_set_id`,
  mutually exclusive), computed attributes (`id`, `env`, `terraform`, `variables` and their object
  fields).
- Example: `examples/data-sources/stackweaver_variables/data-source.tf` - read a workspace's variables
  and reference `data.stackweaver_variables.x.terraform`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_variables`.
