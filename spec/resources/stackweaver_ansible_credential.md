<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_credential
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_credential.go)
---
# stackweaver_ansible_credential

**Native resource - no TFE equivalent.** Manages an Ansible credential: a named, typed secret bundle
scoped to an organization (optionally narrowed to a project) that job templates, jobs, and inventory
sources attach for host access, SCM access, vault decryption, or cloud-inventory auth. Supported
types: `ssh`, `scm`, `vault`, `machine-ssh`, `aws`, `azure`, `gcp`, `vmware`. All secret material is
AES-256-GCM encrypted at rest and **never returned by the API** - see the write-only pattern below.
Model: `core/models/ansible_credential.go` (`AnsibleCredential`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleCredentials` service
(List/Create/Read/Update/Delete) calling the Stackweaver Ansible API over HTTP. The endpoint uses a
**JSON:API-shaped envelope** (`{"data":{"type","attributes","relationships"}}`) with **kebab-case**
attribute keys and a `project` relationship - see the handler in
`backend/internal/api/v2/handlers/ansible/credentials.go`. The native client marshals that envelope
(confirm exact keys against the handler at implement time).

## Write-only secret pattern (CRITICAL)

Every secret field on the model is tagged `json:"-"` (`SSHPrivateKey`, `SSHPassphrase`, `Password`,
`VaultPassword`, `BecomePassword`, `AWSAccessKeyID`, `AWSSecretAccessKey`, `AzureClientSecret`,
`GCPServiceAccount`). These are **accepted on write and never echoed on read**. The API instead
surfaces four presence booleans - `has-ssh-private-key`, `has-password`, `has-vault-password`,
`has-become-password` - and nothing for the AWS/Azure/GCP secrets.

Provider consequences:

- Model each secret as an **`Optional`, `Sensitive`, write-only-style** attribute. Terraform state
  stores the value the practitioner wrote (that is the only place it exists after apply); the Read
  never overwrites it from the API, because the API cannot return it.
- **Do not reconcile secret values from Read** - there is no wire value to compare against, so the
  provider must not mark the attribute changed just because the API omits it. Track presence via the
  `has_*` computed booleans only.
- Rotating a secret is a normal in-place update (send the new value); clearing/keeping is
  configuration-driven, since the API gives no way to observe the old value.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (uuid) | Computed | - | - | no | server-assigned |
| `organization` | string | Required | yes | - | no | org name in the create path; `(organization, name)` unique |
| `project_id` | string (uuid) | Optional | no | server default | no | null → org-scoped; omitted on create resolves to the org's `default` project; updatable in place |
| `name` | string | Required | no | - | no | unique within the org |
| `description` | string | Optional | no | `""` | no | |
| `credential_type` | string | Required | yes | - | no | `ssh`\|`scm`\|`vault`\|`machine-ssh`\|`aws`\|`azure`\|`gcp`\|`vmware`; not in the update request → immutable |
| `username` | string | Optional | no | `""` | no | readable |
| `azure_tenant_id` | string | Optional | no | `""` | no | readable (not a secret) |
| `azure_client_id` | string | Optional | no | `""` | no | readable (not a secret) |
| `ssh_port` | int | Optional+Computed | no | `22` | no | |
| `ssh_become_user` | string | Optional+Computed | no | `root` | no | |
| `ssh_private_key` | string | Optional | no | - | **yes** | **write-only** - never read back |
| `ssh_passphrase` | string | Optional | no | - | **yes** | **write-only** |
| `password` | string | Optional | no | - | **yes** | **write-only** |
| `vault_password` | string | Optional | no | - | **yes** | **write-only** |
| `become_password` | string | Optional | no | - | **yes** | **write-only** |
| `aws_access_key_id` | string | Optional | no | - | **yes** | **write-only** (no `has_*` echo) |
| `aws_secret_access_key` | string | Optional | no | - | **yes** | **write-only** (no `has_*` echo) |
| `azure_client_secret` | string | Optional | no | - | **yes** | **write-only** (no `has_*` echo) |
| `gcp_service_account` | string | Optional | no | - | **yes** | **write-only** (no `has_*` echo) |
| `has_ssh_private_key` | bool | Computed | - | - | no | presence readback |
| `has_password` | bool | Computed | - | - | no | presence readback |
| `has_vault_password` | bool | Computed | - | - | no | presence readback |
| `has_become_password` | bool | Computed | - | - | no | presence readback |

## Wire contract

- **Create:** `POST /organizations/:name/ansible/credentials` - attributes: `name`, `description?`,
  `credential-type`, `username?`, the nine write-only secret attributes (`ssh-private-key`,
  `ssh-passphrase`, `password`, `vault-password`, `become-password`, `aws-access-key-id`,
  `aws-secret-access-key`, `azure-client-secret`, `gcp-service-account`), `azure-tenant-id?`,
  `azure-client-id?`, `ssh-port?`, `ssh-become-user?`; optional `relationships.project.data.id`. →
  `201`, `type: ansible-credentials`.
- **Read:** `GET /ansible/credentials/:id` - returns readable attrs + `has_*` booleans only; **no
  secret values**.
- **Update:** `PATCH /ansible/credentials/:id` - name/description/username/secrets/ssh-options and
  `project` relationship in place. `credential-type` is **not** accepted by the update handler → ForceNew.
- **Delete:** `DELETE /ansible/credentials/:id` → `204`; returns **`409`** if the credential is still
  referenced by a job template, job, or inventory source.
- **JSON:API type:** `ansible-credentials`. Response `relationships` always includes `organization`;
  `project` is set on the request, not echoed in the response relationships. Secret attributes are
  write-only (accepted, never returned). `vault_id` exists on the model but is **not** wired into the
  create/update request structs (see divergences).

## Acceptance criteria (these ARE the test)

Concrete, testable. The `implement` pipeline generates the fixture assertions from these.

1. `apply` of `{organization, name, credential_type = "ssh", username, ssh_private_key}` creates the
   credential; `id`, `name`, `credential_type`, `username` round-trip into state.
2. Re-`plan` after apply shows **no drift** - the write-only secret and the computed `has_*`/`ssh_port`/
   `ssh_become_user` must not produce a perpetual diff.
3. `ssh_private_key` (and every other secret attribute) is **write-only**: it never appears in the Read
   response, and `has_ssh_private_key` reads `true` after it was set.
4. Rotating `ssh_private_key` (new value) applies in place; changing `credential_type` or
   `organization` forces recreate.
5. `project_id` can be set/changed in place without recreate.
6. `destroy` removes it; a subsequent `GET /ansible/credentials/:id` returns 404; attempting to
   `destroy` a credential still attached to a job template / inventory source surfaces the `409`.

## Runtime criterion

The credential has real runtime effect: it is decrypted and injected when a job template / job / a
matching inventory source runs (SSH key for host access, vault password for `--vault-id`, AWS/Azure/GCP
secret for a dynamic-inventory sync). Verified: create an `ssh` credential, attach it to a job template
(or a `custom`/cloud inventory source), launch/dry-run, and assert the run authenticates with the
supplied secret. Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_credential.md` - document the type matrix, which secret
  attribute applies to which `credential_type`, and the write-only behavior (values are never read
  back; use `has_*` to confirm presence).
- Example: `examples/resources/stackweaver_ansible_credential/resource.tf` - a minimal `ssh`
  credential feeding `var.ssh_private_key`, plus a commented `aws` variant.

## Divergences from upstream / TFE

Native resource - no TFE equivalent. **This resource is the intended home for the parked
`tfe_ssh_key`**: a thin TFE-compatible face over the `ssh` credential type can be layered on later
(later scope, not part of this spec). Known gaps in the current API to carry into implementation:
`vault_id` is present on the model but **not** mapped in the create/update request structs (so it is
not settable via the API today - flag before promising it in schema), and the AWS/Azure/GCP/`azure-
client-secret`/`gcp-service-account` secrets have **no `has_*` presence echo**, so their presence
cannot be observed from Read (rely on state only).
