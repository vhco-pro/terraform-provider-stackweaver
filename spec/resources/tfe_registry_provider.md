<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_registry_provider
tfe_alias: tfe_registry_provider
kind: resource
family: registry
origin: forked
backing_api: implemented
client_approach: go-tfe-clean   # value-level divergence, documented: response omits registry-provider-versions/tag-bindings relations; v1 install returns package-metadata JSON, not a 302 (stock go-tfe parses the RegistryProvider unchanged)
status: spec'd
upstream_file: internal/provider/resource_tfe_registry_provider.go
go_tfe_type: RegistryProvider
compat_doc: docs/internal/tfe-compatibility/resources/registry/tfe_registry_provider.md
---
# stackweaver_registry_provider

Registers a public or private provider shell in the organization's private registry. Maps onto
Stackweaver's `provider` model reshaped onto the go-tfe `registry-providers` surface (composite
`:registry_name/:namespace/:name` addressing). The shell is the addressable target that provider
versions/platforms are published under and that `terraform init` installs from.

## Client approach

`go-tfe-clean` **with a documented value-level divergence**. The upstream resource (plugin framework,
`internal/provider/resource_tfe_registry_provider.go:76`) drives `go-tfe`'s
`RegistryProviders.Create/Read/Delete` and the stock `RegistryProvider` JSON:API shape
(`registry-providers`, kebab-case). Stackweaver returns that shape verbatim for every field the
resource reads (`id`, `name`, `namespace`, `registry-name`, `created-at`, `updated-at`,
`permissions.can-delete`, the `organization` relation). Two things differ only in *value/behaviour*,
not in bytes go-tfe must parse: the response **omits** the `registry-provider-versions` and
`tag-bindings` relations (both `omitempty`/optional in the struct, so decoding is unaffected — the
resource never reads them), and the `/v1` install endpoint returns **package-metadata JSON** instead of
a 302 redirect (the install protocol, not this CRUD resource). No wrapper; captured as notes below.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `registry-providers` primary id (opaque UUID); re-read by composite address so it round-trips without drift |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; resolved from provider config when omitted |
| `registry_name` | string | Optional+Computed | yes | `"private"` | no | `private` or `public` (validated `OneOf`) |
| `namespace` | string | Optional+Computed | yes | — | no | private → forced to the org name (must be omitted in config); public → the upstream namespace (required) |
| `name` | string | Required | yes | — | no | provider name |
| `created_at` | string | Computed | — | — | no | iso8601 |
| `updated_at` | string | Computed | — | — | no | iso8601 |

Every configurable attribute is `RequiresReplace` — there is no in-place update (see Wire contract).

## Wire contract

- **Create:** `RegistryProviders.Create(org, RegistryProviderCreateOptions)` →
  `POST /organizations/:org/registry-providers`. Attrs sent: `name`, `namespace`, `registry-name`
  (JSON:API `type` = `registry-providers`). For `private`, the resource forces `namespace` = org name.
- **Read:** `RegistryProviders.Read(RegistryProviderID{org, registry_name, namespace, name}, opts)` →
  `GET /organizations/:org/registry-providers/:registry_name/:namespace/:name` (go-tfe also supports the
  by-id form `GET /registry-providers/:id`).
- **Update:** none — the resource's `Update` returns an error; all attributes are ForceNew, so any change
  recreates.
- **Delete:** `RegistryProviders.Delete(RegistryProviderID{...})` →
  `DELETE /organizations/:org/registry-providers/:registry_name/:namespace/:name`. Cascades to versions,
  platforms and download records and GC's the stored artifacts.
- **JSON:API type:** `registry-providers`. **Divergent (value-level, documented):** the response omits
  the `registry-provider-versions` and `tag-bindings` relations; the `/v1` install endpoint returns
  package-metadata JSON instead of a 302 redirect. Both are parsed/tolerated by stock go-tfe (the omitted
  relations are optional; the install path is outside this resource).

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, name, registry_name = "private"}` creates the provider; `id`, `name`,
   `registry_name`, `namespace` (= org name), `created_at`, `updated_at` round-trip into state.
2. Re-`plan` after apply shows **no drift** (the opaque `id` and computed `namespace` are stable).
3. A `public` provider requires `namespace` and applies with `registry_name = "public"`, `namespace`
   round-tripping to the configured upstream namespace.
4. Changing `name`, `namespace`, `registry_name`, or `organization` forces recreate (no in-place update
   path exists); an attempted in-place update errors.
5. `destroy` removes it; a subsequent `RegistryProviders.Read(...)` returns 404, and any published
   version artifacts are garbage-collected from object storage.
6. The read response omits `registry-provider-versions`/`tag-bindings` relations yet the stock provider
   decodes it without error and reports no drift (divergence is response-shape-safe).

## Runtime criterion

Backs `terraform init`. After a version + platform is published under the shell (publisher signs
`SHA256SUMS` offline with a `stackweaver_registry_gpg_key`, uploads binary + `SHA256SUMS` +
`SHA256SUMS.sig`), `terraform init` against the registry downloads the zip, checks its shasum against
the signed `SHA256SUMS`, and verifies the signature against the advertised public key. The install
endpoint returns package-metadata JSON (`protocols`, `download_url`, `shasums_url`,
`shasums_signature_url`, `shasum`, `signing_keys.gpg_public_keys[]`). Not CRUD-only.

## Docs + example

- Provider docs page: `docs/resources/registry_provider.md` — arguments
  (organization/name/registry_name/namespace), computed `id`/`created_at`/`updated_at`, the
  private-vs-public namespace rule, import by `<org>/<registry_name>/<namespace>/<name>`.
- Example: `examples/resources/stackweaver_registry_provider/resource.tf` — a minimal private provider
  (`registry_name = "private"`, `name`) plus a commented public example.

## Divergences from upstream / TFE

**Value-level (documented), wire-safe:**
- The read/create response **omits the `registry-provider-versions` and `tag-bindings` relations**
  (optional in the go-tfe struct; the resource never reads them; versions are published out-of-band).
- The **v1 install protocol** endpoint returns **package-metadata JSON** rather than a 302 redirect —
  a corrected signing/install model (publisher-signed `SHA256SUMS`), not a CRUD change; the resource is
  unaffected.
- **Agent-pool request forwarding** for providers is not modelled (no attribute exists upstream).

Wire *shape* for the resource's own CRUD is identical to go-tfe, so no client change. Compat source:
`docs/internal/tfe-compatibility/resources/registry/tfe_registry_provider.md:45,78-92`.
