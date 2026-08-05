<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_gcp_oidc_configuration
tfe_alias: tfe_gcp_oidc_configuration
kind: resource
family: oidc
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_gcp_oidc_configuration.go
go_tfe_type: GCPOIDCConfiguration
compat_doc: docs/internal/tfe-compatibility/resources/vcs/tfe_gcp_oidc_configuration.md
---
# stackweaver_gcp_oidc_configuration

Registers a GCP service account plus its Workload Identity Federation provider as an org-scoped OIDC
identity so Terraform runs authenticate to GCP keyless via WIF — no static service-account key. Maps
onto Stackweaver's org-level OIDC configuration record.

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework, `Schema()` at
`internal/provider/resource_tfe_gcp_oidc_configuration.go:62`) drives `go-tfe`'s
`GCPOIDCConfigurations.Create/Read/Update/Delete` verbatim. Stackweaver's `oidc-configurations`
endpoints accept and return the stock `gcp-oidc-configurations` JSON:API shape unchanged (compat:
`docs/internal/tfe-compatibility/resources/vcs/tfe_gcp_oidc_configuration.md`); no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `gcp-oidc-configurations` primary id; `gcpoidc-{16}` format; `UseStateForUnknown` |
| `service_account_email` | string | Required | no | — | no | GCP service account impersonated via WIF |
| `project_number` | string | Required | no | — | no | GCP project number containing the provider + SA |
| `workload_provider_name` | string | Required | no | — | no | full WIF provider resource path (`projects/{num}/locations/global/workloadIdentityPools/{pool}/providers/{provider}`) |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; `RequiresReplace` |

## Wire contract

- **Create:** `GCPOIDCConfigurations.Create(org, GCPOIDCConfigurationCreateOptions)` →
  `POST /organizations/:org/oidc-configurations`. Attrs sent: `service-account-email`,
  `project-number`, `workload-provider-name` (org resolved from the URL, echoed back as the
  `organization` relationship).
- **Read:** `GCPOIDCConfigurations.Read(id)` → `GET /oidc-configurations/:id`.
- **Update:** `GCPOIDCConfigurations.Update(id, GCPOIDCConfigurationUpdateOptions)` →
  `PATCH /oidc-configurations/:id` — the three attrs sent as `omitempty` pointers; all update in place.
- **Delete:** `GCPOIDCConfigurations.Delete(id)` → `DELETE /oidc-configurations/:id`.
- **JSON:API type:** `gcp-oidc-configurations`. Kebab-case attrs (`service-account-email`,
  `project-number`, `workload-provider-name`); `organization` is a back-reference relationship carrying
  the org name. No write-only or sensitive fields. Wire is identical to stock go-tfe.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, service_account_email, project_number, workload_provider_name}` creates
   the config; `id` (`gcpoidc-` prefix) and all three attrs + `organization` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. Updating any of `service_account_email` / `project_number` / `workload_provider_name` applies in
   place — no recreate.
4. Changing `organization` forces recreate (ForceNew).
5. `destroy` removes it; a subsequent `GCPOIDCConfigurations.Read(id)` returns 404.
6. `import` by `id` hydrates all attributes; a follow-up `plan` shows no drift.

## Runtime criterion

Drives run-time behavior. On a Terraform run in an org that has this config, the runner mints a
workload-identity token (`aud = //iam.googleapis.com/{workload_provider_name}`) and the `google`
provider authenticates via Workload Identity Federation with no static key. **Runtime delta vs Azure:**
like AWS, GCP reads the token from a **file** — but it also needs an external-account credential-config
JSON (referencing the token file plus the STS token-exchange + SA-impersonation URLs) pointed at by
`GOOGLE_APPLICATION_CREDENTIALS`; the runner (platform-hosted) or agent (self-hosted) writes both
`0600` files at run time. This is a runtime-only delta; the wire/CRUD contract is unchanged. Verified
indirectly: a run using the config authenticates to GCP keyless. Not `CRUD-only`.

## Docs + example

- Provider docs page: `docs/resources/gcp_oidc_configuration.md` — arguments (organization/
  service_account_email/project_number/workload_provider_name), computed `id`, import by id, and a note
  that a run in the org authenticates to GCP via WIF (token + credential-config written to files).
- Example: `examples/resources/stackweaver_gcp_oidc_configuration/resource.tf` — a minimal config with
  the SA email, project number, and full WIF provider path in a named org.

## Divergences from upstream / TFE

None on the wire — drop-in with `tfe_gcp_oidc_configuration`. The only difference from Azure is at run
time: GCP materializes the token plus an external-account credential-config JSON to files
(`GOOGLE_APPLICATION_CREDENTIALS`) instead of an env-value token (runtime delta, not a wire divergence).
