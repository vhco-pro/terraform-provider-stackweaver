<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_job_template_credential
tfe_alias: n/a
kind: resource
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native — no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + core/models/ansible_playbook.go)
---
# stackweaver_ansible_job_template_credential

**Native relationship resource — no TFE equivalent.** Attaches one Ansible credential to a job
template's multi-credential set (the AWX "one credential per type, multiple vaults with distinct
vault IDs" rule). There is no dedicated model struct: this manages a row in the
`ansible_job_template_credentials` join table declared on
`AnsibleJobTemplate.Credentials` (`core/models/ansible_playbook.go`, ~144, `many2many`). Modeled as
a pure association: **create = attach, delete = detach, no update** — both fields are ForceNew, so
any change replaces the association.

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleJobTemplateCredentials`
service (Attach/List/Detach) calling the Stackweaver Ansible API over HTTP. Handler:
`backend/internal/api/v2/handlers/ansible/template_credentials.go`.

**Mixed envelope.** The attach **request** is plain JSON — `{"credential_id": "<uuid>"}` — while the
**response** is JSON:API-shaped: `{data:{id, type:"ansible-credentials", attributes:{name,
credential-type, vault-id, username}}}`. The client marshals a plain-JSON request and parses a
JSON:API response (confirm exact keys against `formatTemplateCredential` at implement time).

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `id` | string | Computed | — | — | no | synthetic composite `<job_template_id>/<credential_id>` (no server row id) |
| `job_template_id` | string (uuid) | Required | yes | — | no | the template; from URL path |
| `credential_id` | string (uuid) | Required | yes | — | no | credential to attach; must belong to the template's organization |
| `credential_type` | string | Computed | — | — | no | `credential-type` from the read; the per-type uniqueness key |
| `credential_name` | string | Computed | — | — | no | `name` echoed on attach/list |

There is no updatable attribute — every field is either ForceNew or Computed.

## Wire contract

- **Create (attach):** `POST /ansible/job-templates/:id/credentials` with plain-JSON body
  `{"credential_id":"<uuid>"}`. Returns `201` with the attached credential
  (`type:"ansible-credentials"`, attributes `name`/`credential-type`/`vault-id`/`username`).
  Enforces the AWX rule (at most one credential per type; multiple vaults require distinct vault
  IDs) → conflicting attach returns `409`. A credential outside the template's org returns `400`; a
  missing credential returns `404`.
- **Read:** no per-association GET; `GET /ansible/job-templates/:id/credentials` lists the set, and
  the provider selects the row whose id equals `credential_id`. Absence of the row ⇒ resource gone
  (trigger recreate/removal).
- **Update:** none — both keys are ForceNew; any change detaches + reattaches.
- **Delete (detach):** `DELETE /ansible/job-templates/:id/credentials/:credential_id` → `204`.
- **JSON:API type (response):** `ansible-credentials`. Side effect: attaching/detaching a machine
  (SSH) credential also syncs the template's legacy `credential_id` field server-side (not surfaced
  here).

## Acceptance criteria (these ARE the test)

1. `apply` of `{job_template_id, credential_id}` attaches the credential; the association appears in
   `GET /ansible/job-templates/:id/credentials`, and `credential_type`/`credential_name` populate
   into state as Computed.
2. Re-`plan` after apply shows **no drift** (the composite `id` and computed fields are stable).
3. Attaching a **second** credential of the **same type** to the same template fails with `409`
   (AWX one-per-type rule); two vault credentials with distinct vault IDs both attach.
4. Attaching a credential from a different organization fails with `400`; a nonexistent credential
   fails with `404`.
5. Changing either `job_template_id` or `credential_id` forces recreate (detach old, attach new) —
   there is no in-place update path.
6. `destroy` detaches it (`204`); the credential no longer appears in the template's credential list,
   and the underlying credential resource itself is untouched.

## Runtime criterion

The attachment's runtime effect is that a job launched from the template authenticates using the
attached credential(s): a machine credential provides the SSH login, a vault credential decrypts
vaulted vars, etc. Verified: attach a credential, launch (or dry-run) the template, and confirm the
run consumes it. Config-with-real-effect, not dead CRUD.

## Docs + example

- Provider docs page: `docs/resources/ansible_job_template_credential.md` — document the
  one-credential-per-type rule, the ForceNew-only lifecycle, and the org-scoping constraint.
- Example: `examples/resources/stackweaver_ansible_job_template_credential/resource.tf` — a
  `stackweaver_ansible_job_template` plus an Ansible credential, joined by this resource.

## Divergences from upstream / TFE

Native resource — no TFE equivalent. It is an association-only resource (create/delete, no update),
in the spirit of TFE join resources like `tfe_team_access`, but with no read endpoint for a single
row (membership is inferred from the list). The composite `id` is provider-synthesized because the
join table exposes no standalone row identifier. Attaching/detaching an SSH machine credential has a
server-side side effect (syncing the template's legacy single `credential_id`) that is intentionally
not modeled as an attribute here.
