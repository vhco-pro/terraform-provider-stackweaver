<!-- Copyright (c) HashiCorp, Inc. (upstream schema) / VH & Co BV (Stackweaver spec). SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_aws_oidc_configuration
tfe_alias: tfe_aws_oidc_configuration
kind: resource
family: oidc
origin: forked
backing_api: implemented
client_approach: go-tfe-clean
status: spec'd
upstream_file: internal/provider/resource_tfe_aws_oidc_configuration.go
go_tfe_type: AWSOIDCConfiguration
compat_doc: docs/internal/tfe-compatibility/resources/vcs/tfe_aws_oidc_configuration.md
---
# stackweaver_aws_oidc_configuration

Registers an AWS IAM role ARN as an org-scoped OIDC identity so Terraform runs authenticate to AWS
keyless via `AssumeRoleWithWebIdentity` — no static access key. Maps onto Stackweaver's org-level OIDC
configuration record.

## Client approach

`go-tfe-clean`. The upstream resource (plugin framework, `Schema()` at
`internal/provider/resource_tfe_aws_oidc_configuration.go:60`) drives `go-tfe`'s
`AWSOIDCConfigurations.Create/Read/Update/Delete` verbatim. Stackweaver's `oidc-configurations`
endpoints accept and return the stock `aws-oidc-configurations` JSON:API shape unchanged (compat:
`docs/internal/tfe-compatibility/resources/vcs/tfe_aws_oidc_configuration.md`); no wrapper.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | `aws-oidc-configurations` primary id; `awsoidc-{16}` format; `UseStateForUnknown` |
| `role_arn` | string | Required | no | — | no | AWS ARN of the IAM role assumed via web identity |
| `organization` | string | Optional+Computed | yes | provider default | no | org name; `RequiresReplace` |

## Wire contract

- **Create:** `AWSOIDCConfigurations.Create(org, AWSOIDCConfigurationCreateOptions)` →
  `POST /organizations/:org/oidc-configurations`. Attr sent: `role-arn` (org resolved from the URL,
  echoed back as the `organization` relationship).
- **Read:** `AWSOIDCConfigurations.Read(id)` → `GET /oidc-configurations/:id`.
- **Update:** `AWSOIDCConfigurations.Update(id, AWSOIDCConfigurationUpdateOptions)` →
  `PATCH /oidc-configurations/:id` — `role-arn` (plain string on the update options, not a pointer);
  applies in place.
- **Delete:** `AWSOIDCConfigurations.Delete(id)` → `DELETE /oidc-configurations/:id`.
- **JSON:API type:** `aws-oidc-configurations`. Single kebab-case attr `role-arn`; `organization` is a
  back-reference relationship carrying the org name. No write-only or sensitive fields. Wire is
  identical to stock go-tfe.

## Acceptance criteria (these ARE the test)

1. `apply` of `{organization, role_arn}` creates the config; `id` (`awsoidc-` prefix), `role_arn`,
   `organization` round-trip into state.
2. Re-`plan` after apply shows **no drift**.
3. Updating `role_arn` applies in place — no recreate.
4. Changing `organization` forces recreate (ForceNew).
5. `destroy` removes it; a subsequent `AWSOIDCConfigurations.Read(id)` returns 404.
6. `import` by `id` hydrates `role_arn` + `organization`; a follow-up `plan` shows no drift.

## Runtime criterion

Drives run-time behavior. On a Terraform run in an org that has this config, the runner mints a
workload-identity token (`aud = sts.amazonaws.com`) and the run assumes the role via
`AssumeRoleWithWebIdentity` with no static credentials. **Runtime delta vs Azure:** AWS reads the token
from a **file** — the run gets `AWS_ROLE_ARN` + `AWS_WEB_IDENTITY_TOKEN_FILE` (+ `AWS_ROLE_SESSION_NAME`),
so the runner (platform-hosted) or agent (self-hosted) writes the minted token to a `0600` file rather
than passing it as an env value. This is a runtime-only delta; the wire/CRUD contract is unchanged.
Verified indirectly: a run using the config authenticates to AWS keyless. Not `CRUD-only`.

## Docs + example

- Provider docs page: `docs/resources/aws_oidc_configuration.md` — arguments (organization/role_arn),
  computed `id`, import by id, and a note that a run in the org authenticates to AWS keyless (token read
  from a file at run time).
- Example: `examples/resources/stackweaver_aws_oidc_configuration/resource.tf` — a minimal config with a
  role ARN in a named org.

## Divergences from upstream / TFE

None on the wire — drop-in with `tfe_aws_oidc_configuration`. The only difference from Azure is at run
time: AWS materializes the OIDC token to a file (`AWS_WEB_IDENTITY_TOKEN_FILE`) instead of an env value
(runtime delta, not a wire divergence).
