<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_registry_gpg_key
tfe_alias: tfe_registry_gpg_key
kind: data-source
family: registry
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/data_source_registry_gpg_key.go
go_tfe_type: GPGKey
compat_doc: docs/internal/tfe-compatibility/data-sources/registry/tfe_registry_gpg_key.md
---
# stackweaver_registry_gpg_key

Reads a single private-registry GPG key by `id` within an organization, exposing its ASCII-armored key
material and timestamps. Read-only lookup companion to `stackweaver_registry_gpg_key`.

## Client approach

`go-tfe-clean`. The upstream data source is a plugin-framework data source
(`internal/provider/data_source_registry_gpg_key.go:38`) that builds a `tfe.GPGKeyID`
(`registry_name = "private"`, namespace = org, `key_id = id`) and calls `GPGKeys.Read`. Stackweaver's
`GET /api/registry/private/v2/gpg-keys/:namespace/:key_id` returns the stock go-tfe `GPGKey` JSON:API
shape unchanged, so no wrapper
(`docs/internal/tfe-compatibility/data-sources/registry/tfe_registry_gpg_key.md`).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Required | - | - | no | lookup key; the GPG key id |
| `organization` | string | Optional+Computed | - | provider default | no | org name (the key namespace); defaults to provider config |
| `ascii_armor` | string | Computed | - | - | no | ASCII-armored representation of the key |
| `created_at` | string | Computed | - | - | no | ISO8601 create time |
| `updated_at` | string | Computed | - | - | no | ISO8601 update time |

## Wire contract

- **Read/lookup:** `GPGKeys.Read(ctx, GPGKeyID{RegistryName: "private", Namespace: org, KeyID: id})` →
  `GET /api/registry/private/v2/gpg-keys/:namespace/:key_id`. Registry is always `private`. No
  create/update/delete.
- **JSON:API type:** `gpg-keys`. No write-only fields; no divergence from stock go-tfe.

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. Fixture applies a `stackweaver_registry_gpg_key`, then a `data.stackweaver_registry_gpg_key` reading
   it by `id` (+ `organization`).
2. `data...id` equals the backing resource's `id`.
3. `data...ascii_armor` is non-empty (round-trips the applied key material).
4. Re-`plan` after apply shows **no drift**.

## Runtime criterion

Read-only data source. It resolves a private-registry GPG key by id to its ASCII armor and timestamps
so other config can reference the key without hardcoding its material. No mutating runtime effect.

## Docs + example

- Provider docs page: `docs/data-sources/registry_gpg_key.md` - arguments (`id`, `organization`),
  computed `ascii_armor`, `created_at`, `updated_at`.
- Example: `examples/data-sources/stackweaver_registry_gpg_key/data-source.tf` - read a GPG key by id.

## Divergences from upstream / TFE

None. Drop-in with `tfe_registry_gpg_key`.
