<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_registry_gpg_key
tfe_alias: tfe_registry_gpg_key
kind: resource
family: registry
origin: forked
backing_api: implemented
client_approach: go-tfe-clean   # value-level divergence, documented: source/source-url/trust-signature returned empty/default (computed metadata TFE populates); stock go-tfe decodes the GPGKey unchanged
status: spec'd
upstream_file: internal/provider/resource_tfe_registry_gpg_key.go
go_tfe_type: GPGKey
compat_doc: docs/internal/tfe-compatibility/resources/registry/tfe_registry_gpg_key.md
---
# stackweaver_registry_gpg_key

Registers the **public** half of a GPG key pair used to sign releases of private providers in the
organization's private registry. Maps onto Stackweaver's `gpg_key` model, served at the go-tfe registry
paths. The key's public half is advertised to Terraform at provider install time so it can verify the
publisher's signed `SHA256SUMS`.

## Client approach

`go-tfe-clean` **with a documented value-level divergence**. The upstream resource (plugin framework,
`internal/provider/resource_tfe_registry_gpg_key.go:45`) drives `go-tfe`'s
`GPGKeys.Create("private", ...)/Read/Delete` and the stock `GPGKey` JSON:API shape (`gpg-keys`,
kebab-case). Stackweaver populates every field the resource reads - `id` (= `key-id`), `ascii-armor`,
`namespace` (= org name), `created-at`, `updated-at`. It differs only in *value*: the computed metadata
fields `source`, `source-url`, and `trust-signature` come back **empty/default** (TFE derives these from
its registry catalog; Stackweaver does not populate them, and the resource does not read them). The
struct still decodes unchanged (`source`/`trust-signature` are plain strings, `source-url` a
`*string`), so no wrapper - captured as a note below.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | the **key id** (16-char/64-bit long id), NOT a UUID; the provider addresses by `{namespace}/{key_id}` |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; used as the private-registry `namespace` |
| `ascii_armor` | string | Required | yes | - | no | ASCII-armored public key |
| `created_at` | string | Computed | - | - | no | iso8601 |
| `updated_at` | string | Computed | - | - | no | iso8601 |

Every configurable attribute is `RequiresReplace` - there is no in-place update.

## Wire contract

- **Create:** `GPGKeys.Create("private", GPGKeyCreateOptions)` →
  `POST /api/registry/private/v2/gpg-keys`. Attrs sent: `namespace` (= org name), `ascii-armor`
  (JSON:API `type` = `gpg-keys`). Registry paths are **not** under `/api/v2/`.
- **Read:** `GPGKeys.Read(GPGKeyID{RegistryName:"private", Namespace:org, KeyID:id})` →
  `GET /api/registry/private/v2/gpg-keys/:namespace/:key_id`. `key-id` is extracted from the armored key
  at create.
- **Update:** none - the resource's `Update` returns an error; both configurable attrs are ForceNew.
- **Delete:** `GPGKeys.Delete(GPGKeyID{...})` → `DELETE /api/registry/private/v2/gpg-keys/:namespace/:key_id`.
- **JSON:API type:** `gpg-keys`. **Divergent (value-level, documented):** `source` / `source-url` /
  `trust-signature` are returned empty/default (computed catalog metadata TFE populates); stock go-tfe
  decodes the response unchanged.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, ascii_armor}` with a valid ASCII-armored public key creates the key; `id`
   (= the parsed key id), `organization`, `ascii_armor`, `created_at`, `updated_at` round-trip into state.
2. Re-`plan` after apply shows **no drift** (the read round-trips the write; empty
   `source`/`source-url`/`trust-signature` cause no drift because the resource does not expose them).
3. `id` equals the 16-char long key id parsed from the armored key (not a UUID).
4. Changing `ascii_armor` or `organization` forces recreate (no in-place update path exists).
5. `destroy` removes it; a subsequent `GET /api/registry/private/v2/gpg-keys/:org/:key_id` returns 404.
6. A malformed / non-single-key `ascii_armor` is rejected (structured `ReadArmoredKeyRing` parse, no
   arbitrary-text acceptance).

## Runtime criterion

Backs signature verification at `terraform init`. The registered public key is advertised in the
provider install response's `signing_keys.gpg_public_keys[]`; at publish time the server verifies the
publisher's detached `SHA256SUMS.sig` against **this** key (in-process
`GPGService.VerifyDetachedSignature`, pure-Go via ProtonMail go-crypto - no `gpg` binary), and Terraform
re-verifies the same signature during `init`. Runtime proven by
`backend/internal/services/registry/gpg_test.go` (sign, verify good, reject tampered). Not CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/registry_gpg_key.md` - arguments (organization/ascii_armor),
  computed `id`/`created_at`/`updated_at`, note that `id` is the key id, import by `<org>/<key_id>`.
- Example: `examples/resources/stackweaver_registry_gpg_key/resource.tf` - a key from an inline
  ASCII-armored public key (or `file()`), typically paired with a `stackweaver_registry_provider`.

## Divergences from upstream / TFE

**Value-level (documented), wire-safe:** `source`, `source-url`, and `trust-signature` are returned
**empty/default** - computed catalog metadata TFE populates that Stackweaver does not, and that the
resource does not read. Wire *shape* is identical to go-tfe (fields present, empty), so no client change;
this is a read-back note only. Compat source:
`docs/internal/tfe-compatibility/resources/registry/tfe_registry_gpg_key.md:41,46-47`.
