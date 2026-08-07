<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_azure_oidc_configuration
tfe_alias: tfe_azure_oidc_configuration
kind: resource
family: oidc
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_azure_oidc_configuration.go
go_tfe_type: AzureOIDCConfiguration
compat_doc: docs/internal/tfe-compatibility/resources/vcs/tfe_azure_oidc_configuration.md
---
# stackweaver_azure_oidc_configuration

Registers an Azure Entra ID application (client + subscription + tenant) as an org-scoped OIDC identity
so Terraform runs can authenticate to Azure keyless via workload identity federation - no static
service-principal secret. Maps onto Stackweaver's org-level OIDC configuration record.

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework, `Schema()` at
`internal/provider/resource_tfe_azure_oidc_configuration.go:62`) drives `go-tfe`'s
`AzureOIDCConfigurations.Create/Read/Update/Delete` verbatim. Stackweaver's `oidc-configurations`
endpoints accept and return the stock `azure-oidc-configurations` JSON:API shape unchanged (compat:
`docs/internal/tfe-compatibility/resources/vcs/tfe_azure_oidc_configuration.md`); no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | - | - | no | `azure-oidc-configurations` primary id; `azoidc-{16}` format; `UseStateForUnknown` |
| `client_id` | string | Required | no | - | no | Entra ID application (client) ID |
| `subscription_id` | string | Required | no | - | no | Azure subscription ID |
| `tenant_id` | string | Required | no | - | no | Entra ID tenant (directory) ID |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; `RequiresReplace` |

## Wire contract

- **Create:** `AzureOIDCConfigurations.Create(org, AzureOIDCConfigurationCreateOptions)` →
  `POST /organizations/:org/oidc-configurations`. Attrs sent: `client-id`, `subscription-id`,
  `tenant-id` (org resolved from the URL, echoed back as the `organization` relationship).
- **Read:** `AzureOIDCConfigurations.Read(id)` → `GET /oidc-configurations/:id`.
- **Update:** `AzureOIDCConfigurations.Update(id, AzureOIDCConfigurationUpdateOptions)` →
  `PATCH /oidc-configurations/:id` - `client-id`/`subscription-id`/`tenant-id` sent as `omitempty`
  pointers; all update in place.
- **Delete:** `AzureOIDCConfigurations.Delete(id)` → `DELETE /oidc-configurations/:id`.
- **JSON:API type:** `azure-oidc-configurations`. Kebab-case attrs (`client-id`, `subscription-id`,
  `tenant-id`); `organization` is a back-reference relationship carrying the org name. No write-only or
  sensitive fields; no null-normalization. Wire is identical to stock go-tfe.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, client_id, subscription_id, tenant_id}` creates the config; `id`
   (`azoidc-` prefix), `client_id`, `subscription_id`, `tenant_id`, `organization` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. Updating `client_id` (or `subscription_id`/`tenant_id`) applies in place - no recreate.
4. Changing `organization` forces recreate (ForceNew).
5. `destroy` removes it; a subsequent `AzureOIDCConfigurations.Read(id)` returns 404.
6. `import` by `id` hydrates all attributes; a follow-up `plan` shows no drift.

## Runtime criterion

Drives run-time behavior. On a Terraform run in an org that has this config, the runner mints a
workload-identity token and the run authenticates to Azure via OIDC (env-value token, no token file -
this is the Azure baseline the AWS/GCP file-based variants diverge from). Verified indirectly: a run
using the config authenticates to Azure with no static credentials. Not `CRUD-only`.

## Docs + example

- Provider docs page: `docs/resources/azure_oidc_configuration.md` - arguments
  (organization/client_id/subscription_id/tenant_id), computed `id`, import by id, and a note that a run
  in the org authenticates to Azure keyless.
- Example: `examples/resources/stackweaver_azure_oidc_configuration/resource.tf` - a minimal config with
  the three Entra IDs in a named org.

## Divergences from upstream / TFE

None on the wire - drop-in with `tfe_azure_oidc_configuration`. Azure is the env-value-token baseline;
AWS/GCP add a runtime token-file delta (documented on those specs), Azure does not.
