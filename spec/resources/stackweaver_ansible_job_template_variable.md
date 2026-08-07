<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_job_template_variable
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_job_template_variable.go)
---
# stackweaver_ansible_job_template_variable

**Native resource - no TFE equivalent.** A single variable scoped to one Ansible job template
(the AWX/TFE analogue of a workspace variable). Each variable is a key/value pair with a category
(`env` or `terraform`) and an optional `sensitive` flag that write-only-masks the stored value.
Model: `core/models/ansible_job_template_variable.go` (`AnsibleJobTemplateVariable`, ~15). IDs use
the `var-{16-char}` format. Modeled as a child resource keyed on `(job_template_id, key)`.

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleJobTemplateVariables`
service (List/Create/Read/Update/Delete) calling the Stackweaver Ansible API over HTTP.

**Envelope is JSON:API (TFE-compatible), not plain JSON.** The handler
(`backend/internal/api/v2/handlers/ansible/job_template_variables.go`) uses
`{data:{type:"vars", attributes:{key,value,description,category,hcl,sensitive}}}`. The JSON:API
`type` must be exactly `"vars"` (mirrors TFE variables). The response also carries a
`relationships.configurable` link to the owning `job-templates` id and a `links.self`. The parent
job-template id comes from the URL path, not the body.

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string (`var-…`) | Computed | - | - | no | server-assigned, 16-char suffix |
| `job_template_id` | string (uuid) | Required | yes | - | no | owning template; from URL path; `(job_template_id,key)` unique |
| `key` | string | Required | no | - | no | unique within the template |
| `value` | string | Required | no | - | yes when `sensitive` | write-only when `sensitive` (see below) |
| `description` | string | Optional | no | `""` | no | |
| `category` | string | Optional+Computed | no | `env` | no | `env` \| `terraform`; defaults to `env` for Ansible |
| `hcl` | bool | Optional | no | `false` | no | TFE-compat; not used for Ansible execution |
| `sensitive` | bool | Optional | no | `false` | no | when true, server encrypts at rest and masks on read |
| `created_at` / `updated_at` | string (rfc3339) | Computed | - | - | no | present on the model; confirm surfacing at implement time |

The `encrypted` model field is an internal at-rest flag driven by `sensitive`; it is not a
user-facing attribute.

## Wire contract

- **Create:** `POST /ansible/job-templates/:id/vars` - `{data:{type:"vars",
  attributes:{key,value,description?,category?,hcl?,sensitive?}}}`. `key` and `value` are required.
  `category` defaults to `env`. Returns `201`. Duplicate `key` → `409`.
- **Read:** no per-variable GET; read via `GET /ansible/job-templates/:id/vars` (list) and select by
  `id`/`key`. Sensitive values are returned masked as `••••••••`.
- **Update:** `PATCH /ansible/job-templates/:id/vars/:variable_id` - same JSON:API shape with
  optional/pointer attributes. Non-empty `key`/`value`/`description`/`category` overwrite; `hcl` and
  `sensitive` are pointers. Toggling `sensitive` re-encrypts (false→true) or decrypts (true→false)
  the stored value server-side.
- **Delete:** `DELETE /ansible/job-templates/:id/vars/:variable_id` → `204`.
- **JSON:API type:** `vars` (request `type` is validated; a wrong type is a 400). Response
  `relationships.configurable.data.type` is `job-templates`.

## Acceptance criteria (these ARE the test)

1. `apply` of `{job_template_id, key, value}` creates the variable; `id` (matching `^var-`), `key`,
   `category` (defaulting to `env`), `hcl`, `sensitive` round-trip into state.
2. Re-`plan` after apply shows **no drift** - computed `category=env`, `hcl=false`, `sensitive=false`
   settle without a perpetual diff.
3. A non-sensitive `value` round-trips verbatim on read; a `sensitive=true` `value` is **write-only**
   - the read returns the masked sentinel `••••••••`, so the provider must retain the configured
   value in state and not treat the mask as drift.
4. Updating `value`, `description`, `category`, or `hcl` applies in place; changing `job_template_id`
   forces recreate.
5. Flipping `sensitive` false→true then true→false applies in place (server re-encrypts/decrypts) and
   does not corrupt the value.
6. Creating a second variable with a duplicate `key` on the same template fails with `409`.
7. `destroy` removes it; it no longer appears in the template's `vars` list.

## Runtime criterion

The variable's runtime effect is that a job launched from the owning template receives it: `env`
variables are injected into the playbook run environment / extra-vars per its category. Verified:
attach a variable, launch (or dry-run) the template, and confirm the value reaches the run (sensitive
values decrypted only at execution). Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_job_template_variable.md` - document the write-only
  masking of `sensitive` values and the `env`/`terraform` categories.
- Example: `examples/resources/stackweaver_ansible_job_template_variable/resource.tf` - a
  `stackweaver_ansible_job_template` plus a plain and a sensitive variable.

## Divergences from upstream / TFE

Native resource - no TFE equivalent, but deliberately TFE-variable-shaped: JSON:API `type` is
`vars`, and `category`/`hcl`/`sensitive`/`description` mirror `tfe_variable`. `hcl` is carried for
TFE compatibility but has no effect on Ansible execution. There is no single-variable GET endpoint -
reads go through the template's variable list.
