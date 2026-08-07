<!-- Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0 -->
---
name: stackweaver_ansible_adhoc_modules
tfe_alias: n/a
kind: data-source
family: ansible
origin: native
backing_api: implemented
client_approach: native-client
status: spec'd
upstream_file: n/a (native - no terraform-provider-tfe equivalent)
go_tfe_type: n/a
compat_doc: n/a (native surface; source of record is this spec + backend/internal/api/v2/handlers/ansible/adhoc.go)
---
# stackweaver_ansible_adhoc_modules

**Native data source - no TFE equivalent.** A read-only helper that returns the **effective** ad hoc
module allowlist for an organization: the modules an AWX-style "Run Command" (ad hoc) execution is
permitted to use. The effective list is the org's configured comma-separated allowlist, or the built-in
AWX default when unset. Backed by an organization attribute (`AnsibleAdHocModules` on
`core/models/organization.go`), not a standalone model; resolved by
`core/services/ansible` `ResolveAdHocModules`. Handler:
`backend/internal/api/v2/handlers/ansible/adhoc.go` (`ListModules`).

## Client approach

`native-client`. Not in `go-tfe`; served by a new `internal/stackweaver` `AnsibleAdHocModules` service
with `List(ctx, org)` → `GET /organizations/:org/ansible/adhoc-modules`. The endpoint returns a
JSON:API-shaped single object (`{"data":{"type":"adhoc-modules","id":<org uuid>,"attributes":
{"modules":[...]}}}`); the native client marshals accordingly. Read-only - the allowlist is an org
setting surfaced here for observability; no create/update/delete. (The org attribute itself is managed
elsewhere; this data source only reads the effective resolution.)

## Schema

| Attribute | Type | Req/Opt/Computed | ForceNew | Default | Sensitive | Notes |
|-----------|------|------------------|----------|---------|-----------|-------|
| `organization` | string | Optional+Computed | - | provider default | no | org name; falls back to the provider default |
| `id` | string (uuid) | Computed | - | - | no | the organization's id (server-returned) |
| `modules` | list(string) | Computed | - | - | no | the effective ad hoc module allowlist |

## Wire contract

- **Read/lookup:** `AnsibleAdHocModules.List(ctx, org)` →
  `GET /organizations/:org/ansible/adhoc-modules`.
- **Create/Update/Delete:** n/a - read-only data source.
- **JSON:API type:** `adhoc-modules`; `id` is the org uuid; single attribute `modules` (array of
  strings). The effective list is `ResolveAdHocModules(org.AnsibleAdHocModules)` - identical to the
  allowlist enforced at ad hoc launch time, so the data source and the enforcement stay in lockstep.
- **Auth:** requires read-Ansible on the org; unknown org → 404, no permission → 403.

## Acceptance criteria (these ARE the test)

Assert against known dev-stack state: an org that has **not** overridden the allowlist, so the built-in
default applies.

1. Reading `data.stackweaver_ansible_adhoc_modules` for the fixture org returns a non-empty `modules`
   list.
2. The computed `id` equals the organization's id.
3. With no org override configured, `modules` contains the default AWX modules - including at least
   `command`, `shell`, `ping`, and `setup` (the default is the constant
   `DefaultAdHocModules` in `core/services/ansible`).
4. Re-`plan` after apply shows **no drift** (`modules`, `id` are Computed-only).

## Runtime criterion

Read-only observability helper. Reports the effective ad hoc allowlist without side effect. The
allowlist it reports is exactly what the backend enforces on `run-command`, so it can be used to
validate that a module is permitted before wiring up an ad hoc execution.

## Docs + example

- Provider docs page: `docs/data-sources/ansible_adhoc_modules.md` - argument `organization`; computed
  `modules` and `id`.
- Example: `examples/data-sources/stackweaver_ansible_adhoc_modules/data-source.tf` - read an org's
  effective ad hoc module allowlist and output it.

## Divergences from upstream / TFE

Native data source - no TFE equivalent. Not backed by its own model - it reads an organization
attribute (`AnsibleAdHocModules`) resolved to an effective list. Read-only: managing the allowlist is
out of scope for this data source.
