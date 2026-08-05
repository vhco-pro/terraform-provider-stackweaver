<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_vault_oidc_configuration
tfe_alias: tfe_vault_oidc_configuration
kind: resource
family: oidc
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_vault_oidc_configuration.go
go_tfe_type: VaultOIDCConfiguration
compat_doc: docs/internal/tfe-compatibility/resources/vcs/tfe_vault_oidc_configuration.md
---
# stackweaver_vault_oidc_configuration

Registers a HashiCorp Vault instance + JWT-auth role as an org-scoped OIDC identity so Terraform runs
log in to Vault via JWT auth and receive a `VAULT_TOKEN` — no static Vault token. Maps onto
Stackweaver's org-level OIDC configuration record.

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework, `Schema()` at
`internal/provider/resource_tfe_vault_oidc_configuration.go:65`) drives `go-tfe`'s
`VaultOIDCConfigurations.Create/Read/Update/Delete` verbatim. Stackweaver's `oidc-configurations`
endpoints accept and return the stock `vault-oidc-configurations` JSON:API shape unchanged (compat:
`docs/internal/tfe-compatibility/resources/vcs/tfe_vault_oidc_configuration.md`); no wrapper. Note the
provider maps HCL `role_name` to the wire attr `role` — this mapping is done by go-tfe itself
(`RoleName` field, jsonapi tag `role`), so it is **not** a Stackweaver divergence.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `vault-oidc-configurations` primary id; `vaultoidc-{16}` format; `UseStateForUnknown` |
| `address` | string | Required | no | — | no | full Vault server URL |
| `role_name` | string | Required | no | — | no | Vault JWT-auth role name; wire attr is `role` (go-tfe mapping) |
| `namespace` | string | Required | no | — | no | JWT auth mount namespace; provider-side **Required** (go-tfe options treat it as optional) |
| `auth_path` | string | Optional+Computed | no | `"jwt"` | no | JWT auth mount path; `stringdefault.StaticString("jwt")` |
| `encoded_cacert` | string | Optional+Computed | no | `""` | no | base64/PEM CA cert for Vault TLS; `stringdefault.StaticString("")` |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; `RequiresReplace` |

## Wire contract

- **Create:** `VaultOIDCConfigurations.Create(org, VaultOIDCConfigurationCreateOptions)` →
  `POST /organizations/:org/oidc-configurations`. Attrs sent: `address`, `role` (from `role_name`),
  `namespace`, `auth-path`, `encoded-cacert` (org resolved from the URL, echoed back as the
  `organization` relationship).
- **Read:** `VaultOIDCConfigurations.Read(id)` → `GET /oidc-configurations/:id`.
- **Update:** `VaultOIDCConfigurations.Update(id, VaultOIDCConfigurationUpdateOptions)` →
  `PATCH /oidc-configurations/:id` — `address`/`role`/`namespace`/`auth-path`/`encoded-cacert` sent as
  `omitempty` pointers; all update in place.
- **Delete:** `VaultOIDCConfigurations.Delete(id)` → `DELETE /oidc-configurations/:id`.
- **JSON:API type:** `vault-oidc-configurations`. Kebab-case attrs; note the wire field is **`role`**
  (HCL `role_name`) — matches go-tfe's `RoleName jsonapi:"attr,role"`. `organization` is a
  back-reference relationship carrying the org name. No write-only or sensitive fields. Wire is
  identical to stock go-tfe.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, address, role_name, namespace}` creates the config; `id` (`vaultoidc-`
   prefix), `address`, `role_name`, `namespace`, `organization` round-trip into state.
2. Re-`plan` after apply shows **no drift**, including the defaults: `auth_path` settles to `"jwt"` and
   `encoded_cacert` to `""` when omitted, with no perpetual diff.
3. `role_name` round-trips as `role_name` in state even though the wire attribute is `role` (the go-tfe
   mapping is transparent).
4. Updating `address` / `role_name` / `namespace` / `auth_path` / `encoded_cacert` applies in place —
   no recreate.
5. Changing `organization` forces recreate (ForceNew).
6. `destroy` removes it; a subsequent `VaultOIDCConfigurations.Read(id)` returns 404.
7. `import` by `id` hydrates all attributes; a follow-up `plan` shows no drift.

## Runtime criterion

Drives run-time behavior, and differently from the cloud providers: Vault requires an **active login**,
not a provider-side token exchange. On a Terraform run in an org that has this config, the runner (1)
mints a workload-identity token with `aud = vault.workload.identity`, (2) calls
`POST {address}/v1/auth/{auth_path}/login` with `{"role": role_name, "jwt": token}` (+ the
`X-Vault-Namespace` header and custom CA when set), and (3) exports `VAULT_ADDR` + `VAULT_TOKEN`
(+ `VAULT_NAMESPACE`/`VAULT_CACERT`) so the run's `vault` provider authenticates with no auth block.
Platform-hosted runner performs the login directly; the self-hosted agent performs it (only it can reach
the customer's Vault). Verified indirectly: a run using the config obtains a `VAULT_TOKEN`. Not
`CRUD-only`.

## Docs + example

- Provider docs page: `docs/resources/vault_oidc_configuration.md` — arguments (organization/address/
  role_name/namespace/auth_path/encoded_cacert), computed `id`, import by id, and notes: (a) `namespace`
  is provider-required even on Vault OSS (any value; only sent as the `X-Vault-Namespace` header, which
  OSS ignores); (b) at run time the Vault JWT role's `bound_audiences` must include
  `vault.workload.identity`.
- Example: `examples/resources/stackweaver_vault_oidc_configuration/resource.tf` — a minimal config with
  address, role_name, namespace, and auth_path in a named org.

## Divergences from upstream / TFE

None on the wire — drop-in with `tfe_vault_oidc_configuration`. Three notes that are **not** wire
divergences:
- **`role_name` → `role`:** the HCL argument is `role_name`; the wire attr is `role`. This mapping is
  built into go-tfe (`RoleName jsonapi:"attr,role"`) and matches TFE exactly — no user action, no
  Stackweaver-side change.
- **`namespace` optionality:** the provider marks `namespace` **Required** even though go-tfe's create
  options treat it as optional. Provider-side behavior, identical to upstream `tfe`.
- **Audience (runtime delta):** the resource has no audience attribute, so Stackweaver mints the fixed
  `aud = vault.workload.identity` (matching Terraform Cloud's default) at run time; the operator sets
  the Vault JWT role's `bound_audiences` to match. Runtime-only, not a wire divergence.
