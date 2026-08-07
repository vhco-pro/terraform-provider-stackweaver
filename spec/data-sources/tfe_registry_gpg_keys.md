<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_registry_gpg_keys
tfe_alias: tfe_registry_gpg_keys
kind: data-source
family: registry
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_registry_gpg_keys.go
go_tfe_type: GPGKey
compat_doc: docs/internal/tfe-compatibility/data-sources/registry/tfe_registry_gpg_keys.md
---
# stackweaver_registry_gpg_keys

Lists all private-registry GPG keys in an organization, exposing each key's id, ASCII armor, and
timestamps. Plural list companion to `stackweaver_registry_gpg_key`.

## Client approach

`go-tfe-clean`. The upstream data source is a plugin-framework data source
(`internal/provider/data_source_registry_gpg_keys.go:47`) that pages
`GPGKeys.ListPrivate(GPGKeyListOptions{Namespaces: [org]})` into a `keys` list. Stackweaver's
`GET /api/registry/private/v2/gpg-keys?filter[namespace]=:org` returns the stock go-tfe `GPGKey` list
shape unchanged, so no wrapper
(`docs/internal/tfe-compatibility/data-sources/registry/tfe_registry_gpg_keys.md`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional+Computed | - | provider default | no | org name (the key namespace); defaults to provider config |
| `id` | string | Computed | - | - | no | set to the organization name |
| `keys` | list(object) | Computed | - | - | no | one object per GPG key (fields below) |

Each `keys[*]` object: `id`, `organization`, `ascii_armor`, `created_at`, `updated_at` (all strings) -
the same fields as `stackweaver_registry_gpg_key`.

## Wire contract

- **Read/lookup:** `GPGKeys.ListPrivate(ctx, GPGKeyListOptions{Namespaces: [org]})` →
  `GET /api/registry/private/v2/gpg-keys?filter[namespace]=:org`, paginated; all pages are accumulated
  into `keys`. No create/update/delete.
- **JSON:API type:** `gpg-keys` (list). `id` is synthesized to the org name (not a resource id). No
  write-only fields; no divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. Fixture applies a `stackweaver_registry_gpg_key`, then a `data.stackweaver_registry_gpg_keys`
   listing the org.
2. The created key's `id` appears in `data...keys[*].id`.
3. `data...id` equals the organization name.
4. Re-`plan` after apply shows **no drift**.

## Runtime criterion

Read-only data source. It enumerates the org's private-registry GPG keys so config can discover keys
without knowing their ids up front. No mutating runtime effect.

## Docs + example

- Provider docs page: `docs/data-sources/registry_gpg_keys.md` - argument (`organization`), computed
  `id` and `keys` list (with the nested object attributes).
- Example: `examples/data-sources/stackweaver_registry_gpg_keys/data-source.tf` - list GPG keys in an
  org.

## Divergences from upstream / TFE

None. Drop-in with `tfe_registry_gpg_keys`.
