<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_registry_providers
tfe_alias: tfe_registry_providers
kind: data-source
family: registry
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_registry_providers.go
go_tfe_type: RegistryProvider
compat_doc: docs/internal/tfe-compatibility/data-sources/registry/tfe_registry_providers.md
---
# stackweaver_registry_providers

Lists the public and private providers in an organization's private registry, optionally filtered by
`registry_name` and a fuzzy `search` string. Plural list companion to `stackweaver_registry_provider`.

## Client approach

`go-tfe-clean`. The upstream data source is a plugin-framework data source
(`internal/provider/data_source_registry_providers.go:51`) that pages `RegistryProviders.List(org)`
into a `providers` list. Stackweaver's `GET /organizations/:org/registry-providers` returns the stock
go-tfe `RegistryProvider` list shape unchanged, so no wrapper
(`docs/internal/tfe-compatibility/data-sources/registry/tfe_registry_providers.md`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional+Computed | - | provider default | no | org name; defaults to provider config |
| `registry_name` | string | Optional | - | - | no | filter: `public` or `private`; validated (OneOf) |
| `search` | string | Optional | - | - | no | fuzzy search over provider name + namespace |
| `id` | string | Computed | - | - | no | set to the organization name |
| `providers` | list(object) | Computed | - | - | no | one object per provider (fields below) |

Each `providers[*]` object: `id`, `organization`, `registry_name`, `namespace`, `name`, `created_at`,
`updated_at` (all strings) - the same fields as `stackweaver_registry_provider`.

## Wire contract

- **Read/lookup:** `RegistryProviders.List(ctx, org, RegistryProviderListOptions{RegistryName, Search})`
  → `GET /organizations/:org/registry-providers`, paginated; all pages are accumulated into
  `providers`. No create/update/delete.
- **JSON:API type:** `registry-providers` (list). `id` is synthesized to the org name (not a resource
  id). No write-only fields; no divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. Fixture applies a private `stackweaver_registry_provider`, then a
   `data.stackweaver_registry_providers` listing the org.
2. The created provider's `name` appears in `data...providers[*].name`.
3. `data...id` equals the organization name.
4. Re-`plan` after apply shows **no drift**.

## Runtime criterion

Read-only data source. It enumerates the org's registry providers (optionally filtered) so config can
discover providers without knowing their ids up front. No mutating runtime effect.

## Docs + example

- Provider docs page: `docs/data-sources/registry_providers.md` - arguments (`organization`,
  `registry_name`, `search`), computed `id` and `providers` list (with the nested object attributes).
- Example: `examples/data-sources/stackweaver_registry_providers/data-source.tf` - list private
  providers in an org.

## Divergences from upstream / TFE

None. Drop-in with `tfe_registry_providers`.
