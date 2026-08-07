<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_registry_provider
tfe_alias: tfe_registry_provider
kind: data-source
family: registry
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_registry_provider.go
go_tfe_type: RegistryProvider
compat_doc: docs/internal/tfe-compatibility/data-sources/registry/tfe_registry_provider.md
---
# stackweaver_registry_provider

Reads a single public or private provider from an organization's private registry by `name`
(+ `registry_name`, `namespace`), exposing its id, namespace, registry name, and timestamps. Read-only
lookup companion to `stackweaver_registry_provider`.

## Client approach

`go-tfe-clean`. The upstream data source is a plugin-framework data source
(`internal/provider/data_source_registry_provider.go:42`) that builds a `tfe.RegistryProviderID` and
calls `RegistryProviders.Read`. Stackweaver's `GET /organizations/:org/registry-providers/:registry/:namespace/:name`
returns the stock go-tfe `RegistryProvider` JSON:API shape unchanged, so no wrapper
(`docs/internal/tfe-compatibility/data-sources/registry/tfe_registry_provider.md`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `name` | string | Required | - | - | no | lookup key; provider name |
| `organization` | string | Optional+Computed | - | provider default | no | org name; defaults to provider config |
| `registry_name` | string | Optional+Computed | - | `private` | no | `public` or `private`; validated (OneOf) |
| `namespace` | string | Optional+Computed | - | org (for private) | no | equals org for private; **required** for public, **forbidden** for private (ValidateConfig) |
| `id` | string | Computed | - | - | no | `registry-providers` JSON:API primary id |
| `created_at` | string | Computed | - | - | no | ISO8601 create time |
| `updated_at` | string | Computed | - | - | no | ISO8601 update time |

## Wire contract

- **Read/lookup:** `RegistryProviders.Read(ctx, RegistryProviderID{org, registry_name, namespace, name}, RegistryProviderReadOptions)`
  → `GET /organizations/:org/registry-providers/:registry_name/:namespace/:name`. For a private provider
  the namespace is forced to the org name; for public it comes from input. No create/update/delete.
- **JSON:API type:** `registry-providers`. `ValidateConfig` enforces: `namespace` required when
  `registry_name = "public"`, and `namespace` forbidden when `registry_name` is `"private"`/null. No
  write-only fields; no divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. Fixture applies a private `stackweaver_registry_provider`, then a
   `data.stackweaver_registry_provider` reading it by `name` (private, so no `namespace`).
2. `data...id` equals the backing resource's `id`.
3. `data...name` equals the resource's `name`.
4. Re-`plan` after apply shows **no drift**.
5. `registry_name`/`namespace` round-trip (private ⇒ `namespace` resolves to the org name).

## Runtime criterion

Read-only data source. It resolves a registry provider by name (+ registry/namespace) to its id and
metadata so other config can reference it without hardcoding the id. No mutating runtime effect.

## Docs + example

- Provider docs page: `docs/data-sources/registry_provider.md` - arguments (`name`, `organization`,
  `registry_name`, `namespace` - with the public-vs-private namespace rule), computed `id`,
  `created_at`, `updated_at`.
- Example: `examples/data-sources/stackweaver_registry_provider/data-source.tf` - read a private
  provider by name.

## Divergences from upstream / TFE

None. Drop-in with `tfe_registry_provider`.
