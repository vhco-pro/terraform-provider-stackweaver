<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_variable
tfe_alias: tfe_variable
kind: resource
family: variables
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_variable.go
go_tfe_type: Variable / VariableSetVariable
compat_doc: docs/internal/tfe-compatibility/resources/variables/tfe_variable.md
---
# stackweaver_variable

Manages a single Terraform or environment variable belonging **either** to a workspace **or** to a
variable set (exactly one of `workspace_id` / `variable_set_id`). Carries key, value, category, and the
`hcl`/`sensitive` flags. Maps 1:1 onto a Stackweaver variable (`core/models/variable.go`).

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework, `internal/provider/resource_tfe_variable.go`)
is an abstraction over two parallel `go-tfe` services chosen by which id is set: `Variables`
(`Create`/`Read`/`Update`/`Delete`) for workspace vars and `VariableSetVariables` for varset vars. Both
speak the stock `vars` JSON:API shape, which Stackweaver serves unchanged
(`docs/internal/tfe-compatibility/resources/variables/tfe_variable.md`). No wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `vars` primary id (Stackweaver UUID) |
| `key` | string | Required | conditional | — | no | `RequiresReplaceIf`: recreate only when `sensitive=true` **and** key changes |
| `value` | string | Optional+Computed | no | `""` | **yes** | conflicts with `value_wo`; never re-read once `sensitive` |
| `value_wo` | string | Optional (write-only) | no | — | **yes** | write-only, conflicts with `value`, requires `value_wo_version`; **not implemented server-side** (see divergences) |
| `value_wo_version` | int64 | Optional | no | — | no | bump to push a new `value_wo`; conflicts with `value`, requires `value_wo` |
| `category` | string | Required | yes | — | no | `RequiresReplace`; one of `terraform`/`env` |
| `description` | string | Optional+Computed | no | `""` | no | |
| `hcl` | bool | Optional+Computed | no | `false` | no | |
| `sensitive` | bool | Optional+Computed | no | `false` | no | `RequiresReplaceIf`: recreate when flipped `true`→`false` |
| `workspace_id` | string | Optional | yes | — | no | `ExactlyOneOf` with `variable_set_id`; must match `ws-…` |
| `variable_set_id` | string | Optional | yes | — | no | `ExactlyOneOf` with `workspace_id`; must match `varset-…` |
| `readable_value` | string | Computed | — | — | no | non-sensitive mirror of `value`; null/empty when `sensitive` or write-only |

## Wire contract

- **Create (workspace):** `Variables.Create(workspace_id, VariableCreateOptions)` →
  `POST /workspaces/:id/vars`. Sends `key`, `value?`, `category`, `hcl?`, `sensitive?`, `description?`.
- **Create (varset):** `VariableSetVariables.Create(variable_set_id, VariableSetVariableCreateOptions)`
  → `POST /varsets/:id/relationships/vars`. Same attrs.
- **Read:** `Variables.Read(ws, id)` → `GET /workspaces/:id/vars/:var_id`, or
  `VariableSetVariables.Read(varset, id)` → `GET /varsets/:id/relationships/vars/:var_id`.
- **Update:** `Variables.Update` → `PATCH /workspaces/:id/vars/:var_id` (or the varset PATCH). `value`
  is sent **only** when it changed from prior state, so unrelated updates never reset a sensitive value.
- **Delete:** `Variables.Delete` / `VariableSetVariables.Delete` → `DELETE …/:var_id` (404 tolerated).
- **JSON:API type:** `vars`. The API **never echoes a sensitive `value`** back on read — the provider
  carries forward the last known plan value into state and nulls `readable_value`. No wire divergence
  from stock go-tfe.

## Acceptance criteria (these ARE the test)

1. `apply` of a workspace var `{key, value, category="env", workspace_id}` creates it; `id`, `key`,
   `value`, `category`, `hcl`, `sensitive`, `workspace_id` round-trip into state.
2. `apply` of a varset var `{key, value, category="terraform", variable_set_id}` creates it and
   round-trips `variable_set_id` (with `workspace_id` null).
3. Re-`plan` after apply shows **no drift** (including for a `sensitive` var whose value is not
   re-readable).
4. A `sensitive = true` variable's `value` is never echoed in the read/API response —
   `readable_value` is null/empty and the value only persists as the last-applied plan value.
5. Updating `description`/`hcl`/`value` applies **in place**; changing `category` recreates (ForceNew);
   changing `key` while `sensitive=true` recreates; flipping `sensitive` `true`→`false` recreates;
   changing `workspace_id` or `variable_set_id` recreates.
6. Setting both `workspace_id` and `variable_set_id` (or neither) fails validation (`ExactlyOneOf`);
   setting both `value` and `value_wo` fails validation (`ConflictsWith`).
7. `destroy` removes it; a subsequent read of `:var_id` returns 404.

## Runtime criterion

Not `CRUD-only`. The variable must actually reach a run: a workspace variable (or a varset variable via
an attached set) is injected into the run's Terraform/env config
(`variableService.GetVariablesWithMetaForRun`). Verified indirectly — a run in the owning workspace
sees the `terraform`/`env` variable with the correct precedence.

## Docs + example

- Provider docs page: `docs/resources/variable.md` — arguments (key/value/category/description/hcl/
  sensitive and the mutually-exclusive workspace_id/variable_set_id), computed `id`/`readable_value`,
  and a clear note that `value_wo` is not supported on Stackweaver (use `value` + `sensitive`).
- Example: `examples/resources/stackweaver_variable/resource.tf` — a workspace env var, a terraform
  var, a sensitive var, and a variable-set var.

## Divergences from upstream / TFE

None on the wire; the `vars` shape round-trips unchanged. **One unimplemented feature:** the write-only
`value_wo` / `value_wo_version` path (upstream `resource_tfe_variable.go:195` +
`determineValueForUpdate`) is **not implemented server-side** on Stackweaver — this is a missing
feature, not a wire divergence (stock go-tfe still sends `value` bytes unchanged). Users needing to keep
a secret out of state should use `value` with `sensitive = true`. Source:
`docs/internal/tfe-compatibility/resources/variables/tfe_variable.md:21,99-107`.
