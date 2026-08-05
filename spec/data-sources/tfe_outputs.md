<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_outputs
tfe_alias: tfe_outputs
kind: data-source
family: state
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_outputs.go
go_tfe_type: Workspace
compat_doc: n/a
---
# stackweaver_outputs

Reads a workspace's current state outputs and exposes them as dynamic values — all outputs under
`values` (sensitive) and just the non-sensitive ones under `nonsensitive_values`. Maps onto
Stackweaver's workspace state-output concept. Read-only: given an organization + workspace name it
resolves the latest state version's outputs.

## Client approach

`go-tfe-clean`. Plugin-framework data source. Reads the workspace with the outputs relation included
via `Workspaces.ReadWithOptions` (`include=outputs`); for each output flagged sensitive it makes a
follow-up `StateVersionOutputs.Read` call to fetch the redacted value. Consumes the stock go-tfe
`Workspace` / `StateVersionOutput` JSON:API shapes unchanged; no wrapper. No compatibility detail doc
exists yet (`docs/internal/tfe-compatibility/data-sources/state/tfe_outputs.md` is absent) — this spec
is the source of record.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `workspace` | string | Required | — | — | no | workspace name whose outputs to read |
| `organization` | string | Optional | — | provider default | no | org name; falls back to the provider's default org |
| `id` | string | Computed | — | — | no | synthetic id `<organization>-<workspace>` |
| `values` | dynamic | Computed | — | — | **yes** | all outputs (name → typed value), including sensitive |
| `nonsensitive_values` | dynamic | Computed | — | — | no | only the non-sensitive outputs |

## Wire contract

- **Read (lookup):** `Workspaces.ReadWithOptions(org, workspace, WorkspaceReadOptions{Include:
  [outputs]})` → `GET /organizations/:org/workspaces/:name?include=outputs`. For every output with
  `Sensitive == true`, an extra `StateVersionOutputs.Read(op.ID)` → `GET /state-version-outputs/:id`
  fetches the real value. Output values are dynamically typed (bool/number/string/list/tuple/object/map
  inferred from the raw JSON). Sets the synthetic `id` to `<org>-<workspace>`, builds `values` from all
  outputs, and `nonsensitive_values` from the non-sensitive subset.
- No create/update/delete — data source.
- **JSON:API type:** `workspaces` (with included `state-version-outputs`). No divergent fields; the
  dynamic typing is client-side inference over the output values.

## Acceptance criteria (these ARE the test)

1. Fixture provisions a workspace with a completed run producing at least one non-sensitive and one
   sensitive output (e.g. via a backing config with `output` blocks), then this data source reads it by
   `organization` + `workspace`; `apply` succeeds.
2. Computed `id` is set and equals `<organization>-<workspace>`.
3. `values` contains every output keyed by name with the correct inferred type and value, including the
   resolved sensitive output.
4. `nonsensitive_values` contains only the non-sensitive outputs (the sensitive key is absent).
5. `values` is marked sensitive in state (the whole object), so assertions read individual keys rather
   than dumping the object.
6. **Plan-null quirk:** `organization` is Optional-not-Computed, so assert the clearly-Computed outputs
   (`id`, `values`, `nonsensitive_values`) rather than the input arg.

## Runtime criterion

Read-only data source. It resolves a workspace's latest state outputs (with sensitive values fetched
via a second call) so a downstream config can consume another workspace's outputs — the cross-workspace
remote-state pattern. The runtime behavior it depends on is that a prior run has written state outputs
for the workspace.

## Docs + example

- Provider docs page: `docs/data-sources/outputs.md` — arguments (`workspace`, `organization`),
  computed attributes (`id`, `values` (sensitive), `nonsensitive_values`), with a note on the
  dynamic typing of output values.
- Example: `examples/data-sources/stackweaver_outputs/data-source.tf` — read another workspace's
  outputs and reference `data.stackweaver_outputs.x.nonsensitive_values["<name>"]`.

## Divergences from upstream / TFE

None. Drop-in with `tfe_outputs` (reads a workspace's state outputs).
