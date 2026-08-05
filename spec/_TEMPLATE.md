<!--
Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec).
SPDX-License-Identifier: MPL-2.0

Per-resource spec template for terraform-provider-stackweaver. Copy to spec/<name>.md and fill every
section. This file IS the contract the automated `implement` pipeline consumes: acceptance criteria
below become the fixture assertions, so they must be concrete and testable. Do not leave a section
blank — write "n/a" with a reason instead.
-->
---
name: stackweaver_<name>
tfe_alias: tfe_<name>
kind: resource            # resource | data-source
family: <family>          # workspaces | projects | teams | tokens | run-tasks | ...
origin: forked            # forked | native
backing_api: implemented  # implemented | partial | blocked  (from the compat matrix)
client_approach: go-tfe-clean   # go-tfe-clean | go-tfe-wrapper | native-client | dropped
status: spec'd            # spec'd | implemented | partial | dropped
upstream_file: internal/provider/resource_tfe_<name>.go   # or n/a for native
go_tfe_type: <Name>       # the go-tfe/v1.go struct, or n/a for native
compat_doc: docs/internal/tfe-compatibility/resources/<family>/tfe_<name>.md
---
# stackweaver_<name>

One-paragraph description: what the resource manages and the Stackweaver concept it maps to.

## Client approach

State `go-tfe-clean` / `go-tfe-wrapper` / `native-client` and WHY. For `go-tfe-wrapper`, describe the
exact wire divergence (which bytes differ from stock go-tfe) that forces the wrapper. For
`native-client`, name the `internal/stackweaver` service method(s) to add.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| ... | | | | | | |

## Wire contract

- **Create:** `<go-tfe method>` → `POST <path>` — request attrs / response attrs.
- **Read:** `<method>` → `GET <path>`.
- **Update:** `<method>` → `PATCH <path>` (or "recreate — ForceNew only").
- **Delete:** `<method>` → `DELETE <path>`.
- **JSON:API type:** `<type>`. Note any field that is write-only (never echoed), null-normalized, or
  divergent from stock go-tfe.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. `apply` creates the resource; these attributes round-trip into state: `...`.
2. Re-`plan` after apply shows **no drift**.
3. `<attribute>` is write-only and never appears in state / read response.
4. Update of `<attribute>` applies in place; update of `<ForceNew attr>` recreates.
5. `destroy` removes it; a subsequent read returns 404.
6. (resource-specific assertions — enumerate every one that matters.)

## Runtime criterion

The behavior behind the resource that must be observed at run time (e.g. "a run using this config
authenticates", "the webhook delivers"). If the resource is config-only with no runtime effect, write
`CRUD-only` and say why.

## Docs + example

- Provider docs page: `docs/resources/<name>.md` — sections/attributes to document.
- Example: `examples/resources/stackweaver_<name>/resource.tf` — the minimal working config.

## Divergences from upstream / TFE

Enumerate anything in/out-of-scope, renamed, dropped, or Stackweaver-extra. "None" is a valid,
explicit answer.
